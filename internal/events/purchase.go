package events

const (
	TopicPurchaseReceiptCompleted = "purchase_receipt.completed.v1"
	TopicPOCreated                = "purchase_order.created.v1"
	TopicPOConfirmed              = "purchase_order.confirmed.v1"
	TopicPOCancelled              = "purchase_order.cancelled.v1"
	TopicGoodsReceiptCreated      = "goods_receipt.created.v1"
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
