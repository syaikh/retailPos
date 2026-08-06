package brand

type Brand struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

type UpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

type ImportRow struct {
	Row         int
	Name        string
	Description string
	IsActive    bool
}
