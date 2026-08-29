package customer

import (
	"context"

	"retail-pos-system/internal/shared"
)

// GroupNameProvider is the customer-group-owned read side of the
// customer module's group-name enrichment. customer_groups is owned by the
// referensi bounded context (internal/customergroup); customer no longer queries
// that table directly and instead routes group-name lookups through this port.
// internal/customergroup provides the production implementation
// (customergroup.NameLookup); the composition root MUST wire it
// via SetCustomerGroupNameProvider before any customer read that returns a
// group name — an unwired repository fails fast at runtime.
type GroupNameProvider interface {
	// CustomerGroupNamesByIDs returns customer-group names keyed by group ID.
	// IDs with no matching group are absent from the map.
	CustomerGroupNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}
