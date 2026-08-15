package sale

import (
	"errors"

	"retail-pos-system/internal/shared"
)

// ==================== DOMAIN ERRORS ====================

var (
	ErrPaymentTotalMismatch     = errors.New("total payments do not match sale total amount")
	ErrDuplicatePaymentMethod   = errors.New("duplicate payment method in split payment")
	ErrPaymentMethodInactive    = errors.New("payment method is not active")
	ErrPaymentReferenceRequired = errors.New("reference number is required for this payment method")
	ErrZeroPaymentAmount        = errors.New("payment amount must be greater than zero")
	ErrInvalidPaymentMethod     = errors.New("invalid payment method code")
	ErrMaxPaymentsExceeded      = errors.New("maximum number of payment entries exceeded")
	ErrMultipleCashPayments     = errors.New("only one cash payment per transaction is allowed")
)

const MaxPaymentsPerSale = 10

// ==================== ENTITIES ====================

type Sale struct {
	ID            int       `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	CashierID     int       `json:"cashier_id"`
	ShiftID       *int      `json:"shift_id,omitempty"`
	CustomerID    *int      `json:"customer_id,omitempty"`
	CustomerName  string    `json:"customer_name,omitempty"`
	HoldNote      string    `json:"hold_note,omitempty"`
	StoreID       *int      `json:"store_id,omitempty"`
	Subtotal      int       `json:"subtotal"`
	Discount      int       `json:"discount"`
	Tax           int       `json:"tax"`
	TotalAmount   int       `json:"total_amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	Items         []Item    `json:"items,omitempty"`
	Payments      []Payment `json:"payments,omitempty"`
	CreatedAt     string    `json:"created_at,omitempty"`
	UpdatedAt     string    `json:"updated_at,omitempty"`

	// consignmentRecords is the checkout-time consignment snapshot stashed by
	// finalizeSaleItems and persisted to consignment_sale_items right after the
	// sale row is created. It is deliberately unexported: it is an internal
	// sale-flow payload, not part of the API/DTO shape.
	consignmentRecords []shared.ConsignmentSaleRecord
}

type Item struct {
	ID                int      `json:"id"`
	SaleID            int      `json:"sale_id"`
	ProductID         int      `json:"product_id"`
	Name              string   `json:"name"`
	Quantity          int      `json:"quantity"`
	UnitPrice         int      `json:"unit_price"`
	Subtotal          int      `json:"subtotal"`
	DPPAmount         int      `json:"dpp_amount"`
	TaxAmount         int      `json:"tax_amount"`
	PricingRuleID     *int     `json:"pricing_rule_id,omitempty"`
	PricingRuleName   *string  `json:"pricing_rule_name,omitempty"`
	PricingRuleType   *string  `json:"pricing_rule_type,omitempty"`
	Type              *string  `json:"pricing_type,omitempty"`
	OriginalPrice     *int     `json:"original_price,omitempty"`
	Cost              int      `json:"cost,omitempty"`
	TaxClassID        *int     `json:"tax_class_id,omitempty"`
	TaxRate           *float64 `json:"tax_rate,omitempty"`
	SnapshotCreatedAt string   `json:"snapshot_created_at,omitempty"`
	ProductName       string   `json:"product_name,omitempty"`
}

type Payment struct {
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

type ExportRow struct {
	InvoiceNumber string `json:"invoice_number"`
	CreatedAt     string `json:"created_at"`
	CustomerName  string `json:"customer_name"`
	ItemCount     int    `json:"items_count"`
	PaymentMethod string `json:"payment_method"`
	TotalAmount   int    `json:"total_amount"`
}

type CreateRequest struct {
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
	Items         []Item                 `json:"items"`
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

// Caller identifies the authenticated actor behind a parked-sale operation.
// It drives the P2-6 ownership/scope rules: cashiers are restricted to their
// own parked sales, managers may recall any parked sale (but never cancel, and
// may only complete a sale when it carries a parked_sale_id), and
// superadmin/admin are elevated over all parked sales.
type Caller struct {
	UserID  int
	Role    string
	StoreID *int
}

// IsElevated reports whether the caller bypasses cashier/manager scoping
// (superadmin and admin).
func (c Caller) IsElevated() bool {
	return c.Role == "superadmin" || c.Role == "admin"
}

// IsManager reports whether the caller is a manager (recall-only rules).
func (c Caller) IsManager() bool {
	return c.Role == "manager"
}

// ownerScope returns the cashier owner filter to apply at the repository level.
// A nil scope means no cashier restriction (manager/elevated). All other roles
// (cashier, staff, unknown) are treated as owner-scoped so they can never touch
// another cashier's parked sale.
func (c Caller) ownerScope() *int {
	if c.IsElevated() || c.IsManager() {
		return nil
	}
	uid := c.UserID
	return &uid
}

// storeScope returns the caller's store filter, if any. It is applied on top of
// ownership so that manager/elevated users (nil owner scope) are still confined
// to their own store, mirroring GetAllSales/GetSaleByID. Users issued a token
// without a store claim (nil) are unscoped by design.
func (c Caller) storeScope() *int {
	return c.StoreID
}
