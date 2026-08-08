package customer

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// CustomerGroupCountsLookup is the customer-owned implementation of the
// customergroup module's CustomerCountProvider port.
type CustomerGroupCountsLookup struct{}

// CustomerGroupCounts returns the number of customers per customer group,
// keyed by group ID. Groups with no customers are absent from the map.
func (CustomerGroupCountsLookup) CustomerGroupCounts(ctx context.Context, db shared.DBPool) (map[int]int, error) {
	rows, err := db.Query(ctx, `
		SELECT customer_group_id, COUNT(*)
		FROM customers
		WHERE customer_group_id IS NOT NULL
		GROUP BY customer_group_id`)
	if err != nil {
		return nil, fmt.Errorf("count customers by customer group: %w", err)
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var id int
		var cnt int64
		if err := rows.Scan(&id, &cnt); err != nil {
			return nil, fmt.Errorf("scan customer group count: %w", err)
		}
		counts[id] = int(cnt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer group counts: %w", err)
	}
	return counts, nil
}
