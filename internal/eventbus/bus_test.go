package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBus_PublishSubscribe(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 1)
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
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
			[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			done <- struct{}{}
			return nil
		},
	))
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	))

	for i := 0; i < 1100; i++ {
		_ = bus.Publish(context.Background(), "sale.created", nil)
	}

	time.Sleep(200 * time.Millisecond)
}

func TestBus_ConcurrentSafety(t *testing.T) {
	bus := New()
	go bus.Run()
	defer bus.Shutdown()

	done := make(chan struct{}, 100)
	bus.Subscribe(NewListenerFunc(
		[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
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
		[]EventType{SaleCreated},
		func(ctx context.Context, event Event) error {
			return nil
		},
	))
}

func TestListenerFunc_ImplementsInterface(t *testing.T) {
	var _ Listener = (*ListenerFunc)(nil)
}
