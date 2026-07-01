package progress

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Status string

const (
	StatusQueued       Status = "queued"
	StatusParsing      Status = "parsing"
	StatusValidating   Status = "validating"
	StatusPreviewReady Status = "preview_ready"
	StatusConfirmed    Status = "confirmed"
	StatusImporting    Status = "importing"
	StatusCompleted    Status = "completed"
	StatusFailed       Status = "failed"
	StatusCancelled    Status = "cancelled"
)

type Progress struct {
	JobID       int64  `json:"job_id"`
	Status      Status `json:"status"`
	ProgressPct int    `json:"progress_pct"`
	TotalRows   int    `json:"total_rows"`
	Processed   int    `json:"processed"`
	Errors      int    `json:"errors"`
	StartedAt   string `json:"started_at"`
	DurationMs  int    `json:"duration_ms,omitempty"`
}

type Repository interface {
	CreateJob(ctx context.Context, module, schemaVersion, filename string, userID, storeID int) (int64, error)
	UpdateStatus(ctx context.Context, jobID int64, status Status) error
	UpdateProgress(ctx context.Context, jobID int64, processed, total, errors int) error
	GetProgress(ctx context.Context, jobID int64) (*Progress, error)
	RequestCancel(ctx context.Context, jobID int64) error
	IsCancelRequested(ctx context.Context, jobID int64) (bool, error)
}

type Engine struct {
	repo Repository
}

func NewEngine(repo Repository) *Engine {
	return &Engine{repo: repo}
}

func (e *Engine) CreateJob(ctx context.Context, module, schemaVersion, filename string, userID, storeID int) (int64, error) {
	return e.repo.CreateJob(ctx, module, schemaVersion, filename, userID, storeID)
}

func (e *Engine) SetStatus(ctx context.Context, jobID int64, status Status) error {
	return e.repo.UpdateStatus(ctx, jobID, status)
}

func (e *Engine) UpdateProgress(ctx context.Context, jobID int64, processed, total, errors int) error {
	return e.repo.UpdateProgress(ctx, jobID, processed, total, errors)
}

func (e *Engine) GetProgress(ctx context.Context, jobID int64) (*Progress, error) {
	return e.repo.GetProgress(ctx, jobID)
}

func (e *Engine) RequestCancel(ctx context.Context, jobID int64) error {
	return e.repo.RequestCancel(ctx, jobID)
}

func (e *Engine) IsCancelRequested(ctx context.Context, jobID int64) (bool, error) {
	return e.repo.IsCancelRequested(ctx, jobID)
}

type InMemoryStore struct {
	mu   sync.RWMutex
	jobs map[int64]*inMemoryJob
	next int64
}

type inMemoryJob struct {
	JobID           int64
	Module          string
	SchemaVersion   string
	Filename        string
	Status          Status
	TotalRows       int
	Processed       int
	Errors          int
	StartedAt       time.Time
	CompletedAt     *time.Time
	UserID          int
	StoreID         int
	CancelRequested bool
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		jobs: make(map[int64]*inMemoryJob),
		next: 1,
	}
}

func (s *InMemoryStore) CreateJob(_ context.Context, module, schemaVersion, filename string, userID, storeID int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.next
	s.next++
	s.jobs[id] = &inMemoryJob{
		JobID:         id,
		Module:        module,
		SchemaVersion: schemaVersion,
		Filename:      filename,
		Status:        StatusQueued,
		StartedAt:     time.Now(),
		UserID:        userID,
		StoreID:       storeID,
	}
	return id, nil
}

func (s *InMemoryStore) UpdateStatus(_ context.Context, jobID int64, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %d not found", jobID)
	}
	job.Status = status
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		now := time.Now()
		job.CompletedAt = &now
	}
	return nil
}

func (s *InMemoryStore) UpdateProgress(_ context.Context, jobID int64, processed, total, errors int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %d not found", jobID)
	}
	job.Processed = processed
	job.TotalRows = total
	job.Errors = errors
	return nil
}

func (s *InMemoryStore) GetProgress(_ context.Context, jobID int64) (*Progress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job %d not found", jobID)
	}
	p := &Progress{
		JobID:       job.JobID,
		Status:      job.Status,
		TotalRows:   job.TotalRows,
		Processed:   job.Processed,
		Errors:      job.Errors,
		StartedAt:   job.StartedAt.Format(time.RFC3339),
	}
	if job.TotalRows > 0 {
		p.ProgressPct = (job.Processed * 100) / job.TotalRows
	}
	if job.CompletedAt != nil {
		p.DurationMs = int(job.CompletedAt.Sub(job.StartedAt).Milliseconds())
	}
	return p, nil
}

func (s *InMemoryStore) RequestCancel(_ context.Context, jobID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %d not found", jobID)
	}
	job.CancelRequested = true
	return nil
}

func (s *InMemoryStore) IsCancelRequested(_ context.Context, jobID int64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return false, fmt.Errorf("job %d not found", jobID)
	}
	return job.CancelRequested, nil
}
