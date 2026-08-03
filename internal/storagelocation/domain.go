package storagelocation

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

type StorageLocationCreateRequest struct {
	Code        string `json:"code" binding:"required,max=50"`
	Name        string `json:"name" binding:"required,max=100"`
	WarehouseID *int   `json:"warehouse_id"`
	StoreID     *int   `json:"store_id"`
	Notes       string `json:"notes"`
}

type StorageLocationUpdateRequest struct {
	Code        *string `json:"code" binding:"omitempty,max=50"`
	Name        *string `json:"name" binding:"omitempty,max=100"`
	WarehouseID *int    `json:"warehouse_id"`
	StoreID     *int    `json:"store_id"`
	Notes       *string `json:"notes"`
	IsActive    *bool   `json:"is_active"`
}
