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

// ProductCostsByIDs returns product unit costs keyed by product ID. IDs with
// no matching product are absent from the map. Cost is governed by the
// product.cost.view permission on display paths; this provider serves backend
// business reads (e.g. stock opname posting computes adjustment line totals
// from the product cost).
func (ProductMetaLookup) ProductCostsByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]int, error) {
	if len(ids) == 0 {
		return map[int]int{}, nil
	}
	costs := make(map[int]int, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, COALESCE(cost, 0)
		FROM products
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list product costs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, cost int
		if err := rows.Scan(&id, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan product cost: %w", err)
		}
		costs[id] = cost
	}
	return costs, rows.Err()
}

// ScopeProductIDs returns the active product universe covered by a
// product-scoped stock opname scope (store/category/brand/supplier/product, or
// "manual" for every active product). Warehouse/location scopes are resolved
// from product_stock by internal/inventory, not here. Scope IDs ≤ 0 yield an
// empty result (products.supplier membership lives in product_suppliers, also
// owned by internal/product).
func (ProductMetaLookup) ScopeProductIDs(ctx context.Context, db shared.DBPool, scopeType string, scopeID int64) ([]int, error) {
	var query string
	var args []interface{}
	switch scopeType {
	case "store":
		query = `SELECT id FROM products WHERE store_id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "category":
		query = `SELECT id FROM products WHERE category_id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "brand":
		query = `SELECT id FROM products WHERE brand_id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "supplier":
		query = `SELECT DISTINCT product_id FROM product_suppliers WHERE supplier_id = $1`
	case "product":
		query = `SELECT id FROM products WHERE id = $1 AND deleted_at IS NULL AND status = 'active'`
	case "manual":
		query = `SELECT id FROM products WHERE deleted_at IS NULL AND status = 'active'`
	default:
		return nil, nil
	}
	if scopeType != "manual" {
		args = append(args, scopeID)
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scope products for %s #%d: %w", scopeType, scopeID, err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan scope product: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SnapshotProducts returns the active product catalog rows of the stock
// opname snapshot read-model, ordered by name. When ids is empty all active
// products are returned. Stock quantities are NOT included — they are owned
// by internal/inventory (product_stock) and merged by the consumer.
func (ProductMetaLookup) SnapshotProducts(ctx context.Context, db shared.DBPool, ids []int) ([]shared.SnapshotProduct, error) {
	query := `
		SELECT p.id, p.name, p.sku, COALESCE(p.barcode, ''), p.unit_of_measure_id
		FROM products p
		WHERE p.deleted_at IS NULL AND p.status = 'active'`
	args := []interface{}{}
	if len(ids) > 0 {
		args = append(args, ids)
		query += ` AND p.id = ANY($1::int[])`
	}
	query += ` ORDER BY p.name ASC`
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot products: %w", err)
	}
	defer rows.Close()
	var out []shared.SnapshotProduct
	for rows.Next() {
		var p shared.SnapshotProduct
		var uomID *int
		if err := rows.Scan(&p.ProductID, &p.Name, &p.SKU, &p.Barcode, &uomID); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot product: %w", err)
		}
		p.UOMID = uomID
		out = append(out, p)
	}
	return out, rows.Err()
}
