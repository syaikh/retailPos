// Package salereturn implements the Sales Return (Retur Penjualan) feature,
// PRD docs/prd/PRD-Sales-Return.md and the amended plan in
// docs/design/sales-return-plan.md.
//
// Phase 1 scope (backend only):
//   - A customer returns damaged items they bought; the returned unit is
//     written off (never restocked). Per line, the system offers a
//     replacement (same product handed over, stock deducted) when stock is
//     available, otherwise a refund of the tax-inclusive line value to the
//     original payment method (cash from the drawer / non-cash back).
//   - Refunds above the app_settings 'return.approval_threshold' wait for a
//     manager (sale.return.approve) within the
//     'return.approval_timeout_minutes' window; the stock/outcome decision is
//     re-run inside the approval transaction.
//   - A completed return adjusts the processing cashier's open shift running
//     totals via internal/shift's TotalUpdater (ApplyReturnAdjustment); with
//     no open shift the return is rejected.
//   - Consignment-owned lines are rejected outright (follow-up work will add
//     consignment restock).
//
// The module owns exactly five tables (internal/archtest strictModuleTables):
// sale_returns, sale_return_items, sale_return_replacements,
// sale_return_payments, return_approvals. Every cross-module read/write goes
// through consumer-side ports in ports.go (ADR_Modular_Monolith_Module_Boundaries).
package salereturn

import (
	"errors"
	"time"
)

// Return and item lifecycle values (migration 067 check constraints).
const (
	ReturnStatusPendingApproval = "pending_approval"
	ReturnStatusCompleted       = "completed"
	ReturnStatusCancelled       = "cancelled"
	ReturnStatusExpired         = "expired"

	ReturnTypeReplacement = "replacement"
	ReturnTypeRefund      = "refund"

	// Phase 1 accepts only the PRD-locked 'damaged' condition: every returned
	// unit is written off, so a functional unit would be indistinguishable
	// from an exchange of sellable goods.
	ConditionDamaged    = "damaged"
	ConditionFunctional = "functional"

	// Phase 1 never restocks a returned unit (review decision B1): the outcome
	// is always 'written_off'. 'restocked' is reserved by the schema for the
	// future follow-up that inspects and re-shelves functional returns.
	OutcomeWrittenOff = "written_off"
	OutcomeRestocked  = "restocked"
)

// Domain errors. The Indonesian messages are the PRD §12 user-facing texts;
// they intentionally start lowercase (revive error-strings).
var (
	ErrSaleNotFound     = errors.New("sale not found")
	ErrSaleNotCompleted = errors.New("sale is not completed")
	ErrSaleItemNotFound = errors.New("sale item not found")
	ErrStoreMismatch    = errors.New("sale belongs to another store")
	// ErrShiftNotOpen replaces the PRD's "skip the shift adjustment and accept
	// drift" edge case (§12.1 #1): a refund must not be posted without a
	// drawer to pay it from, so the whole return is rejected.
	ErrShiftNotOpen        = errors.New("shift sudah ditutup, retur tidak dapat diproses")
	ErrQtyExceedsSold      = errors.New("qty retur melebihi qty terjual")
	ErrReturnWindowExpired = errors.New("return window expired")
	// ErrConsignmentNotSupported rejects consignment-owned lines until the
	// consignment restock follow-up lands (review decision).
	ErrConsignmentNotSupported = errors.New("retur untuk barang konsinyasi belum didukung")
	ErrConditionUnsupported    = errors.New("hanya kondisi 'rusak' (damaged) yang didukung pada fase ini")

	ErrReturnNotFound   = errors.New("return not found")
	ErrReturnNotPending = errors.New("return is not pending approval")
	ErrApprovalExpired  = errors.New("approval window has expired")
	ErrInvalidReturn    = errors.New("invalid return request")

	// ErrStockChanged aborts an immediate return whose refund grew past the
	// approval threshold when a replacement line demoted to refund during the
	// stock deduction (a concurrent sale took the stock). The deduction
	// cannot be undone in place, so the transaction rolls back — undoing it —
	// and the cashier retries; the retry plans the demoted line as a refund
	// from the start and correctly routes to pending approval.
	ErrStockChanged = errors.New("stok berubah saat memproses retur, silakan coba lagi")
)

// Actor identifies the caller for audit logging (PRD §13) and for the
// cashier/shift association of a processed return.
type Actor struct {
	ID        int
	Username  string
	Role      string
	StoreID   *int
	IPAddress string
	UserAgent string
}

// ItemRequest is one requested return line: how many units of one original
// sale line. The replacement/refund split and all amounts are computed
// server-side; client-supplied amounts are deliberately not part of the
// contract.
type ItemRequest struct {
	SaleItemID int
	Quantity   int
	Notes      *string
}

// ProcessRequest is the POST /sales/:id/return body.
type ProcessRequest struct {
	Reason    string
	Condition string
	Notes     *string
	Items     []ItemRequest
}

// Return is the sale_returns header plus, on detail reads, its lines,
// refund payments and approval record.
type Return struct {
	ID                int
	ReturnNumber      string
	OriginalSaleID    int
	CashierID         int
	StoreID           int
	CustomerID        *int
	ShiftID           int
	ReturnType        string
	Reason            string
	Condition         string
	Notes             *string
	Subtotal          int
	Discount          int
	RestockingFee     int
	TaxReversal       int
	TotalRefundAmount int
	Status            string
	ChangeDue         int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ExpiresAt         *time.Time

	Items    []ReturnItem
	Payments []ReturnPayment
	Approval *Approval
}

// ReturnItem is one sale_return_items row: a returned slice of an original
// sale line. Subtotal/DppAmount/TaxAmount follow the same inclusive-tax math
// as the sale line they mirror (see lineRefund).
type ReturnItem struct {
	ID           int
	SaleItemID   int
	ProductID    int
	ProductName  string
	Quantity     int
	UnitPrice    int
	Subtotal     int
	DppAmount    int
	TaxAmount    int
	TaxRate      float64
	Condition    string
	Outcome      string
	Notes        *string
	Replacements []Replacement
}

// Replacement is one sale_return_replacements row: units of the same product
// handed back to the customer in exchange (their stock was deducted).
type Replacement struct {
	ID        int
	ItemID    int
	ProductID int
	Quantity  int
}

// ReturnPayment is one sale_return_payments row: how much of the refund was
// paid out through one payment method.
type ReturnPayment struct {
	ID                int
	PaymentMethodID   int
	PaymentMethodCode string
	Amount            int
}

// Approval is the return_approvals row governing a pending return.
type Approval struct {
	ID             int
	RequestedBy    int
	ApprovedBy     *int
	Status         string
	Reason         *string
	RejectedReason *string
	ExpiresAt      time.Time
}
