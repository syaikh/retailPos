package customergroup

import (
	"context"

	"retail-pos-system/internal/shared"
)

// PricingLookup is the customer-group-owned implementation of the pricing
// module's consumer-side port (pricing.CustomerGroupExistsProvider, structural
// typing — no import of internal/pricing needed). internal/customergroup is
// the canonical owner of the customer_groups table (ADR
// Modular_Monolith_Module_Boundaries §2.8 Referensi), so the existence check
// the price resolver runs before resolving group-scoped rules is computed here
// rather than via a direct query inside internal/pricing.
type PricingLookup struct{}

// CustomerGroupExists reports whether the customer group ID exists.
func (PricingLookup) CustomerGroupExists(ctx context.Context, db shared.DBPool, customerGroupID int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM customer_groups WHERE id = $1)`, customerGroupID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
