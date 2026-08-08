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

// StockSetItem is the minimal input for setting a product's global stock to an
// absolute value (upserting the global product_stock row and syncing the
// products.stock column). It is the cross-module contract between
// internal/stockopname (consumer) and internal/inventory (single-writer of
// product_stock).
type StockSetItem struct {
	ProductID int
	Quantity  int
}

// StockRowSet is the input for setting a product's stock row to an absolute
// value, either the store-scoped row (StoreID != nil) or the global row
// (StoreID == nil, i.e. NULL warehouse/store/location). It is the cross-module
// contract between internal/product (consumer) and internal/inventory
// (single-writer of product_stock).
type StockRowSet struct {
	ProductID int
	StoreID   *int
	Quantity  int
}

// LocationStockReconcile is the input for a location-scoped stock reconcile:
// apply a signed delta to a product's rack row and recompute the global row
// from the reconciled rack share (max(global-rack, 0) + newRack). It is the
// cross-module contract between internal/stockopname (consumer) and
// internal/inventory (single-writer of product_stock).
type LocationStockReconcile struct {
	ProductID   int
	LocationID  int
	WarehouseID *int
	StoreID     *int
	Delta       int
}
