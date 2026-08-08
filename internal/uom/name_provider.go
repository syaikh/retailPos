package uom

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// UnitNameLookup is the uom-owned implementation of the stock opname module's
// consumer-side port (stockopname.UOMNameProvider, structural typing — no
// import of internal/stockopname needed). internal/uom is the canonical owner
// of the units_of_measure table (ADR Modular_Monolith_Module_Boundaries §2.8
// Katalog), so the unit-of-measure name lookups that stockopname uses when
// building stock snapshots are computed here rather than via a cross-context
// JOIN inside internal/stockopname.
type UnitNameLookup struct{}

// UnitNamesByIDs returns unit-of-measure names keyed by ID. IDs with no
// matching unit are absent from the map.
func (UnitNameLookup) UnitNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, COALESCE(name, '')
		FROM units_of_measure
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list unit names: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("failed to scan unit name: %w", err)
		}
		names[id] = name
	}
	return names, rows.Err()
}
