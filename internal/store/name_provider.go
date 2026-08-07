package store

import (
	"context"

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
