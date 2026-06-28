package inventory

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

type InventoryMovement struct {
	ID          int    `json:"id"`
	ProductID   int    `json:"product_id"`
	Quantity    int    `json:"quantity_change"`
	Type        string `json:"type"`
	ReferenceID *int   `json:"reference_id,omitempty"`
	UserID      *int   `json:"user_id,omitempty"`
	Notes       string `json:"notes"`
	CreatedAt   string `json:"created_at"`
}
