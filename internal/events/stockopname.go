package events

const (
	TopicStockOpnameCreated   = "stock_opname.created"
	TopicStockOpnameOpened    = "stock_opname.opened"
	TopicStockOpnameSubmitted = "stock_opname.submitted"
	TopicStockOpnameApproved  = "stock_opname.approved"
	TopicStockOpnamePosted    = "stock_opname.posted"
	TopicStockOpnameClosed    = "stock_opname.closed"
	TopicStockOpnameRejected  = "stock_opname.rejected"
	TopicStockOpnameRecount   = "stock_opname.needs_recount"
	TopicStockOpnameCancelled = "stock_opname.cancelled"
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
