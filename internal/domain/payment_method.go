package domain

type PaymentMethod struct {
	ID                 int     `json:"id"`
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	IsActive           bool    `json:"is_active"`
	RequiresReference  bool    `json:"requires_reference"`
	SortOrder          int     `json:"sort_order"`
	CreatedAt          string  `json:"created_at,omitempty"`
}
