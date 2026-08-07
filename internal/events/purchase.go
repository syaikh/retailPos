package events

const (
	TopicPurchaseReceiptCompleted = "PurchaseReceiptCompleted"
	TopicPOCreated                = "purchase_order.created"
	TopicPOConfirmed              = "purchase_order.confirmed"
	TopicPOCancelled              = "purchase_order.cancelled"
	TopicGoodsReceiptCreated      = "goods_receipt.created"
)

// PurchaseOrderEvent is the cross-module payload published when a purchase
// order transitions state (created/confirmed/cancelled). StoreID 0 means the
// order is not store-scoped.
type PurchaseOrderEvent struct {
	POID     int    `json:"po_id"`
	PONumber string `json:"po_number"`
	StoreID  int    `json:"store_id"`
}

// GoodsReceiptCreated is the cross-module payload published after a goods
// receipt is created. StoreID 0 means the receipt is not store-scoped.
type GoodsReceiptCreated struct {
	GRID                int    `json:"gr_id"`
	GRNumber            string `json:"gr_number"`
	DeliveryOrderNumber string `json:"delivery_order_number"`
	POID                int    `json:"po_id"`
	PONumber            string `json:"po_number"`
	StoreID             int    `json:"store_id"`
}

type PurchaseReceiptCompleted struct {
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
