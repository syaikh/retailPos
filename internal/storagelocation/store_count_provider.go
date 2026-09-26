package storagelocation

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// StoreLocationCount is the storagelocation-owned implementation of the store
// module's consumer-side StorageLocationCountProvider port (structural typing
// — no import of internal/store needed). internal/storagelocation is the
// canonical single-writer of storage_locations (ADR
// Modular_Monolith_Module_Boundaries §2.8 Referensi), so the onboarding
// readiness check counts a store's locations here.
type StoreLocationCount struct{}

// StorageLocationCountByStore returns the number of active storage locations
// assigned to a store. A store needs at least one so store-scoped stock
// adjustments have somewhere to land.
func (StoreLocationCount) StorageLocationCountByStore(ctx context.Context, db shared.DBPool, storeID int) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM storage_locations
		WHERE store_id = $1 AND is_active = true`, storeID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count storage locations by store: %w", err)
	}
	return count, nil
}
