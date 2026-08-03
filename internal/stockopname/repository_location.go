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

// UpdateLocationStock reconciles a product's rack row after a location-scoped
// stock opname posting. The rack row is set to its counted quantity (delta =
// physical - rack_old), and the global row is recomputed so the rack's share of
// global is replaced by the counted figure:
//
//	global_new = max(global_old - rack_old, 0) + rack_new   (rack_new = rack_old + delta)
//
// This keeps the global total correct even when rack and global drifted apart
// (sales deduct global only; set/transfer touch rack rows only), instead of
// blindly mirroring the rack delta onto global. The floor keeps the global row
// from going negative when a rack row was over-set without a matching global
// increase. Rack and global rows are created lazily when absent (starting at
// 0). warehouseID and storeID identify the rack's owning scope and are only
// used when the rack row has to be created.
func (r *Repository) UpdateLocationStock(ctx context.Context, tx pgx.Tx, productID, locationID int, warehouseID, storeID *int, delta int) error {
	if delta == 0 {
		return nil
	}

	// Lock and read both rows so the reconcile is race-free within the tx.
	var rackQty, globalQty int
	row := tx.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND location_id = $2
		FOR UPDATE
	`, productID, locationID)
	if err := row.Scan(&rackQty); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to lock location stock: %w", err)
	}
	row = tx.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
		FOR UPDATE
	`, productID)
	if err := row.Scan(&globalQty); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to lock global stock: %w", err)
	}

	newRack := rackQty + delta
	newGlobal := max(globalQty-rackQty, 0) + newRack

	tag, err := tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND location_id = $3
	`, newRack, productID, locationID)
	if err != nil {
		return fmt.Errorf("failed to update location stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, warehouse_id, store_id, location_id, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, productID, newRack, warehouseID, storeID, locationID)
		if err != nil {
			return fmt.Errorf("failed to insert location stock: %w", err)
		}
	}

	tag, err = tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, newGlobal, productID)
	if err != nil {
		return fmt.Errorf("failed to update global stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, NOW())
		`, productID, newGlobal)
		if err != nil {
			return fmt.Errorf("failed to insert global stock: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `UPDATE products SET stock = $1, updated_at = NOW() WHERE id = $2`, newGlobal, productID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock column: %w", err)
	}
	return nil
}
