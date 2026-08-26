// Package queue manages print jobs: an in-memory store plus a worker that renders
// and dispatches them. The store is the source of truth for job status; the
// backend (the POS) remains the source of truth for the transaction itself.
package queue

import (
	"sort"
	"sync"
	"time"

	"print-agent/internal/receipt"
)

// Status is a print job lifecycle state.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusPrinting  Status = "printing"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Job is a single print job.
type Job struct {
	ID        string          `json:"job_id"`
	Status    Status          `json:"status"`
	Receipt   receipt.Receipt `json:"receipt"`
	Branding  receipt.Branding `json:"branding"`
	Error     string          `json:"error"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Store is an in-memory, concurrency-safe job store with a dispatch channel.
// It is intentionally not persisted: on agent restart, reprint is available from
// the backend (the POS) which retains the transaction.
type Store struct {
	mu   sync.Mutex
	jobs map[string]*Job
	ch   chan string
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{
		jobs: make(map[string]*Job),
		ch:   make(chan string, 1024),
	}
}

// Enqueue registers a new job and signals the worker.
func (s *Store) Enqueue(j *Job) {
	s.mu.Lock()
	j.Status = StatusQueued
	now := time.Now()
	j.CreatedAt = now
	j.UpdatedAt = now
	s.jobs[j.ID] = j
	s.mu.Unlock()
	s.ch <- j.ID
}

// Requeue re-signals an existing (failed) job for the worker.
func (s *Store) Requeue(id string) {
	s.ch <- id
}

// Channel returns the dispatch channel of queued job IDs.
func (s *Store) Channel() <-chan string {
	return s.ch
}

// List returns a snapshot of all jobs, ordered oldest first.
func (s *Store) List() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	sort.Slice(jobs, func(i, k int) bool {
		return jobs[i].CreatedAt.Before(jobs[k].CreatedAt)
	})
	return jobs
}

// Get returns a job by ID.
func (s *Store) Get(id string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	return j, ok
}

// Update changes a job's status and optional error message.
func (s *Store) Update(id string, status Status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.Status = status
		j.Error = errMsg
		j.UpdatedAt = time.Now()
	}
}
