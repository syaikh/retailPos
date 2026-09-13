package shift

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
)

type stubStoreNameProvider struct{ names map[int]string }

func (p stubStoreNameProvider) StoreNamesByIDs(_ context.Context, _ shared.DBPool, ids []int) (map[int]string, error) {
	out := make(map[int]string, len(ids))
	for _, id := range ids {
		if name, ok := p.names[id]; ok {
			out[id] = name
		}
	}
	return out, nil
}

type stubUsernameProvider struct{ names map[int]string }

func (p stubUsernameProvider) UsernamesByIDs(_ context.Context, _ shared.DBPool, ids []int) (map[int]string, error) {
	out := make(map[int]string, len(ids))
	for _, id := range ids {
		if name, ok := p.names[id]; ok {
			out[id] = name
		}
	}
	return out, nil
}

func newMockRepo(t *testing.T) (pgxmock.PgxPoolIface, *Repository, context.Context) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	repo := NewRepository(mock)
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	repo.SetStoreNameProvider(stubStoreNameProvider{names: map[int]string{}})
	repo.SetUsernameProvider(stubUsernameProvider{names: map[int]string{}})
	repo.SetCartSessionChecker(sale.CartSessionProvider{})
	return mock, repo, context.Background()
}

func TestRepositoryMock_ErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()
	row8 := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at", "created_at", "updated_at"}).
			AddRow(1, 1, nil, "open", 100000, now, now, now)
	}
	rowShift := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at", "created_at"}).
			AddRow(1, 1, nil, "open", 100000, now, now)
	}
	summaryRow := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0)
	}

	t.Run("open shift begin error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin().WillReturnError(boom)
		_, err := repo.OpenShift(ctx, 1, nil, 100000)
		assert.ErrorContains(t, err, "failed to begin transaction")
	})

	t.Run("open shift insert error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM shifts").WithArgs(1).WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery("INSERT INTO shifts").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(boom)
		_, err := repo.OpenShift(ctx, 1, nil, 100000)
		assert.ErrorContains(t, err, "failed to open shift")
	})

	t.Run("open shift commit error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM shifts").WithArgs(1).WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery("INSERT INTO shifts").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(row8())
		mock.ExpectCommit().WillReturnError(boom)
		_, err := repo.OpenShift(ctx, 1, nil, 100000)
		assert.ErrorContains(t, err, "failed to commit shift")
	})

	t.Run("close shift begin error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin().WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to begin transaction")
	})

	t.Run("close shift summary error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1, 1).WillReturnRows(rowShift())
		mock.ExpectQuery("SELECT COUNT").WithArgs(1).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to calculate shift summary")
	})

	t.Run("close shift update error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1, 1).WillReturnRows(rowShift())
		mock.ExpectQuery("SELECT COUNT").WithArgs(1).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnRows(summaryRow())
		mock.ExpectQuery("UPDATE shifts").WithArgs(100000, 0, 0, 0, 0, 0, (*string)(nil), false, 1).WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to close shift")
	})

	t.Run("close shift commit error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1, 1).WillReturnRows(rowShift())
		mock.ExpectQuery("SELECT COUNT").WithArgs(1).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnRows(summaryRow())
		mock.ExpectQuery("UPDATE shifts").WithArgs(100000, 0, 0, 0, 0, 0, (*string)(nil), false, 1).WillReturnRows(
			pgxmock.NewRows([]string{"closed_at", "updated_at"}).AddRow(now, now))
		mock.ExpectCommit().WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to commit shift close")
	})

	t.Run("review shift exec error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("UPDATE shifts").WithArgs(1, 2).WillReturnError(boom)
		_, err := repo.ReviewShift(ctx, 2, 1)
		assert.ErrorContains(t, err, "failed to review shift")
	})

	t.Run("review shift no rows affected", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("UPDATE shifts").WithArgs(1, 2).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		_, err := repo.ReviewShift(ctx, 2, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending review or not found")
	})

	t.Run("flag for review exec error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("UPDATE shifts").WithArgs(1).WillReturnError(boom)
		err := repo.FlagForReview(ctx, 1)
		assert.ErrorContains(t, err, "failed to flag shift for review")
	})

	t.Run("get shift with live sales get error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1).WillReturnError(boom)
		_, _, err := repo.GetShiftWithLiveSales(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("get shift with live sales summary error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "user_id", "store_id", "status", "opening_balance",
				"closing_balance", "cash_sales", "non_cash_sales", "total_sales", "transaction_count",
				"discrepancy", "notes", "needs_review", "reviewed_by", "reviewed_at",
				"opened_at", "closed_at", "created_at", "updated_at",
			}).AddRow(1, 1, nil, "open", 0, nil, 0, 0, 0, 0, nil, nil, false, nil, nil,
				now, nil, now, now))
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnError(boom)
		_, _, err := repo.GetShiftWithLiveSales(ctx, 1)
		assert.ErrorContains(t, err, "failed to query live cash sales")
	})

	t.Run("list shifts count error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnError(boom)
		_, _, err := repo.ListShifts(ctx, ownershipScopeEmpty(), "", nil, "", 10, 0, "opened_at", "DESC")
		assert.ErrorContains(t, err, "failed to count shifts")
	})

	t.Run("list shifts list error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(10, 0).WillReturnError(boom)
		_, _, err := repo.ListShifts(ctx, ownershipScopeEmpty(), "", nil, "", 10, 0, "opened_at", "DESC")
		assert.ErrorContains(t, err, "failed to list shifts")
	})
}

func TestRepositoryMock_GetActiveShiftByUserID_FullData(t *testing.T) {
	mock, repo, ctx := newMockRepo(t)
	storeID := int64(7)
	closing := int64(120000)
	disc := int64(5000)
	notes := "some notes"
	reviewedBy := int64(3)
	now := time.Now()

	mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1).WillReturnRows(
		pgxmock.NewRows([]string{
			"id", "user_id", "store_id", "status", "opening_balance", "closing_balance",
			"cash_sales", "non_cash_sales", "total_sales", "transaction_count",
			"discrepancy", "notes", "needs_review", "reviewed_by", "reviewed_at",
			"opened_at", "closed_at", "created_at", "updated_at",
		}).AddRow(1, 1, storeID, "open", 100000, closing,
			0, 0, 0, 0, disc, notes, true, reviewedBy, now,
			now, now, now, now))
	repo.SetStoreNameProvider(stubStoreNameProvider{names: map[int]string{7: "Store A"}})
	mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnRows(
		pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(100000, 50000, 150000, 3))

	shift, err := repo.GetActiveShiftByUserID(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, shift)
	assert.Equal(t, "Store A", shift.StoreName)
	require.NotNil(t, shift.StoreID)
	assert.Equal(t, 7, *shift.StoreID)
	require.NotNil(t, shift.ClosingBalance)
	assert.Equal(t, 120000, *shift.ClosingBalance)
	require.NotNil(t, shift.Discrepancy)
	assert.Equal(t, 5000, *shift.Discrepancy)
	require.NotNil(t, shift.Notes)
	assert.Equal(t, "some notes", *shift.Notes)
	require.NotNil(t, shift.ReviewedBy)
	assert.Equal(t, 3, *shift.ReviewedBy)
	assert.NotEmpty(t, shift.ReviewedAt)
	assert.NotEmpty(t, shift.ClosedAt)
	assert.Equal(t, 100000, shift.CashSales)
	assert.Equal(t, 50000, shift.NonCashSales)
	assert.Equal(t, 150000, shift.TotalSales)
	assert.Equal(t, 3, shift.TransactionCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func ownershipScopeEmpty() ownership.Scope {
	return ownership.Scope{}
}

func TestRepositoryMock_ListOpenShiftsOlderThan(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WillReturnError(errors.New("boom"))
		_, err := repo.ListOpenShiftsOlderThan(ctx, time.Now().Add(-24*time.Hour))
		assert.ErrorContains(t, err, "failed to list old open shifts")
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}).
				AddRow("not_an_int", 1, nil, "open", 100000, time.Now()))
		_, err := repo.ListOpenShiftsOlderThan(ctx, time.Now().Add(-24*time.Hour))
		assert.Error(t, err)
	})

	t.Run("empty result", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}))
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, time.Now().Add(-24*time.Hour))
		require.NoError(t, err)
		assert.Len(t, shifts, 0)
	})

	t.Run("with store_id", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		now := time.Now()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}).
				AddRow(1, 1, int64(5), "open", 100000, now))
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, now.Add(-1*time.Hour))
		require.NoError(t, err)
		require.Len(t, shifts, 1)
		require.NotNil(t, shifts[0].StoreID)
		assert.Equal(t, 5, *shifts[0].StoreID)
		assert.Equal(t, 100000, shifts[0].OpeningBalance)
	})

	t.Run("without store_id", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		now := time.Now()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}).
				AddRow(2, 2, nil, "open", 50000, now))
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, now.Add(-1*time.Hour))
		require.NoError(t, err)
		require.Len(t, shifts, 1)
		assert.Nil(t, shifts[0].StoreID)
		assert.Equal(t, 2, shifts[0].UserID)
	})
}

// beginMockTx opens a transaction against the pgxmock pool. The repository's
// tx-taking methods chain their queries through this tx.
func beginMockTx(ctx context.Context, t *testing.T, mock pgxmock.PgxPoolIface) pgx.Tx {
	t.Helper()
	mock.ExpectBegin()
	tx, err := mock.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	return tx
}

func TestRepositoryMock_CreateCashMovement(t *testing.T) {
	boom := errors.New("boom")

	t.Run("invalid movement type returns before touching the database", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		tx := beginMockTx(ctx, t, mock)
		_, err := repo.CreateCashMovement(ctx, tx, 1, 1, "deposit", 1000, nil)
		assert.ErrorIs(t, err, ErrInvalidMovementType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("status query error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		tx := beginMockTx(ctx, t, mock)
		mock.ExpectQuery("SELECT status FROM shifts").WithArgs(1).WillReturnError(boom)
		_, err := repo.CreateCashMovement(ctx, tx, 1, 1, "paid_in", 1000, nil)
		assert.ErrorContains(t, err, "failed to check shift status")
	})

	t.Run("shift not found", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		tx := beginMockTx(ctx, t, mock)
		mock.ExpectQuery("SELECT status FROM shifts").WithArgs(1).WillReturnError(pgx.ErrNoRows)
		_, err := repo.CreateCashMovement(ctx, tx, 1, 1, "paid_in", 1000, nil)
		assert.ErrorContains(t, err, "shift not found")
	})

	t.Run("owner query error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		tx := beginMockTx(ctx, t, mock)
		mock.ExpectQuery("SELECT status FROM shifts").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"status"}).AddRow("open"))
		mock.ExpectQuery("SELECT user_id FROM shifts").WithArgs(1).WillReturnError(boom)
		_, err := repo.CreateCashMovement(ctx, tx, 1, 1, "paid_in", 1000, nil)
		assert.ErrorContains(t, err, "failed to get shift owner")
	})

	t.Run("insert error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		tx := beginMockTx(ctx, t, mock)
		mock.ExpectQuery("SELECT status FROM shifts").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"status"}).AddRow("open"))
		mock.ExpectQuery("SELECT user_id FROM shifts").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"user_id"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO cash_movements").WithArgs(1, 1, "paid_in", 1000, nil).WillReturnError(boom)
		_, err := repo.CreateCashMovement(ctx, tx, 1, 1, "paid_in", 1000, nil)
		assert.ErrorContains(t, err, "failed to create cash movement")
	})
}

func TestRepositoryMock_ShiftCashMovementSummary(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnError(errors.New("boom"))
		_, err := repo.ShiftCashMovementSummary(ctx, 1)
		assert.ErrorContains(t, err, "failed to get cash movement summary")
	})

	t.Run("computes net effect from scanned totals", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COALESCE").WithArgs(2).WillReturnRows(
			pgxmock.NewRows([]string{"cash_drops", "paid_ins", "paid_outs"}).AddRow(50000, 10000, 2000))
		sum, err := repo.ShiftCashMovementSummary(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, 50000, sum.CashDrops)
		assert.Equal(t, 10000, sum.PaidIns)
		assert.Equal(t, 2000, sum.PaidOuts)
		assert.Equal(t, -42000, sum.NetEffect)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepositoryMock_ListCashMovements(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("FROM cash_movements").WithArgs(1).WillReturnError(errors.New("boom"))
		_, err := repo.ListCashMovements(ctx, 1)
		assert.ErrorContains(t, err, "failed to list cash movements")
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("FROM cash_movements").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id", "shift_id", "user_id", "type", "amount", "description", "created_at"}).
				AddRow("not_an_int", 1, 1, "paid_in", 1000, nil, time.Now()))
		_, err := repo.ListCashMovements(ctx, 1)
		assert.ErrorContains(t, err, "failed to scan cash movement")
	})

	t.Run("empty result", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("FROM cash_movements").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id", "shift_id", "user_id", "type", "amount", "description", "created_at"}))
		movements, err := repo.ListCashMovements(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, movements, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepositoryMock_GetShiftReportData_CashMovementSummary(t *testing.T) {
	now := time.Now()
	reportRow := func(shiftID int) *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"id", "user_id", "store_id", "status", "opening_balance",
			"closing_balance", "cash_sales", "non_cash_sales", "total_sales", "transaction_count",
			"discrepancy", "notes", "needs_review", "reviewed_by", "reviewed_at",
			"opened_at", "closed_at", "created_at", "updated_at",
		}).AddRow(shiftID, 1, nil, "open", 100000, nil, 0, 0, 0, 0, nil, nil, false, nil, nil,
			now, nil, now, now)
	}

	t.Run("summary query failure is swallowed and report returned with zero summary", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1).WillReturnRows(reportRow(1))
		mock.ExpectQuery("SELECT COALESCE").WithArgs(1).WillReturnError(errors.New("boom"))

		report, err := repo.GetShiftReportData(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, report)
		assert.Equal(t, 1, report.ID)
		assert.Equal(t, CashMovementSummary{}, report.CashMovementSummary)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful summary is populated in the report", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(2).WillReturnRows(reportRow(2))
		mock.ExpectQuery("SELECT COALESCE").WithArgs(2).WillReturnRows(
			pgxmock.NewRows([]string{"cash_drops", "paid_ins", "paid_outs"}).AddRow(100000, 10000, 25000))

		report, err := repo.GetShiftReportData(ctx, 2)
		require.NoError(t, err)
		require.NotNil(t, report)
		assert.Equal(t, 100000, report.CashMovementSummary.CashDrops)
		assert.Equal(t, 10000, report.CashMovementSummary.PaidIns)
		assert.Equal(t, 25000, report.CashMovementSummary.PaidOuts)
		assert.Equal(t, -115000, report.CashMovementSummary.NetEffect)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
