package shift

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// TotalUpdater is the production implementation of the sale module's
// consumer-side port (sale.ShiftTotalUpdater). It is the canonical single-writer
// of the shifts running totals and runs against the caller's transaction so a
// sale and its shift contribution commit atomically (Unit of Work, see
// ADR_Cross_Module_Transaction_Strategy §2.2).
type TotalUpdater struct{}

// UpdateShiftTotals accumulates a completed sale's contribution onto its shift's
// running totals. Shifts that are no longer 'open' — e.g. a shift that closed
// concurrently while the sale was committing — return an error so the caller can
// reject the late addition instead of silently dropping the contribution.
func (TotalUpdater) UpdateShiftTotals(ctx context.Context, tx pgx.Tx, c shared.ShiftSaleContribution) error {
	tag, err := tx.Exec(ctx, `
		UPDATE shifts
		SET cash_sales = cash_sales + $1,
		    non_cash_sales = non_cash_sales + $2,
		    total_sales = total_sales + $3,
		    transaction_count = transaction_count + 1,
		    updated_at = NOW()
		WHERE id = $4 AND status = 'open'
	`, c.CashSales, c.NonCashSales, c.TotalAmount, c.ShiftID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("shift is not open or no longer exists; sale contribution rejected")
	}
	return nil
}
