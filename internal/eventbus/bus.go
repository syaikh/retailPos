package eventbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"retail-pos-system/internal/shared"
)

var (
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
)

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
		return nil
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
