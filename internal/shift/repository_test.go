package shift

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/shared"
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

func insertTestUser(t *testing.T, ctx context.Context, roleID int) int {
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

func createOpenShift(t *testing.T, ctx context.Context, repo *Repository, userID int) *Shift {
	t.Helper()
	shift, err := repo.OpenShift(ctx, userID, nil, 100000)
	require.NoError(t, err)
	require.NotNil(t, shift)
	return shift
}

func TestShiftRepository_OpenShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("open shift success", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)
		assert.Equal(t, userID, shift.UserID)
		assert.Equal(t, "open", shift.Status)
		assert.Equal(t, 100000, shift.OpeningBalance)
		assert.NotEmpty(t, shift.OpenedAt)
		assert.NotEmpty(t, shift.CreatedAt)
	})

	t.Run("cannot open second shift for same user", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		createOpenShift(t, ctx, repo, userID)
		_, err := repo.OpenShift(ctx, userID, nil, 50000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already has an open shift")
	})
}

func TestShiftRepository_CloseShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("close shift success", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)

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
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)

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
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)
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
	ctx := context.Background()

	t.Run("returns active shift", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)

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
	ctx := context.Background()

	t.Run("lists shifts", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		createOpenShift(t, ctx, repo, userID)

		shifts, total, err := repo.ListShifts(ctx, ownership.Scope{}, "", nil, "", 10, 0, "opened_at", "DESC")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(shifts), 1)
	})

	t.Run("filters by user_id", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		createOpenShift(t, ctx, repo, userID)

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
	ctx := context.Background()

	t.Run("gets shift by ID", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)

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
	ctx := context.Background()

	userA := insertTestUser(t, ctx, 1)
	userB := insertTestUser(t, ctx, 1)
	shiftA := createOpenShift(t, ctx, repo, userA)
	shiftB := createOpenShift(t, ctx, repo, userB)

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
	ctx := context.Background()

	userA := insertTestUser(t, ctx, 1)
	userB := insertTestUser(t, ctx, 1)
	shiftA := createOpenShift(t, ctx, repo, userA)

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
	ctx := context.Background()

	t.Run("review shift marks as reviewed", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)

		closed, err := repo.CloseShift(ctx, shift.ID, userID, 200000, nil)
		require.NoError(t, err)
		assert.True(t, closed.NeedsReview)

		reviewerID := insertTestUser(t, ctx, 2)
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
}

func TestShiftRepository_CloseAll(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("closes all open shifts for user", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift1 := createOpenShift(t, ctx, repo, userID)

		userID2 := insertTestUser(t, ctx, 1)
		shift2 := createOpenShift(t, ctx, repo, userID2)

		var custID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO customers (name, email, phone)
			VALUES ('Test Customer', 'test@test.com', '08123456789')
			RETURNING id
		`).Scan(&custID)
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, discount, tax, total_amount, created_at, shift_id)
			VALUES
				('INV-1', $1, $2, 'Cash', 'completed', 100000, 0, 10000, 110000, NOW(), $3),
				('INV-2', $1, $2, 'Cash', 'completed', 200000, 0, 20000, 220000, NOW(), $4),
				('INV-3', $1, $2, 'Card', 'completed', 150000, 0, 15000, 165000, NOW(), $4)
		`, userID, custID, shift1.ID, shift2.ID)
		require.NoError(t, err)

		closedIDs, err := repo.CloseAll(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, closedIDs, 1)
		assert.Contains(t, closedIDs, shift1.ID)

		closed1, _ := repo.GetShiftByID(ctx, ownership.Scope{}, shift1.ID)
		assert.Equal(t, "closed", closed1.Status)
		require.NotNil(t, closed1.ClosingBalance)
		assert.Equal(t, 0, *closed1.ClosingBalance)
		assert.Equal(t, 110000, closed1.CashSales)
		assert.Equal(t, 0, closed1.NonCashSales)
		assert.Equal(t, 110000, closed1.TotalSales)
		assert.Equal(t, 1, closed1.TransactionCount)
		assert.Equal(t, 0, *closed1.Discrepancy)
		require.NotNil(t, closed1.Notes)
		assert.Equal(t, "Closed by admin via CloseAll", *closed1.Notes)
		require.NotNil(t, closed1.ClosedAt)

		stillOpen, _ := repo.GetShiftByID(ctx, ownership.Scope{}, shift2.ID)
		assert.Equal(t, "open", stillOpen.Status)
	})

	t.Run("no open shifts returns empty", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		closedIDs, err := repo.CloseAll(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, closedIDs)
	})
}

func TestShiftRepository_GetShiftWithLiveSales(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("returns shift with live cash sales", func(t *testing.T) {
		userID := insertTestUser(t, ctx, 1)
		shift := createOpenShift(t, ctx, repo, userID)

		var custID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO customers (name, email, phone)
			VALUES ('Test Customer', 'test@test.com', '08123456789')
			RETURNING id
		`).Scan(&custID)
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, discount, tax, total_amount, created_at, shift_id)
			VALUES
				('INV-AUDIT', $1, $2, 'Cash', 'completed', 50000, 0, 5000, 55000, NOW(), $3),
				('INV-AUDIT-2', $1, $2, 'Card', 'completed', 75000, 0, 7500, 82500, NOW(), $3)
		`, userID, custID, shift.ID)
		require.NoError(t, err)

		got, liveCash, err := repo.GetShiftWithLiveSales(ctx, shift.ID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, got.ID)
		assert.Equal(t, 55000, liveCash)
	})
}
