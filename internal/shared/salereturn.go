package shared

import "time"

// Cross-module contracts for the Sales Return feature (internal/salereturn,
// PRD docs/prd/PRD-Sales-Return.md). The types live here so that internal/sale
// (seller), internal/shift (shift totals), internal/inventory (stock),
// internal/consignment (consignment guard) and internal/appsettings (policy
// settings) can each implement a consumer-side port for salereturn without
// importing it (ADR_Modular_Monolith_Module_Boundaries).

// ReturnSale is the sale header as needed by the return flow: enough to
// validate eligibility (store, status, age) and to compute the refund split
// (change_due, payments). It is returned by internal/sale's SaleReader
// implementation and read on the caller's transaction, so the sale cannot
// change between validation and posting.
type ReturnSale struct {
	ID            int
	InvoiceNumber string
	CashierID     int
	CustomerID    *int
	StoreID       int
	Status        string
	TotalAmount   int
	ChangeDue     int
	CreatedAt     time.Time
	Items         []ReturnSaleItem
	Payments      []ReturnSalePayment
}

// ReturnSaleItem is one line of the original sale that a return may target.
// The monetary fields mirror sale_items: Subtotal/DppAmount/TaxAmount cover
// the WHOLE line (Quantity units) and are tax-inclusive at the line level
// (see sale.computeLineTotals: Subtotal = Quantity*UnitPrice, Dpp = round of
// Subtotal/(1+rate), Tax = Subtotal-Dpp).
type ReturnSaleItem struct {
	ID          int
	ProductID   int
	ProductName string
	Quantity    int
	UnitPrice   int
	Subtotal    int
	DppAmount   int
	TaxAmount   int
	TaxRate     float64
}

// ReturnSalePayment is one payment row of the original sale, with the payment
// method resolved so the return flow can split a refund into cash and non-cash
// without joining payment_methods itself.
type ReturnSalePayment struct {
	PaymentMethodID   int
	PaymentMethodCode string
	Amount            int
}

// ShiftReturnAdjustment is one return's effect on a shift's running totals.
// It is the cross-module contract between internal/salereturn (consumer) and
// internal/shift (single-writer of shifts running totals): a completed return
// reduces cash_sales by the cash paid out of the drawer, non_cash_sales by
// the non-cash refund and total_sales by the total refund. A return is NOT a
// sale transaction, so TransactionCount is deliberately not part of the
// adjustment.
type ShiftReturnAdjustment struct {
	ShiftID       int
	CashRefund    int
	NonCashRefund int
	TotalRefund   int
}

// ShiftReturnTotals is the aggregate refund paid out by one shift, reported
// by internal/shift's ReturnTotalsProvider so internal/shift can net returns
// out of its own stored/live totals (close-time netting, live fallback).
// Rows with status 'pending_approval' are excluded: the money has not left
// the drawer yet.
type ShiftReturnTotals struct {
	CashRefund    int
	NonCashRefund int
	TotalRefund   int
}
