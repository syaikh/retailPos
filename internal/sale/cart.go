package sale

import "errors"

// ==================== CART DOMAIN ERRORS ====================

var (
	ErrCartNotFound          = errors.New("cart session not found")
	ErrCartNotOpen           = errors.New("cart session is not open")
	ErrCartExpired           = errors.New("cart session has expired")
	ErrCartItemNotFound      = errors.New("cart item not found")
	ErrCartItemQuantity      = errors.New("quantity must be greater than zero")
	ErrCartAlreadyCheckedOut = errors.New("cart session already checked out")
	ErrCartNotOwned          = errors.New("cart session does not belong to the authenticated cashier")
)

func (ci CartItem) ToSaleItem() SaleItem {
	return SaleItem{
		ProductID:         ci.ProductID,
		Name:              ci.ProductName,
		ProductName:       ci.ProductName,
		Quantity:          ci.Quantity,
		UnitPrice:         ci.UnitPrice,
		Subtotal:          ci.Subtotal,
		DPPAmount:         ci.DPPAmount,
		TaxAmount:         ci.TaxAmount,
		PricingRuleID:     ci.PricingRuleID,
		PricingRuleName:   ci.PricingRuleName,
		PricingRuleType:   ci.PricingRuleType,
		PricingType:       ci.PricingType,
		OriginalPrice:     intPtr(ci.OriginalPrice),
		Cost:              ci.Cost,
		TaxClassID:        ci.TaxClassID,
		TaxRate:           ci.TaxRate,
		SnapshotCreatedAt: ci.SnapshotCreatedAt,
	}
}

// ==================== CART ENTITIES ====================

type CartSession struct {
	ID          int        `json:"id"`
	CashierID   int        `json:"cashier_id"`
	StoreID     *int       `json:"store_id,omitempty"`
	ShiftID     *int       `json:"shift_id,omitempty"`
	CustomerID  *int       `json:"customer_id,omitempty"`
	Status      string     `json:"status"`
	Subtotal    int        `json:"subtotal"`
	Discount    int        `json:"discount"`
	Tax         int        `json:"tax"`
	TotalAmount int        `json:"total_amount"`
	ExpiredAt   *string    `json:"expired_at,omitempty"`
	Items       []CartItem `json:"items,omitempty"`
	CreatedAt   string     `json:"created_at,omitempty"`
	UpdatedAt   string     `json:"updated_at,omitempty"`
}

type CartItem struct {
	ID                int      `json:"id"`
	CartSessionID     int      `json:"cart_session_id"`
	ProductID         int      `json:"product_id"`
	ProductName       string   `json:"product_name"`
	Quantity          int      `json:"quantity"`
	UnitPrice         int      `json:"unit_price"`
	OriginalPrice     int      `json:"original_price"`
	Discount          int      `json:"discount"`
	PricingRuleID     *int     `json:"pricing_rule_id,omitempty"`
	PricingRuleName   *string  `json:"pricing_rule_name,omitempty"`
	PricingRuleType   *string  `json:"pricing_rule_type,omitempty"`
	PricingType       *string  `json:"pricing_type,omitempty"`
	Cost              int      `json:"cost"`
	TaxClassID        *int     `json:"tax_class_id,omitempty"`
	TaxRate           *float64 `json:"tax_rate,omitempty"`
	SnapshotCreatedAt string   `json:"snapshot_created_at,omitempty"`
	Subtotal          int      `json:"subtotal"`
	DPPAmount         int      `json:"dpp_amount"`
	TaxAmount         int      `json:"tax_amount"`
}

// ==================== CART REQUEST/EVENT TYPES ====================

type CreateCartRequest struct {
	StoreID    *int `json:"store_id"`
	ShiftID    *int `json:"shift_id"`
	CustomerID *int `json:"customer_id"`
}

type AddCartItemRequest struct {
	ProductID       int  `json:"product_id" binding:"required"`
	Quantity        int  `json:"quantity" binding:"required,min=1"`
	CustomerGroupID *int `json:"customer_group_id"`
	StoreID         *int `json:"store_id"`
	ShiftID         *int `json:"shift_id"`
	CustomerID      *int `json:"customer_id"`
}

type UpdateCartItemQuantityRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

type UpdateCartCustomerRequest struct {
	CustomerID *int `json:"customer_id"`
}

type CartCheckedOutEvent struct {
	CartID   int `json:"cart_id"`
	SaleID   int `json:"sale_id"`
	CashierID int `json:"cashier_id"`
}
