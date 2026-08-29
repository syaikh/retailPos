package consignment

import "errors"

var (
	ErrConsignmentNotFound          = errors.New("consignment arrangement not found")
	ErrActiveArrangementExists      = errors.New("an active arrangement already exists for this supplier and store")
	ErrNotConsignmentSupplier       = errors.New("supplier is not flagged as consignment")
	ErrConflictStoreStock           = errors.New("product still has store-owned stock; ownership cannot be consigned")
	ErrConflictOtherSupplier        = errors.New("product is consignment-owned by another supplier")
	ErrPendingReturnBlocksTransfer  = errors.New("product has an open pending return; ownership cannot be transferred")
	ErrInsufficientConsignmentStock = errors.New("insufficient consignment stock")
	ErrProductNotFound              = errors.New("product not found")
	ErrInvalidQty                   = errors.New("quantity must be greater than zero")
	ErrInvalidReason                = errors.New("invalid pending return reason")
	ErrInvalidShareType             = errors.New("store share type must be percentage or fixed_amount")
	ErrInvalidShareValue            = errors.New("store share value must be greater than zero")
	ErrInvalidShareValueForType     = errors.New("percentage store share value must be less than 100")
	ErrArrangementEnded             = errors.New("consignment arrangement is ended")
	ErrPendingReturnNotFound        = errors.New("pending return not found")
	ErrReturnNotFound               = errors.New("consignment return not found")
	ErrEmptySettlement              = errors.New("supplier has no unsettled consignment sales")
	ErrSettlementNotFound           = errors.New("consignment settlement not found")
	ErrSettlementAlreadyPaid        = errors.New("consignment settlement is already paid")
	ErrInvalidPayoutAmount          = errors.New("payout amount must match the settlement payable")
	ErrPaymentMethodNotFound        = errors.New("payment method not found")
)

const (
	StatusActive = "active"
	StatusEnded  = "ended"
)

const (
	ShareTypePercentage  = "percentage"
	ShareTypeFixedAmount = "fixed_amount"
)

const (
	PendingReturnOpen     = "open"
	PendingReturnReturned = "returned"
)

const (
	SettlementPendingPayment = "pending_payment"
	SettlementPaid           = "paid"
)

const (
	ReasonDamaged        = "damaged"
	ReasonExpired        = "expired"
	ReasonCustomerReturn = "customer_return"
	ReasonOther          = "other"
)

const (
	MovementTypeConsignmentReceipt        = "consignment_receipt"
	MovementTypeConsignmentReturn         = "consignment_return"
	MovementTypeConsignmentPendingReturn  = "consignment_pending_return"
	MovementTypeConsignmentCustomerReturn = "consignment_customer_return"
)

// Arrangement is one consignment partnership (one supplier + one store). The
// lazy Ended status is computed on read from last_visit_at (2 weeks) and is
// persisted on the next write.
type Arrangement struct {
	ID           int     `json:"id"`
	SupplierID   int     `json:"supplier_id"`
	SupplierName string  `json:"supplier_name,omitempty"`
	StoreID      int     `json:"store_id"`
	StoreName    string  `json:"store_name,omitempty"`
	Status       string  `json:"status"`
	LastVisitAt  *string `json:"last_visit_at,omitempty"`
	EndedAt      *string `json:"ended_at,omitempty"`
	CreatedBy    int     `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	Terms        []Term  `json:"terms,omitempty"`
}

// Term is the agreed price and store-share for one product of an arrangement.
// Exactly one share type is enforced; the value is a percentage (0-100) or a
// fixed amount (IDR per unit).
type Term struct {
	ID              int     `json:"id"`
	ArrangementID   int     `json:"arrangement_id"`
	ProductID       int     `json:"product_id"`
	ProductSKU      string  `json:"product_sku,omitempty"`
	ProductName     string  `json:"product_name,omitempty"`
	Price           int     `json:"price"`
	StoreShareType  string  `json:"store_share_type"`
	StoreShareValue float64 `json:"store_share_value"`
	EffectiveFrom   string  `json:"effective_from,omitempty"`
	CreatedBy       int     `json:"created_by"`
	CreatedAt       string  `json:"created_at"`
}

// Receipt records the ACCEPTED quantity of consigned goods after inspection.
// Rejected goods are never recorded (BR-08).
type Receipt struct {
	ID                 int           `json:"id"`
	ReceiptNumber      string        `json:"receipt_number"`
	SupplierID         int           `json:"supplier_id"`
	SupplierName       string        `json:"supplier_name,omitempty"`
	StoreID            int           `json:"store_id"`
	ArrangementID      int           `json:"arrangement_id"`
	ReceivedBy         int           `json:"received_by"`
	ReceivedByUsername string        `json:"received_by_username,omitempty"`
	ReceivedAt         string        `json:"received_at"`
	Notes              string        `json:"notes,omitempty"`
	CreatedAt          string        `json:"created_at"`
	Items              []ReceiptItem `json:"items,omitempty"`
}

type ReceiptItem struct {
	ID                   int     `json:"id"`
	ConsignmentReceiptID int     `json:"consignment_receipt_id"`
	ProductID            int     `json:"product_id"`
	ProductSKU           string  `json:"product_sku,omitempty"`
	ProductName          string  `json:"product_name,omitempty"`
	AcceptedQty          int     `json:"accepted_qty"`
	Price                int     `json:"price"`
	StoreShareType       string  `json:"store_share_type"`
	StoreShareValue      float64 `json:"store_share_value"`
	Notes                string  `json:"notes,omitempty"`
}

// StockRow is one row of the consignment ownership ledger.
type StockRow struct {
	ProductID        int     `json:"product_id"`
	ProductSKU       string  `json:"product_sku,omitempty"`
	ProductName      string  `json:"product_name,omitempty"`
	SupplierID       int     `json:"supplier_id"`
	SupplierName     string  `json:"supplier_name,omitempty"`
	ArrangementID    int     `json:"arrangement_id"`
	StoreID          int     `json:"store_id"`
	AvailableQty     int     `json:"available_qty"`
	PendingReturnQty int     `json:"pending_return_qty"`
	UpdatedAt        *string `json:"updated_at,omitempty"`
}

// PendingReturn is a simple record of goods pulled off the display and not yet
// handed back to the supplier. It still counts as consignment ownership.
type PendingReturn struct {
	ID            int     `json:"id"`
	SupplierID    int     `json:"supplier_id"`
	ProductID     int     `json:"product_id"`
	ProductSKU    string  `json:"product_sku,omitempty"`
	ProductName   string  `json:"product_name,omitempty"`
	ArrangementID int     `json:"arrangement_id"`
	StoreID       int     `json:"store_id"`
	Qty           int     `json:"qty"`
	Reason        string  `json:"reason"`
	Notes         string  `json:"notes,omitempty"`
	Status        string  `json:"status"`
	ReturnedAt    *string `json:"returned_at,omitempty"`
	CreatedBy     int     `json:"created_by"`
	CreatedAt     string  `json:"created_at"`
}

// Return is the formal hand-back of goods to the supplier.
type Return struct {
	ID                 int          `json:"id"`
	ReturnNumber       string       `json:"return_number"`
	SupplierID         int          `json:"supplier_id"`
	SupplierName       string       `json:"supplier_name,omitempty"`
	StoreID            int          `json:"store_id"`
	ArrangementID      int          `json:"arrangement_id"`
	ReturnedBy         int          `json:"returned_by"`
	ReturnedByUsername string       `json:"returned_by_username,omitempty"`
	ReturnedAt         string       `json:"returned_at"`
	Notes              string       `json:"notes,omitempty"`
	CreatedAt          string       `json:"created_at"`
	Items              []ReturnItem `json:"items,omitempty"`
}

type ReturnItem struct {
	ID                  int    `json:"id"`
	ConsignmentReturnID int    `json:"consignment_return_id"`
	ProductID           int    `json:"product_id"`
	ProductSKU          string `json:"product_sku,omitempty"`
	ProductName         string `json:"product_name,omitempty"`
	Qty                 int    `json:"qty"`
	Reason              string `json:"reason"`
	PendingReturnID     *int   `json:"pending_return_id,omitempty"`
	Notes               string `json:"notes,omitempty"`
}

// Settlement is a full settlement of ALL unsettled consignment sales of one
// supplier. Partial settlement is never allowed.
type Settlement struct {
	ID               int              `json:"id"`
	SettlementNumber string           `json:"settlement_number"`
	SupplierID       int              `json:"supplier_id"`
	SupplierName     string           `json:"supplier_name,omitempty"`
	StoreID          int              `json:"store_id"`
	TotalSaleValue   int              `json:"total_sale_value"`
	TotalStoreShare  int              `json:"total_store_share"`
	TotalPayable     int              `json:"total_payable"`
	Status           string           `json:"status"`
	CreatedBy        int              `json:"created_by"`
	CreatedAt        string           `json:"created_at"`
	PaidAt           *string          `json:"paid_at,omitempty"`
	Items            []SettlementItem `json:"items,omitempty"`
	Payouts          []Payout         `json:"payouts,omitempty"`
}

type SettlementItem struct {
	ID                      int    `json:"id"`
	ConsignmentSettlementID int    `json:"consignment_settlement_id"`
	ConsignmentSaleItemID   int    `json:"consignment_sale_item_id"`
	ProductID               int    `json:"product_id"`
	ProductName             string `json:"product_name,omitempty"`
	Quantity                int    `json:"quantity"`
	UnitPrice               int    `json:"unit_price"`
	Subtotal                int    `json:"subtotal"`
	StoreShare              int    `json:"store_share"`
}

// Payout is the money-out record to a supplier, decoupled from sale payments.
type Payout struct {
	ID                int    `json:"id"`
	PayoutNumber      string `json:"payout_number"`
	SettlementID      int    `json:"settlement_id"`
	PaymentMethodID   int    `json:"payment_method_id"`
	PaymentMethodCode string `json:"payment_method_code,omitempty"`
	PaymentMethodName string `json:"payment_method_name,omitempty"`
	Amount            int    `json:"amount"`
	ReferenceNumber   string `json:"reference_number,omitempty"`
	PaidBy            int    `json:"paid_by"`
	PaidByUsername    string `json:"paid_by_username,omitempty"`
	PaidAt            string `json:"paid_at"`
	Notes             string `json:"notes,omitempty"`
	CreatedAt         string `json:"created_at"`
}

// SaleItemRecord is one unsettled consignment sale line, used by the settlement
// preview and settlement item builder.
type SaleItemRecord struct {
	ID               int     `json:"id"`
	SaleID           int     `json:"sale_id"`
	InvoiceNumber    string  `json:"invoice_number,omitempty"`
	ProductID        int     `json:"product_id"`
	ProductName      string  `json:"product_name,omitempty"`
	Quantity         int     `json:"quantity"`
	UnitPrice        int     `json:"unit_price"`
	Subtotal         int     `json:"subtotal"`
	StoreShareType   string  `json:"store_share_type"`
	StoreShareValue  float64 `json:"store_share_value"`
	StoreShareAmount int     `json:"store_share_amount"`
	CreatedAt        string  `json:"created_at"`
}

// ---- Requests ----

type CreateArrangementRequest struct {
	SupplierID int `json:"supplier_id" binding:"required"`
	StoreID    int `json:"store_id"`
}

type SetTermsRequest struct {
	ProductID       int     `json:"product_id" binding:"required"`
	Price           int     `json:"price" binding:"required,min=0"`
	StoreShareType  string  `json:"store_share_type" binding:"required"`
	StoreShareValue float64 `json:"store_share_value" binding:"required"`
}

type ReceiptItemRequest struct {
	ProductID   int    `json:"product_id" binding:"required"`
	AcceptedQty int    `json:"accepted_qty" binding:"required,min=1"`
	Notes       string `json:"notes"`
}

type ReceiptRequest struct {
	ArrangementID int                  `json:"arrangement_id" binding:"required"`
	Notes         string               `json:"notes"`
	Items         []ReceiptItemRequest `json:"items" binding:"required,min=1"`
}

type CreatePendingReturnRequest struct {
	ProductID int    `json:"product_id" binding:"required"`
	Qty       int    `json:"qty" binding:"required,min=1"`
	Reason    string `json:"reason" binding:"required"`
	Notes     string `json:"notes"`
}

type ReturnItemRequest struct {
	ProductID       int    `json:"product_id" binding:"required"`
	Qty             int    `json:"qty" binding:"required,min=1"`
	Reason          string `json:"reason" binding:"required"`
	PendingReturnID *int   `json:"pending_return_id"`
	Notes           string `json:"notes"`
}

type ReturnRequest struct {
	ArrangementID int                 `json:"arrangement_id" binding:"required"`
	Notes         string              `json:"notes"`
	Items         []ReturnItemRequest `json:"items" binding:"required,min=1"`
}

type CreateSettlementRequest struct {
	SupplierID int `json:"supplier_id" binding:"required"`
}

type CreatePayoutRequest struct {
	PaymentMethodID int    `json:"payment_method_id" binding:"required"`
	Amount          int    `json:"amount" binding:"required,min=1"`
	ReferenceNumber string `json:"reference_number"`
	Notes           string `json:"notes"`
}
