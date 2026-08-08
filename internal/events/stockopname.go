package events

const (
	TopicStockOpnameCreated   = "stock_opname.created.v1"
	TopicStockOpnameOpened    = "stock_opname.opened.v1"
	TopicStockOpnameSubmitted = "stock_opname.submitted.v1"
	TopicStockOpnameApproved  = "stock_opname.approved.v1"
	TopicStockOpnamePosted    = "stock_opname.posted.v1"
	TopicStockOpnameClosed    = "stock_opname.closed.v1"
	TopicStockOpnameRejected  = "stock_opname.rejected.v1"
	TopicStockOpnameRecount   = "stock_opname.needs_recount.v1"
	TopicStockOpnameCancelled = "stock_opname.cancelled.v1"
)

// StockOpnameStatusChanged is the cross-module payload published on a stock
// opname status transition. Consumers (websocket broadcast) must rely on this
// DTO instead of importing the stockopname module. StoreID 0 means the session
// is global (not store-scoped).
type StockOpnameStatusChanged struct {
	SessionID     int    `json:"session_id"`
	SessionNumber string `json:"session_number"`
	Status        string `json:"status"`
	StoreID       int    `json:"store_id"`
}
