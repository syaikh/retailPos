package supplier

import (
	"errors"
	"time"
)

var (
	ErrSupplierNotFound      = errors.New("supplier not found")
	ErrSupplierCodeExists    = errors.New("supplier code already exists")
	ErrInvalidSupplier       = errors.New("invalid supplier data")
	ErrInvalidEmail          = errors.New("invalid email format")
	ErrInvalidPhone          = errors.New("invalid phone format")
	ErrProductSupplierExists = errors.New("product-supplier link already exists")
	ErrProductSupplierNotFound = errors.New("product-supplier link not found")
	ErrMultiplePreferred     = errors.New("only one preferred supplier allowed per product")
)

// Supplier represents a supplier entity. See ADR-003.
type Supplier struct {
	ID          int        `json:"id"           db:"id"`
	Name        string     `json:"name"         db:"name"         validate:"required"`
	Code        string     `json:"code"         db:"code"         validate:"required"`
	ContactName *string    `json:"contact_name,omitempty" db:"contact_name"`
	Phone       *string    `json:"phone,omitempty"        db:"phone"`
	Email       *string    `json:"email,omitempty"        db:"email"        validate:"omitempty,email"`
	Address     *string    `json:"address,omitempty"      db:"address"`
	Notes       *string    `json:"notes,omitempty"        db:"notes"`
	IsActive    bool       `json:"is_active"              db:"is_active"`
	StoreID     *int       `json:"store_id,omitempty"     db:"store_id"`
	CreatedAt   string     `json:"created_at,omitempty"   db:"created_at"`
	UpdatedAt   string     `json:"updated_at,omitempty"   db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"   db:"deleted_at"`
}

// ProductSupplier represents the many-to-many relationship between
// products and suppliers with per-supplier-per-product metadata.
// See ADR-003.
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
