package storagelocation

import "errors"

// ErrStoreForbidden is returned when a store-scoped caller tries to read or
// write a storage location owned by a different store (or by a warehouse with
// no store). Handlers map it to HTTP 403. A nil caller store (superadmin)
// never triggers it.
var ErrStoreForbidden = errors.New("storage location is not in your store")

// ErrNotFound is the sentinel for a missing storage location. Handlers map it
// to HTTP 404 so genuine not-found responses are distinguishable from DB or
// provider failures (which surface as 500).
var ErrNotFound = errors.New("storage location not found")

// ErrInternal wraps unexpected persistence failures (DB outages, driver
// errors). Handlers map it to HTTP 500 so they are not conflated with
// validation failures, which surface as plain errors and map to HTTP 400.
var ErrInternal = errors.New("internal storage location error")

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
