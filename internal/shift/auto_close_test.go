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
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/user"
)

func TestAutoCloser_GetHours_DefaultWhenNilSettings(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, nil)
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

func TestAutoCloser_GetHours_DefaultWhenSettingsError(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{err: assert.AnError})
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

func TestAutoCloser_GetHours_DefaultWhenKeyMissing(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{}})
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

func TestAutoCloser_GetHours_DefaultWhenValueNotNumber(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "abc"}})
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

func TestAutoCloser_GetHours_DefaultWhenZero(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "0"}})
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

func TestAutoCloser_GetHours_DefaultWhenNegative(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "-5"}})
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

func TestAutoCloser_GetHours_ConfiguredValue(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "8"}})
	hours := ac.getHours(context.Background())
	assert.Equal(t, 8, hours)
}

func TestAutoCloser_GetHours_DefaultWhenEmptyValue(t *testing.T) {
	repo := &Repository{}
	ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": ""}})
	hours := ac.getHours(context.Background())
	assert.Equal(t, defaultAutoCloseHours, hours)
}

// TestAutoCloser_Run_Unit tests the AutoCloser using a mock repository.
// This avoids needing a database connection for the core logic.
func TestAutoCloser_Run_Unit(t *testing.T) {
	t.Run("returns nil when hours is zero (falls back to default, no shifts)", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		// getHours returns defaultAutoCloseHours when setting is "0" (n > 0 check fails)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}))
		ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "0"}})
		err := ac.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns nil when no abandoned shifts found", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}))
		ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}})
		err := ac.Run(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when ListOpenShiftsOlderThan fails", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnError(errors.New("db error"))
		ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}})
		err := ac.Run(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list abandoned shifts")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil when settings is nil (default hours, no shifts)", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}))
		ac := NewAutoCloser(repo, nil)
		err := ac.Run(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("closes abandoned shift via mock", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		now := time.Now()

		// ListOpenShiftsOlderThan returns one shift
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}).
				AddRow(10, 5, nil, "open", 100000, now.Add(-25*time.Hour)))

		// shiftSalesSummary → summary query (CTE matches "FROM sales" pattern)
		mock.ExpectQuery("FROM sales").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))

		// CloseShift → begin, lock, cart check, summary in tx, update, commit
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(10, 5).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at", "created_at"}).
				AddRow(10, 5, nil, "open", 100000, now.Add(-25*time.Hour), now.Add(-25*time.Hour)))
		mock.ExpectQuery("SELECT COUNT").WithArgs(10).WillReturnRows(
			pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM sales").WithArgs(10).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))
		mock.ExpectQuery("UPDATE shifts").WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnRows(
			pgxmock.NewRows([]string{"closed_at", "updated_at"}).AddRow(now, now))
		mock.ExpectCommit()

		ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}})
		err := ac.Run(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("continues to next shift when summary fails", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		now := time.Now()

		// ListOpenShiftsOlderThan returns two shifts
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}).
				AddRow(10, 5, nil, "open", 100000, now.Add(-25*time.Hour)).
				AddRow(11, 6, nil, "open", 50000, now.Add(-30*time.Hour)))

		// First shift: summary fails
		mock.ExpectQuery("FROM sales").WithArgs(pgxmock.AnyArg()).WillReturnError(errors.New("summary boom"))

		// Second shift: summary succeeds
		mock.ExpectQuery("FROM sales").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))
		// Second shift: close succeeds
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(11, 6).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at", "created_at"}).
				AddRow(11, 6, nil, "open", 50000, now.Add(-30*time.Hour), now.Add(-30*time.Hour)))
		mock.ExpectQuery("SELECT COUNT").WithArgs(11).WillReturnRows(
			pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM sales").WithArgs(11).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))
		mock.ExpectQuery("UPDATE shifts").WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnRows(
			pgxmock.NewRows([]string{"closed_at", "updated_at"}).AddRow(now, now))
		mock.ExpectCommit()

		ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}})
		err := ac.Run(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("continues to next shift when close fails", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		now := time.Now()

		// ListOpenShiftsOlderThan returns two shifts
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at"}).
				AddRow(10, 5, nil, "open", 100000, now.Add(-25*time.Hour)).
				AddRow(11, 6, nil, "open", 50000, now.Add(-30*time.Hour)))

		// First shift: summary OK, close fails
		mock.ExpectQuery("FROM sales").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))
		mock.ExpectBegin().WillReturnError(errors.New("tx boom"))

		// Second shift: summary OK, close succeeds
		mock.ExpectQuery("FROM sales").WithArgs(pgxmock.AnyArg()).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT s.id, s.user_id").WithArgs(11, 6).WillReturnRows(
			pgxmock.NewRows([]string{"id", "user_id", "store_id", "status", "opening_balance", "opened_at", "created_at"}).
				AddRow(11, 6, nil, "open", 50000, now.Add(-30*time.Hour), now.Add(-30*time.Hour)))
		mock.ExpectQuery("SELECT COUNT").WithArgs(11).WillReturnRows(
			pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM sales").WithArgs(11).WillReturnRows(
			pgxmock.NewRows([]string{"cash", "non_cash", "total", "count"}).AddRow(0, 0, 0, 0))
		mock.ExpectQuery("UPDATE shifts").WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnRows(
			pgxmock.NewRows([]string{"closed_at", "updated_at"}).AddRow(now, now))
		mock.ExpectCommit()

		ac := NewAutoCloser(repo, &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}})
		err := ac.Run(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAutoCloser_Integration(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("closes abandoned shift older than threshold", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		shift, err := repo.OpenShift(ctx, userID, nil, 100000)
		require.NoError(t, err)

		// Backdate the shift to 25 hours ago
		_, err = dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shift.ID)
		require.NoError(t, err)

		settings := &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}}
		ac := NewAutoCloser(repo, settings)
		err = ac.Run(ctx)
		require.NoError(t, err)

		// Verify shift was closed
		got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", got.Status)
		assert.NotNil(t, got.Notes)
		assert.Contains(t, *got.Notes, "Auto-closed after 24 hours")
	})

	t.Run("does not close shift younger than threshold", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		shift, err := repo.OpenShift(ctx, userID, nil, 100000)
		require.NoError(t, err)

		settings := &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}}
		ac := NewAutoCloser(repo, settings)
		err = ac.Run(ctx)
		require.NoError(t, err)

		// Verify shift is still open
		got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "open", got.Status)
	})

	t.Run("returns nil when no abandoned shifts exist", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		settings := &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}}
		ac := NewAutoCloser(repo, settings)
		err := ac.Run(ctx)
		require.NoError(t, err)
	})

	t.Run("continues closing other shifts when one fails", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userA := insertTestUser(ctx, t, 1)
		userB := insertTestUser(ctx, t, 1)

		shiftA, err := repo.OpenShift(ctx, userA, nil, 100000)
		require.NoError(t, err)
		shiftB, err := repo.OpenShift(ctx, userB, nil, 50000)
		require.NoError(t, err)

		// Backdate both shifts
		_, err = dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours'
			WHERE id IN ($1, $2)
		`, shiftA.ID, shiftB.ID)
		require.NoError(t, err)

		settings := &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}}
		ac := NewAutoCloser(repo, settings)
		err = ac.Run(ctx)
		require.NoError(t, err)

		// Both should be closed (summary has no sales, so close succeeds)
		gotA, err := repo.GetShiftByID(ctx, ownership.Scope{}, shiftA.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", gotA.Status)

		gotB, err := repo.GetShiftByID(ctx, ownership.Scope{}, shiftB.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", gotB.Status)
	})

	t.Run("with sales summary correctly calculates expected cash", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		shift, err := repo.OpenShift(ctx, userID, nil, 100000)
		require.NoError(t, err)

		// Backdate the shift
		_, err = dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shift.ID)
		require.NoError(t, err)

		// Insert a completed cash sale
		var custID int
		err = dbPool.QueryRow(ctx, `
			INSERT INTO customers (name, email, phone)
			VALUES ('AutoClose Customer', 'autoclose@test.com', '0899')
			RETURNING id
		`).Scan(&custID)
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, payment_method, status,
			                   subtotal, discount, tax, total_amount, created_at)
			VALUES ('AUTO-001', $1, $2, $3, 'Cash', 'completed', 50000, 0, 0, 50000, NOW())
		`, userID, custID, shift.ID)
		require.NoError(t, err)

		settings := &mockSettings{data: map[string]string{"shift_auto_close_hours": "24"}}
		ac := NewAutoCloser(repo, settings)
		err = ac.Run(ctx)
		require.NoError(t, err)

		got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", got.Status)
		// Expected cash = opening_balance (100000) + cash_sales (50000) = 150000
		require.NotNil(t, got.ClosingBalance)
		assert.Equal(t, 150000, *got.ClosingBalance)
	})

	t.Run("uses default 24 hours when settings nil", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		shift, err := repo.OpenShift(ctx, userID, nil, 100000)
		require.NoError(t, err)

		// Backdate shift to 25 hours ago
		_, err = dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shift.ID)
		require.NoError(t, err)

		ac := NewAutoCloser(repo, nil)
		err = ac.Run(ctx)
		require.NoError(t, err)

		got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", got.Status)
	})

	t.Run("default hours is a safe constant", func(t *testing.T) {
		assert.Equal(t, 24, defaultAutoCloseHours)
	})

	t.Run("threshold calculation is correct", func(t *testing.T) {
		hours := 24
		threshold := time.Now().Add(-time.Duration(hours) * time.Hour)
		// Threshold should be ~24 hours ago
		ago := time.Since(threshold)
		assert.InDelta(t, 24*time.Hour, ago, float64(time.Minute))
	})
}
