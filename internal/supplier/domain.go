package supplier

import (
	"errors"
	"time"

	"retail-pos-system/internal/shared"
)

var (
	ErrSupplierNotFound      = errors.New("supplier not found")
	ErrSupplierCodeExists    = errors.New("supplier code already exists")
	ErrInvalidSupplier       = errors.New("invalid supplier data")
	ErrInvalidEmail          = errors.New("invalid email format")
	ErrInvalidPhone          = errors.New("invalid phone format")
	ErrProductSupplierExists = errors.New("product-supplier link already exists")
	ErrMultiplePreferred     = errors.New("only one preferred supplier allowed per product")
)

// ErrProductSupplierNotFound is the supplier-side alias for the shared sentinel
// so that both internal/supplier and the product_suppliers owner
// (internal/product) can reference it without importing each other.
var ErrProductSupplierNotFound = shared.ErrProductSupplierNotFound

// Supplier represents a supplier entity. See ADR-003.
type Supplier struct {
	ID            int        `json:"id"           db:"id"`
	Name          string     `json:"name"         db:"name"         validate:"required"`
	Code          string     `json:"code"         db:"code"         validate:"required"`
	ContactName   *string    `json:"contact_name,omitempty" db:"contact_name"`
	Phone         *string    `json:"phone,omitempty"        db:"phone"`
	Email         *string    `json:"email,omitempty"        db:"email"        validate:"omitempty,email"`
	Address       *string    `json:"address,omitempty"      db:"address"`
	Notes         *string    `json:"notes,omitempty"        db:"notes"`
	IsActive      bool       `json:"is_active"              db:"is_active"`
	IsConsignment bool       `json:"is_consignment"         db:"is_consignment"`
	StoreID       *int       `json:"store_id,omitempty"     db:"store_id"`
	CreatedAt     string     `json:"created_at,omitempty"   db:"created_at"`
	UpdatedAt     string     `json:"updated_at,omitempty"   db:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"   db:"deleted_at"`
}

// ProductSupplier represents the many-to-many relationship between
// products and suppliers with per-supplier-per-product metadata.
// See ADR-003. It is the supplier-side alias for the cross-module DTO owned by
// internal/shared (the product_suppliers table itself is katalog-owned and
// written through the product-supplied ProductSupplierStore port).
type ProductSupplier = shared.ProductSupplier
