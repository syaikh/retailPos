package customergroup

type CustomerGroup struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	IsActive      bool   `json:"is_active"`
	CustomerCount int    `json:"customer_count"`
	Color         string `json:"color,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type CreateRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type UpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
	Color       *string `json:"color"`
}

type ImportRow struct {
	Row         int
	Name        string
	Description string
	IsActive    bool
	Color       string
}
