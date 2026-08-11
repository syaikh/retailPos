package shift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// TestShiftTotalUpdater covers the production implementation of the sale module's
// consumer-side port. It is exercised in the tx of the caller (as sale checkout
// does) to mirror the Unit of Work constraint.
func TestShiftTotalUpdater(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	updater := TotalUpdater{}

	setupShift := func(t *testing.T, status string) int {
		t.Helper()
		userID := insertTestUser(ctx, t, 1)
		var shiftID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, $2, 0, NOW()) RETURNING id
		`, userID, status).Scan(&shiftID)
		require.NoError(t, err)
		return shiftID
	}

	getShiftTotals := func(t *testing.T, shiftID int) (cashSales, nonCashSales, totalSales, txCount int) {
		t.Helper()
		err := dbPool.QueryRow(ctx, `
			SELECT cash_sales, non_cash_sales, total_sales, transaction_count
			FROM shifts WHERE id = $1
		`, shiftID).Scan(&cashSales, &nonCashSales, &totalSales, &txCount)
		require.NoError(t, err)
		return
	}

	apply := func(t *testing.T, shiftID int, contribution shared.ShiftSaleContribution) {
		t.Helper()
		contribution.ShiftID = shiftID
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		require.NoError(t, updater.UpdateShiftTotals(ctx, tx, contribution))
		require.NoError(t, tx.Commit(ctx))
	}

	t.Run("cash contribution updates cash_sales only", func(t *testing.T) {
		shiftID := setupShift(t, "open")
		apply(t, shiftID, shared.ShiftSaleContribution{TotalAmount: 10000, CashSales: 10000})

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 10000, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, 10000, totalSales)
		assert.Equal(t, 1, txCount)
	})

	t.Run("non-cash contribution updates non_cash_sales only", func(t *testing.T) {
		shiftID := setupShift(t, "open")
		apply(t, shiftID, shared.ShiftSaleContribution{TotalAmount: 20000, NonCashSales: 20000})

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 0, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 20000, totalSales)
		assert.Equal(t, 1, txCount)
	})

	t.Run("mixed contribution updates both and accumulates", func(t *testing.T) {
		shiftID := setupShift(t, "open")
		apply(t, shiftID, shared.ShiftSaleContribution{TotalAmount: 50000, CashSales: 30000, NonCashSales: 20000})
		apply(t, shiftID, shared.ShiftSaleContribution{TotalAmount: 10000, CashSales: 10000})

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 40000, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 60000, totalSales)
		assert.Equal(t, 2, txCount)
	})

	t.Run("closed shift rejects contribution", func(t *testing.T) {
		shiftID := setupShift(t, "closed")

		contribution := shared.ShiftSaleContribution{ShiftID: shiftID, TotalAmount: 10000, CashSales: 10000}
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		err = updater.UpdateShiftTotals(ctx, tx, contribution)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shift is not open")
		require.NoError(t, tx.Rollback(ctx))

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 0, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, 0, totalSales)
		assert.Equal(t, 0, txCount)
	})

	t.Run("missing shift rejects contribution", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		err = updater.UpdateShiftTotals(ctx, tx, shared.ShiftSaleContribution{ShiftID: 999999, TotalAmount: 10000, CashSales: 10000})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shift is not open")
		require.NoError(t, tx.Rollback(ctx))
	})
}
