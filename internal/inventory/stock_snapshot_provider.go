package inventory

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// StockSnapshotProvider is the inventory-owned implementation of the stock
// opname module's consumer-side port (stockopname.StockSnapshotProvider,
// structural typing — no import of internal/stockopname needed).
// internal/inventory is the canonical owner of product_stock
// (ADR_Modular_Monolith_Module_Boundaries §2.8 transaksional), so the
// warehouse/location scope product lookups and the snapshot stock-quantity
// reads used when building stock opname snapshots are computed here rather
// than via cross-context SELECTs inside internal/stockopname.
type StockSnapshotProvider struct{}

// ScopeProductIDs returns the product universe of a stock-scoped stock opname
// scope (warehouse/location), read from product_stock. Other scope types are
// resolved by internal/product and are not handled here (returns nil, nil).
func (StockSnapshotProvider) ScopeProductIDs(ctx context.Context, db shared.DBPool, scopeType string, scopeID int64) ([]int, error) {
	var query string
	switch scopeType {
	case "warehouse":
		query = `SELECT DISTINCT product_id FROM product_stock WHERE warehouse_id = $1`
	case "location":
		query = `SELECT DISTINCT product_id FROM product_stock WHERE location_id = $1`
	default:
		return nil, nil
	}
	rows, err := db.Query(ctx, query, scopeID)
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

// SnapshotQuantities returns product_stock quantities for the given product
// ids, keyed by product id. Only products WITH a matching stock row are
// present in the map so consumers can preserve join semantics (the global
// snapshot inner-joins product_stock while the rack snapshot left-joins and
// defaults to 0). A nil locationID reads the global stock row
// (warehouse/store/location NULL); a non-nil locationID reads the rack row.
func (StockSnapshotProvider) SnapshotQuantities(ctx context.Context, db shared.DBPool, ids []int, locationID *int) (map[int]int, error) {
	stock := make(map[int]int, len(ids))
	if len(ids) == 0 {
		return stock, nil
	}
	var query string
	var args []interface{}
	if locationID == nil {
		query = `
			SELECT product_id, quantity FROM product_stock
			WHERE product_id = ANY($1::int[]) AND warehouse_id IS NULL AND store_id IS NULL`
		args = []interface{}{ids}
	} else {
		query = `
			SELECT product_id, quantity FROM product_stock
			WHERE product_id = ANY($1::int[]) AND location_id = $2`
		args = []interface{}{ids, *locationID}
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot quantities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot quantity: %w", err)
		}
		stock[pid] = qty
	}
	return stock, rows.Err()
}
