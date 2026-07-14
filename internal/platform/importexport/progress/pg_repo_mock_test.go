package progress

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgRepository_CreateJob(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(42))
	mock.ExpectQuery("INSERT INTO import_jobs").
		WithArgs("product", "1.0", "test.csv", 1, (*int)(nil)).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	id, err := repo.CreateJob(context.Background(), "product", "1.0", "test.csv", 1, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_CreateJob_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	sid := 1
	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(43))
	mock.ExpectQuery("INSERT INTO import_jobs").
		WithArgs("product", "1.0", "test.csv", 1, &sid).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	id, err := repo.CreateJob(context.Background(), "product", "1.0", "test.csv", 1, sid)
	require.NoError(t, err)
	assert.Equal(t, int64(43), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_UpdateStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs").
		WithArgs("importing", nil, int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.UpdateStatus(context.Background(), 1, StatusImporting)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_UpdateStatus_Completed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs").
		WithArgs("completed", pgxmock.AnyArg(), int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.UpdateStatus(context.Background(), 1, StatusCompleted)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_UpdateStatus_Failed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs").
		WithArgs("failed", pgxmock.AnyArg(), int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.UpdateStatus(context.Background(), 1, StatusFailed)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_UpdateStatus_Cancelled(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs").
		WithArgs("cancelled", pgxmock.AnyArg(), int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.UpdateStatus(context.Background(), 1, StatusCancelled)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_SetErrorReport(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs SET error_report_path").
		WithArgs("/path/to/report.csv", int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.SetErrorReport(context.Background(), 1, "/path/to/report.csv")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_UpdateProgress(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs").
		WithArgs(100, 45, 3, 2, 45, 0, int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.UpdateProgress(context.Background(), 1, 45, 100, 2, 45, 3)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_UpdateProgress_ZeroTotal(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs").
		WithArgs(0, 0, 0, 0, 0, 0, int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.UpdateProgress(context.Background(), 1, 0, 0, 0, 0, 0)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_GetProgress(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	completed := now.Add(time.Minute)
	rows := pgxmock.NewRows([]string{"id", "module", "status", "total_rows", "inserted", "updated", "error_count", "progress_pct", "error_report_path", "started_at", "completed_at"}).
		AddRow(int64(1), "product", "completed", 100, 95, 3, 2, 100, "", &now, &completed)
	mock.ExpectQuery("SELECT (.+) FROM import_jobs WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	p, err := repo.GetProgress(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, p.Status)
	assert.Equal(t, 100, p.ProgressPct)
	assert.Greater(t, p.DurationMs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_GetProgress_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM import_jobs WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	repo := NewPgRepository(mock)
	_, err = repo.GetProgress(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_GetProgress_NoCompletionTime(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "module", "status", "total_rows", "inserted", "updated", "error_count", "progress_pct", "error_report_path", "started_at", "completed_at"}).
		AddRow(int64(1), "product", "importing", 100, 50, 0, 0, 50, "", &now, nil)
	mock.ExpectQuery("SELECT (.+) FROM import_jobs WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	p, err := repo.GetProgress(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, StatusImporting, p.Status)
	assert.Equal(t, 0, p.DurationMs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_ListJobs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	completed := now.Add(time.Minute)
	rows := pgxmock.NewRows([]string{"id", "module", "status", "total_rows", "inserted", "updated", "error_count", "progress_pct", "error_report_path", "started_at", "completed_at"}).
		AddRow(int64(1), "product", "completed", 100, 98, 2, 0, 100, "", &now, &completed).
		AddRow(int64(2), "product", "importing", 50, 30, 0, 0, 60, "", &now, nil)
	mock.ExpectQuery("SELECT (.+) FROM import_jobs WHERE module = \\$1").
		WithArgs("product", 10).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	jobs, err := repo.ListJobs(context.Background(), "product", 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
	assert.Equal(t, StatusCompleted, jobs[0].Status)
	assert.Equal(t, StatusImporting, jobs[1].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_ListJobs_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "module", "status", "total_rows", "inserted", "updated", "error_count", "progress_pct", "error_report_path", "started_at", "completed_at"})
	mock.ExpectQuery("SELECT (.+) FROM import_jobs WHERE module = \\$1").
		WithArgs("product", 10).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	jobs, err := repo.ListJobs(context.Background(), "product", 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_ListJobs_CompletedJobHasDuration(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	completed := now.Add(5 * time.Second)
	rows := pgxmock.NewRows([]string{"id", "module", "status", "total_rows", "inserted", "updated", "error_count", "progress_pct", "error_report_path", "started_at", "completed_at"}).
		AddRow(int64(1), "product", "completed", 100, 100, 0, 0, 100, "", &now, &completed)
	mock.ExpectQuery("SELECT (.+) FROM import_jobs WHERE module = \\$1").
		WithArgs("product", 10).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	jobs, err := repo.ListJobs(context.Background(), "product", 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Greater(t, jobs[0].DurationMs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_RequestCancel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE import_jobs SET cancel_requested = true").
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPgRepository(mock)
	err = repo.RequestCancel(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_IsCancelRequested(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"cancel_requested"}).AddRow(true)
	mock.ExpectQuery("SELECT cancel_requested FROM import_jobs WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	cancelled, err := repo.IsCancelRequested(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, cancelled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_IsCancelRequested_False(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"cancel_requested"}).AddRow(false)
	mock.ExpectQuery("SELECT cancel_requested FROM import_jobs WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	repo := NewPgRepository(mock)
	cancelled, err := repo.IsCancelRequested(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, cancelled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgRepository_IsCancelRequested_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT cancel_requested FROM import_jobs WHERE id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	repo := NewPgRepository(mock)
	_, err = repo.IsCancelRequested(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
