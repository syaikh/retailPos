package storagelocation

import "errors"

// ErrStoreForbidden is returned when a store-scoped caller tries to read or
// write a storage location owned by a different store (or by a warehouse with
// no store). Handlers map it to HTTP 403. A nil caller store (superadmin)
// never triggers it.
var ErrStoreForbidden = errors.New("storage location is not in your store")

type StorageLocation struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	WarehouseID *int   `json:"warehouse_id,omitempty"`
	StoreID     *int   `json:"store_id,omitempty"`
	Notes       string `json:"notes,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type CreateRequest struct {
	Code        string `json:"code" binding:"required,max=50"`
	Name        string `json:"name" binding:"required,max=100"`
	WarehouseID *int   `json:"warehouse_id"`
	StoreID     *int   `json:"store_id"`
	Notes       string `json:"notes"`
}

type UpdateRequest struct {
	Code        *string `json:"code" binding:"omitempty,max=50"`
	Name        *string `json:"name" binding:"omitempty,max=100"`
	WarehouseID *int    `json:"warehouse_id"`
	StoreID     *int    `json:"store_id"`
	Notes       *string `json:"notes"`
	IsActive    *bool   `json:"is_active"`
}
