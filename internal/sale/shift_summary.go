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
	WITH sales_base AS (
		SELECT s.id, s.total_amount
		FROM sales s
		WHERE s.shift_id = $1
		  AND s.status = 'completed'
	),
	payments AS (
		SELECT sp.sale_id, LOWER(COALESCE(sp.payment_method_code, '')) AS method, sp.amount
		FROM sale_payments sp
		WHERE sp.sale_id IN (SELECT id FROM sales_base)
	),
	legacy AS (
		SELECT s.id, LOWER(s.payment_method) AS method, s.total_amount AS amount
		FROM sales s
		WHERE s.id IN (SELECT id FROM sales_base)
		  AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.sale_id = s.id)
	)
	SELECT
		COALESCE((SELECT SUM(amount) FROM payments WHERE method = 'cash'), 0)
			+ COALESCE((SELECT SUM(amount) FROM legacy WHERE method = 'cash'), 0),
		COALESCE((SELECT SUM(amount) FROM payments WHERE method <> 'cash'), 0)
			+ COALESCE((SELECT SUM(amount) FROM legacy WHERE method <> 'cash'), 0),
		COALESCE((SELECT SUM(total_amount) FROM sales_base), 0),
		(SELECT COUNT(*) FROM sales_base)
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
