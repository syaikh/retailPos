package consignment

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// UsageProvider is the consignment-owned implementation of the supplier
// module's consumer-side ConsignmentUsageCounter port (structural typing — no
// import of internal/supplier needed). internal/consignment owns
// consignment_arrangements and consignment_stock, so whether a supplier is still
// tied to live consignment is answered here rather than via direct SQL inside
// internal/supplier.
type UsageProvider struct{}

// CountActiveConsignmentsBySupplier counts the supplier's arrangements that are
// still live: an active arrangement, or an ended one that still holds
// consignment stock (available or awaiting return) for the supplier. The stock
// half matters because ending an arrangement does not necessarily clear its
// stock, and a supplier with unsettled consignment stock cannot be retired even
// though no arrangement row is active.
func (UsageProvider) CountActiveConsignmentsBySupplier(ctx context.Context, db shared.DBPool, supplierID int) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM consignment_arrangements a
		WHERE a.supplier_id = $1
		  AND (
		    a.status = $2
		    OR EXISTS (
		      SELECT 1 FROM consignment_stock s
		      WHERE s.arrangement_id = a.id
		        AND (s.available_qty > 0 OR s.pending_return_qty > 0)
		    )
		  )
	`, supplierID, StatusActive).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active consignments by supplier: %w", err)
	}
	return count, nil
}
