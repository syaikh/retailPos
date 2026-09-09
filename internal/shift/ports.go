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

// StoreNameProvider resolves store display names for shift listings and
// close/active reads. The stores table is owned by the referensi bounded
// context (internal/store); shift no longer LEFT JOINs stores directly (ADR
// audit finding, stores/users) and instead routes the read through this port.
// The composition root MUST wire it via SetStoreNameProvider before any read
// that needs a store name — an unwired repository fails fast at runtime.
type StoreNameProvider interface {
	StoreNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}

// UsernameProvider resolves cashier usernames for shift listings and detail
// reads. The users table is owned by the platform bounded context
// (internal/user); shift routes the read through this port instead of a
// direct JOIN. SetUsernameProvider MUST be wired before any read that needs a
// username.
type UsernameProvider interface {
	UsernamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}

// PaymentBreakdownProvider returns payment method breakdowns for a shift.
// Implemented by internal/sale (the canonical writer of sale_payments).
type PaymentBreakdownProvider interface {
	PaymentMethodBreakdown(ctx context.Context, db shared.DBPool, shiftID int) ([]shared.PaymentMethodTotal, error)
}

// CartSessionChecker is the consumer-side port for checking open cart sessions
// during shift close. The cart_sessions table is owned by the sale bounded
// context (internal/sale); shift previously queried it directly (ADR
// crossContextDebt audit finding, shift→cart_sessions) and instead routes the
// read through this port. The implementation MUST run against the caller's tx
// to observe uncommitted cart inserts within the same Unit of Work.
// internal/sale provides the production implementation; the composition root
// MUST wire it via SetCartSessionChecker before any shift-close path runs — an
// unwired repository fails fast at runtime.
type CartSessionChecker interface {
	// OpenCartCount returns the number of open cart sessions for the given
	// shift, within the caller's transaction.
	OpenCartCount(ctx context.Context, tx pgx.Tx, shiftID int) (int, error)
}
