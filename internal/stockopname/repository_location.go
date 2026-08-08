package stockopname

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// --- location-scoped stock opnames ---

// GetLocationScope returns the warehouse and store a storage location belongs
// to. It errors when the location does not exist or is inactive. The read is
// routed through the LocationScopeProvider port owned by internal/storagelocation.
func (r *Repository) GetLocationScope(ctx context.Context, db shared.DBPool, locationID int) (*int, *int, error) {
	if r.locationScopeProvider == nil {
		return nil, nil, errors.New("stockopname repository: location scope provider not wired; call SetLocationScopeProvider")
	}
	rack, err := r.locationScopeProvider.GetRack(ctx, db, locationID)
	if err != nil {
		if errors.Is(err, shared.ErrLocationNotFound) {
			return nil, nil, ErrLocationNotFound
		}
		return nil, nil, fmt.Errorf("failed to load storage location scope: %w", err)
	}
	if !rack.IsActive {
		return nil, nil, ErrLocationInactive
	}
	return rack.WarehouseID, rack.StoreID, nil
}

// LoadSnapshotProductsByLocation returns the rack-stock snapshot for the given
// active product ids on a specific storage location. Products without a rack
// row are included with expected quantity 0.
func (r *Repository) LoadSnapshotProductsByLocation(ctx context.Context, db shared.DBPool, locationID int, ids []int) ([]SessionItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := db.Query(ctx, `
		SELECT ps.product_id, p.name, p.sku, COALESCE(p.barcode, ''), p.unit_of_measure_id,
		       COALESCE(ps.quantity, 0)
		FROM products p
		LEFT JOIN product_stock ps ON ps.product_id = p.id AND ps.location_id = $2
		WHERE p.id = ANY($1::int[]) AND p.status = 'active' AND p.deleted_at IS NULL
		ORDER BY p.name ASC
	`, ids, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to load location snapshot products: %w", err)
	}
	defer rows.Close()
	var items []SessionItem
	var uomIDs []sql.NullInt64
	for rows.Next() {
		var it SessionItem
		var uomID sql.NullInt64
		if err := rows.Scan(&it.ProductID, &it.ProductName, &it.SKU, &it.Barcode, &uomID, &it.OpeningQty); err != nil {
			return nil, fmt.Errorf("failed to scan location snapshot product: %w", err)
		}
		items = append(items, it)
		uomIDs = append(uomIDs, uomID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.fillUOMNames(ctx, db, items, uomIDs); err != nil {
		return nil, err
	}
	return items, nil
}

// LockStockForLocation locks the rack stock rows of the given products on a
// location and returns the current rack quantities (0 when no rack row exists).
// The lock is taken inside the caller's tx via the StockLocker port owned by
// internal/inventory.
func (r *Repository) LockStockForLocation(ctx context.Context, tx pgx.Tx, productIDs []int, locationID int) (map[int]int, error) {
	if r.stockLocker == nil {
		return nil, errors.New("stockopname repository: stock locker not wired; call SetStockLocker")
	}
	return r.stockLocker.LockLocationStock(ctx, tx, productIDs, locationID)
}
