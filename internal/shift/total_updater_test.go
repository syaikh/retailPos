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

	setupShift := func(t *testing.T, status string) (int, int) {
		t.Helper()
		userID := insertTestUser(ctx, t, 1)
		var shiftID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, $2, 0, NOW()) RETURNING id
		`, userID, status).Scan(&shiftID)
		require.NoError(t, err)
		return shiftID, userID
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

	apply := func(t *testing.T, shiftID int, cashierID int, contribution shared.ShiftSaleContribution) {
		t.Helper()
		contribution.ShiftID = shiftID
		contribution.CashierID = cashierID
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		require.NoError(t, updater.UpdateShiftTotals(ctx, tx, contribution))
		require.NoError(t, tx.Commit(ctx))
	}

	t.Run("cash contribution updates cash_sales only", func(t *testing.T) {
		shiftID, cashierID := setupShift(t, "open")
		apply(t, shiftID, cashierID, shared.ShiftSaleContribution{TotalAmount: 10000, CashSales: 10000})

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 10000, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, 10000, totalSales)
		assert.Equal(t, 1, txCount)
	})

	t.Run("non-cash contribution updates non_cash_sales only", func(t *testing.T) {
		shiftID, cashierID := setupShift(t, "open")
		apply(t, shiftID, cashierID, shared.ShiftSaleContribution{TotalAmount: 20000, NonCashSales: 20000})

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 0, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 20000, totalSales)
		assert.Equal(t, 1, txCount)
	})

	t.Run("mixed contribution updates both and accumulates", func(t *testing.T) {
		shiftID, cashierID := setupShift(t, "open")
		apply(t, shiftID, cashierID, shared.ShiftSaleContribution{TotalAmount: 50000, CashSales: 30000, NonCashSales: 20000})
		apply(t, shiftID, cashierID, shared.ShiftSaleContribution{TotalAmount: 10000, CashSales: 10000})

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 40000, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 60000, totalSales)
		assert.Equal(t, 2, txCount)
	})

	t.Run("closed shift rejects contribution", func(t *testing.T) {
		shiftID, cashierID := setupShift(t, "closed")

		contribution := shared.ShiftSaleContribution{ShiftID: shiftID, CashierID: cashierID, TotalAmount: 10000, CashSales: 10000}
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		err = updater.UpdateShiftTotals(ctx, tx, contribution)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shift is not open")
		assert.ErrorIs(t, err, shared.ErrShiftNotOpen)
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
		err = updater.UpdateShiftTotals(ctx, tx, shared.ShiftSaleContribution{ShiftID: 999999, CashierID: 1, TotalAmount: 10000, CashSales: 10000})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shift is not open")
		assert.ErrorIs(t, err, shared.ErrShiftNotOpen)
		require.NoError(t, tx.Rollback(ctx))
	})

	t.Run("another user's open shift rejects contribution", func(t *testing.T) {
		shiftID, _ := setupShift(t, "open")
		foreignCashierID := insertTestUser(ctx, t, 1)

		contribution := shared.ShiftSaleContribution{ShiftID: shiftID, CashierID: foreignCashierID, TotalAmount: 10000, CashSales: 10000}
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		err = updater.UpdateShiftTotals(ctx, tx, contribution)
		require.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrShiftNotOpen)
		require.NoError(t, tx.Rollback(ctx))

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 0, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, 0, totalSales)
		assert.Equal(t, 0, txCount)
	})
}

// TestTotalUpdater_ApplyReturnAdjustment covers salereturn's consumer-side port:
// a completed return nets its refund out of the shift's running totals. The
// semantics deliberately differ from UpdateShiftTotals — no user_id guard
// (the approver posts onto the creator's shift), no transaction_count change
// (a return is not a sale), and the 'open' guard only.
func TestTotalUpdater_ApplyReturnAdjustment(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	updater := TotalUpdater{}

	setupShift := func(t *testing.T, status string) (int, int) {
		t.Helper()
		userID := insertTestUser(ctx, t, 1)
		var shiftID int
		err := dbPool.QueryRow(ctx, `
			INSERT INTO shifts (user_id, status, opening_balance, opened_at)
			VALUES ($1, $2, 0, NOW()) RETURNING id
		`, userID, status).Scan(&shiftID)
		require.NoError(t, err)
		return shiftID, userID
	}

	// seedTotals writes known running totals so the assertion reads the net
	// effect of the adjustment rather than the accumulation of prior activity.
	seedTotals := func(t *testing.T, shiftID, cash, nonCash, total, txCount int) {
		t.Helper()
		_, err := dbPool.Exec(ctx, `
			UPDATE shifts SET cash_sales = $2, non_cash_sales = $3, total_sales = $4, transaction_count = $5
			WHERE id = $1
		`, shiftID, cash, nonCash, total, txCount)
		require.NoError(t, err)
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

	apply := func(t *testing.T, adj shared.ShiftReturnAdjustment) error {
		t.Helper()
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()
		err = updater.ApplyReturnAdjustment(ctx, tx, adj)
		if err != nil {
			require.NoError(t, tx.Rollback(ctx))
			return err
		}
		require.NoError(t, tx.Commit(ctx))
		return nil
	}

	t.Run("nets cash non-cash and total without touching transaction_count", func(t *testing.T) {
		shiftID, _ := setupShift(t, "open")
		seedTotals(t, shiftID, 50000, 30000, 80000, 3)

		err := apply(t, shared.ShiftReturnAdjustment{
			ShiftID: shiftID, CashRefund: 20000, NonCashRefund: 10000, TotalRefund: 30000,
		})
		require.NoError(t, err)

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 30000, cashSales)
		assert.Equal(t, 20000, nonCashSales)
		assert.Equal(t, 50000, totalSales)
		assert.Equal(t, 3, txCount, "a return is not a sale transaction; transaction_count must not change")
	})

	t.Run("adjusts a shift regardless of its owner", func(t *testing.T) {
		// The port carries no user id at all: the approver may post onto the
		// creator's shift. This subtest pins that the adjustment succeeds on a
		// shift owned by an unrelated user.
		shiftID, ownerID := setupShift(t, "open")
		seedTotals(t, shiftID, 50000, 0, 50000, 1)
		foreignUserID := insertTestUser(ctx, t, 1)
		require.NotEqual(t, ownerID, foreignUserID)

		require.NoError(t, apply(t, shared.ShiftReturnAdjustment{ShiftID: shiftID, CashRefund: 5000, TotalRefund: 5000}))

		cashSales, _, totalSales, _ := getShiftTotals(t, shiftID)
		assert.Equal(t, 45000, cashSales)
		assert.Equal(t, 45000, totalSales)
	})

	t.Run("refund exceeding running totals drives them negative", func(t *testing.T) {
		// Boundary: there is no floor on the columns, so an over-refund (or a
		// refund posted from another shift's sale) surfaces as a negative
		// figure instead of being silently clamped to zero.
		shiftID, _ := setupShift(t, "open")
		seedTotals(t, shiftID, 10000, 5000, 15000, 1)

		require.NoError(t, apply(t, shared.ShiftReturnAdjustment{
			ShiftID: shiftID, CashRefund: 20000, NonCashRefund: 5000, TotalRefund: 25000,
		}))

		cashSales, nonCashSales, totalSales, _ := getShiftTotals(t, shiftID)
		assert.Equal(t, -10000, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, -10000, totalSales)
	})

	t.Run("closed shift rejects adjustment", func(t *testing.T) {
		shiftID, _ := setupShift(t, "closed")
		seedTotals(t, shiftID, 50000, 0, 50000, 1)

		err := apply(t, shared.ShiftReturnAdjustment{ShiftID: shiftID, CashRefund: 5000, TotalRefund: 5000})
		require.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrShiftNotOpen)
		assert.Contains(t, err.Error(), "return adjustment rejected")

		cashSales, nonCashSales, totalSales, txCount := getShiftTotals(t, shiftID)
		assert.Equal(t, 50000, cashSales)
		assert.Equal(t, 0, nonCashSales)
		assert.Equal(t, 50000, totalSales)
		assert.Equal(t, 1, txCount)
	})

	t.Run("missing shift rejects adjustment", func(t *testing.T) {
		err := apply(t, shared.ShiftReturnAdjustment{ShiftID: 999999, CashRefund: 5000, TotalRefund: 5000})
		require.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrShiftNotOpen)
	})
}
