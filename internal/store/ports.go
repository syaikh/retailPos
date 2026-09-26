package store

import (
	"context"

	"retail-pos-system/internal/shared"
)

// Consumer-side ports used to compute a store's onboarding readiness. Each is
// owned by the module that owns the underlying table (ADR Modular Monolith
// Module Boundaries §2.8); internal/store is hardened to stores/warehouses
// only (internal/archtest), so readiness reads never join users, products or
// storage_locations directly. The composition root (internal/wiring) MUST wire
// all three before GET /stores/:id/readiness runs — an unwired repository
// fails fast at the readiness load point.

// StaffCountsProvider is the user-owned port that reports how many active
// accounts a store has per role. internal/user owns users/roles (platform).
type StaffCountsProvider interface {
	StaffCountsByStore(ctx context.Context, db shared.DBPool, storeID int) (map[string]int, error)
}

// SellableStockProvider is the product-owned port that reports catalog
// health: how many active products exist and how many of them have no
// sellable stock left. internal/product owns products/v_products_full
// (katalog).
type SellableStockProvider interface {
	SellableStockStats(ctx context.Context, db shared.DBPool) (activeProducts int, zeroStockProducts int, err error)
}

// StorageLocationCountProvider is the storagelocation-owned port that counts
// a store's active storage locations (referensi).
type StorageLocationCountProvider interface {
	StorageLocationCountByStore(ctx context.Context, db shared.DBPool, storeID int) (int, error)
}
