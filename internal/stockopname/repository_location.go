package stockopname

import (
	"context"
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
// LoadSnapshotProductsByLocation returns the rack-stock snapshot for the given
// active product ids on a specific storage location. Products without a rack
// row are included with expected quantity 0. The catalog rows come from the
// product-owned ProductCatalogProvider and the rack quantities from the
// inventory-owned StockSnapshotProvider.
func (r *Repository) LoadSnapshotProductsByLocation(ctx context.Context, db shared.DBPool, locationID int, ids []int) ([]SessionItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	catalog, err := r.snapshotCatalog(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	catalogIDs := make([]int, 0, len(catalog))
	for _, p := range catalog {
		catalogIDs = append(catalogIDs, p.ProductID)
	}
	quantities, err := r.snapshotQuantities(ctx, db, catalogIDs, &locationID)
	if err != nil {
		return nil, err
	}
	return r.buildSnapshotItems(ctx, db, catalog, quantities, true)
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
