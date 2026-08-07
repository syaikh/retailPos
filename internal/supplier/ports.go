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
type ProductSupplierStore interface {
	// CreateLink inserts a new product-supplier link row.
	CreateLink(ctx context.Context, db shared.DBPool, ps *ProductSupplier) error
	// DeleteLink removes the product-supplier link row.
	DeleteLink(ctx context.Context, db shared.DBPool, productID, supplierID int) error
	// GetLink returns a single link row, or shared.ErrProductSupplierNotFound.
	GetLink(ctx context.Context, db shared.DBPool, productID, supplierID int) (*ProductSupplier, error)
	// GetPreferredLink returns the preferred link row for a product, or
	// shared.ErrProductSupplierNotFound when none is preferred.
	GetPreferredLink(ctx context.Context, db shared.DBPool, productID int) (*ProductSupplier, error)
	// SetPreferredLink makes the given link the product's single preferred one
	// (clearing any other preferred row for the product first).
	SetPreferredLink(ctx context.Context, db shared.DBPool, productID, supplierID int) error
	// UpdateLink updates the per-supplier metadata of an existing link row.
	UpdateLink(ctx context.Context, db shared.DBPool, ps *ProductSupplier) error
	// ListLinksByProduct returns the raw link rows of a product ordered by
	// is_preferred DESC. Supplier enrichment (name/code) is the consumer's
	// responsibility on its own suppliers table.
	ListLinksByProduct(ctx context.Context, db shared.DBPool, productID int) ([]ProductSupplier, error)
	// ListLinksBySupplier returns the link rows of a supplier with the joined
	// product name/SKU (both product_suppliers and products are katalog-owned).
	ListLinksBySupplier(ctx context.Context, db shared.DBPool, supplierID int) ([]ProductSupplier, error)
	// HasPreferredLink reports whether the product has a preferred supplier.
	HasPreferredLink(ctx context.Context, db shared.DBPool, productID int) (bool, error)
}
