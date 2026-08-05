package events

const (
	TopicPurchaseReceiptCompleted = "PurchaseReceiptCompleted"
)

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
