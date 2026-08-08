package customergroup

import (
	"context"

	"retail-pos-system/internal/shared"
)

// CustomerCountProvider resolves the number of customers per customer group.
// The customergroup module does not own the customers table, so it reads
// customer counts through this port instead of querying the table directly.
type CustomerCountProvider interface {
	// CustomerGroupCounts returns customer counts keyed by customer group ID.
	// Customer groups with no customers are absent from the map.
	CustomerGroupCounts(ctx context.Context, db shared.DBPool) (map[int]int, error)
}
