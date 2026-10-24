package shared

import "math"

// SplitTaxInclusive splits a tax-INCLUSIVE line amount into its DPP (the rounded
// tax base) and the tax portion. It is the single source of truth for the
// DPP/tax arithmetic that internal/sale (computeLineTotals, checkout) and
// internal/salereturn (lineRefund, refund) must agree on, so the rounding rule
// cannot drift between a sale and its later refund.
//
// The line subtotal is tax-inclusive (subtotal = qty * unit_price). DPP is the
// subtotal with the tax backed out, rounded to the nearest rupiah; the tax is
// the remainder, which guarantees DPP + tax == subtotal exactly (no rounding
// leak). A non-positive taxRate means the line carries no tax, so the whole
// amount is DPP and the tax is zero.
func SplitTaxInclusive(subtotal int, taxRate float64) (dpp, tax int) {
	if taxRate <= 0 {
		return subtotal, 0
	}
	dpp = int(math.Round(float64(subtotal) * 100.0 / (100.0 + taxRate)))
	return dpp, subtotal - dpp
}
