package sale

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CartSessionProvider is the sale-owned implementation of the shift module's
// consumer-side port (shift.CartSessionChecker, structural typing — no import
// of internal/shift needed). internal/sale is the canonical owner of the
// cart_sessions table (ADR Modular_Monolith_Module_Boundaries §2.8
// Transaksional), so the open-cart count check that internal/shift uses during
// shift close is computed here rather than via a direct SQL query.
type CartSessionProvider struct{}

// OpenCartCount returns the number of open cart sessions for the given shift,
// within the caller's transaction.
func (CartSessionProvider) OpenCartCount(ctx context.Context, tx pgx.Tx, shiftID int) (int, error) {
	var count int
	err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM cart_sessions WHERE shift_id = $1 AND status = 'open'`, shiftID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
