package eventbus

import (
	"context"
	"log"
	"sync"
	"time"
)

type Bus struct {
	mu         sync.RWMutex
	closed     bool
	listeners  map[EventType][]Listener
	eventCh    chan Event
	dispatchWg sync.WaitGroup // tracks inflight dispatch goroutines
	loopWg     sync.WaitGroup // tracks Run() loop
}

func New() *Bus {
	return &Bus{
		listeners: make(map[EventType][]Listener),
		eventCh:   make(chan Event, 1000),
	}
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
		return nil
	default:
		log.Printf("[eventbus] channel full, dropping event: %s", topic)
		return nil
	}
}

// Run memproses event dari channel sampai channel di-close dan kosong.
// Gunakan Shutdown() untuk graceful stop.
func (b *Bus) Run() {
	b.loopWg.Add(1)
	defer b.loopWg.Done()
	for event := range b.eventCh {
		b.dispatch(event)
	}
}

// dispatch meneruskan event ke semua listener yang terdaftar.
// Listener berjalan di goroutine terpisah dengan shallow copy event.
// Payload WAJIB read-only oleh listener (aturan + race detector).
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
			ctx := e.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			if err := l.HandleEvent(ctx, e); err != nil {
				log.Printf("[eventbus] listener error: %v", err)
			}
		}()
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
	b.loopWg.Wait()    // wait for Run() to finish draining channel
	b.dispatchWg.Wait() // wait for all inflight listener goroutines to finish
}
