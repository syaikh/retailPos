package customergroup

import (
	"context"

	"retail-pos-system/internal/shared"
)

// CustomerGroupNameLookup is the customer-group-owned implementation of the
// customer module's consumer-side port (customer.CustomerGroupNameProvider,
// structural typing — no import of internal/customer needed). internal/customergroup
// is the canonical owner of the customer_groups table (ADR
// Modular_Monolith_Module_Boundaries §2.8 Referensi), so the group-name lookups
// that internal/customer uses to enrich customer rows are computed here rather
// than via a direct JOIN inside internal/customer.
type CustomerGroupNameLookup struct{}

// CustomerGroupNamesByIDs returns customer-group names keyed by group ID. IDs
// with no matching group are absent from the map.
func (CustomerGroupNameLookup) CustomerGroupNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM customer_groups
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}
