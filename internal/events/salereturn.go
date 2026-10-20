package events

const (
	TopicSaleReturned            = "sale.returned.v1"
	TopicReturnApprovalRequested = "sale.return.approval_requested.v1"
	TopicReturnApprovalResolved  = "sale.return.approval_resolved.v1"
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

// ReturnApprovalRequested is the cross-module payload published after a return
// is parked pending manager approval (PRD §7). Consumers (websocket broadcast)
// must rely on this DTO instead of importing the salereturn module.
//
// The payload is deliberately self-contained (invoice, cashier, amount,
// deadline) so the manager's live notification can be rendered without a
// follow-up API call; ReturnNumber doubles as the human-readable handle the
// approval queue already shows. StoreID 0 means "global" and broadcasts to
// every store (storeIDPtr maps it to nil).
type ReturnApprovalRequested struct {
	ReturnID          int    `json:"return_id"`
	ReturnNumber      string `json:"return_number"`
	OriginalSaleID    int    `json:"original_sale_id"`
	InvoiceNumber     string `json:"invoice_number"`
	CashierID         int    `json:"cashier_id"`
	CashierName       string `json:"cashier_name"`
	StoreID           int    `json:"store_id"`
	TotalRefundAmount int    `json:"total_refund_amount"`
	ItemCount         int    `json:"item_count"`
	ExpiresAt         string `json:"expires_at"`
}

// ReturnApprovalResolved is the cross-module payload published after a pending
// return is approved, rejected or expired (PRD §7). Status is the terminal
// return status ("completed", "cancelled" or "expired") so the cashier's live
// queue drops the request and the downstream consumers can branch on it.
type ReturnApprovalResolved struct {
	ReturnID     int    `json:"return_id"`
	ReturnNumber string `json:"return_number"`
	StoreID      int    `json:"store_id"`
	Status       string `json:"status"`
}
