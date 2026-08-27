package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"print-agent/internal/receipt"
)

// fakeTransport records every job id it was asked to write.
type fakeTransport struct {
	mu    sync.Mutex
	wrote []string
}

func (f *fakeTransport) Write(id string, _ []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wrote = append(f.wrote, id)
	return nil
}

func (f *fakeTransport) Close() error   { return nil }
func (f *fakeTransport) Type() string   { return "fake" }

func (f *fakeTransport) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.wrote)
}

func TestWorkerDrainsPendingJobsOnCancel(t *testing.T) {
	store := NewStore()
	ft := &fakeTransport{}
	w := NewWorker(store, ft)
	w.SetRenderer(func(receipt.Receipt, receipt.Branding) ([]byte, error) {
		return []byte("escpos"), nil
	})

	const n = 5
	for i := 0; i < n; i++ {
		id := "job-" + string(rune('A'+i))
		store.Enqueue(&Job{ID: id, Receipt: receipt.Receipt{InvoiceNumber: id}})
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// Let the worker start processing, then shut it down.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit after context cancel")
	}

	if got := ft.count(); got != n {
		t.Fatalf("expected %d jobs drained, got %d", n, got)
	}
}
