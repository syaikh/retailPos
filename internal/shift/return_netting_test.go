package shift

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// stubReturnTotals is a local stand-in for salereturn's production
// implementation of ReturnTotalsProvider. It lives in the shift package on
// purpose: the archtest module boundary forbids importing internal/salereturn
// from internal/shift, so the port is exercised structurally, exactly as
// internal/wiring wires the real one.
type stubReturnTotals struct {
	totals  shared.ShiftReturnTotals
	err     error
	txErr   error
	calls   int
	txCalls int
}

func (s *stubReturnTotals) ReturnTotals(_ context.Context, _ shared.DBPool, _ int) (shared.ShiftReturnTotals, error) {
	s.calls++
	return s.totals, s.err
}

func (s *stubReturnTotals) ReturnTotalsInTx(_ context.Context, _ pgx.Tx, _ int) (shared.ShiftReturnTotals, error) {
	s.txCalls++
	return s.totals, s.txErr
}

// insertShiftSale records a completed sale against a shift, mirroring the
// fixtures used by the sales-summary tests: 'Cash' lands in cash_sales, any
// other method in non_cash_sales.
func insertShiftSale(ctx context.Context, t *testing.T, userID, shiftID int, invoice, method string, amount int) {
	t.Helper()
	// customer_id is NULL explicitly: TruncateTestData empties customers, so
	// the column default of 1 would violate the foreign key.
	_, err := dbPool.Exec(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, discount, tax, total_amount, created_at, shift_id)
		VALUES ($1, $2, NULL, $3, 'completed', $4, 0, 0, $4, NOW(), $5)
	`, invoice, userID, method, amount, shiftID)
	require.NoError(t, err)
}

// The close-time summary is recomputed from GROSS sales rows, so it must be
// reduced by this shift's completed refunds before it overwrites the persisted
// running totals (which ApplyReturnAdjustment already reduced live). These
// tests pin both halves of that contract: the net figure lands on the closed
// shift, and a failing provider aborts the close instead of closing gross.
func TestShiftReturnNetting_CloseShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	t.Run("nets completed refunds out of the persisted totals", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-NET-CASH", "Cash", 55000)

		// The provider reports the refunds this shift has already paid out;
		// the port is a stub, so no return row is needed to prove the netting.
		stub := &stubReturnTotals{
			totals: shared.ShiftReturnTotals{CashRefund: 5000, TotalRefund: 5000},
		}
		repo.SetReturnTotalsProvider(stub)

		// Closing balance a cashier who paid the refund would count:
		// opening 100000 + gross cash 55000 - refund 5000.
		closed, err := repo.CloseShift(ctx, shift.ID, userID, 150000, nil)
		require.NoError(t, err)
		assert.Equal(t, 50000, closed.CashSales, "close-time summary must be net of cash refunds")
		assert.Equal(t, 0, closed.NonCashSales)
		assert.Equal(t, 50000, closed.TotalSales)
		assert.Equal(t, 1, closed.TransactionCount, "returns must not change transaction_count")
		require.NotNil(t, closed.Discrepancy)
		assert.Equal(t, 0, *closed.Discrepancy,
			"a balanced close proves the same netting was applied to the drawer reconciliation")
		assert.False(t, closed.NeedsReview)

		// The net figure must be what is persisted, not just what is returned.
		var cashSales, totalSales, txCount int
		require.NoError(t, dbPool.QueryRow(ctx, `
			SELECT cash_sales, total_sales, transaction_count FROM shifts WHERE id = $1
		`, shift.ID).Scan(&cashSales, &totalSales, &txCount))
		assert.Equal(t, 50000, cashSales)
		assert.Equal(t, 50000, totalSales)
		assert.Equal(t, 1, txCount)

		// ReturnTotals is read through the standalone entry point only when a
		// path runs outside a tx; CloseShift goes through ReturnTotalsInTx.
		assert.Zero(t, stub.calls)
		assert.Equal(t, 1, stub.txCalls, "close must read return totals inside its own transaction")
	})

	t.Run("provider failure aborts the close and leaves the shift open", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-NET-ERR", "Cash", 55000)
		repo.SetReturnTotalsProvider(&stubReturnTotals{err: errors.New("returns unavailable")})

		_, err := repo.CloseShift(ctx, shift.ID, userID, 150000, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to calculate return totals")
		assert.ErrorContains(t, err, "returns unavailable")

		var status string
		require.NoError(t, dbPool.QueryRow(ctx, `SELECT status FROM shifts WHERE id = $1`, shift.ID).Scan(&status))
		assert.Equal(t, "open", status, "a close that cannot net refunds must not persist gross totals")
	})
}

// GetActiveShiftByUserID re-derives totals from live sales only while the
// stored running totals are still zero; that re-derived figure is gross, so it
// must be netted exactly like the close-time summary.
func TestShiftReturnNetting_GetActiveShiftLiveFallback(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	t.Run("live fallback is net of completed refunds", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-LIVE-NET", "Cash", 55000)
		stub := &stubReturnTotals{
			totals: shared.ShiftReturnTotals{CashRefund: 5000, TotalRefund: 5000},
		}
		repo.SetReturnTotalsProvider(stub)

		active, err := repo.GetActiveShiftByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, shift.ID, active.ID)
		assert.Equal(t, 50000, active.CashSales)
		assert.Equal(t, 0, active.NonCashSales)
		assert.Equal(t, 50000, active.TotalSales)
		assert.Equal(t, 1, active.TransactionCount)
		assert.Equal(t, 1, stub.calls, "the standalone entry point serves the live fallback")
		assert.Zero(t, stub.txCalls)
	})

	t.Run("provider failure keeps the stored totals instead of writing gross", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-LIVE-ERR", "Cash", 55000)
		repo.SetReturnTotalsProvider(&stubReturnTotals{err: errors.New("returns unavailable")})

		active, err := repo.GetActiveShiftByUserID(ctx, userID)
		require.NoError(t, err, "the live fallback degrades to the stored figure rather than failing the read")
		assert.Equal(t, 0, active.CashSales,
			"stored totals win when return totals cannot be read; the gross 55000 must not leak through")
		assert.Equal(t, 0, active.TotalSales)
	})
}

// GetShiftWithLiveSales re-derives live cash from the gross sales rows every
// time it is read, so it is the one reader that must subtract refunds on each
// call: a persisted net total proves nothing here because this path never
// writes.
func TestShiftReturnNetting_GetShiftWithLiveSales(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	t.Run("live cash is net of completed refunds", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-LIVECASH-NET", "Cash", 55000)
		stub := &stubReturnTotals{
			totals: shared.ShiftReturnTotals{CashRefund: 5000, TotalRefund: 5000},
		}
		repo.SetReturnTotalsProvider(stub)

		got, liveCash, err := repo.GetShiftWithLiveSales(ctx, ownershipScopeEmpty(), shift.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, shift.ID, got.ID)
		assert.Equal(t, 50000, liveCash,
			"the drawer holds gross 55000 minus the 5000 refund already paid out")
		assert.Equal(t, 1, stub.calls, "the standalone entry point serves this read")
		assert.Zero(t, stub.txCalls, "this read runs outside a transaction")
	})

	t.Run("provider failure surfaces instead of reporting gross cash", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-LIVECASH-ERR", "Cash", 55000)
		repo.SetReturnTotalsProvider(&stubReturnTotals{err: errors.New("returns unavailable")})

		_, _, err := repo.GetShiftWithLiveSales(ctx, ownershipScopeEmpty(), shift.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query return totals")
		assert.ErrorContains(t, err, "returns unavailable")
	})
}

// AutoCloser derives the expected closing balance as
// opening + gross cash - cash refunds + net movements, then hands the same
// figure to CloseShift, which re-derives summary - refunds. Both sides must
// agree or every auto-close of a shift with refunds would flag a phantom
// discrepancy of exactly +CashRefund.
func TestAutoCloser_ReturnNetting(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	backdate := func(t *testing.T, shiftID int) {
		t.Helper()
		_, err := dbPool.Exec(ctx, `
			UPDATE shifts SET opened_at = NOW() - INTERVAL '25 hours' WHERE id = $1
		`, shiftID)
		require.NoError(t, err)
	}

	t.Run("auto-close nets refunds so the drawer reconciles", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-AUTO-CASH", "Cash", 55000)
		backdate(t, shift.ID)
		repo.SetReturnTotalsProvider(&stubReturnTotals{
			totals: shared.ShiftReturnTotals{CashRefund: 5000, TotalRefund: 5000},
		})

		closer := NewAutoCloser(repo, nil) // nil settings -> 24h default threshold
		require.NoError(t, closer.Run(ctx))

		got, err := repo.GetShiftByID(ctx, ownershipScopeEmpty(), shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", got.Status)
		require.NotNil(t, got.ClosingBalance)
		assert.Equal(t, 150000, *got.ClosingBalance, "opening 100000 + gross cash 55000 - refund 5000")
		assert.Equal(t, 50000, got.CashSales)
		require.NotNil(t, got.Discrepancy)
		assert.Equal(t, 0, *got.Discrepancy, "the refund must not surface as a +5000 phantom discrepancy")
		assert.False(t, got.NeedsReview)
		require.NotNil(t, got.Notes)
		assert.Contains(t, *got.Notes, "Auto-closed")
	})

	t.Run("provider failure skips the shift instead of closing it gross", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-AUTO-ERR", "Cash", 55000)
		backdate(t, shift.ID)
		repo.SetReturnTotalsProvider(&stubReturnTotals{err: errors.New("returns unavailable")})

		closer := NewAutoCloser(repo, nil)
		require.NoError(t, closer.Run(ctx), "one unnettable shift must not fail the whole run")

		got, err := repo.GetShiftByID(ctx, ownershipScopeEmpty(), shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "open", got.Status, "the shift is deferred to a later run rather than closed with a wrong balance")
	})

	t.Run("young shifts are untouched", func(t *testing.T) {
		repo := newTestRepo(t)
		userID := insertTestUser(ctx, t, 1)
		shift := createOpenShift(ctx, t, repo, userID)
		insertShiftSale(ctx, t, userID, shift.ID, "INV-AUTO-YOUNG", "Cash", 55000)
		repo.SetReturnTotalsProvider(&stubReturnTotals{
			totals: shared.ShiftReturnTotals{CashRefund: 5000, TotalRefund: 5000},
		})

		closer := NewAutoCloser(repo, nil)
		require.NoError(t, closer.Run(ctx))

		got, err := repo.GetShiftByID(ctx, ownershipScopeEmpty(), shift.ID)
		require.NoError(t, err)
		assert.Equal(t, "open", got.Status)
		assert.Empty(t, got.ClosedAt)
	})
}
