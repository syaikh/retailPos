package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StoreNamesProvider is the store-owned implementation of the shift module's
// consumer-side port (shift.StoreNameProvider, structural typing — no import
// of internal/shift needed). internal/store is the canonical owner of the
// stores table (ADR Modular_Monolith_Module_Boundaries §2.8 Referensi), so
// shift listing/detail reads resolve store names here rather than via a direct
// JOIN on stores.
type StoreNamesProvider struct{}

// StoreNamesByIDs returns a map of store id -> name for the given ids. IDs
// without a store row (e.g. deleted) are absent from the result map.
func (StoreNamesProvider) StoreNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM stores
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

// WarehouseNamesProvider is the store-owned implementation of the stockopname
// module's consumer-side scope-name read (stockopname.ScopeNameResolver,
// structural typing — no import of internal/stockopname needed). internal/store
// owns the warehouses table (ADR §2.8 Referensi), so warehouse scope names are
// resolved here rather than via a correlated subquery inside internal/stockopname.
type WarehouseNamesProvider struct{}

// WarehouseNamesByIDs returns a map of warehouse id -> name for the given ids.
// IDs without a warehouse row are absent from the result map.
func (WarehouseNamesProvider) WarehouseNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM warehouses
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

// WarehouseStoreIDProvider is the store-owned implementation of the stockopname
// module's consumer-side warehouse->store read
// (stockopname.WarehouseStoreIDProvider, structural typing). Warehouse scope
// sessions derive their store_id from the warehouse's linked store, resolved
// here instead of a direct warehouses query inside internal/stockopname.
type WarehouseStoreIDProvider struct{}

// WarehouseStoreID returns the store_id linked to a warehouse, or nil when the
// warehouse does not exist or has no linked store.
func (WarehouseStoreIDProvider) WarehouseStoreID(ctx context.Context, db shared.DBPool, warehouseID int) (*int, error) {
	var storeID *int
	err := db.QueryRow(ctx, `SELECT store_id FROM warehouses WHERE id = $1`, warehouseID).Scan(&storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return storeID, nil
}
