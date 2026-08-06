package shared

import "errors"

// ErrInsufficientStock is returned when a stock deduction cannot be fulfilled
// because the available quantity is lower than the requested quantity. It lives
// in shared so that consumer-side ports (internal/sale) and provider-side
// implementations (internal/inventory) can reference the same sentinel without
// importing each other.
var ErrInsufficientStock = errors.New("insufficient stock")

// StockDeductItem is the minimal input for a stock deduction. It is the
// cross-module contract between internal/sale (consumer) and internal/inventory
// (single-writer of product_stock, see ADR_Modular_Monolith_Module_Boundaries).
type StockDeductItem struct {
	ProductID int
	Quantity  int
}
