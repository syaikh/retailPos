package shared

import "errors"

// ShiftSaleContribution is the share of a single completed sale that is
// accumulated onto its shift's running totals. It is the cross-module contract
// between internal/sale (consumer) and internal/shift (single-writer of the
// shifts running totals, see ADR_Modular_Monolith_Module_Boundaries).
type ShiftSaleContribution struct {
	ShiftID      int
	CashierID    int
	TotalAmount  int
	CashSales    int
	NonCashSales int
}

// ErrShiftNotOpen is returned when a sale contribution targets a shift that is
// no longer 'open' (closed concurrently, already closed, or nonexistent).
// Cross-module sentinel so consumers (sale checkout) can map it to a client
// error instead of an internal failure.
var ErrShiftNotOpen = errors.New("shift is not open or no longer exists")
