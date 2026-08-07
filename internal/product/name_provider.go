package product

import (
	"context"

	"retail-pos-system/internal/shared"
)

// ProductNameLookup is the product-owned implementation of the sale module's
// consumer-side port (sale.ProductNameProvider, structural typing — no import
// of internal/sale needed). internal/product is the canonical owner of the
// products table (ADR Modular_Monolith_Module_Boundaries §2.8 Katalog), so the
// name lookups that internal/sale uses for listing/detail/export enrichment
// and free-text search resolution are computed here rather than via direct
// SQL inside internal/sale.
type ProductNameLookup struct{}

// ProductNamesByIDs returns product names keyed by product ID. IDs with no
// matching product are absent from the map.
func (ProductNameLookup) ProductNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM products
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

// ProductIDsByName returns the IDs of products whose name ILIKE-matches the
// given search pattern (caller supplies the '%' pattern).
func (ProductNameLookup) ProductIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error) {
	rows, err := db.Query(ctx, `
		SELECT id
		FROM products
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
