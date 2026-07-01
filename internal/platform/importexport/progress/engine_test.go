package progress

import (
	"context"
	"testing"
	"time"
)

func TestCreateJob(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, err := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected job id 1, got %d", id)
	}
}

func TestStatusLifecycle(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)

	p, err := e.GetProgress(context.Background(), id)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}
	if p.Status != StatusQueued {
		t.Fatalf("expected queued, got %s", p.Status)
	}

	_ = e.SetStatus(context.Background(), id, StatusParsing)
	_ = e.SetStatus(context.Background(), id, StatusValidating)
	_ = e.SetStatus(context.Background(), id, StatusPreviewReady)

	p, _ = e.GetProgress(context.Background(), id)
	if p.Status != StatusPreviewReady {
		t.Fatalf("expected preview_ready, got %s", p.Status)
	}
}

func TestProgressCalculation(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	_ = e.UpdateProgress(context.Background(), id, 25, 100, 0, 10, 15)

	p, _ := e.GetProgress(context.Background(), id)
	if p.ProgressPct != 25 {
		t.Fatalf("expected 25%%, got %d%%", p.ProgressPct)
	}
	if p.Processed != 25 {
		t.Fatalf("expected 25 processed, got %d", p.Processed)
	}
}

func TestCancelRequest(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)

	cancelled, _ := e.IsCancelRequested(context.Background(), id)
	if cancelled {
		t.Fatal("should not be cancelled initially")
	}

	_ = e.RequestCancel(context.Background(), id)

	cancelled, _ = e.IsCancelRequested(context.Background(), id)
	if !cancelled {
		t.Fatal("should be cancelled after request")
	}
}

func TestCompletedDuration(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	time.Sleep(time.Millisecond)
	_ = e.SetStatus(context.Background(), id, StatusCompleted)

	p, _ := e.GetProgress(context.Background(), id)
	if p.DurationMs <= 0 {
		t.Fatalf("expected duration > 0, got %d", p.DurationMs)
	}
}

func TestJobNotFound(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	_, err := e.GetProgress(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestMultipleJobs(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id1, _ := e.CreateJob(context.Background(), "products", "1.0.0", "a.csv", 1, 1)
	id2, _ := e.CreateJob(context.Background(), "categories", "1.0.0", "b.csv", 2, 1)

	if id2 != id1+1 {
		t.Fatalf("expected sequential ids: %d then %d", id1, id2)
	}

	_ = e.UpdateProgress(context.Background(), id1, 10, 100, 0, 0, 0)
	_ = e.UpdateProgress(context.Background(), id2, 50, 200, 2, 0, 0)

	p1, _ := e.GetProgress(context.Background(), id1)
	p2, _ := e.GetProgress(context.Background(), id2)

	if p1.ProgressPct != 10 {
		t.Fatalf("job1: expected 10%%, got %d%%", p1.ProgressPct)
	}
	if p2.ProgressPct != 25 {
		t.Fatalf("job2: expected 25%%, got %d%%", p2.ProgressPct)
	}
}
