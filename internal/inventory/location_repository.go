package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type queryer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ListLocationStock returns rack-level stock rows, optionally filtered by a
// single product and/or a single location.
func (r *Repository) ListLocationStock(ctx context.Context, productID, locationID int) ([]LocationStockItem, error) {
	query := `
		SELECT ps.product_id, COALESCE(p.sku, ''), COALESCE(p.name, ''), ps.location_id,
		       COALESCE(sl.code, ''), COALESCE(sl.name, ''), ps.quantity
		FROM product_stock ps
		JOIN storage_locations sl ON sl.id = ps.location_id
		JOIN products p ON p.id = ps.product_id
		WHERE ps.location_id IS NOT NULL`
	args := []interface{}{}
	if productID > 0 {
		args = append(args, productID)
		query += fmt.Sprintf(" AND ps.product_id = $%d", len(args))
	}
	if locationID > 0 {
		args = append(args, locationID)
		query += fmt.Sprintf(" AND ps.location_id = $%d", len(args))
	}
	query += ` ORDER BY sl.name ASC, p.name ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list location stock: %w", err)
	}
	defer rows.Close()

	var items []LocationStockItem
	for rows.Next() {
		var it LocationStockItem
		if err := rows.Scan(&it.ProductID, &it.SKU, &it.Name, &it.LocationID, &it.LocationCode, &it.LocationName, &it.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan location stock: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// LoadLocationForStock returns the rack metadata for writing rack stock rows.
// The caller must check IsActive before writing.
func (r *Repository) LoadLocationForStock(ctx context.Context, locationID int) (*LocationRack, error) {
	return r.loadLocationForStock(ctx, r.db, locationID)
}

func (r *Repository) loadLocationForStock(ctx context.Context, q queryer, locationID int) (*LocationRack, error) {
	var rack LocationRack
	var warehouseID, storeID *int
	err := q.QueryRow(ctx, `
		SELECT id, code, name, warehouse_id, store_id, is_active
		FROM storage_locations WHERE id = $1
	`, locationID).Scan(&rack.ID, &rack.Code, &rack.Name, &warehouseID, &storeID, &rack.IsActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("failed to load storage location %d: %w", locationID, err)
	}
	rack.WarehouseID = warehouseID
	rack.StoreID = storeID
	return &rack, nil
}

// SetLocationStock records how much of a product sits in a rack, creating the
// rack row on first use. The rack row mirrors the rack's warehouse_id/store_id.
// Global stock is intentionally left unchanged (rack rows are bookkeeping).
func (r *Repository) SetLocationStock(ctx context.Context, productID, locationID, quantity, userID int) error {
	if quantity < 0 {
		return ErrNegativeQuantity
	}
	rack, err := r.LoadLocationForStock(ctx, locationID)
	if err != nil {
		return err
	}
	if !rack.IsActive {
		return ErrLocationInactive
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// quantity_change is a signed delta (matching every other movement type),
	// so read the current rack figure before overwriting it.
	var oldQty int
	err = tx.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND location_id = $2
		FOR UPDATE
	`, productID, locationID).Scan(&oldQty)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to read current location stock: %w", err)
	}

	if err := r.upsertLocationRow(ctx, tx, productID, locationID, rack.WarehouseID, rack.StoreID, quantity); err != nil {
		return err
	}
	notes := fmt.Sprintf("Set rack stock for product #%d at %s (%s) to %d", productID, rack.Code, rack.Name, quantity)
	if err := r.insertLocationMovement(ctx, tx, productID, locationID, quantity-oldQty, "location_set", userID, notes); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit location stock set: %w", err)
	}
	return nil
}

// TransferLocationStock moves quantity between two racks. Global stock is
// unchanged. Both rack rows are locked to keep concurrent transfers correct.
func (r *Repository) TransferLocationStock(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int) error {
	if quantity <= 0 {
		return ErrNonPositiveQuantity
	}
	if fromLocationID == toLocationID {
		return ErrSameLocation
	}
	fromRack, err := r.LoadLocationForStock(ctx, fromLocationID)
	if err != nil {
		return err
	}
	if !fromRack.IsActive {
		return ErrLocationInactive
	}
	toRack, err := r.LoadLocationForStock(ctx, toLocationID)
	if err != nil {
		return err
	}
	if !toRack.IsActive {
		return ErrLocationInactive
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fromQty, toQty, err := r.lockLocationRows(ctx, tx, productID, fromLocationID, toLocationID)
	if err != nil {
		return err
	}
	if fromQty < quantity {
		return ErrInsufficientLocationStock
	}

	if err := r.upsertLocationRow(ctx, tx, productID, fromLocationID, fromRack.WarehouseID, fromRack.StoreID, fromQty-quantity); err != nil {
		return err
	}
	if err := r.upsertLocationRow(ctx, tx, productID, toLocationID, toRack.WarehouseID, toRack.StoreID, toQty+quantity); err != nil {
		return err
	}
	fromNotes := fmt.Sprintf("Transfer out %d from %s (%s) to %s (%s)", quantity, fromRack.Code, fromRack.Name, toRack.Code, toRack.Name)
	toNotes := fmt.Sprintf("Transfer in %d to %s (%s) from %s (%s)", quantity, toRack.Code, toRack.Name, fromRack.Code, fromRack.Name)
	if err := r.insertLocationMovement(ctx, tx, productID, fromLocationID, -quantity, "location_transfer", userID, fromNotes); err != nil {
		return err
	}
	if err := r.insertLocationMovement(ctx, tx, productID, toLocationID, quantity, "location_transfer", userID, toNotes); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit location transfer: %w", err)
	}
	return nil
}

func (r *Repository) upsertLocationRow(ctx context.Context, tx queryer, productID, locationID int, warehouseID, storeID *int, quantity int) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO product_stock (product_id, warehouse_id, store_id, location_id, quantity, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT ON CONSTRAINT uq_product_stock
		DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = NOW()
	`, productID, warehouseID, storeID, locationID, quantity)
	if err != nil {
		return fmt.Errorf("failed to upsert location stock: %w", err)
	}
	return nil
}

func (r *Repository) lockLocationRows(ctx context.Context, tx queryer, productID, fromLocationID, toLocationID int) (int, int, error) {
	rows, err := tx.Query(ctx, `
		SELECT location_id, COALESCE(quantity, 0) FROM product_stock
		WHERE product_id = $1 AND (location_id = $2 OR location_id = $3)
		FOR UPDATE
	`, productID, fromLocationID, toLocationID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to lock location stock: %w", err)
	}
	defer rows.Close()

	fromQty, toQty := 0, 0
	for rows.Next() {
		var loc, qty int
		if err := rows.Scan(&loc, &qty); err != nil {
			return 0, 0, fmt.Errorf("failed to scan location stock: %w", err)
		}
		if loc == fromLocationID {
			fromQty = qty
		} else {
			toQty = qty
		}
	}
	return fromQty, toQty, rows.Err()
}

func (r *Repository) insertLocationMovement(ctx context.Context, tx queryer, productID, locationID, quantityChange int, mtype string, userID int, notes string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes)
		VALUES ($1, $2, $3, $4, 'storage_locations', $5, $6)
	`, productID, quantityChange, mtype, locationID, userID, notes)
	if err != nil {
		return fmt.Errorf("failed to record inventory movement: %w", err)
	}
	return nil
}
