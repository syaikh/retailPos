package report

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultReportRefreshDebounce is the base retry delay after consecutive refresh
// failures when no REPORT_REFRESH_DEBOUNCE is configured.
const DefaultReportRefreshDebounce = 30 * time.Second

// retryBackoffCapMultiplier bounds the exponential retry backoff after
// consecutive refresh failures (base * 2^n, capped at this multiple).
const retryBackoffCapMultiplier = 8

// RefreshFunc executes a single full refresh of the reporting materialized views.
type RefreshFunc func(ctx context.Context) error

// RefreshMetrics exposes atomic counters describing coordinator behaviour, used
// for observability and unit tests.
type RefreshMetrics struct {
	Started       atomic.Int64 // refresh executions started
	Succeeded     atomic.Int64 // refresh executions that returned nil
	Failed        atomic.Int64 // refresh executions that returned an error
	Concurrent    atomic.Int64 // refresh executions currently in flight
	MaxConcurrent atomic.Int64 // peak concurrent refresh executions (invariant: <= 1)
}

// RefreshCoordinator refreshes the reporting materialized views at each
// period boundary (every hour at :00). It guarantees at most one refresh at a
// time and retries failures with exponential backoff.
//
// SaleCreated events no longer trigger refreshes; the coordinator refreshes
// on a fixed boundary schedule so completed hours/days are always up to date.
type RefreshCoordinator struct {
	mu        sync.Mutex
	started   bool
	closed    bool
	cancel    context.CancelFunc
	done      chan struct{}
	failures  int
	debounce  time.Duration
	refresh   RefreshFunc
	metrics   RefreshMetrics
}

// NewRefreshCoordinator creates a coordinator that refreshes reporting data at
// each hour boundary. A non-positive debounce falls back to
// DefaultReportRefreshDebounce for retry backoff.
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
	}
}

// MarkDirty is a no-op kept for backward compatibility with existing
// SaleCreated listeners. The coordinator now refreshes on a fixed boundary
// schedule and does not need dirty signals.
func (c *RefreshCoordinator) MarkDirty() {}

// Start launches the single background refresh worker. It performs an immediate
// refresh to catch up after downtime, then refreshes at each hour boundary.
// It is idempotent.
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
// timer leaks across the process lifecycle. It is idempotent.
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

// IsDirty always returns false; the coordinator no longer tracks dirty state.
func (c *RefreshCoordinator) IsDirty() bool {
	return false
}

// Metrics exposes the coordinator counters for observability and tests.
func (c *RefreshCoordinator) Metrics() *RefreshMetrics {
	return &c.metrics
}

// run is the single worker goroutine. It refreshes immediately on startup,
// then at each hour boundary, with retry backoff on failure.
func (c *RefreshCoordinator) run(ctx context.Context) {
	defer close(c.done)

	c.refreshWithRetry(ctx)

	for {
		next := c.nextBoundary()
		timer := time.NewTimer(next.Sub(time.Now()))

		select {
		case <-timer.C:
			timer.Stop()
			c.refreshWithRetry(ctx)
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// nextBoundary returns the next hour boundary (e.g., if now is 10:37, returns
// 11:00). The MV refresh at that boundary finalizes the completed hour.
func (c *RefreshCoordinator) nextBoundary() time.Time {
	now := time.Now()
	return now.Truncate(time.Hour).Add(time.Hour)
}

// refreshWithRetry runs refreshOnce and retries with exponential backoff until
// it succeeds or the context is cancelled.
func (c *RefreshCoordinator) refreshWithRetry(ctx context.Context) {
	for {
		err := c.refreshOnce(ctx)
		if err == nil {
			return
		}
		c.mu.Lock()
		delay := c.retryDelay()
		c.mu.Unlock()
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			timer.Stop()
			continue
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// refreshOnce runs a single refresh synchronously on the worker and updates
// the metrics under the coordinator lock.
func (c *RefreshCoordinator) refreshOnce(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
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
		slog.Error("report refresh failed",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
			"next_delay", c.retryDelay().String(),
		)
	} else {
		c.metrics.Succeeded.Add(1)
		c.failures = 0
		slog.Info("report refresh succeeded",
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
	c.mu.Unlock()
	return err
}

// retryDelay returns the delay before the next refresh run: the base debounce
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
