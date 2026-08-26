package queue

import (
	"testing"

	"print-agent/internal/receipt"
)

func TestStoreEnqueueGetUpdate(t *testing.T) {
	s := NewStore()
	j := &Job{ID: "j1", Receipt: receipt.Receipt{InvoiceNumber: "X"}}
	s.Enqueue(j)

	got, ok := s.Get("j1")
	if !ok {
		t.Fatal("job not found")
	}
	if got.Status != StatusQueued {
		t.Fatalf("expected queued, got %s", got.Status)
	}

	s.Update("j1", StatusCompleted, "")
	if got.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", got.Status)
	}
}

func TestStoreRequeue(t *testing.T) {
	s := NewStore()
	s.Enqueue(&Job{ID: "j2", Receipt: receipt.Receipt{InvoiceNumber: "Y"}})
	// drain
	<-s.Channel()
	s.Update("j2", StatusFailed, "boom")
	s.Requeue("j2")
	select {
	case id := <-s.Channel():
		if id != "j2" {
			t.Fatalf("expected j2, got %s", id)
		}
	default:
		t.Fatal("Requeue did not push to channel")
	}
}
