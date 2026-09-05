package shift

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/user"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

var userSeq int

func insertTestUser(ctx context.Context, t *testing.T, roleID int) int {
	t.Helper()
	userSeq++
	var id int
	username := fmt.Sprintf("shift_user_%d", userSeq)
	email := fmt.Sprintf("shift_%d@test.com", userSeq)
	err := dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ($1, $2, 'hash', $3)
		RETURNING id
	`, username, email, roleID).Scan(&id)
	require.NoError(t, err)
	return id
}

func createOpenShift(ctx context.Context, t *testing.T, repo *Repository, userID int) *Shift {
	t.Helper()
	shift, err := repo.OpenShift(ctx, userID, nil, 100000)
	require.NoError(t, err)
	require.NotNil(t, shift)
	return shift
}

func TestShiftRepository_OpenShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("open shift success", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		assert.Equal(t, userID, shift.UserID)
		assert.Equal(t, "open", shift.Status)
		assert.Equal(t, 100000, shift.OpeningBalance)
		assert.NotEmpty(t, shift.OpenedAt)
		assert.NotEmpty(t, shift.CreatedAt)
	})

	t.Run("cannot open second shift for same user", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		createOpenShift(ctx, t, repo, userID)
		_, err := repo.OpenShift(ctx, userID, nil, 50000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already has an open shift")
	})

	t.Run("concurrent open shift yields exactly one open shift", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)

		const attempts = 8
		errs := make(chan error, attempts)
		for i := 0; i < attempts; i++ {
			go func() {
				_, err := repo.OpenShift(context.Background(), userID, nil, 100000)
				errs <- err
			}()
		}

		var success, openShiftErr int
		for i := 0; i < attempts; i++ {
			if err := <-errs; err != nil {
				assert.Contains(t, err.Error(), "already has an open shift")
				openShiftErr++
			} else {
				success++
			}
		}
		assert.Equal(t, 1, success, "exactly one concurrent OpenShift should succeed")
		assert.Equal(t, attempts-1, openShiftErr)

		count := 0
		err := dbPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM shifts WHERE user_id = $1 AND status = 'open'
		`, userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "exactly one open shift row should exist")
	})
}

func TestShiftRepository_CloseShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("close shift success", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		closed, err := repo.CloseShift(ctx, shift.ID, userID, 100000, nil)
		require.NoError(t, err)
		assert.Equal(t, "closed", closed.Status)
		assert.Equal(t, 100000, *closed.ClosingBalance)
		assert.Equal(t, 0, closed.CashSales)
		assert.Equal(t, 0, closed.TotalSales)
		assert.Equal(t, 0, closed.TransactionCount)
		require.NotNil(t, closed.Discrepancy)
		assert.Equal(t, 0, *closed.Discrepancy)
		assert.False(t, closed.NeedsReview)
		assert.NotEmpty(t, closed.ClosedAt)
	})

	t.Run("close shift with discrepancy needs review", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		closed, err := repo.CloseShift(ctx, shift.ID, userID, 200000, nil)
		require.NoError(t, err)
		require.NotNil(t, closed.Discrepancy)
		assert.Equal(t, 100000, *closed.Discrepancy)
		assert.True(t, closed.NeedsReview)
	})

	t.Run("close non-existent shift returns error", func(t *testing.T) {
		_, err := repo.CloseShift(ctx, 999999, 1, 100000, nil)
		assert.Error(t, err)
	})

	t.Run("close shift with notes", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		notes := "test notes"
		closed, err := repo.CloseShift(ctx, shift.ID, userID, 100000, &notes)
		require.NoError(t, err)
		require.NotNil(t, closed.Notes)
		assert.Equal(t, "test notes", *closed.Notes)
	})
}

func TestShiftRepository_GetActiveShiftByUserID(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("returns active shift", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		active, err := repo.GetActiveShiftByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, active.ID)
		assert.Equal(t, "open", active.Status)
	})

	t.Run("returns error when no active shift", func(t *testing.T) {
		_, err := repo.GetActiveShiftByUserID(ctx, 999999)
		assert.Error(t, err)
	})
}

func TestShiftRepository_ListShifts(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("lists shifts", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		createOpenShift(ctx, t, repo, userID)

		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(shifts), 1)
	})

	t.Run("filters by user_id", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		createOpenShift(ctx, t, repo, userID)

		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{UserID: &userID}, "", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, s := range shifts {
			assert.Equal(t, userID, s.UserID)
		}
	})
}

func TestShiftRepository_GetShiftByID(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("gets shift by ID", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, got.ID)
		assert.Equal(t, shift.UserID, got.UserID)
		assert.Equal(t, "open", got.Status)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetShiftByID(ctx, ownership.Scope{}, 999999)
		assert.Error(t, err)
	})
}

func TestShiftRepository_ListShifts_OwnershipScope(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userA := insertTestUser(ctx, t, 1)
	userB := insertTestUser(ctx, t, 1)
	shiftA := createOpenShift(ctx, t, repo, userA)
	shiftB := createOpenShift(ctx, t, repo, userB)

	t.Run("all-access scope returns shifts for every user", func(t *testing.T) {
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 2)
		foundA, foundB := false, false
		for _, s := range shifts {
			if s.ID == shiftA.ID {
				foundA = true
			}
			if s.ID == shiftB.ID {
				foundB = true
			}
		}
		assert.True(t, foundA, "all-access scope must include shift A")
		assert.True(t, foundB, "all-access scope must include shift B")
	})

	t.Run("restricted scope only returns own shifts", func(t *testing.T) {
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{UserID: &userA}, "", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, shifts, 1)
		assert.Equal(t, shiftA.ID, shifts[0].ID)
		assert.Equal(t, userA, shifts[0].UserID)
	})
}

func TestShiftRepository_GetShiftByID_OwnershipScope(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userA := insertTestUser(ctx, t, 1)
	userB := insertTestUser(ctx, t, 1)
	shiftA := createOpenShift(ctx, t, repo, userA)

	t.Run("restricted owner can read own shift", func(t *testing.T) {
		got, err := repo.GetShiftByID(ctx, ownership.Scope{UserID: &userA}, shiftA.ID)
		require.NoError(t, err)
		assert.Equal(t, shiftA.ID, got.ID)
	})

	t.Run("restricted non-owner cannot read shift", func(t *testing.T) {
		_, err := repo.GetShiftByID(ctx, ownership.Scope{UserID: &userB}, shiftA.ID)
		assert.Error(t, err, "non-owner must not be able to read another user's shift")
	})

	t.Run("all-access scope reads any shift", func(t *testing.T) {
		got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shiftA.ID)
		require.NoError(t, err)
		assert.Equal(t, shiftA.ID, got.ID)
	})
}

func TestShiftRepository_ReviewShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("review shift marks as reviewed", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		closed, err := repo.CloseShift(ctx, shift.ID, userID, 200000, nil)
		require.NoError(t, err)
		assert.True(t, closed.NeedsReview)

		reviewerID := insertTestUser(ctx, t, 2)
		reviewed, err := repo.ReviewShift(ctx, closed.ID, reviewerID)
		require.NoError(t, err)
		assert.False(t, reviewed.NeedsReview)
		require.NotNil(t, reviewed.ReviewedBy)
		assert.Equal(t, reviewerID, *reviewed.ReviewedBy)
		assert.NotEmpty(t, reviewed.ReviewedAt)
	})

	t.Run("review non-existent shift", func(t *testing.T) {
		_, err := repo.ReviewShift(ctx, 999999, 1)
		assert.Error(t, err)
	})

	t.Run("review already reviewed shift fails", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		closed, err := repo.CloseShift(ctx, shift.ID, userID, 200000, nil)
		require.NoError(t, err)

		reviewerID := insertTestUser(ctx, t, 2)
		_, err = repo.ReviewShift(ctx, closed.ID, reviewerID)
		require.NoError(t, err)

		_, err = repo.ReviewShift(ctx, closed.ID, reviewerID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending review")
	})

	t.Run("review open shift fails", func(t *testing.T) {
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		_, err := repo.ReviewShift(ctx, shift.ID, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending review")
	})
}

func TestShiftRepository_OpenShift_WithStore(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	var storeID int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Shift Store') RETURNING id`).Scan(&storeID)
	require.NoError(t, err)

	userID := insertTestUser(ctx, t, 1)
	shift, err := repo.OpenShift(ctx, userID, &storeID, 100000)
	require.NoError(t, err)
	require.NotNil(t, shift.StoreID)
	assert.Equal(t, storeID, *shift.StoreID)
}

func TestShiftRepository_CloseShift_WithStore(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	var storeID int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Shift Store') RETURNING id`).Scan(&storeID)
	require.NoError(t, err)

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	var custID int
	err = dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, email, phone)
		VALUES ('Shift Customer', 'shift@test.com', '08111111111')
		RETURNING id
	`).Scan(&custID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		UPDATE shifts SET store_id = $1 WHERE id = $2
	`, storeID, shift.ID)
	require.NoError(t, err)

	closed, err := repo.CloseShift(ctx, shift.ID, userID, 100000, nil)
	require.NoError(t, err)
	require.NotNil(t, closed.StoreID)
	assert.Equal(t, storeID, *closed.StoreID)
	assert.Equal(t, "Shift Store", closed.StoreName)
}

func TestShiftRepository_GetActiveShiftByUserID_LiveSales(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	var custID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, email, phone)
		VALUES ('Shift Customer', 'shift@test.com', '08111111111')
		RETURNING id
	`).Scan(&custID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, discount, tax, total_amount, created_at, shift_id)
		VALUES
			('INV-LIVE-1', $1, $2, 'Cash', 'completed', 50000, 0, 5000, 55000, NOW(), $3),
			('INV-LIVE-2', $1, $2, 'Card', 'completed', 30000, 0, 3000, 33000, NOW(), $3)
	`, userID, custID, shift.ID)
	require.NoError(t, err)

	active, err := repo.GetActiveShiftByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, shift.ID, active.ID)
	assert.Equal(t, 55000, active.CashSales)
	assert.Equal(t, 33000, active.NonCashSales)
	assert.Equal(t, 88000, active.TotalSales)
	assert.Equal(t, 2, active.TransactionCount)
}

func TestShiftRepository_ListShifts_Filters(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	var storeID int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Shift Store') RETURNING id`).Scan(&storeID)
	require.NoError(t, err)

	userID := insertTestUser(ctx, t, 1)

	balanced := createOpenShift(ctx, t, repo, userID)
	_, err = repo.CloseShift(ctx, balanced.ID, userID, 100000, nil)
	require.NoError(t, err)

	surplus := createOpenShift(ctx, t, repo, userID)
	_, err = repo.CloseShift(ctx, surplus.ID, userID, 200000, nil)
	require.NoError(t, err)

	shortage := createOpenShift(ctx, t, repo, userID)
	_, err = repo.CloseShift(ctx, shortage.ID, userID, 50000, nil)
	require.NoError(t, err)

	t.Run("filter by status", func(t *testing.T) {
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "closed", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, shifts, 3)
		for _, s := range shifts {
			assert.Equal(t, "closed", s.Status)
		}
	})

	t.Run("filter by needs_review", func(t *testing.T) {
		val := true
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", &val, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, shifts, 1)
		assert.True(t, shifts[0].NeedsReview)
	})

	reviewerID := insertTestUser(ctx, t, 2)
	_, err = repo.ReviewShift(ctx, surplus.ID, reviewerID)
	require.NoError(t, err)

	t.Run("filter by balanced discrepancy", func(t *testing.T) {
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "balanced", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, s := range shifts {
			assert.Zero(t, *s.Discrepancy)
		}
	})

	t.Run("filter by surplus discrepancy", func(t *testing.T) {
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "surplus", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, s := range shifts {
			assert.Greater(t, *s.Discrepancy, 0)
		}
	})

	t.Run("filter by shortage discrepancy", func(t *testing.T) {
		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "shortage", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, s := range shifts {
			assert.Less(t, *s.Discrepancy, 0)
		}
	})

	t.Run("closed shift with store and review fields populated", func(t *testing.T) {
		var updatedID int
		err := dbPool.QueryRow(ctx, `UPDATE shifts SET store_id = $1, notes = $2 WHERE id = $3 RETURNING id`,
			storeID, "list notes", balanced.ID).Scan(&updatedID)
		require.NoError(t, err)

		shifts, _, err := repo.ListShifts(ctx, ownership.Scope{}, "closed", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		var found Shift
		for _, s := range shifts {
			if s.ID == updatedID {
				found = s
			}
		}
		assert.Equal(t, storeID, *found.StoreID)
		assert.Equal(t, "Shift Store", found.StoreName)
		require.NotNil(t, found.Notes)
		assert.Equal(t, "list notes", *found.Notes)
		assert.NotEmpty(t, found.ClosedAt)
	})
}

func TestShiftRepository_ListShifts_InvalidSort(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	createOpenShift(ctx, t, repo, userID)

	shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "", 10, 0, "bogus_col", "sideways")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(shifts), 1)
}

func TestShiftRepository_GetShiftByID_WithStore(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	var storeID int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Shift Store') RETURNING id`).Scan(&storeID)
	require.NoError(t, err)

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	_, err = dbPool.Exec(ctx, `UPDATE shifts SET store_id = $1, status = 'closed', closed_at = NOW() WHERE id = $2`, storeID, shift.ID)
	require.NoError(t, err)

	got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
	require.NoError(t, err)
	assert.Equal(t, storeID, *got.StoreID)
	assert.Equal(t, "Shift Store", got.StoreName)
	assert.NotEmpty(t, got.ClosedAt)
}

func TestShiftRepository_GetShiftWithLiveSales_NotFound(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	_, _, err := repo.GetShiftWithLiveSales(ctx, 999999)
	assert.Error(t, err)
}

func TestShiftRepository_GetActiveShiftByUserID_AllFields(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	reviewerID := insertTestUser(ctx, t, 2)
	_, err := dbPool.Exec(ctx, `
		UPDATE shifts
		SET closing_balance = 150000, discrepancy = 50000, notes = 'review me',
		    needs_review = true, reviewed_by = $1, reviewed_at = NOW(),
		    closed_at = NOW()
		WHERE id = $2
	`, reviewerID, shift.ID)
	require.NoError(t, err)

	active, err := repo.GetActiveShiftByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, active.ClosingBalance)
	assert.Equal(t, 150000, *active.ClosingBalance)
	require.NotNil(t, active.Discrepancy)
	assert.Equal(t, 50000, *active.Discrepancy)
	require.NotNil(t, active.Notes)
	assert.Equal(t, "review me", *active.Notes)
	require.NotNil(t, active.ReviewedBy)
	assert.Equal(t, reviewerID, *active.ReviewedBy)
	assert.NotEmpty(t, active.ReviewedAt)
	assert.NotEmpty(t, active.ClosedAt)
}

func TestShiftRepository_GetShiftByID_AllFields(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	reviewerID := insertTestUser(ctx, t, 2)
	_, err := dbPool.Exec(ctx, `
		UPDATE shifts
		SET closing_balance = 150000, discrepancy = 50000, notes = 'review me',
		    needs_review = true, reviewed_by = $1, reviewed_at = NOW(),
		    closed_at = NOW(), status = 'closed'
		WHERE id = $2
	`, reviewerID, shift.ID)
	require.NoError(t, err)

	got, err := repo.GetShiftByID(ctx, ownership.Scope{}, shift.ID)
	require.NoError(t, err)
	require.NotNil(t, got.ClosingBalance)
	assert.Equal(t, 150000, *got.ClosingBalance)
	require.NotNil(t, got.Discrepancy)
	assert.Equal(t, 50000, *got.Discrepancy)
	require.NotNil(t, got.Notes)
	assert.Equal(t, "review me", *got.Notes)
	require.NotNil(t, got.ReviewedBy)
	assert.Equal(t, reviewerID, *got.ReviewedBy)
	assert.NotEmpty(t, got.ReviewedAt)
	assert.NotEmpty(t, got.ClosedAt)
}

func TestShiftRepository_CloseShift_SalesSummary(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	shift := createOpenShift(ctx, t, repo, userID)

	var custID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, email, phone)
		VALUES ('Shift Customer', 'shift@test.com', '08111111111')
		RETURNING id
	`).Scan(&custID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, discount, tax, total_amount, created_at, shift_id)
		VALUES
			('INV-CASH', $1, $2, 'Cash', 'completed', 50000, 0, 5000, 55000, NOW(), $3),
			('INV-CASH2', $1, $2, 'Cash', 'completed', 30000, 0, 3000, 33000, NOW(), $3),
			('INV-NONCASH', $1, $2, 'Card', 'completed', 20000, 0, 2000, 22000, NOW(), $3)
	`, userID, custID, shift.ID)
	require.NoError(t, err)

	closed, err := repo.CloseShift(ctx, shift.ID, userID, 88000, nil)
	require.NoError(t, err)
	assert.Equal(t, 88000, closed.CashSales)
	assert.Equal(t, 22000, closed.NonCashSales)
	assert.Equal(t, 110000, closed.TotalSales)
	assert.Equal(t, 3, closed.TransactionCount)
	require.NotNil(t, closed.Discrepancy)
	assert.Equal(t, -100000, *closed.Discrepancy)
	assert.True(t, closed.NeedsReview)
}

func TestShiftRepository_OpenShift_Duplicate(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	userID := insertTestUser(ctx, t, 1)
	createOpenShift(ctx, t, repo, userID)
	_, err := repo.OpenShift(ctx, userID, nil, 50000)
	assert.ErrorContains(t, err, "already has an open shift")
}

func TestShiftRepository_ListOpenShiftsOlderThan(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetStoreNameProvider(store.NamesProvider{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetSalesSummaryProvider(sale.ShiftSummaryProvider{})
	ctx := context.Background()

	t.Run("returns open shifts older than threshold", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		// Backdate the shift to 25 hours ago
		_, err := dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shift.ID)
		require.NoError(t, err)

		threshold := time.Now().Add(-24 * time.Hour)
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, threshold)
		require.NoError(t, err)
		assert.Len(t, shifts, 1)
		assert.Equal(t, shift.ID, shifts[0].ID)
	})

	t.Run("excludes shifts younger than threshold", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		createOpenShift(ctx, t, repo, userID)

		threshold := time.Now().Add(-24 * time.Hour)
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, threshold)
		require.NoError(t, err)
		assert.Len(t, shifts, 0)
	})

	t.Run("excludes closed shifts", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)

		// Close the shift
		_, err := repo.CloseShift(ctx, shift.ID, userID, 100000, nil)
		require.NoError(t, err)

		// Backdate the opened_at
		_, err = dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shift.ID)
		require.NoError(t, err)

		threshold := time.Now().Add(-24 * time.Hour)
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, threshold)
		require.NoError(t, err)
		assert.Len(t, shifts, 0)
	})

	t.Run("returns multiple old open shifts", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		userA := insertTestUser(ctx, t, 1)
		userB := insertTestUser(ctx, t, 1)
		shiftA := createOpenShift(ctx, t, repo, userA)
		shiftB := createOpenShift(ctx, t, repo, userB)

		_, err := dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '30 hours'
			WHERE id IN ($1, $2)
		`, shiftA.ID, shiftB.ID)
		require.NoError(t, err)

		threshold := time.Now().Add(-24 * time.Hour)
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, threshold)
		require.NoError(t, err)
		assert.Len(t, shifts, 2)
	})

	t.Run("returns empty when no shifts exist", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		threshold := time.Now().Add(-24 * time.Hour)
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, threshold)
		require.NoError(t, err)
		assert.Len(t, shifts, 0)
	})

	t.Run("populates store_id and opened_at correctly", func(t *testing.T) {
		_ = shared.TruncateTestData(dbPool)

		var storeID int
		err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Test Store') RETURNING id`).Scan(&storeID)
		require.NoError(t, err)

		userID := insertTestUser(ctx, t, 1)
		shift, err := repo.OpenShift(ctx, userID, &storeID, 100000)
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shift.ID)
		require.NoError(t, err)

		threshold := time.Now().Add(-24 * time.Hour)
		shifts, err := repo.ListOpenShiftsOlderThan(ctx, threshold)
		require.NoError(t, err)
		require.Len(t, shifts, 1)
		require.NotNil(t, shifts[0].StoreID)
		assert.Equal(t, storeID, *shifts[0].StoreID)
		assert.NotEmpty(t, shifts[0].OpenedAt)
	})
}
