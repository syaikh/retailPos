package supplier

import (
	"context"

	"retail-pos-system/internal/shared"
)

// ConsignmentSupplierProvider is the supplier-owned implementation of the
// consignment module's consumer-side SupplierStore port (structural typing — no
// import of internal/consignment needed). internal/supplier owns the suppliers
// table (ADR §2.8 Referensi), including the is_consignment flag added by the
// consignment migration, so consignment ownership questions about suppliers are
// answered here rather than via direct SQL inside internal/consignment.
type ConsignmentSupplierProvider struct{}

// IsConsignmentSupplier reports whether the supplier is flagged for consignment
// (is_consignment = true). A missing supplier row reports false with no error —
// the consignment module treats unknown suppliers as non-consignment.
func (ConsignmentSupplierProvider) IsConsignmentSupplier(ctx context.Context, db shared.DBPool, supplierID int) (bool, error) {
	var flagged bool
	err := db.QueryRow(ctx, `
		SELECT is_consignment FROM suppliers WHERE id = $1
	`, supplierID).Scan(&flagged)
	if err != nil {
		return false, err
	}
	return flagged, nil
}

// ListConsignmentSuppliers returns the id/name identity of every supplier
// flagged for consignment, ordered by name. The consignment module uses this to
// populate supplier pickers and to restrict arrangements to flagged suppliers.
func (ConsignmentSupplierProvider) ListConsignmentSuppliers(ctx context.Context, db shared.DBPool) ([]shared.SupplierRef, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, is_consignment
		FROM suppliers
		WHERE is_consignment = TRUE AND deleted_at IS NULL
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []shared.SupplierRef
	for rows.Next() {
		var ref shared.SupplierRef
		if err := rows.Scan(&ref.ID, &ref.Name, &ref.IsConsignment); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// SupplierNamesByIDs returns a map of supplier id -> name for the given ids.
// IDs without a supplier row are absent from the result map.
func (ConsignmentSupplierProvider) SupplierNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
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
