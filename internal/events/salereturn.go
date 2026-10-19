package events

const (
	TopicSaleReturned = "sale.returned.v1"
)

// SaleReturned is the cross-module payload published after a sales return is
// completed (auto-approved or approved by a manager). Consumers (reporting
// invalidation, websocket broadcast, future restock follow-ups) must rely on
// this DTO instead of importing the salereturn module. Returns awaiting
// approval publish nothing yet: no stock moved and no money left the drawer.
type SaleReturned struct {
	ID                int    `json:"id"`
	ReturnNumber      string `json:"return_number"`
	OriginalSaleID    int    `json:"original_sale_id"`
	StoreID           int    `json:"store_id"`
	ReturnType        string `json:"return_type"`
	TotalRefundAmount int    `json:"total_refund_amount"`
	ItemCount         int    `json:"item_count"`
}
