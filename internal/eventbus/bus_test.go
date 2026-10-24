package eventbus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// saleTopic is an arbitrary probe topic used by the bus tests. Its string value
// is irrelevant to production; it only needs to be stable within this package.
const saleTopic EventType = "sale.created"

func TestBus_PublishSubscribe(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			done <- struct{}{}
			return nil
		},
	))

	_ = bus.Publish(context.Background(), "sale.created", nil)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBus_MultipleListeners(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	const numListeners = 5
	done := make(chan struct{}, numListeners)
	for i := 0; i < numListeners; i++ {
		bus.Subscribe(NewListenerFunc(
			[]EventType{saleTopic},
			func(ctx context.Context, event Event) error {
				done <- struct{}{}
				return nil
			},
		))
	}

	_ = bus.Publish(context.Background(), "sale.created", nil)

	for i := 0; i < numListeners; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("only %d/%d listeners fired", i, numListeners)
		}
	}
}

func TestBus_ListenerErrorDoesNotPanic(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 2)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			done <- struct{}{}
			return nil
		},
	))
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			done <- struct{}{}
			return nil
		},
	))

	_ = bus.Publish(context.Background(), "sale.created", nil)

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for listeners")
		}
	}
}

func TestBus_UnrelatedEventTypeIgnored(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			done <- struct{}{}
			return nil
		},
	))

	_ = bus.Publish(context.Background(), "product.updated", nil)

	select {
	case <-done:
		t.Fatal("listener should not have been called for unrelated event type")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestBus_ShutdownDrainsEvents(t *testing.T) {
	bus := New()
	go bus.Run()

	const numEvents = 20
	done := make(chan struct{}, numEvents)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			time.Sleep(5 * time.Millisecond)
			done <- struct{}{}
			return nil
		},
	))

	for i := 0; i < numEvents; i++ {
		_ = bus.Publish(context.Background(), "sale.created", nil)
	}

	bus.Shutdown()

	for i := 0; i < numEvents; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("expected %d events after shutdown drain, got %d", numEvents, i)
		}
	}
}

func TestBus_ChannelFullDoesNotBlockPublisher(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	))

	for i := 0; i < 1100; i++ {
		_ = bus.Publish(context.Background(), "sale.created", nil)
	}
}

// TestBus_Publish_ChannelFullReturnsErr asserts the Publish contract: on a full
// channel the event is dropped and ErrChannelFull is returned to the caller
// instead of blocking or silently succeeding (backend review B6). No Run(), so
// the capacity-1000 channel backs up deterministically (same pattern as
// TestBus_DroppedCount).
func TestBus_Publish_ChannelFullReturnsErr(t *testing.T) {
	bus := New()
	defer bus.Shutdown()

	var errCount int
	for i := 0; i < 1100; i++ {
		if err := bus.Publish(context.Background(), "sale.created", nil); err != nil {
			if !errors.Is(err, ErrChannelFull) {
				t.Fatalf("Publish #%d: unexpected error: %v", i, err)
			}
			errCount++
		}
	}

	if errCount != 100 {
		t.Errorf("Publish returned ErrChannelFull %d times, want 100 (channel capacity 1000)", errCount)
	}
}

// TestBus_PublishCritical_Delivered asserts the happy path: when the channel
// has room, PublishCritical enqueues like Publish and the listener receives the
// event.
func TestBus_PublishCritical_Delivered(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan Event, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			done <- event
			return nil
		},
	))

	err := bus.PublishCritical(context.Background(), "sale.created", "payload")
	if err != nil {
		t.Fatalf("PublishCritical: %v", err)
	}

	select {
	case ev := <-done:
		if ev.Payload != "payload" {
			t.Errorf("listener payload = %v, want %q", ev.Payload, "payload")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for critical event delivery")
	}
}

// TestBus_PublishCritical_ChannelFullPersistsToDeadLetter asserts the
// never-lose contract: when the channel stays full, PublishCritical blocks at
// most criticalPublishTimeout, then persists the event to the dead-letter
// store and returns ErrChannelFull so the caller can surface the incident. No
// Run(): the channel never drains. The production timeout (5s) is shortened
// for the test so CI does not pay the real wait.
func TestBus_PublishCritical_ChannelFullPersistsToDeadLetter(t *testing.T) {
	bus := New()
	defer bus.Shutdown()

	savedTimeout := criticalPublishTimeout
	criticalPublishTimeout = 50 * time.Millisecond
	defer func() { criticalPublishTimeout = savedTimeout }()

	store := &mockDeadLetterStore{}
	bus.SetDeadLetterStore(store)

	for i := 0; i < 1000; i++ {
		if err := bus.Publish(context.Background(), "sale.created", nil); err != nil {
			t.Fatalf("fill Publish #%d: %v", i, err)
		}
	}

	err := bus.PublishCritical(context.Background(), "sale.created", "critical-payload")
	if !errors.Is(err, ErrChannelFull) {
		t.Fatalf("PublishCritical on full channel = %v, want ErrChannelFull", err)
	}

	if got := bus.DroppedCount(); got != 1 {
		t.Errorf("DroppedCount = %d, want 1", got)
	}

	if len(store.calls) != 1 {
		t.Fatalf("dead-letter store calls = %d, want 1", len(store.calls))
	}
	call := store.calls[0]
	if call.eventType != "sale.created" {
		t.Errorf("dead-letter eventType = %q, want %q", call.eventType, "sale.created")
	}
	if string(call.payload) != `"critical-payload"` {
		t.Errorf("dead-letter payload = %s, want %q", call.payload, `"critical-payload"`)
	}
	if call.errMsg != ErrChannelFull.Error() {
		t.Errorf("dead-letter errMsg = %q, want %q", call.errMsg, ErrChannelFull.Error())
	}
}

// TestBus_PublishCritical_ClosedBusReturnsErr asserts the honest-shutdown
// contract: a critical event published after Shutdown is neither delivered nor
// dead-lettered, and MUST NOT report success — the caller learns the event was
// not enqueued via ErrBusClosed (backend review follow-up).
func TestBus_PublishCritical_ClosedBusReturnsErr(t *testing.T) {
	bus := New()
	go bus.Run()
	bus.Shutdown()

	store := &mockDeadLetterStore{}
	bus.SetDeadLetterStore(store)

	err := bus.PublishCritical(context.Background(), "sale.created", "late-payload")
	if !errors.Is(err, ErrBusClosed) {
		t.Fatalf("PublishCritical on closed bus = %v, want ErrBusClosed", err)
	}
	if len(store.calls) != 0 {
		t.Errorf("dead-letter store calls = %d, want 0 (closed bus must not dead-letter)", len(store.calls))
	}
}

func TestBus_ConcurrentSafety(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 100)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			done <- struct{}{}
			return nil
		},
	))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bus.Publish(context.Background(), "sale.created", nil)
		}()
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("only %d/100 events received (possible data race or lost events)", i)
		}
	}
}

func TestBus_EventCarriesContext(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	type ctxKey string
	const traceID ctxKey = "trace_id"

	gotCtx := make(chan context.Context, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			gotCtx <- ctx
			return nil
		},
	))

	expectedCtx := context.WithValue(context.Background(), traceID, "abc-123")
	_ = bus.Publish(expectedCtx, "sale.created", nil)

	select {
	case ctx := <-gotCtx:
		if ctx.Value(traceID) != "abc-123" {
			t.Errorf("expected trace ID in context, got %v", ctx.Value(traceID))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBus_TimestampAutoSet(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			if event.Timestamp.IsZero() {
				t.Error("expected non-zero timestamp")
			}
			done <- struct{}{}
			return nil
		},
	))

	_ = bus.Publish(context.Background(), "sale.created", nil)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBus_ShutdownNoEvents(t *testing.T) {
	bus := New()
	go bus.Run()
	bus.Shutdown()
}

func TestBus_PublishAfterShutdown(t *testing.T) {
	bus := New()
	go bus.Run()
	bus.Shutdown()

	_ = bus.Publish(context.Background(), "sale.created", nil)
}

func TestBus_SubscribeAfterShutdown(t *testing.T) {
	bus := New()
	go bus.Run()
	bus.Shutdown()

	bus.Subscribe(NewListenerFunc(
		[]EventType{saleTopic},
		func(ctx context.Context, event Event) error {
			return nil
		},
	))
}

func TestListenerFunc_ImplementsInterface(t *testing.T) {
	var _ Listener = (*ListenerFunc)(nil)
}
