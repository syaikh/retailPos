package sale

import "errors"

// ==================== DOMAIN ERRORS ====================

var (
	ErrPaymentTotalMismatch      = errors.New("total payments do not match sale total amount")
	ErrDuplicatePaymentMethod    = errors.New("duplicate payment method in split payment")
	ErrPaymentMethodInactive     = errors.New("payment method is not active")
	ErrPaymentReferenceRequired  = errors.New("reference number is required for this payment method")
	ErrZeroPaymentAmount         = errors.New("payment amount must be greater than zero")
	ErrInvalidPaymentMethod      = errors.New("invalid payment method code")
	ErrMaxPaymentsExceeded       = errors.New("maximum number of payment entries exceeded")
	ErrMultipleCashPayments      = errors.New("only one cash payment per transaction is allowed")
)

const MaxPaymentsPerSale = 10

// ==================== ENTITIES ====================

type Sale struct {
	ID            int           `json:"id"`
	InvoiceNumber string        `json:"invoice_number"`
	CashierID     int           `json:"cashier_id"`
	ShiftID       *int          `json:"shift_id,omitempty"`
	CustomerID    *int          `json:"customer_id,omitempty"`
	CustomerName  string        `json:"customer_name,omitempty"`
	StoreID       *int          `json:"store_id,omitempty"`
	Subtotal      int           `json:"subtotal"`
	Discount      int           `json:"discount"`
	Tax           int           `json:"tax"`
	TotalAmount   int           `json:"total_amount"`
	PaymentMethod string        `json:"payment_method"`
	Status        string        `json:"status"`
	Items         []SaleItem    `json:"items,omitempty"`
	Payments      []SalePayment `json:"payments,omitempty"`
	CreatedAt     string        `json:"created_at,omitempty"`
	UpdatedAt     string        `json:"updated_at,omitempty"`
}

type SaleItem struct {
	ID               int     `json:"id"`
	SaleID           int     `json:"sale_id"`
	ProductID        int     `json:"product_id"`
	Name             string  `json:"name"`
	Quantity         int     `json:"quantity"`
	UnitPrice        int     `json:"unit_price"`
	Subtotal         int     `json:"subtotal"`
	DPPAmount        int     `json:"dpp_amount"`
	TaxAmount        int     `json:"tax_amount"`
	PricingRuleID    *int    `json:"pricing_rule_id,omitempty"`
	PricingRuleName  *string `json:"pricing_rule_name,omitempty"`
	PricingRuleType  *string `json:"pricing_rule_type,omitempty"`
	PricingType      *string `json:"pricing_type,omitempty"`
	OriginalPrice    *int    `json:"original_price,omitempty"`
	Cost             int     `json:"cost,omitempty"`
	TaxClassID       *int    `json:"tax_class_id,omitempty"`
	TaxRate          *float64 `json:"tax_rate,omitempty"`
	SnapshotCreatedAt string  `json:"snapshot_created_at,omitempty"`
	ProductName      string  `json:"product_name,omitempty"`
}

type SalePayment struct {
	ID                int    `json:"id"`
	SaleID            int    `json:"sale_id"`
	PaymentMethodID   int    `json:"payment_method_id"`
	PaymentMethodCode string `json:"payment_method_code"`
	Amount            int    `json:"amount"`
	ReferenceNumber   string `json:"reference_number,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
}

type CreatePaymentRequest struct {
	PaymentMethodCode string `json:"payment_method_code" binding:"required"`
	Amount            int    `json:"amount" binding:"required,min=1"`
	ReferenceNumber   string `json:"reference_number"`
}

type SaleExportRow struct {
	InvoiceNumber string `json:"invoice_number"`
	CreatedAt     string `json:"created_at"`
	CustomerName  string `json:"customer_name"`
	ItemCount     int    `json:"items_count"`
	PaymentMethod string `json:"payment_method"`
	TotalAmount   int    `json:"total_amount"`
}

type SaleCreateRequest struct {
	InvoiceNumber string                 `json:"invoice_number"`
	CashierID     int                    `json:"cashier_id"`
	ShiftID       *int                   `json:"shift_id,omitempty"`
	StoreID       *int                   `json:"store_id,omitempty"`
	Subtotal      int                    `json:"subtotal"`
	Discount      int                    `json:"discount"`
	Tax           int                    `json:"tax"`
	TotalAmount   int                    `json:"total_amount"`
	PaymentMethod string                 `json:"payment_method"`
	CustomerID    *int                   `json:"customer_id,omitempty"`
	Items         []SaleItem             `json:"items"`
	Payments      []CreatePaymentRequest `json:"payments"`
}

type PaymentMethod struct {
	ID                int    `json:"id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	IsActive          bool   `json:"is_active"`
	RequiresReference bool   `json:"requires_reference"`
	SortOrder         int    `json:"sort_order"`
	CreatedAt         string `json:"created_at,omitempty"`
}
