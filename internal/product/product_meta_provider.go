package product

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// ProductMetaLookup is the product-owned implementation of the inventory
// module's consumer-side port (inventory.ProductMetaProvider, structural
// typing — no import of internal/inventory needed). internal/product is the
// canonical owner of the products table
// (ADR_Modular_Monolith_Module_Boundaries §2.8 Katalog), so the sku/name
// lookups that internal/inventory uses for rack-stock listing enrichment are
// computed here rather than via a cross-context JOIN inside internal/inventory.
type ProductMetaLookup struct{}

// ProductMetasByIDs returns product identity rows (sku/name) keyed by product
// ID. IDs with no matching product are absent from the map.
func (ProductMetaLookup) ProductMetasByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductMeta, error) {
	if len(ids) == 0 {
		return map[int]shared.ProductMeta{}, nil
	}
	metas := make(map[int]shared.ProductMeta, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, COALESCE(sku, ''), COALESCE(name, '')
		FROM products
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list product meta: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var m shared.ProductMeta
		if err := rows.Scan(&id, &m.SKU, &m.Name); err != nil {
			return nil, fmt.Errorf("failed to scan product meta: %w", err)
		}
		metas[id] = m
	}
	return metas, rows.Err()
}
