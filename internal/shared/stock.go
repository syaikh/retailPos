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
// absolute value (upserting the global product_stock row). It is the
// cross-module contract between internal/stockopname (consumer) and
// internal/inventory (single-writer of product_stock).
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

// InventoryMovement is one row of the inventory_movements ledger, owned by
// internal/inventory (ADR Modular_Monolith_Module_Boundaries §2.8
// transaksional). It is the cross-module contract between internal/stockopname
// (consumer) and internal/inventory (single-writer of the movement ledger):
// stock opname posting appends adjustment movements within the posting Unit of
// Work (ADR_Cross_Module_Transaction_Strategy), so the write runs on the
// caller's tx.
type InventoryMovement struct {
	ProductID      int
	QuantityChange int
	Type           string
	ReferenceID    int
	ReferenceTable string
	UserID         int
	Notes          string
}

// ConsignmentStockDelta is a signed adjustment to a product's global
// product_stock row plus a matching inventory_movements ledger entry. It is the
// cross-module contract between internal/consignment (consumer) and
// internal/inventory (single-writer of product_stock and the movement ledger,
// ADR Modular_Monolith_Module_Boundaries §2.8 transaksional): consignment
// receipts/pending returns/returns write through this port inside the caller's
// Unit of Work (ADR_Cross_Module_Transaction_Strategy), so each write runs on
// the caller's tx. The consignment ownership ledger (consignment_stock) is
// separate and owned by internal/consignment.
type ConsignmentStockDelta struct {
	ProductID      int
	Delta          int
	MovementType   string
	ReferenceID    int
	ReferenceTable string
	UserID         int
	Notes          string
}

// ConsignmentCheckoutItem is one sale line fed to the checkout-time consignment
// resolver. It is the cross-module contract between internal/sale (consumer of
// the ConsignmentCheckout port) and internal/consignment (owner of the
// consignment_stock ledger). The unit price is the ACTUAL sale unit price from
// the pricing engine (BR-15/AC-C11: sales follow existing pricing; settlement
// uses the price when the sale happened), so the resolver can snapshot the
// store share without trusting client prices.
type ConsignmentCheckoutItem struct {
	ProductID int
	Quantity  int
	UnitPrice int
}

// ConsignmentSaleRecord is the checkout-time snapshot of one consignment sale
// line, produced by the resolver and persisted to consignment_sale_items after
// the sale row is created. StoreShareType/StoreShareValue are snapshotted from
// the term in effect at sale time (BR-19/AC-C11); the store share amount is
// derived from the ACTUAL sale unit price (PRD §10.2/§10.3), not the term price.
type ConsignmentSaleRecord struct {
	SaleID          int
	InvoiceNumber   string
	ProductID       int
	SupplierID      int
	ArrangementID   int
	StoreID         int
	Quantity        int
	UnitPrice       int
	Subtotal        int
	StoreShareType  string
	StoreShareValue float64
}
