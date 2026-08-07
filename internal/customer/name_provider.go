package customer

import (
	"context"

	"retail-pos-system/internal/shared"
)

// CustomerNameLookup is the customer-owned implementation of the sale module's
// consumer-side port (sale.CustomerNameProvider, structural typing — no import
// of internal/sale needed). internal/customer is the canonical owner of the
// customers table (ADR Modular_Monolith_Module_Boundaries §2.8 Referensi), so
// the name lookups that internal/sale uses for listing/detail/export enrichment
// and free-text search resolution are computed here rather than via direct SQL
// inside internal/sale.
type CustomerNameLookup struct{}

// CustomerNamesByIDs returns customer names keyed by customer ID. IDs with no
// matching customer are absent from the map.
func (CustomerNameLookup) CustomerNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM customers
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

// CustomerIDsByName returns the IDs of customers whose name ILIKE-matches the
// given search pattern (caller supplies the '%' pattern).
func (CustomerNameLookup) CustomerIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error) {
	rows, err := db.Query(ctx, `
		SELECT id
		FROM customers
		WHERE name ILIKE $1
	`, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
