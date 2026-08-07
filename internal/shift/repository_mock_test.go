package shift

import (
	"context"
	"errors"
	"testing"
	"time"

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
		mock.ExpectQuery("SELECT id FROM shifts").WithArgs(1).WillReturnError(errors.New("no rows"))
		mock.ExpectQuery("INSERT INTO shifts").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(boom)
		_, err := repo.OpenShift(ctx, 1, nil, 100000)
		assert.ErrorContains(t, err, "failed to open shift")
	})

	t.Run("open shift commit error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM shifts").WithArgs(1).WillReturnError(errors.New("no rows"))
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
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to calculate shift summary")
	})

	t.Run("close shift update error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1, 1).WillReturnRows(rowShift())
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnRows(summaryRow())
		mock.ExpectQuery("UPDATE shifts").WithArgs(100000, 0, 0, 0, 0, 0, (*string)(nil), false, 1).WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to close shift")
	})

	t.Run("close shift commit error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(1, 1).WillReturnRows(rowShift())
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnRows(summaryRow())
		mock.ExpectQuery("UPDATE shifts").WithArgs(100000, 0, 0, 0, 0, 0, (*string)(nil), false, 1).WillReturnRows(
			pgxmock.NewRows([]string{"closed_at", "updated_at"}).AddRow(now, now))
		mock.ExpectCommit().WillReturnError(boom)
		_, err := repo.CloseShift(ctx, 1, 1, 100000, nil)
		assert.ErrorContains(t, err, "failed to commit shift close")
	})

	t.Run("close all begin error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin().WillReturnError(boom)
		_, err := repo.CloseAll(ctx, 1)
		assert.ErrorContains(t, err, "failed to begin transaction")
	})

	t.Run("close all query error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id FROM shifts").WithArgs(1).WillReturnError(boom)
		_, err := repo.CloseAll(ctx, 1)
		assert.ErrorContains(t, err, "failed to query open shifts")
	})

	t.Run("close all scan error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id FROM shifts").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id", "extra"}).AddRow(1, 2))
		_, err := repo.CloseAll(ctx, 1)
		assert.ErrorContains(t, err, "failed to scan shift id")
	})

	t.Run("close all summary error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id FROM shifts").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnError(boom)
		_, err := repo.CloseAll(ctx, 1)
		assert.ErrorContains(t, err, "failed to calculate shift summary")
	})

	t.Run("close all update error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id FROM shifts").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("FROM sales").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))
		mock.ExpectExec("UPDATE shifts").WithArgs(0, 0, 0, 0, 0, 0, "Closed by admin via CloseAll", 1).WillReturnError(boom)
		_, err := repo.CloseAll(ctx, 1)
		assert.ErrorContains(t, err, "failed to close shift 1")
	})

	t.Run("review shift exec error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("UPDATE shifts").WithArgs(1, 2).WillReturnError(boom)
		_, err := repo.ReviewShift(ctx, 2, 1)
		assert.ErrorContains(t, err, "failed to review shift")
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
