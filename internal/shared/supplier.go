package shared

import "errors"

// SupplierRef is the id/name identity of a supplier, used by consumers that
// render supplier pickers/labels without importing internal/supplier. The
// IsConsignment flag lets the consignment module render only flagged suppliers.
type SupplierRef struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	IsConsignment bool   `json:"is_consignment"`
}

// ErrProductSupplierNotFound is returned when a product-supplier link row does
// not exist. It lives in shared so that internal/supplier (consumer) and
// internal/product (single-writer of product_suppliers) can reference the same
// sentinel without importing each other (both are isolated modules).
var ErrProductSupplierNotFound = errors.New("product-supplier link not found")

// ProductSupplier is a row of the product_suppliers link table. It is the
// cross-module contract between internal/supplier (consumer) and
// internal/product (single-writer of product_suppliers, see
// ADR_Modular_Monolith_Module_Boundaries §2.8 Katalog). The Joined fields are
// enrichment populated by whichever side owns the joined table:
// SupplierName/SupplierCode by internal/supplier on its own suppliers table,
// ProductName/ProductSKU by internal/product on products.
type ProductSupplier struct {
	ID           int     `json:"id"             db:"id"`
	ProductID    int     `json:"product_id"     db:"product_id"     validate:"gt=0"`
	SupplierID   int     `json:"supplier_id"    db:"supplier_id"    validate:"gt=0"`
	SupplierSKU  *string `json:"supplier_sku,omitempty" db:"supplier_sku"`
	UnitCost     int     `json:"unit_cost"      db:"unit_cost"      validate:"gte=0"`
	LeadTimeDays int     `json:"lead_time_days" db:"lead_time_days" validate:"gte=0"`
	IsPreferred  bool    `json:"is_preferred"   db:"is_preferred"`
	Notes        *string `json:"notes,omitempty" db:"notes"`
	CreatedAt    string  `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt    string  `json:"updated_at,omitempty" db:"updated_at"`

	// Joined fields (populated on read with JOIN queries)
	SupplierName *string `json:"supplier_name,omitempty" db:"supplier_name"`
	SupplierCode *string `json:"supplier_code,omitempty" db:"supplier_code"`
	ProductName  *string `json:"product_name,omitempty"  db:"product_name"`
	ProductSKU   *string `json:"product_sku,omitempty"   db:"product_sku"`
}
