package customer

type Customer struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Phone           *string `json:"phone,omitempty"`
	Email           *string `json:"email,omitempty"`
	Address         *string `json:"address,omitempty"`
	TaxID           *string `json:"tax_id,omitempty"`
	CustomerGroupID *int    `json:"customer_group_id,omitempty"`
	CustomerGroupName *string `json:"customer_group_name,omitempty"`
	LoyaltyPoints   int     `json:"loyalty_points"`
	TotalSpent      int     `json:"total_spent"`
	LastPurchaseAt  *string `json:"last_purchase_at,omitempty"`
	Note            *string `json:"note,omitempty"`
	StoreID         *int    `json:"store_id,omitempty"`
	IsActive        bool    `json:"is_active"`
	IsWalkIn        bool    `json:"is_walk_in"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

type CustomerCreateRequest struct {
	Name            string  `json:"name"`
	Phone           string  `json:"phone,omitempty"`
	Email           string  `json:"email,omitempty"`
	Address         *string `json:"address,omitempty"`
	TaxID           *string `json:"tax_id,omitempty"`
	CustomerGroupID *int    `json:"customer_group_id,omitempty"`
	Note            *string `json:"note,omitempty"`
}

type CustomerUpdateRequest struct {
	Name            *string `json:"name,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	Email           *string `json:"email,omitempty"`
	Address         *string `json:"address,omitempty"`
	TaxID           *string `json:"tax_id,omitempty"`
	CustomerGroupID *int    `json:"customer_group_id"`
	Note            *string `json:"note,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

type CustomerImportRow struct {
	Row             int
	Name            string
	Phone           string
	Email           string
	Address         string
	Note            string
	TaxID           string
	CustomerGroupID *int
	IsActive        bool
	StoreID         *int
}
