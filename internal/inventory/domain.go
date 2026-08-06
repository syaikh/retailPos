package inventory

import "errors"

var (
	ErrInsufficientLocationStock = errors.New("insufficient stock at source location")
	ErrLocationInactive          = errors.New("storage location is inactive")
	ErrLocationNotFound          = errors.New("storage location not found")
	ErrSameLocation              = errors.New("source and destination location must differ")
	ErrNegativeQuantity          = errors.New("quantity must not be negative")
	ErrNonPositiveQuantity       = errors.New("quantity must be positive")
)

// LocationStockItem is a rack-level stock row: how much of a product sits in a
// specific storage location. Rack rows are a sub-account of the global stock.
type LocationStockItem struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	LocationID   int    `json:"location_id"`
	LocationCode string `json:"location_code"`
	LocationName string `json:"location_name"`
	Quantity     int    `json:"quantity"`
}

// LocationRack carries the storage-location metadata needed to write rack
// stock rows. Rack rows mirror the rack's warehouse_id/store_id so global-row
// queries (warehouse_id IS NULL AND store_id IS NULL) stay unambiguous.
type LocationRack struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	WarehouseID *int   `json:"warehouse_id,omitempty"`
	StoreID     *int   `json:"store_id,omitempty"`
	IsActive    bool   `json:"is_active"`
}

type ProductStock struct {
	ID              int    `json:"id"`
	ProductID       int    `json:"product_id"`
	WarehouseID     *int   `json:"warehouse_id,omitempty"`
	StoreID         *int   `json:"store_id,omitempty"`
	Quantity        int    `json:"quantity"`
	ReorderPoint    int    `json:"reorder_point"`
	ReorderQuantity int    `json:"reorder_quantity"`
	LastRestockedAt string `json:"last_restocked_at,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type StockAdjustedEvent struct {
	ProductID      int
	QuantityChange int
	UserID         int
	Notes          string
}

// StockAdjustment is one product's stock delta to apply within a batch.
type StockAdjustment struct {
	ProductID      int
	QuantityChange int
}

type Movement struct {
	ID          int    `json:"id"`
	ProductID   int    `json:"product_id"`
	Quantity    int    `json:"quantity_change"`
	Type        string `json:"type"`
	ReferenceID *int   `json:"reference_id,omitempty"`
	UserID      *int   `json:"user_id,omitempty"`
	Notes       string `json:"notes"`
	CreatedAt   string `json:"created_at"`
}
