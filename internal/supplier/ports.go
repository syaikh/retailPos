package supplier

import (
	"context"

	"retail-pos-system/internal/shared"
)

// ProductSupplierStore is the consumer-side port for the product_suppliers
// link table (katalog-owned, see ADR_Modular_Monolith_Module_Boundaries §2.8).
// internal/supplier does not own that table, so every read and write is
// delegated to the product-owned implementation; the composition root MUST wire
// it via Repository.SetProductSupplierStore before the repository is used,
// otherwise the affected methods fail fast.
//
// storeID is the caller's resolved scope (audit D3 Option C, migration
// 058_supplier_terms_store_scope.sql): nil is unrestricted (superadmin),
// non-nil is one store. A link row's store_id holds the store whose commercial
// terms it carries and NULL means global terms every store inherits. See the
// store-scoping contract on product.SupplierLinkStore for the read/write
// asymmetry: reads see global plus own store, writes touch only the caller's
// own row so one store cannot reprice another's terms.
type ProductSupplierStore interface {
	// CreateLink inserts a new product-supplier link row.
	CreateLink(ctx context.Context, db shared.DBPool, ps *ProductSupplier) error
	// DeleteLink removes the caller's own product-supplier link row.
	DeleteLink(ctx context.Context, db shared.DBPool, productID, supplierID int, storeID *int) error
	// GetLink returns a single link row visible in the caller's scope, or
	// shared.ErrProductSupplierNotFound.
	GetLink(ctx context.Context, db shared.DBPool, productID, supplierID int, storeID *int) (*ProductSupplier, error)
	// GetPreferredLink returns the preferred link row in effect for the caller's
	// scope, or shared.ErrProductSupplierNotFound when none is preferred.
	GetPreferredLink(ctx context.Context, db shared.DBPool, productID int, storeID *int) (*ProductSupplier, error)
	// SetPreferredLink makes the given link the product's single preferred one for
	// the caller's own store (clearing that store's previous choice first).
	SetPreferredLink(ctx context.Context, db shared.DBPool, productID, supplierID int, storeID *int) error
	// UpdateLink updates the per-supplier metadata of an existing link row.
	UpdateLink(ctx context.Context, db shared.DBPool, ps *ProductSupplier) error
	// ListLinksByProduct returns the link rows of a product in effect for the
	// caller's scope, ordered by is_preferred DESC. Supplier enrichment
	// (name/code) is the consumer's responsibility on its own suppliers table.
	ListLinksByProduct(ctx context.Context, db shared.DBPool, productID int, storeID *int) ([]ProductSupplier, error)
	// ListLinksBySupplier returns the link rows of a supplier in effect for the
	// caller's scope, with the joined product name/SKU (both product_suppliers
	// and products are katalog-owned).
	ListLinksBySupplier(ctx context.Context, db shared.DBPool, supplierID int, storeID *int) ([]ProductSupplier, error)
	// HasPreferredLink reports whether a preferred supplier is in effect for the
	// caller's scope.
	HasPreferredLink(ctx context.Context, db shared.DBPool, productID int, storeID *int) (bool, error)
	// CountLinksBySupplier counts every product_suppliers row that references the
	// supplier, across every store. Unlike the list methods it takes no store
	// scope: product_suppliers has no soft-delete column and a link in any store
	// is a live reference the delete guard must not silently cascade away.
	CountLinksBySupplier(ctx context.Context, db shared.DBPool, supplierID int) (int, error)
}

// PurchaseUsageCounter is the purchase-owned port that answers how many open
// purchase orders still depend on a supplier, implemented by internal/purchase
// (structural typing — no import of internal/purchase needed). "Open" is the
// purchase module's definition: draft, confirmed, and partial_received orders
// still owe goods and name a supplier; fully_received and cancelled are
// terminal.
type PurchaseUsageCounter interface {
	CountOpenPurchaseOrdersBySupplier(ctx context.Context, db shared.DBPool, supplierID int) (int, error)
}

// ConsignmentUsageCounter is the consignment-owned port that answers how many
// live consignment arrangements still depend on a supplier, implemented by
// internal/consignment (structural typing). An arrangement is live when it is
// active or still holds stock, because a supplier tied to unsettled consignment
// stock cannot be retired even after the arrangement row was ended.
type ConsignmentUsageCounter interface {
	CountActiveConsignmentsBySupplier(ctx context.Context, db shared.DBPool, supplierID int) (int, error)
}

// SupplierUsage is the cross-module reference breakdown the delete and
// deactivate guards report. Each field is owned by a different module and is
// answered through the counters above; internal/supplier owns only suppliers,
// so it cannot run these counts itself (internal/archtest enforces that).
type SupplierUsage struct {
	ProductLinks       int `json:"product_links"`
	OpenPurchaseOrders int `json:"open_purchase_orders"`
	ActiveConsignments int `json:"active_consignments"`
}

// Total is the count that blocks a soft delete: any product link, open purchase
// order, or active consignment arrangement makes the supplier unsafe to remove.
func (u SupplierUsage) Total() int {
	return u.ProductLinks + u.OpenPurchaseOrders + u.ActiveConsignments
}

// InFlight is the count that blocks deactivation. Product links do not block it
// — a link is reversible and an inactive supplier is hidden from new selection
// — but an open purchase order or a live consignment arrangement names work the
// supplier is still owed, so deactivating would strand an in-flight transaction.
func (u SupplierUsage) InFlight() int {
	return u.OpenPurchaseOrders + u.ActiveConsignments
}

// Empty reports whether the supplier has no live references at all.
func (u SupplierUsage) Empty() bool {
	return u.Total() == 0
}
