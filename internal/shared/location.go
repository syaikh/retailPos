package shared

import "errors"

// ErrLocationNotFound is returned when a storage location cannot be found. It
// lives in shared so that internal/storagelocation (provider) and
// internal/inventory (consumer) can reference the same sentinel without
// importing each other (both are isolated modules).
var ErrLocationNotFound = errors.New("storage location not found")

// LocationRack is the rack metadata contract between internal/storagelocation
// (single-writer of storage_locations, Referensi) and internal/inventory
// (consumer). internal/inventory must not import internal/storagelocation, so
// this DTO lives in internal/shared.
type LocationRack struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	WarehouseID *int   `json:"warehouse_id,omitempty"`
	StoreID     *int   `json:"store_id,omitempty"`
	IsActive    bool   `json:"is_active"`
}
