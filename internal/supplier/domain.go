package supplier

import (
	"errors"
	"time"
)

var (
	ErrSupplierNotFound      = errors.New("supplier not found")
	ErrSupplierCodeExists    = errors.New("supplier code already exists")
	ErrInvalidSupplier       = errors.New("invalid supplier data")
	ErrProductSupplierExists = errors.New("product-supplier link already exists")
	ErrProductSupplierNotFound = errors.New("product-supplier link not found")
	ErrMultiplePreferred     = errors.New("only one preferred supplier allowed per product")
)

// Supplier represents a supplier entity. See ADR-003.
type Supplier struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	ContactName *string    `json:"contact_name,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Email       *string    `json:"email,omitempty"`
	Address     *string    `json:"address,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	IsActive    bool       `json:"is_active"`
	StoreID     *int       `json:"store_id,omitempty"`
	CreatedAt   string     `json:"created_at,omitempty"`
	UpdatedAt   string     `json:"updated_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// ProductSupplier represents the many-to-many relationship between
// products and suppliers with per-supplier-per-product metadata.
// See ADR-003.
type ProductSupplier struct {
	ID           int     `json:"id"`
	ProductID    int     `json:"product_id"`
	SupplierID   int     `json:"supplier_id"`
	SupplierSKU  *string `json:"supplier_sku,omitempty"`
	UnitCost     int     `json:"unit_cost"`
	LeadTimeDays int     `json:"lead_time_days"`
	IsPreferred  bool    `json:"is_preferred"`
	Notes        *string `json:"notes,omitempty"`
	CreatedAt    string  `json:"created_at,omitempty"`
	UpdatedAt    string  `json:"updated_at,omitempty"`

	// Joined fields (populated on read with JOIN queries)
	SupplierName *string `json:"supplier_name,omitempty"`
	SupplierCode *string `json:"supplier_code,omitempty"`
	ProductName  *string `json:"product_name,omitempty"`
	ProductSKU   *string `json:"product_sku,omitempty"`
}
