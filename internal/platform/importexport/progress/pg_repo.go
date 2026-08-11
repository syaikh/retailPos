package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// compile-time interface check
var _ Repository = (*PgRepository)(nil)

type PgRepository struct {
	db shared.DBPool
}

func NewPgRepository(db shared.DBPool) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) CreateJob(ctx context.Context, module, schemaVersion, filename string, userID, storeID int) (int64, error) {
	var id int64
	var storeIDVal *int
	if storeID > 0 {
		storeIDVal = &storeID
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO import_jobs (module, schema_version, filename, status, user_id, store_id, started_at)
		VALUES ($1, $2, $3, 'queued', $4, $5, NOW())
		RETURNING id
	`, module, schemaVersion, filename, userID, storeIDVal).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create import job: %w", err)
	}
	return id, nil
}

func (r *PgRepository) UpdateStatus(ctx context.Context, jobID int64, status Status) error {
	var completedAt interface{}
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		completedAt = time.Now().In(shared.JakartaLocation())
	}
	_, err := r.db.Exec(ctx, `
		UPDATE import_jobs
		SET status = $1, completed_at = COALESCE($2, completed_at), updated_at = NOW()
		WHERE id = $3
	`, string(status), completedAt, jobID)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}

func (r *PgRepository) SetErrorReport(ctx context.Context, jobID int64, errorReport string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE import_jobs SET error_report_path = $1, updated_at = NOW()
		WHERE id = $2
	`, errorReport, jobID)
	if err != nil {
		return fmt.Errorf("set error report: %w", err)
	}
	return nil
}

func (r *PgRepository) UpdateProgress(ctx context.Context, jobID int64, processed, total, errors, inserted, updated int) error {
	progressPct := 0
	if total > 0 {
		progressPct = (processed * 100) / total
	}
	durationMs := 0
	_, err := r.db.Exec(ctx, `
		UPDATE import_jobs
		SET total_rows = GREATEST(total_rows, $1),
		    inserted = $2,
		    updated = $3,
		    error_count = $4,
		    progress_pct = $5,
		    duration_ms = CASE WHEN $6 > 0 THEN $6 ELSE duration_ms END,
		    updated_at = NOW()
		WHERE id = $7
	`, total, inserted, updated, errors, progressPct, durationMs, jobID)
	if err != nil {
		return fmt.Errorf("update import progress: %w", err)
	}
	return nil
}

func (r *PgRepository) GetProgress(ctx context.Context, jobID int64) (*Progress, error) {
	var job struct {
		ID          int64
		Module      string
		Status      string
		TotalRows   int
		Inserted    int
		Updated     int
		ErrorCount  int
		ProgressPct int
		ErrorReport string
		StartedAt   *time.Time
		CompletedAt *time.Time
	}
	err := r.db.QueryRow(ctx, `
		SELECT id, module, status, total_rows, inserted, updated, error_count, progress_pct,
		       COALESCE(error_report_path, ''), started_at, completed_at
		FROM import_jobs WHERE id = $1
	`, jobID).Scan(
		&job.ID, &job.Module, &job.Status, &job.TotalRows, &job.Inserted,
		&job.Updated, &job.ErrorCount, &job.ProgressPct, &job.ErrorReport,
		&job.StartedAt, &job.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job %d not found", jobID)
		}
		return nil, fmt.Errorf("get progress: %w", err)
	}

	p := &Progress{
		JobID:       job.ID,
		Status:      Status(job.Status),
		ProgressPct: job.ProgressPct,
		TotalRows:   job.TotalRows,
		Processed:   job.Inserted + job.Updated + job.ErrorCount,
		Inserted:    job.Inserted,
		Updated:     job.Updated,
		Errors:      job.ErrorCount,
		ErrorReport: job.ErrorReport,
	}
	if job.StartedAt != nil {
		p.StartedAt = job.StartedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if job.CompletedAt != nil {
		p.DurationMs = int(job.CompletedAt.Sub(*job.StartedAt).Milliseconds())
	}
	return p, nil
}

// GetJobMeta resolves a job's module and owning store. Returns (module,
// storeID, nil); storeID is nil for jobs not bound to a store.
func (r *PgRepository) GetJobMeta(ctx context.Context, jobID int64) (string, *int, error) {
	var module string
	var storeID *int
	err := r.db.QueryRow(ctx, `
		SELECT module, store_id FROM import_jobs WHERE id = $1
	`, jobID).Scan(&module, &storeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil, fmt.Errorf("job %d not found", jobID)
		}
		return "", nil, fmt.Errorf("get job meta: %w", err)
	}
	return module, storeID, nil
}

func (r *PgRepository) ListJobs(ctx context.Context, module string, limit int, storeID *int) ([]*Progress, error) {
	query := `
		SELECT id, module, status, total_rows, inserted, updated, error_count, progress_pct,
		       COALESCE(error_report_path, ''), started_at, completed_at
		FROM import_jobs
		WHERE module = $1`
	args := []interface{}{module}
	next := 2
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", next)
		args = append(args, *storeID)
		next++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", next)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	result := make([]*Progress, 0)
	for rows.Next() {
		var job struct {
			ID          int64
			Module      string
			Status      string
			TotalRows   int
			Inserted    int
			Updated     int
			ErrorCount  int
			ProgressPct int
			ErrorReport string
			StartedAt   *time.Time
			CompletedAt *time.Time
		}
		if err := rows.Scan(
			&job.ID, &job.Module, &job.Status, &job.TotalRows, &job.Inserted,
			&job.Updated, &job.ErrorCount, &job.ProgressPct, &job.ErrorReport,
			&job.StartedAt, &job.CompletedAt,
		); err != nil {
			continue
		}
		p := &Progress{
			JobID:       job.ID,
			Status:      Status(job.Status),
			ProgressPct: job.ProgressPct,
			TotalRows:   job.TotalRows,
			Processed:   job.Inserted + job.Updated + job.ErrorCount,
			Inserted:    job.Inserted,
			Updated:     job.Updated,
			Errors:      job.ErrorCount,
			ErrorReport: job.ErrorReport,
		}
		if job.StartedAt != nil {
			p.StartedAt = job.StartedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if job.CompletedAt != nil {
			p.DurationMs = int(job.CompletedAt.Sub(*job.StartedAt).Milliseconds())
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list jobs rows: %w", err)
	}

	return result, nil
}

func (r *PgRepository) RequestCancel(ctx context.Context, jobID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE import_jobs SET cancel_requested = true, updated_at = NOW()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("request cancel: %w", err)
	}
	return nil
}

func (r *PgRepository) IsCancelRequested(ctx context.Context, jobID int64) (bool, error) {
	var cancelled bool
	err := r.db.QueryRow(ctx, `
		SELECT cancel_requested FROM import_jobs WHERE id = $1
	`, jobID).Scan(&cancelled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, fmt.Errorf("job %d not found", jobID)
		}
		return false, fmt.Errorf("is cancel requested: %w", err)
	}
	return cancelled, nil
}
