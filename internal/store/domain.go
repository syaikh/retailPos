package store

import "errors"

// ErrNotFound is the sentinel for a missing store. Handlers map it to 404.
var ErrNotFound = errors.New("store not found")

// ErrInternal wraps unexpected persistence failures (DB outages, driver
// errors). Handlers map it to 500 so an outage is never reported as 400/404.
var ErrInternal = errors.New("internal store error")

type Store struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address,omitempty"`
	Phone     string `json:"phone,omitempty"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at,omitempty"`
}

type CreateRequest struct {
	Name    string `json:"name" binding:"required,max=100"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type UpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=100"`
	Address  *string `json:"address"`
	Phone    *string `json:"phone"`
	IsActive *bool   `json:"is_active"`
}

type Warehouse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Address   string `json:"address,omitempty"`
	StoreID   *int   `json:"store_id,omitempty"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at,omitempty"`
}

type ImportRow struct {
	Row      int
	Name     string
	Address  string
	Phone    string
	IsActive bool
}
