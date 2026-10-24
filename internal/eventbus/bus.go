package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"retail-pos-system/internal/shared"
)

var (
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
	// criticalPublishTimeout bounds how long PublishCritical blocks waiting for
	// the dispatch loop to drain the full channel before falling back to
	// dead-letter persistence. Kept short: the publish happens post-commit on a
	// background context, so a brief stall is acceptable but must not hang the
	// caller indefinitely.
	criticalPublishTimeout = 5 * time.Second
)

// ErrChannelFull is returned by Publish (and PublishCritical after exhausting
// its wait and dead-letter fallback) when the internal event channel is at
// capacity and the event could not be enqueued. Callers publishing
// business-state events MUST NOT discard this error: a dropped event silently
// loses the state change it carried (backend review B6).
var ErrChannelFull = errors.New("eventbus: channel full, event not enqueued")

// ErrBusClosed is returned by PublishCritical when the bus has been shut down,
// so a critical event published into the shutdown window is surfaced rather
// than silently reported as delivered (backend review follow-up).
var ErrBusClosed = errors.New("eventbus: bus is closed, event not enqueued")

// MetricsSnapshot is a point-in-time copy of event processing counters.
type MetricsSnapshot struct {
	EventsPublished         int64
	EventsConsumed          int64
	EventsFailed            int64
	EventProcessingDuration int64
}

// DeadLetterStore abstracts dead-letter persistence.
type DeadLetterStore interface {
	Store(ctx context.Context, eventType string, payload []byte, errMsg string) error
}

// PgDeadLetterStore implements DeadLetterStore backed by PostgreSQL.
type PgDeadLetterStore struct {
	pool shared.DBPool
}

// NewPgDeadLetterStore creates a new PostgreSQL-backed dead-letter store.
func NewPgDeadLetterStore(pool shared.DBPool) *PgDeadLetterStore {
	return &PgDeadLetterStore{pool: pool}
}

// Store inserts a failed event into the dead_letter_events table.
func (s *PgDeadLetterStore) Store(ctx context.Context, eventType string, payload []byte, errMsg string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO dead_letter_events (event_type, payload, error, created_at) VALUES ($1, $2, $3, NOW())`,
		eventType, payload, errMsg,
	)
	return err
}

type busMetrics struct {
	published          atomic.Int64
	consumed           atomic.Int64
	failed             atomic.Int64
	processingDuration atomic.Int64
}

type Bus struct {
	mu              sync.RWMutex
	closed          bool
	listeners       map[EventType][]Listener
	eventCh         chan Event
	dispatchWg      sync.WaitGroup
	loopWg          sync.WaitGroup
	loopStarted     chan struct{}
	dropCount       atomic.Int64
	metrics         busMetrics
	deadLetterStore DeadLetterStore
}

func New() *Bus {
	return &Bus{
		listeners:   make(map[EventType][]Listener),
		eventCh:     make(chan Event, 1000),
		loopStarted: make(chan struct{}),
	}
}

// SetDeadLetterStore configures the store used for events that exhaust all retries.
func (b *Bus) SetDeadLetterStore(store DeadLetterStore) {
	b.deadLetterStore = store
}

func (b *Bus) Subscribe(listener Listener) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	for _, et := range listener.EventTypes() {
		b.listeners[et] = append(b.listeners[et], listener)
	}
}

func (b *Bus) Publish(ctx context.Context, topic string, payload interface{}) error {
	evt := Event{
		Type:      EventType(topic),
		Payload:   payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return nil
	}
	select {
	case b.eventCh <- evt:
		b.metrics.published.Add(1)
		return nil
	default:
		b.dropCount.Add(1)
		slog.Warn("eventbus channel full, dropping event", "event", topic, "total_drops", b.dropCount.Load())
		return ErrChannelFull
	}
}

// PublishCritical publishes an event whose loss would corrupt business state
// (e.g. purchase_receipt.completed, which drives stock deductions). Unlike
// Publish, it does not silently drop when the channel is full: it first tries
// a non-blocking enqueue, then blocks (bounded by criticalPublishTimeout) for
// the dispatch loop to drain space, and finally — if the event still cannot be
// enqueued — persists it to the dead-letter store so the payload is never
// lost and an operator can replay it. It returns ErrChannelFull only in that
// final persisted-but-not-delivered case, so callers can surface the incident,
// and ErrBusClosed when the bus has shut down (the event is NOT delivered and
// NOT dead-lettered — the server is going down, so the caller must not treat
// it as a success).
func (b *Bus) PublishCritical(ctx context.Context, topic string, payload interface{}) error {
	evt := Event{
		Type:      EventType(topic),
		Payload:   payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	// Fast path: non-blocking enqueue when the channel has room.
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	select {
	case b.eventCh <- evt:
		b.metrics.published.Add(1)
		b.mu.RUnlock()
		return nil
	default:
	}
	b.mu.RUnlock()

	// Channel full: block, bounded, for the dispatch loop to drain space. The
	// consumer (Run) receives from eventCh without holding the lock, so holding
	// RLock here cannot deadlock it; a writer (Subscribe/Shutdown) would wait at
	// most criticalPublishTimeout.
	timer := time.NewTimer(criticalPublishTimeout)
	defer timer.Stop()
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	select {
	case b.eventCh <- evt:
		b.metrics.published.Add(1)
		b.mu.RUnlock()
		return nil
	case <-timer.C:
	}
	b.mu.RUnlock()

	// Still full after waiting: persist to dead-letter (WITHOUT holding the
	// lock, since this performs a bounded DB write) so the payload survives for
	// manual replay rather than being silently discarded.
	b.dropCount.Add(1)
	slog.Error("eventbus channel full for critical event, persisting to dead-letter",
		"event", topic, "total_drops", b.dropCount.Load())
	b.persistCriticalDrop(evt)
	return ErrChannelFull
}

// persistCriticalDrop writes a critical event that could not be enqueued to
// the dead-letter store, mirroring deadLetter but for an undelivered (rather
// than listener-failed) event. A nil store or marshal failure is logged; the
// caller has already returned ErrChannelFull to surface the incident.
func (b *Bus) persistCriticalDrop(event Event) {
	store := b.deadLetterStore
	if store == nil {
		slog.Error("eventbus: no dead-letter store configured; critical event payload lost",
			"event", string(event.Type))
		return
	}

	var payloadBytes []byte
	if event.Payload != nil {
		marshalErr := func() error {
			p, err := json.Marshal(event.Payload)
			if err != nil {
				return err
			}
			payloadBytes = p
			return nil
		}()
		if marshalErr != nil {
			slog.Error("eventbus: failed to marshal payload for critical dead letter",
				"event", string(event.Type), "error", marshalErr)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Store(ctx, string(event.Type), payloadBytes, ErrChannelFull.Error()); err != nil {
		slog.Error("eventbus: failed to persist critical dropped event to dead letter",
			"event", string(event.Type), "error", err)
	}
}

// Run memproses event dari channel sampai channel di-close dan kosong.
// Gunakan Shutdown() untuk graceful stop.
func (b *Bus) Run() {
	b.loopWg.Add(1)
	close(b.loopStarted)
	defer b.loopWg.Done()
	for event := range b.eventCh {
		b.dispatch(event)
	}
}

// dispatch meneruskan event ke semua listener yang terdaftar.
// Setiap listener dijalankan di goroutine terpisah dengan retry + exponential backoff.
// Setelah maxRetries kegagalan, event dikirim ke dead-letter store.
func (b *Bus) dispatch(event Event) {
	b.mu.RLock()
	listeners := b.listeners[event.Type]
	b.mu.RUnlock()

	for _, listener := range listeners {
		l := listener
		e := event
		b.dispatchWg.Add(1)
		go func() {
			defer b.dispatchWg.Done()
			start := time.Now()
			b.metrics.consumed.Add(1)

			parentCtx := e.Ctx
			if parentCtx == nil {
				parentCtx = context.Background()
			}

			var lastErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					delay := retryBaseDelay * time.Duration(1<<(attempt-1))
					timer := time.NewTimer(delay)
					select {
					case <-timer.C:
					case <-parentCtx.Done():
						timer.Stop()
					}
					if parentCtx.Err() != nil {
						lastErr = parentCtx.Err()
						break
					}
				}

				ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
				lastErr = l.HandleEvent(ctx, e)
				cancel()

				if lastErr == nil {
					break
				}
				slog.Error("eventbus listener error",
					"event", string(e.Type),
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"error", lastErr,
				)
			}

			b.metrics.processingDuration.Add(time.Since(start).Nanoseconds())

			if lastErr != nil {
				b.metrics.failed.Add(1)
				b.deadLetter(e, lastErr)
			}
		}()
	}
}

// deadLetter stores a permanently failed event in the dead-letter store.
func (b *Bus) deadLetter(event Event, err error) {
	store := b.deadLetterStore
	if store == nil {
		return
	}

	var payloadBytes []byte
	if event.Payload != nil {
		var marshalErr error
		payloadBytes, marshalErr = json.Marshal(event.Payload)
		if marshalErr != nil {
			slog.Error("eventbus: failed to marshal payload for dead letter", "event", string(event.Type), "error", marshalErr)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if storeErr := store.Store(ctx, string(event.Type), payloadBytes, err.Error()); storeErr != nil {
		slog.Error("eventbus: failed to store dead letter event",
			"event", string(event.Type),
			"error", storeErr,
		)
	}
}

// Metrics returns a snapshot of event processing counters.
func (b *Bus) Metrics() MetricsSnapshot {
	return MetricsSnapshot{
		EventsPublished:         b.metrics.published.Load(),
		EventsConsumed:          b.metrics.consumed.Load(),
		EventsFailed:            b.metrics.failed.Load(),
		EventProcessingDuration: b.metrics.processingDuration.Load(),
	}
}

// Shutdown gracefully menghentikan bus.
// 1. Tutup eventCh agar tidak ada event baru masuk.
// 2. Tunggu semua event di channel selesai diproses (drain pattern).
// 3. Tunggu semua dispatch goroutine selesai.
// Tidak ada event yang hilang.
func (b *Bus) Shutdown() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	b.mu.Unlock()

	close(b.eventCh)
	select {
	case <-b.loopStarted:
		b.loopWg.Wait()
	default:
	}
	b.dispatchWg.Wait()
}

func (b *Bus) DroppedCount() int64 {
	return b.dropCount.Load()
}
