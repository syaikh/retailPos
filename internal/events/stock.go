package events

const TopicStockAdjusted = "stock.adjusted.v1"

// StockAdjusted is the cross-module payload published after inventory stock
// changes. Consumers (websocket broadcast) must rely on this DTO instead of
// importing the inventory module.
type StockAdjusted struct {
	ProductID      int    `json:"product_id"`
	QuantityChange int    `json:"quantity_change"`
	UserID         int    `json:"user_id"`
	Notes          string `json:"notes"`
}
