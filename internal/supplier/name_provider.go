package supplier

import (
	"context"

	"retail-pos-system/internal/shared"
)

// SupplierNamesProvider is the supplier-owned implementation of the stockopname
// module's consumer-side scope-name read (stockopname.ScopeNameResolver,
// structural typing — no import of internal/stockopname needed). internal/supplier
// owns the suppliers table (ADR §2.8 Referensi), so supplier scope names are
// resolved here rather than via a correlated subquery inside internal/stockopname.
type SupplierNamesProvider struct{}

// SupplierNamesByIDs returns a map of supplier id -> name for the given ids.
// IDs without a supplier row are absent from the result map.
func (SupplierNamesProvider) SupplierNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM suppliers
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
