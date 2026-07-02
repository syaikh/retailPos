package uom

type UnitOfMeasure struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type UOMCreateRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

type UOMUpdateRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

type UOMImportRow struct {
	Row         int
	Code        string
	Name        string
	Description string
	IsActive    bool
}
