package domain

type Category struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug,omitempty"`
	Description  string `json:"description,omitempty"`
	IsActive     bool   `json:"is_active"`
	ProductCount int    `json:"product_count,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type CategoryCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type CategoryUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active,omitempty"`
}