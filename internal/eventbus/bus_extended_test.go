package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockDeadLetterStore records calls for assertions.
type mockDeadLetterStore struct {
	calls    []deadLetterCall
	storeErr error
}

type deadLetterCall struct {
	eventType string
	payload   []byte
	errMsg    string
}

func (m *mockDeadLetterStore) Store(_ context.Context, eventType string, payload []byte, errMsg string) error {
	m.calls = append(m.calls, deadLetterCall{eventType: eventType, payload: payload, errMsg: errMsg})
	return m.storeErr
}

// probeEvent is a private event type used only by waitForBus to avoid
// interfering with metrics in tests that also subscribe to SaleCreated.
const probeEvent EventType = "__probe__"

// waitForBus ensures Run() has started by publishing a probe event.
func waitForBus(bus *Bus) {
	done := make(chan struct{})
	var once sync.Once
	bus.Subscribe(NewListenerFunc(
		[]EventType{probeEvent},
		func(ctx context.Context, event Event) error {
			once.Do(func() { close(done) })
			return nil
		},
	))
	go bus.Run()
	_ = bus.Publish(context.Background(), string(probeEvent), nil)
	<-done
}

func TestBus_Metrics(t *testing.T) {
	bus := New()
	waitForBus(bus)

	var count atomic.Int32
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			count.Add(1)
			return nil
		},
	))

	// Read baseline (includes probe event from waitForBus).
	baseline := bus.Metrics()

	const n = 5
	for i := 0; i < n; i++ {
		_ = bus.Publish(context.Background(), "sale.created", nil)
	}

	bus.Shutdown()

	m := bus.Metrics()
	if m.EventsPublished != baseline.EventsPublished+int64(n) {
		t.Errorf("EventsPublished = %d, want %d", m.EventsPublished, baseline.EventsPublished+int64(n))
	}
	if m.EventsConsumed != baseline.EventsConsumed+int64(n) {
		t.Errorf("EventsConsumed = %d, want %d", m.EventsConsumed, baseline.EventsConsumed+int64(n))
	}
	if m.EventsFailed != baseline.EventsFailed {
		t.Errorf("EventsFailed = %d, want %d", m.EventsFailed, baseline.EventsFailed)
	}
	if m.EventProcessingDuration <= baseline.EventProcessingDuration {
		t.Errorf("EventProcessingDuration = %d, want > %d", m.EventProcessingDuration, baseline.EventProcessingDuration)
	}
}

func TestBus_MetricsWithFailure(t *testing.T) {
	bus := New()
	waitForBus(bus)

	errTest := errors.New("listener failure")

	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return errTest
		},
	))

	// Read baseline (includes probe event from waitForBus).
	baseline := bus.Metrics()

	ctx, cancel := context.WithCancel(context.Background())
	_ = bus.Publish(ctx, "sale.created", nil)

	// Cancel context quickly to abort retries.
	time.AfterFunc(200*time.Millisecond, cancel)

	bus.Shutdown()

	m := bus.Metrics()
	if m.EventsFailed != baseline.EventsFailed+1 {
		t.Errorf("EventsFailed = %d, want %d", m.EventsFailed, baseline.EventsFailed+1)
	}
	if m.EventsConsumed != baseline.EventsConsumed+1 {
		t.Errorf("EventsConsumed = %d, want %d", m.EventsConsumed, baseline.EventsConsumed+1)
	}
}

func TestBus_DroppedCount(t *testing.T) {
	bus := New()

	// Do not start Run(): with no consumer the capacity-1000 event channel
	// backs up, so the 1100th Publish overflows deterministically instead of
	// relying on dispatcher scheduling to fall behind.
	for i := 0; i < 1100; i++ {
		_ = bus.Publish(context.Background(), "sale.created", nil)
	}

	dropped := bus.DroppedCount()
	if dropped != 100 {
		t.Errorf("DroppedCount = %d, want 100 (channel capacity 1000)", dropped)
	}

	bus.Shutdown()
}

func TestBus_DoubleShutdown(t *testing.T) {
	bus := New()
	waitForBus(bus)
	bus.Shutdown()
	// Second shutdown should be a no-op, no panic.
	bus.Shutdown()
}

func TestBus_SetDeadLetterStore(t *testing.T) {
	origRetries := maxRetries
	origDelay := retryBaseDelay
	maxRetries = 0
	retryBaseDelay = time.Millisecond
	defer func() {
		maxRetries = origRetries
		retryBaseDelay = origDelay
	}()

	bus := New()
	waitForBus(bus)

	store := &mockDeadLetterStore{}
	bus.SetDeadLetterStore(store)

	errTest := errors.New("permanent failure")
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return errTest
		},
	))

	_ = bus.Publish(context.Background(), "sale.created", map[string]string{"key": "value"})

	bus.Shutdown()

	if len(store.calls) != 1 {
		t.Fatalf("dead letter store called %d times, want 1", len(store.calls))
	}
	call := store.calls[0]
	if call.eventType != "sale.created" {
		t.Errorf("eventType = %q, want %q", call.eventType, "sale.created")
	}
	if call.errMsg != errTest.Error() {
		t.Errorf("errMsg = %q, want %q", call.errMsg, errTest.Error())
	}
	var payload map[string]string
	if err := json.Unmarshal(call.payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload["key"] != "value" {
		t.Errorf("payload[key] = %q, want %q", payload["key"], "value")
	}
}

func TestBus_DeadLetterNilStore(t *testing.T) {
	bus := New()
	waitForBus(bus)

	errTest := errors.New("permanent failure")
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return errTest
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	_ = bus.Publish(ctx, "sale.created", nil)
	time.AfterFunc(200*time.Millisecond, cancel)

	bus.Shutdown()

	m := bus.Metrics()
	if m.EventsFailed != 1 {
		t.Errorf("EventsFailed = %d, want 1", m.EventsFailed)
	}
}

func TestBus_DeadLetterUnmarshalablePayload(t *testing.T) {
	bus := New()
	waitForBus(bus)

	store := &mockDeadLetterStore{}
	bus.SetDeadLetterStore(store)

	errTest := errors.New("permanent failure")
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return errTest
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	_ = bus.Publish(ctx, "sale.created", make(chan int))
	time.AfterFunc(200*time.Millisecond, cancel)

	bus.Shutdown()

	// Store should NOT be called because payload can't be marshalled.
	if len(store.calls) != 0 {
		t.Errorf("dead letter store called %d times, want 0 (unmarshalable payload)", len(store.calls))
	}
}

func TestBus_DeadLetterStoreError(t *testing.T) {
	bus := New()
	waitForBus(bus)

	store := &mockDeadLetterStore{storeErr: errors.New("db down")}
	bus.SetDeadLetterStore(store)

	errTest := errors.New("permanent failure")
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return errTest
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	_ = bus.Publish(ctx, "sale.created", "hello")
	time.AfterFunc(300*time.Millisecond, cancel)

	bus.Shutdown()

	if len(store.calls) != 1 {
		t.Errorf("dead letter store called %d times, want 1", len(store.calls))
	}
}

func TestBus_NilContextFallback(t *testing.T) {
	bus := New()
	waitForBus(bus)

	done := make(chan struct{}, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			if ctx == nil {
				t.Error("context should not be nil in listener")
			}
			done <- struct{}{}
			return nil
		},
	))

	// Inject event with nil Ctx directly into the channel.
	bus.eventCh <- Event{
		Type:      SaleCreated,
		Payload:   nil,
		Ctx:       nil,
		Timestamp: time.Now(),
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event with nil context fallback")
	}

	bus.Shutdown()
}

func TestBus_DispatchCancelledContextDuringRetry(t *testing.T) {
	bus := New()
	waitForBus(bus)

	var attempts atomic.Int32
	errTest := errors.New("always fail")
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			attempts.Add(1)
			return errTest
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	_ = bus.Publish(ctx, "sale.created", nil)

	// Cancel after first attempt to abort retries.
	time.AfterFunc(100*time.Millisecond, cancel)

	bus.Shutdown()

	a := attempts.Load()
	if a < 1 {
		t.Errorf("attempts = %d, want >= 1", a)
	}
	// With cancellation after 100ms, should have fewer attempts than maxRetries+1 (4).
	if a > 2 {
		t.Errorf("attempts = %d, want <= 2 (context cancelled to abort retries)", a)
	}
}

func TestBus_NilPayloadDeadLetter(t *testing.T) {
	bus := New()
	waitForBus(bus)

	store := &mockDeadLetterStore{}
	bus.SetDeadLetterStore(store)

	errTest := errors.New("failure")
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return errTest
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	_ = bus.Publish(ctx, "sale.created", nil)
	time.AfterFunc(200*time.Millisecond, cancel)

	bus.Shutdown()

	if len(store.calls) != 1 {
		t.Fatalf("dead letter store called %d times, want 1", len(store.calls))
	}
	if len(store.calls[0].payload) != 0 {
		t.Errorf("payload = %v, want empty []byte for nil payload", store.calls[0].payload)
	}
}
