package purchase

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// SupplierUsageProvider is the purchase-owned implementation of the supplier
// module's consumer-side PurchaseUsageCounter port (structural typing — no
// import of internal/supplier needed). internal/purchase owns purchase_orders,
// so whether a supplier is still named by an open order is answered here rather
// than via direct SQL inside internal/supplier.
type SupplierUsageProvider struct{}

// openPurchaseOrderStatuses are the statuses in which a purchase order still
// owes goods and therefore still depends on its supplier. fully_received and
// cancelled are terminal: the goods arrived or the order was abandoned, so the
// supplier can be retired without stranding anything.
var openPurchaseOrderStatuses = []string{
	StatusDraft,
	StatusConfirmed,
	StatusPartialReceived,
}

// CountOpenPurchaseOrdersBySupplier counts the supplier's open purchase orders.
func (SupplierUsageProvider) CountOpenPurchaseOrdersBySupplier(ctx context.Context, db shared.DBPool, supplierID int) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM purchase_orders
		WHERE supplier_id = $1 AND status = ANY($2)
	`, supplierID, openPurchaseOrderStatuses).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count open purchase orders by supplier: %w", err)
	}
	return count, nil
}
