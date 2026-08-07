package inventory

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"retail-pos-system/internal/shared"
)

type queryer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ListLocationStock returns rack-level stock rows, optionally filtered by a
// single product and/or a single location. The rack rows are read from the
// owned product_stock table and then enriched with storage_locations and
// products metadata through consumer-side ports
// (LocationRackProvider/ProductMetaProvider) instead of cross-context JOINs.
func (r *Repository) ListLocationStock(ctx context.Context, productID, locationID int) ([]LocationStockItem, error) {
	query := `
		SELECT ps.product_id, ps.location_id, ps.quantity
		FROM product_stock ps
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
	query += ` ORDER BY ps.location_id ASC, ps.product_id ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list location stock: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		ProductID  int
		LocationID int
		Quantity   int
	}
	var raws []rawRow
	locSet := map[int]struct{}{}
	prodSet := map[int]struct{}{}
	for rows.Next() {
		var rw rawRow
		if err := rows.Scan(&rw.ProductID, &rw.LocationID, &rw.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan location stock: %w", err)
		}
		raws = append(raws, rw)
		locSet[rw.LocationID] = struct{}{}
		prodSet[rw.ProductID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list location stock: %w", err)
	}

	racks, err := r.locationProvider().RacksByIDs(ctx, r.db, sortedKeys(locSet))
	if err != nil {
		return nil, err
	}
	metas, err := r.productMetaProvider().ProductMetasByIDs(ctx, r.db, sortedKeys(prodSet))
	if err != nil {
		return nil, err
	}
	rackByID := make(map[int]LocationRack, len(racks))
	for _, rack := range racks {
		rackByID[rack.ID] = rack
	}

	items := make([]LocationStockItem, 0, len(raws))
	for _, rw := range raws {
		rack, ok := rackByID[rw.LocationID]
		if !ok {
			rack = LocationRack{}
		}
		meta, ok := metas[rw.ProductID]
		if !ok {
			meta = shared.ProductMeta{}
		}
		items = append(items, LocationStockItem{
			ProductID:    rw.ProductID,
			SKU:          meta.SKU,
			Name:         meta.Name,
			LocationID:   rw.LocationID,
			LocationCode: rack.Code,
			LocationName: rack.Name,
			Quantity:     rw.Quantity,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LocationName != items[j].LocationName {
			return items[i].LocationName < items[j].LocationName
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func sortedKeys(set map[int]struct{}) []int {
	keys := make([]int, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// LoadLocationForStock returns the rack metadata for writing rack stock rows.
// The caller must check IsActive before writing. The storage_locations read is
// delegated to the LocationRackProvider (internal/storagelocation).
func (r *Repository) LoadLocationForStock(ctx context.Context, locationID int) (*LocationRack, error) {
	return r.locationProvider().GetRack(ctx, r.db, locationID)
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
