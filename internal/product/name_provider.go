package product

import (
	"context"

	"retail-pos-system/internal/shared"
)

// NameLookup is the product-owned implementation of the sale module's
// consumer-side port (sale.ProductNameProvider, structural typing — no import
// of internal/sale needed). internal/product is the canonical owner of the
// products table (ADR Modular_Monolith_Module_Boundaries §2.8 Katalog), so the
// name lookups that internal/sale uses for listing/detail/export enrichment
// and free-text search resolution are computed here rather than via direct
// SQL inside internal/sale.
type NameLookup struct{}

// ProductNamesByIDs returns product names keyed by product ID. IDs with no
// matching product are absent from the map.
func (NameLookup) ProductNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
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

// maxSearchIDs bounds how many product IDs a free-text search may resolve to.
// A leading-wildcard ILIKE (e.g. "%a%") matches a large fraction of the catalog,
// and the caller feeds the result into an id = ANY($n) filter — an uncapped
// resolution produces a pathologically large IN list. Truncating to the first
// maxSearchIDs (ordered by id, so the result is deterministic) keeps the
// downstream filter bounded; a search broad enough to hit the cap is already too
// imprecise to be useful, so the truncation is an acceptable trade-off.
const maxSearchIDs = 1000

// ProductIDsByName returns the IDs of products whose name ILIKE-matches the
// given search pattern (caller supplies the '%' pattern). At most maxSearchIDs
// IDs are returned (see maxSearchIDs).
func (NameLookup) ProductIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error) {
	rows, err := db.Query(ctx, `
		SELECT id
		FROM products
		WHERE name ILIKE $1
		ORDER BY id
		LIMIT $2
	`, search, maxSearchIDs)
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
