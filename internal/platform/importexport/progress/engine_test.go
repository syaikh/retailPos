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
	if p.Inserted != 10 {
		t.Fatalf("expected 10 inserted, got %d", p.Inserted)
	}
	if p.Updated != 15 {
		t.Fatalf("expected 15 updated, got %d", p.Updated)
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
	if p1.Inserted != 0 || p1.Updated != 0 {
		t.Fatalf("job1: expected inserted=0 updated=0, got %d/%d", p1.Inserted, p1.Updated)
	}
	if p2.Inserted != 0 || p2.Updated != 0 {
		t.Fatalf("job2: expected inserted=0 updated=0, got %d/%d", p2.Inserted, p2.Updated)
	}
}

func TestSetErrorReport(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	err := e.SetErrorReport(context.Background(), id, "row 3: invalid price")
	if err != nil {
		t.Fatalf("SetErrorReport failed: %v", err)
	}

	p, _ := e.GetProgress(context.Background(), id)
	if p.ErrorReport != "row 3: invalid price" {
		t.Fatalf("expected error report, got %q", p.ErrorReport)
	}
}

func TestSetErrorReport_JobNotFound(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	err := e.SetErrorReport(context.Background(), 999, "error")
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestListJobs_FilterByModule(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	_, _ = e.CreateJob(context.Background(), "products", "1.0.0", "a.csv", 1, 1)
	_, _ = e.CreateJob(context.Background(), "categories", "1.0.0", "b.csv", 2, 1)
	_, _ = e.CreateJob(context.Background(), "products", "1.0.0", "c.csv", 3, 1)

	jobs, err := e.ListJobs(context.Background(), "products", 50)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 products jobs, got %d", len(jobs))
	}
}

func TestListJobs_FilterByModule_NoneFound(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	_, _ = e.CreateJob(context.Background(), "products", "1.0.0", "a.csv", 1, 1)

	jobs, err := e.ListJobs(context.Background(), "categories", 50)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestListJobs_Limit(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	for i := 0; i < 5; i++ {
		_, _ = e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	}

	jobs, err := e.ListJobs(context.Background(), "products", 3)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestListJobs_EmptyModuleFilter(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	_, _ = e.CreateJob(context.Background(), "products", "1.0.0", "a.csv", 1, 1)
	_, _ = e.CreateJob(context.Background(), "categories", "1.0.0", "b.csv", 2, 1)

	jobs, err := e.ListJobs(context.Background(), "", 50)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs with empty filter, got %d", len(jobs))
	}
}

func TestUpdateStatus_FailedSetsCompletedAt(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	time.Sleep(time.Millisecond)
	_ = e.SetStatus(context.Background(), id, StatusFailed)

	p, _ := e.GetProgress(context.Background(), id)
	if p.DurationMs <= 0 {
		t.Fatalf("expected duration > 0 for failed status, got %d", p.DurationMs)
	}
}

func TestUpdateStatus_CancelledSetsCompletedAt(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	time.Sleep(time.Millisecond)
	_ = e.SetStatus(context.Background(), id, StatusCancelled)

	p, _ := e.GetProgress(context.Background(), id)
	if p.DurationMs <= 0 {
		t.Fatalf("expected duration > 0 for cancelled status, got %d", p.DurationMs)
	}
}

func TestUpdateStatus_JobNotFound(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	err := e.SetStatus(context.Background(), 999, StatusCompleted)
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestRequestCancel_JobNotFound(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	err := e.RequestCancel(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestIsCancelRequested_JobNotFound(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	_, err := e.IsCancelRequested(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestGetProgress_ZeroTotalRows(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)

	p, _ := e.GetProgress(context.Background(), id)
	if p.ProgressPct != 0 {
		t.Fatalf("expected 0%% for zero total rows, got %d%%", p.ProgressPct)
	}
}

func TestListJobs_WithCompletedDuration(t *testing.T) {
	store := NewInMemoryStore()
	e := NewEngine(store)

	id, _ := e.CreateJob(context.Background(), "products", "1.0.0", "test.csv", 1, 1)
	time.Sleep(time.Millisecond)
	_ = e.SetStatus(context.Background(), id, StatusCompleted)

	jobs, err := e.ListJobs(context.Background(), "products", 50)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].DurationMs <= 0 {
		t.Fatalf("expected duration > 0 in list, got %d", jobs[0].DurationMs)
	}
}
