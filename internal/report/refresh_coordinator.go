package report

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultReportRefreshDebounce is the debounce window between SaleCreated-driven
// materialized view refreshes when no REPORT_REFRESH_DEBOUNCE is configured.
const DefaultReportRefreshDebounce = 30 * time.Second

// retryBackoffCapMultiplier bounds the exponential retry backoff after
// consecutive refresh failures (debounce * 2^n, capped at this multiple).
const retryBackoffCapMultiplier = 8

// RefreshFunc executes a single full refresh of the reporting materialized views.
type RefreshFunc func(ctx context.Context) error

// RefreshMetrics exposes atomic counters describing coordinator behaviour, used
// for observability and unit tests.
type RefreshMetrics struct {
	Scheduled     atomic.Int64 // refresh runs armed by the debounce worker
	Started       atomic.Int64 // refresh executions started
	Succeeded     atomic.Int64 // refresh executions that returned nil
	Failed        atomic.Int64 // refresh executions that returned an error
	Coalesced     atomic.Int64 // MarkDirty calls folded into an already-pending refresh
	Concurrent    atomic.Int64 // refresh executions currently in flight
	MaxConcurrent atomic.Int64 // peak concurrent refresh executions (invariant: <= 1)
}

// RefreshCoordinator coalesces SaleCreated-driven materialized view refreshes.
//
// A single background worker is the only caller of RefreshFunc, so at most one
// refresh can execute at any time (single-flight). SaleCreated listeners only
// call MarkDirty, which arms a debounce timer; many events within the window
// coalesce into one refresh, and events arriving while a refresh runs simply
// keep the store dirty so a follow-up refresh is scheduled afterwards.
//
// State machine:
//
//	CLEAN ── MarkDirty() ──▶ DIRTY_WAITING ── debounce elapsed ──▶ REFRESHING
//	REFRESHING ── success + no new marks ──▶ CLEAN
//	REFRESHING ── success + new marks ──▶ DIRTY_WAITING (fresh debounce)
//	REFRESHING ── failure ──▶ DIRTY_WAITING (exponential backoff retry)
//
// Refresh failures are owned by the coordinator and never propagated back to the
// SaleCreated event lifecycle or the event bus.
type RefreshCoordinator struct {
	mu                    sync.Mutex
	started               bool
	closed                bool
	cancel                context.CancelFunc
	done                  chan struct{}
	dirty                 bool
	timerArmed            bool
	markCount             int64
	refreshStartMarkCount int64
	failures              int
	debounce              time.Duration
	refresh               RefreshFunc
	signal                chan struct{}
	metrics               RefreshMetrics
}

// NewRefreshCoordinator creates a coordinator that refreshes reporting data via
// refresh at most once per debounce window. A non-positive debounce falls back
// to DefaultReportRefreshDebounce.
func NewRefreshCoordinator(debounce time.Duration, refresh RefreshFunc) *RefreshCoordinator {
	if debounce <= 0 {
		debounce = DefaultReportRefreshDebounce
	}
	if refresh == nil {
		refresh = func(ctx context.Context) error { return nil }
	}
	return &RefreshCoordinator{
		done:     make(chan struct{}),
		debounce: debounce,
		refresh:  refresh,
		signal:   make(chan struct{}, 1),
	}
}

// MarkDirty records that the reporting data may be stale. It never blocks, never
// triggers a refresh directly, and never fails; the background worker schedules
// the actual refresh. This is the only entry point SaleCreated consumers touch.
func (c *RefreshCoordinator) MarkDirty() {
	c.mu.Lock()
	c.markCount++
	if c.dirty {
		c.metrics.Coalesced.Add(1)
		coalesced := c.metrics.Coalesced.Load()
		c.mu.Unlock()
		slog.Debug("report refresh coalesced", "total_marks", c.markCount, "coalesced", coalesced)
	} else {
		c.dirty = true
		c.mu.Unlock()
	}
	select {
	case c.signal <- struct{}{}:
	default:
	}
}

// Start launches the single background refresh worker. It is idempotent.
func (c *RefreshCoordinator) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return
	}
	c.started = true
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.run(ctx)
}

// Shutdown cancels the worker and waits for it to exit, so no goroutine or
// debounce timer leaks across the process lifecycle. It is idempotent.
func (c *RefreshCoordinator) Shutdown() {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return
	}
	c.closed = true
	cancel := c.cancel
	c.mu.Unlock()
	cancel()
	<-c.done
}

// IsDirty reports whether reporting data is currently considered stale.
func (c *RefreshCoordinator) IsDirty() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dirty
}

// Metrics exposes the coordinator counters for observability and tests.
func (c *RefreshCoordinator) Metrics() *RefreshMetrics {
	return &c.metrics
}

// run is the single worker goroutine. It is the only place that invokes
// RefreshFunc, guaranteeing at most one refresh runs at a time.
func (c *RefreshCoordinator) run(ctx context.Context) {
	defer close(c.done)

	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}

		switch {
		case c.dirty && !c.timerArmed:
			delay := c.retryDelay()
			if timer == nil {
				timer = time.NewTimer(delay)
			} else {
				timer.Reset(delay)
			}
			c.timerArmed = true
			c.metrics.Scheduled.Add(1)
			c.mu.Unlock()
			slog.Debug("report refresh scheduled",
				"delay", delay.String(),
				"scheduled_total", c.metrics.Scheduled.Load(),
			)

		case c.dirty && c.timerArmed:
			c.mu.Unlock()
			select {
			case <-timer.C:
				c.mu.Lock()
				c.timerArmed = false
				c.mu.Unlock()
				c.refreshOnce(ctx)
			case <-c.signal:
				// A new sale arrived while the debounce timer was pending; the
				// timer keeps running so all events coalesce into one refresh.
			case <-ctx.Done():
				return
			}

		default:
			// Clean: park until a SaleCreated marks the store dirty.
			c.mu.Unlock()
			select {
			case <-c.signal:
			case <-ctx.Done():
				return
			}
		}
	}
}

// refreshOnce runs a single refresh synchronously on the worker and updates the
// dirty state under the coordinator lock.
func (c *RefreshCoordinator) refreshOnce(ctx context.Context) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.refreshStartMarkCount = c.markCount
	c.mu.Unlock()

	c.metrics.Started.Add(1)
	cur := c.metrics.Concurrent.Add(1)
	for {
		max := c.metrics.MaxConcurrent.Load()
		if cur <= max || c.metrics.MaxConcurrent.CompareAndSwap(max, cur) {
			break
		}
	}
	defer c.metrics.Concurrent.Add(-1)

	slog.Debug("report refresh started", "started_total", c.metrics.Started.Load())
	start := time.Now()
	err := c.refresh(ctx)

	c.mu.Lock()
	if err != nil {
		c.metrics.Failed.Add(1)
		c.failures++
		c.dirty = true
		slog.Error("report refresh failed",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
			"next_delay", c.retryDelay().String(),
		)
	} else {
		c.metrics.Succeeded.Add(1)
		c.failures = 0
		if c.markCount == c.refreshStartMarkCount {
			c.dirty = false
		} else {
			slog.Debug("report refresh caught new data",
				"new_marks", c.markCount-c.refreshStartMarkCount,
			)
		}
		slog.Info("report refresh succeeded",
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
	c.mu.Unlock()
}

// retryDelay returns the delay before the next refresh run: the debounce window
// when healthy, growing exponentially after consecutive failures (bounded).
// Must be called with c.mu held.
func (c *RefreshCoordinator) retryDelay() time.Duration {
	const capBits = 3 // log2(retryBackoffCapMultiplier)
	shift := c.failures
	if shift > capBits {
		shift = capBits
	}
	return c.debounce << shift
}
