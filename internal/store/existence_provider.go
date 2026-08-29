package store

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
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
