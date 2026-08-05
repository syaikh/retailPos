package purchase

import "errors"

var (
	ErrPurchaseOrderNotFound         = errors.New("purchase order not found")
	ErrPurchaseOrderNotDraft         = errors.New("purchase order is not in draft status")
	ErrPurchaseOrderAlreadyConfirmed = errors.New("purchase order is already confirmed")
	ErrPurchaseOrderCancelled        = errors.New("purchase order is cancelled")
	ErrPurchaseOrderHasReceipts      = errors.New("cannot cancel purchase order that has receipts")
	ErrPOItemNotFound                = errors.New("purchase order item not found")
	ErrOverReceiving                 = errors.New("received quantity exceeds remaining ordered quantity")
	ErrInvalidReceivingQty           = errors.New("invalid receiving quantity")
	ErrDuplicatePOItem               = errors.New("duplicate product in purchase order items")
	ErrInvalidPOStatusForReceiving   = errors.New("purchase order is not in a receivable status")
	ErrNoItemsToReceive              = errors.New("no items to receive")
	ErrInvalidInput                  = errors.New("invalid purchase order input")
)

const (
	StatusDraft           = "draft"
	StatusConfirmed       = "confirmed"
	StatusPartialReceived = "partial_received"
	StatusFullyReceived   = "fully_received"
	StatusCancelled       = "cancelled"
	StatusWaitingApproval = "waiting_approval"
	StatusRejected        = "rejected"
)

type DomainEvent string

const (
	EventPOCreated           DomainEvent = "purchase_order.created"
	EventPOConfirmed         DomainEvent = "purchase_order.confirmed"
	EventPOCancelled         DomainEvent = "purchase_order.cancelled"
	EventGoodsReceiptCreated DomainEvent = "goods_receipt.created"
)

type PurchaseOrder struct {
	ID                      int                 `json:"id"`
	PONumber                string              `json:"po_number"`
	SupplierID              int                 `json:"supplier_id"`
	SupplierName            string              `json:"supplier_name"`
	StoreID                 int                 `json:"store_id"`
	WarehouseID             *int                `json:"warehouse_id,omitempty"`
	Status                  string              `json:"status"`
	ExpectedDate            string              `json:"expected_date,omitempty"`
	PaymentTerm             string              `json:"payment_term,omitempty"`
	DeliveryAddress         string              `json:"delivery_address,omitempty"`
	SupplierReferenceNumber string              `json:"supplier_reference_number,omitempty"`
	ApprovalStatus          string              `json:"approval_status,omitempty"`
	PaymentStatus           string              `json:"payment_status,omitempty"`
	InvoiceStatus           string              `json:"invoice_status,omitempty"`
	CurrencyCode            string              `json:"currency_code,omitempty"`
	ExchangeRate            int                 `json:"exchange_rate,omitempty"`
	ApprovedBy              *int                `json:"approved_by,omitempty"`
	ApprovedAt              string              `json:"approved_at,omitempty"`
	Subtotal                int                 `json:"subtotal"`
	DiscountAmount          int                 `json:"discount_amount"`
	TaxAmount               int                 `json:"tax_amount"`
	GrandTotal              int                 `json:"grand_total"`
	Notes                   string              `json:"notes,omitempty"`
	ConfirmedAt             string              `json:"confirmed_at,omitempty"`
	ConfirmedBy             *int                `json:"confirmed_by,omitempty"`
	CancelledAt             string              `json:"cancelled_at,omitempty"`
	CancelledBy             *int                `json:"cancelled_by,omitempty"`
	CreatedBy               int                 `json:"created_by"`
	UpdatedBy               int                 `json:"updated_by"`
	CreatedAt               string              `json:"created_at"`
	UpdatedAt               string              `json:"updated_at"`
	Items                   []PurchaseOrderItem `json:"items,omitempty"`
}

type PurchaseOrderItem struct {
	ID              int    `json:"id"`
	PurchaseOrderID int    `json:"purchase_order_id"`
	ProductID       int    `json:"product_id"`
	QtyOrdered      int    `json:"qty_ordered"`
	QtyReceived     int    `json:"qty_received"`
	UnitCost        int    `json:"unit_cost"`
	DiscountAmount  int    `json:"discount_amount"`
	Subtotal        int    `json:"subtotal"`
	ProductName     string `json:"product_name"`
	SKU             string `json:"sku,omitempty"`
	Barcode         string `json:"barcode,omitempty"`
	UOMID           *int   `json:"uom_id,omitempty"`
	UOMName         string `json:"uom_name,omitempty"`
	Notes           string `json:"notes,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type GoodsReceipt struct {
	ID                  int                `json:"id"`
	GRNumber            string             `json:"gr_number"`
	PurchaseOrderID     int                `json:"purchase_order_id"`
	StoreID             int                `json:"store_id"`
	ReceivedBy          int                `json:"received_by"`
	ReceivedAt          string             `json:"received_at"`
	DeliveryOrderNumber string             `json:"delivery_order_number,omitempty"`
	ShippingMethod      string             `json:"shipping_method,omitempty"`
	DriverName          string             `json:"driver_name,omitempty"`
	VehiclePlateNumber  string             `json:"vehicle_plate_number,omitempty"`
	Notes               string             `json:"notes,omitempty"`
	CreatedAt           string             `json:"created_at"`
	Items               []GoodsReceiptItem `json:"items,omitempty"`
}

type GoodsReceiptItem struct {
	ID                  int    `json:"id"`
	GoodsReceiptID      int    `json:"goods_receipt_id"`
	PurchaseOrderItemID int    `json:"purchase_order_item_id"`
	ProductID           int    `json:"product_id"`
	QtyGood             int    `json:"qty_good"`
	QtyDamaged          int    `json:"qty_damaged"`
	UnitCost            int    `json:"unit_cost"`
	ProductName         string `json:"product_name"`
	SupplierID          *int   `json:"supplier_id,omitempty"`
	Notes               string `json:"notes,omitempty"`
	CreatedAt           string `json:"created_at"`
}

type CreatePurchaseOrderRequest struct {
	SupplierID              int                   `json:"supplier_id" binding:"required"`
	StoreID                 *int                  `json:"store_id,omitempty"`
	WarehouseID             *int                  `json:"warehouse_id,omitempty"`
	ExpectedDate            string                `json:"expected_date,omitempty"`
	PaymentTerm             string                `json:"payment_term,omitempty"`
	DeliveryAddress         string                `json:"delivery_address,omitempty"`
	SupplierReferenceNumber string                `json:"supplier_reference_number,omitempty"`
	Notes                   string                `json:"notes,omitempty"`
	Items                   []CreatePOItemRequest `json:"items" binding:"required"`
}

type CreatePOItemRequest struct {
	ProductID      int     `json:"product_id" binding:"required"`
	QtyOrdered     int     `json:"qty_ordered" binding:"required,min=1"`
	UnitCost       int     `json:"unit_cost" binding:"min=0"`
	DiscountAmount int     `json:"discount_amount"`
	Notes          *string `json:"notes,omitempty"`
}

type UpdatePurchaseOrderRequest struct {
	SupplierID              int                   `json:"supplier_id" binding:"required"`
	ExpectedDate            string                `json:"expected_date,omitempty"`
	PaymentTerm             string                `json:"payment_term,omitempty"`
	DeliveryAddress         string                `json:"delivery_address,omitempty"`
	SupplierReferenceNumber string                `json:"supplier_reference_number,omitempty"`
	Notes                   string                `json:"notes,omitempty"`
	Items                   []UpdatePOItemRequest `json:"items" binding:"required"`
}

type UpdatePOItemRequest struct {
	ID             int     `json:"id"`
	ProductID      int     `json:"product_id" binding:"required"`
	QtyOrdered     int     `json:"qty_ordered" binding:"required,min=1"`
	UnitCost       int     `json:"unit_cost" binding:"required,min=0"`
	DiscountAmount int     `json:"discount_amount"`
	Notes          *string `json:"notes,omitempty"`
}

type CreateGoodsReceiptRequest struct {
	PurchaseOrderID    int                   `json:"purchase_order_id" binding:"required"`
	ShippingMethod     string                `json:"shipping_method,omitempty"`
	DriverName         string                `json:"driver_name,omitempty"`
	VehiclePlateNumber string                `json:"vehicle_plate_number,omitempty"`
	Notes              string                `json:"notes,omitempty"`
	Items              []CreateGRItemRequest `json:"items" binding:"required"`
}

type CreateGRItemRequest struct {
	PurchaseOrderItemID int     `json:"purchase_order_item_id" binding:"required"`
	QtyGood             int     `json:"qty_good" binding:"min=0"`
	QtyDamaged          int     `json:"qty_damaged" binding:"min=0"`
	Notes               *string `json:"notes,omitempty"`
}

type CreateGRItemInput struct {
	PurchaseOrderItemID int
	QtyGood             int
	QtyDamaged          int
	Notes               *string
}

type PurchaseReceiptPayload struct {
	POID    int                   `json:"po_id"`
	GRID    int                   `json:"gr_id"`
	StoreID int                   `json:"store_id"`
	UserID  int                   `json:"user_id"`
	Items   []PurchaseReceiptItem `json:"items"`
}

type PurchaseReceiptItem struct {
	ProductID int `json:"product_id"`
	QtyGood   int `json:"qty_good"`
	UnitCost  int `json:"unit_cost"`
}
