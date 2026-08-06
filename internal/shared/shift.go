package shared

// ShiftSaleContribution is the share of a single completed sale that is
// accumulated onto its shift's running totals. It is the cross-module contract
// between internal/sale (consumer) and internal/shift (single-writer of the
// shifts running totals, see ADR_Modular_Monolith_Module_Boundaries).
type ShiftSaleContribution struct {
	ShiftID      int
	TotalAmount  int
	CashSales    int
	NonCashSales int
}
