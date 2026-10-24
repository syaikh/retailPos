package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// GroupLookup is the customer-owned implementation of the sale module's
// consumer-side port (sale.CustomerGroupReader, structural typing — no import
// of internal/sale needed). internal/customer is the canonical owner of the
// customers table (ADR Modular_Monolith_Module_Boundaries §2.8 Referensi), so
// the customer→group membership read used when the cart's customer changes
// lives here rather than as direct SQL inside internal/sale.
type GroupLookup struct{}

// CustomerGroupID returns the customer's customer_group_id. A customer with no
// group, and a nonexistent customer id, both yield (nil, nil): no group means
// base pricing, never an error. Pricing rules are scoped by customer group, so
// the cart customer-swap path re-resolves every line snapshot against this
// value (backend review B5).
func (GroupLookup) CustomerGroupID(ctx context.Context, db shared.DBPool, customerID int) (*int, error) {
	var gid *int
	err := db.QueryRow(ctx, `
		SELECT customer_group_id
		FROM customers
		WHERE id = $1
	`, customerID).Scan(&gid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read customer group for customer %d: %w", customerID, err)
	}
	return gid, nil
}
