package shift

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// SalesSummaryProvider is the consumer-side port for the sale subsystem's
// completed-sales totals. Shift close and live-summary reads previously ran
// SQL against sales/sale_payments directly (ADR audit finding, line 26);
// internal/sale is the canonical single-writer of those tables and provides
// the production implementation. The composition root MUST wire it via
// SetSalesSummaryProvider before any summary read runs — an unwired
// repository fails fast at runtime.
//
// Two entry points are exposed because summaries are read both standalone
// (live cash sales for an open shift) and inside the caller's transaction
// (shift close must observe the same snapshot as the shift row it locks).
type SalesSummaryProvider interface {
	ShiftSummary(ctx context.Context, db shared.DBPool, shiftID int) (shared.ShiftSaleSummary, error)
	ShiftSummaryInTx(ctx context.Context, tx pgx.Tx, shiftID int) (shared.ShiftSaleSummary, error)
}
