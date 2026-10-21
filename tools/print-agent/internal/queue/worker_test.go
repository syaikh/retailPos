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

func (f *fakeTransport) Close() error { return nil }
func (f *fakeTransport) Type() string { return "fake" }

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

// jobSnapshot reads a job's status and error under the store's own lock: the
// store hands out raw pointers, so unlocked field reads race with a live
// worker's Update calls under -race.
func jobSnapshot(store *Store, id string) (Status, string, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	j, ok := store.jobs[id]
	if !ok {
		return "", "", false
	}
	return j.Status, j.Error, true
}

// waitForJob polls (lock-protected) until the job reaches the wanted status.
func waitForJob(t *testing.T, store *Store, id string, want Status) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if status, _, ok := jobSnapshot(store, id); ok && status == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	status, errMsg, ok := jobSnapshot(store, id)
	if !ok {
		t.Fatalf("job %s not found", id)
	}
	t.Fatalf("job %s status %q, want %q (error: %s)", id, status, want, errMsg)
}

func TestWorkerDispatchesReturnJobsToTheReturnRenderer(t *testing.T) {
	store := NewStore()
	ft := &fakeTransport{}
	w := NewWorker(store, ft)
	w.SetRenderer(func(receipt.Receipt, receipt.Branding) ([]byte, error) {
		return []byte("sale"), nil
	})
	var gotReturn receipt.ReturnReceipt
	w.SetReturnRenderer(func(r receipt.ReturnReceipt, _ receipt.Branding) ([]byte, error) {
		gotReturn = r
		return []byte("return"), nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	store.Enqueue(&Job{
		ID:     "ret-1",
		Type:   "return",
		Return: receipt.ReturnReceipt{InvoiceNumber: "RETUR-1"},
	})
	// A return job must reach the return renderer even though a sale renderer
	// is configured: the type decides, not the payload's shape.
	waitForJob(t, store, "ret-1", StatusCompleted)
	if gotReturn.InvoiceNumber != "RETUR-1" {
		t.Fatalf("return renderer received %q, want RETUR-1", gotReturn.InvoiceNumber)
	}
}

func TestWorkerFailsReturnJobWithoutReturnRenderer(t *testing.T) {
	store := NewStore()
	ft := &fakeTransport{}
	w := NewWorker(store, ft)
	w.SetRenderer(func(receipt.Receipt, receipt.Branding) ([]byte, error) {
		return []byte("sale"), nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	store.Enqueue(&Job{ID: "ret-2", Type: "return", Return: receipt.ReturnReceipt{InvoiceNumber: "RETUR-2"}})
	waitForJob(t, store, "ret-2", StatusFailed)

	if _, errMsg, _ := jobSnapshot(store, "ret-2"); errMsg == "" {
		t.Fatal("expected a failure reason")
	}
	if ft.count() != 0 {
		t.Fatal("nothing must be written when no return renderer is configured")
	}
}
