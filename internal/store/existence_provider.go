package store

import (
	"context"
	"errors"
	"fmt"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

// ExistenceProvider is the store-owned implementation of the
// storagelocation module's consumer-side port
// (storagelocation.ExistenceProvider, structural typing — no import of
// internal/storagelocation needed). internal/store is the canonical owner of
// the stores and warehouses tables (ADR Modular_Monolith_Module_Boundaries §2.8
// Referensi), so storage-location store/warehouse reference validation is
// resolved here rather than via a direct SELECT COUNT(*) inside
// internal/storagelocation.
type ExistenceProvider struct{}

// StoreExists reports whether a store with the given id exists.
func (ExistenceProvider) StoreExists(ctx context.Context, db shared.DBPool, storeID int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE id = $1)`, storeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check store exists: %w", err)
	}
	return exists, nil
}

// WarehouseExists reports whether a warehouse with the given id exists.
func (ExistenceProvider) WarehouseExists(ctx context.Context, db shared.DBPool, warehouseID int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1)`, warehouseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check warehouse exists: %w", err)
	}
	return exists, nil
}

// WarehouseStoreID returns the store_id linked to a warehouse, or nil when the
// warehouse does not exist or has no linked store. Semantics match
// WarehouseStoreIDProvider (name_provider.go) so both ports agree.
func (ExistenceProvider) WarehouseStoreID(ctx context.Context, db shared.DBPool, warehouseID int) (*int, error) {
	var storeID *int
	err := db.QueryRow(ctx, `SELECT store_id FROM warehouses WHERE id = $1`, warehouseID).Scan(&storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve warehouse store id: %w", err)
	}
	return storeID, nil
}

// WarehouseIDsByStoreID returns every warehouse id linked to the given store.
func (ExistenceProvider) WarehouseIDsByStoreID(ctx context.Context, db shared.DBPool, storeID int) ([]int, error) {
	rows, err := db.Query(ctx, `SELECT id FROM warehouses WHERE store_id = $1`, storeID)
	if err != nil {
		return nil, fmt.Errorf("list warehouse ids by store: %w", err)
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan warehouse id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
