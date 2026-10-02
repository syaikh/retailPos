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
}
