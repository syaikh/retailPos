package storagelocation

import (
	"context"

	"retail-pos-system/internal/shared"
)

// ExistenceProvider is the consumer-side port that resolves existence of
// stores and warehouses, both owned by the referensi bounded context
// (internal/store, ADR Modular_Monolith_Module_Boundaries §2.8 Referensi).
// storage_locations validates store/warehouse references at create/update, so
// storagelocation routes those checks here instead of direct SELECT COUNT(*)
// over stores/warehouses. The composition root MUST wire it via
// SetStoreExistenceProvider before any create/update path runs — an unwired
// repository fails fast at runtime.
type ExistenceProvider interface {
	StoreExists(ctx context.Context, db shared.DBPool, storeID int) (bool, error)
	WarehouseExists(ctx context.Context, db shared.DBPool, warehouseID int) (bool, error)
}
