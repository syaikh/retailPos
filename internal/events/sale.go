package events

const (
	TopicSaleCreated = "sale.created.v1"
)

// SaleCreated is the cross-module payload published after a sale is completed.
// Consumers (report dashboard invalidation, websocket broadcast) must rely on
// this DTO instead of importing the sale module.
type SaleCreated struct {
	ID            int    `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	StoreID       *int   `json:"store_id"`
	TotalAmount   int    `json:"total_amount"`
	ItemCount     int    `json:"item_count"`
}
