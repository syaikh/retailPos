package stockopname

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- location-scoped stock opnames ---

// GetLocationScope returns the warehouse and store a storage location belongs
// to. It errors when the location does not exist or is inactive.
func (r *Repository) GetLocationScope(ctx context.Context, q queryer, locationID int) (*int, *int, error) {
	var warehouseID, storeID *int
	var isActive bool
	err := q.QueryRow(ctx, `
		SELECT warehouse_id, store_id, is_active FROM storage_locations WHERE id = $1
	`, locationID).Scan(&warehouseID, &storeID, &isActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrLocationNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load storage location scope: %w", err)
	}
	if !isActive {
		return nil, nil, ErrLocationInactive
	}
	return warehouseID, storeID, nil
}

// LoadSnapshotProductsByLocation returns the rack-stock snapshot for the given
// active product ids on a specific storage location. Products without a rack
// row are included with expected quantity 0.
func (r *Repository) LoadSnapshotProductsByLocation(ctx context.Context, q queryer, locationID int, ids []int) ([]SessionItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT ps.product_id, p.name, p.sku, COALESCE(p.barcode, ''), COALESCE(u.name, 'pcs'),
		       COALESCE(ps.quantity, 0)
		FROM products p
		LEFT JOIN product_stock ps ON ps.product_id = p.id AND ps.location_id = $2
		LEFT JOIN units_of_measure u ON u.id = p.unit_of_measure_id
		WHERE p.id = ANY($1::int[]) AND p.status = 'active' AND p.deleted_at IS NULL
		ORDER BY p.name ASC
	`, ids, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to load location snapshot products: %w", err)
	}
	defer rows.Close()
	var items []SessionItem
	for rows.Next() {
		var it SessionItem
		if err := rows.Scan(&it.ProductID, &it.ProductName, &it.SKU, &it.Barcode, &it.UOMName, &it.OpeningQty); err != nil {
			return nil, fmt.Errorf("failed to scan location snapshot product: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// LockStockForLocation locks the rack stock rows of the given products on a
// location and returns the current rack quantities (0 when no rack row exists).
func (r *Repository) LockStockForLocation(ctx context.Context, tx pgx.Tx, productIDs []int, locationID int) (map[int]int, error) {
	stock := make(map[int]int, len(productIDs))
	if len(productIDs) == 0 {
		return stock, nil
	}
	for _, pid := range productIDs {
		stock[pid] = 0
	}
	rows, err := tx.Query(ctx, `
		SELECT product_id, quantity FROM product_stock
		WHERE product_id = ANY($1::int[]) AND location_id = $2
		FOR UPDATE
	`, productIDs, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock location stock: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			return nil, fmt.Errorf("failed to scan location stock: %w", err)
		}
		stock[pid] = qty
	}
	return stock, rows.Err()
}
