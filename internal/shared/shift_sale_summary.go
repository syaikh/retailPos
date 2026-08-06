package shared

// ShiftSaleSummary is the cross-module contract for the completed-sales totals
// of a shift. internal/sale (canonical single-writer of sales/sale_payments)
// computes it for internal/shift, which persists the result on the shifts row
// at close time.
type ShiftSaleSummary struct {
	TotalCashSales    int
	TotalNonCashSales int
	TotalSales        int
	TotalTransactions int
}
