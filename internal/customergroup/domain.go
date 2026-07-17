package customergroup

type CustomerGroup struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type CustomerGroupCreateRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description"`
}

type CustomerGroupUpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type CustomerGroupImportRow struct {
	Row         int
	Name        string
	Description string
	IsActive    bool
}
