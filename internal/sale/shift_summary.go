package sale

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// ShiftSummaryProvider is the sale-owned implementation of the shift module's
// consumer-side port (shift.SalesSummaryProvider, structural typing — no
// import of internal/shift needed). internal/sale is the canonical single-writer
// of sales/sale_payments (ADR_Modular_Monolith_Module_Boundaries §2.8
// Transaksional), so the completed-sales totals that shift persists at close
// time are computed here rather than inside internal/shift.
type ShiftSummaryProvider struct{}

const shiftSaleSummarySQL = `
	SELECT
		COALESCE(SUM(CASE WHEN LOWER(COALESCE(sp.payment_method_code, s.payment_method)) = 'cash' THEN COALESCE(sp.amount, s.total_amount) ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN LOWER(COALESCE(sp.payment_method_code, s.payment_method)) != 'cash' THEN COALESCE(sp.amount, s.total_amount) ELSE 0 END), 0),
		COALESCE(SUM(s.total_amount), 0),
		COUNT(DISTINCT s.id)
	FROM sales s
	LEFT JOIN sale_payments sp ON sp.sale_id = s.id
	WHERE s.shift_id = $1
	  AND s.status = 'completed'
`

// ShiftSummary returns a shift's completed-sales totals read outside any caller
// transaction (live summaries for open shifts).
func (ShiftSummaryProvider) ShiftSummary(ctx context.Context, db shared.DBPool, shiftID int) (shared.ShiftSaleSummary, error) {
	var s shared.ShiftSaleSummary
	err := db.QueryRow(ctx, shiftSaleSummarySQL, shiftID).Scan(
		&s.TotalCashSales, &s.TotalNonCashSales, &s.TotalSales, &s.TotalTransactions,
	)
	return s, err
}

// ShiftSummaryInTx returns a shift's completed-sales totals against the
// caller's transaction so a close observes the same snapshot as the shift row
// it locks.
func (ShiftSummaryProvider) ShiftSummaryInTx(ctx context.Context, tx pgx.Tx, shiftID int) (shared.ShiftSaleSummary, error) {
	var s shared.ShiftSaleSummary
	err := tx.QueryRow(ctx, shiftSaleSummarySQL, shiftID).Scan(
		&s.TotalCashSales, &s.TotalNonCashSales, &s.TotalSales, &s.TotalTransactions,
	)
	return s, err
}
