package storagelocation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// RackProvider is the storagelocation-owned implementation of the inventory
// module's consumer-side port (inventory.LocationRackProvider, structural
// typing — no import of internal/inventory needed). internal/storagelocation
// is the canonical single-writer of the storage_locations table
// (ADR_Modular_Monolith_Module_Boundaries §2.8 Referensi), so rack metadata
// reads for internal/inventory rack-stock bookkeeping are computed here rather
// than via a cross-context JOIN inside internal/inventory.
type RackProvider struct{}

// GetRack returns a single storage location's rack metadata, or
// shared.ErrLocationNotFound when it does not exist.
func (RackProvider) GetRack(ctx context.Context, db shared.DBPool, locationID int) (*shared.LocationRack, error) {
	var rack shared.LocationRack
	var warehouseID, storeID *int
	err := db.QueryRow(ctx, `
		SELECT id, code, name, warehouse_id, store_id, is_active
		FROM storage_locations WHERE id = $1
	`, locationID).Scan(&rack.ID, &rack.Code, &rack.Name, &warehouseID, &storeID, &rack.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrLocationNotFound
		}
		return nil, fmt.Errorf("failed to load storage location %d: %w", locationID, err)
	}
	rack.WarehouseID = warehouseID
	rack.StoreID = storeID
	return &rack, nil
}

// RacksByIDs returns rack metadata for the given location IDs. IDs with no
// matching location are absent from the result.
func (RackProvider) RacksByIDs(ctx context.Context, db shared.DBPool, ids []int) ([]shared.LocationRack, error) {
	if len(ids) == 0 {
		return []shared.LocationRack{}, nil
	}
	rows, err := db.Query(ctx, `
		SELECT id, code, name, warehouse_id, store_id, is_active
		FROM storage_locations
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage locations: %w", err)
	}
	defer rows.Close()

	var racks []shared.LocationRack
	for rows.Next() {
		var rack shared.LocationRack
		var warehouseID, storeID *int
		if err := rows.Scan(&rack.ID, &rack.Code, &rack.Name, &warehouseID, &storeID, &rack.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan storage location: %w", err)
		}
		rack.WarehouseID = warehouseID
		rack.StoreID = storeID
		racks = append(racks, rack)
	}
	return racks, rows.Err()
}
