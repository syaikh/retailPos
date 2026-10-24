package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSplitTaxInclusive pins the shared tax-inclusive split that
// sale.computeLineTotals (checkout) and salereturn.lineRefund (refund) both
// delegate to, so a sale and its later refund can never disagree on the
// rounding: DPP is the rounded tax base, the tax is the remainder, and
// DPP + tax always reconstructs the tax-inclusive subtotal exactly.
func TestSplitTaxInclusive(t *testing.T) {
	tests := []struct {
		name     string
		subtotal int
		rate     float64
		wantDPP  int
		wantTax  int
	}{
		{name: "zero rate is fully DPP", subtotal: 3500, rate: 0, wantDPP: 3500, wantTax: 0},
		{name: "negative rate is fully DPP", subtotal: 3500, rate: -5, wantDPP: 3500, wantTax: 0},
		{name: "11 percent rounds the base up", subtotal: 5000, rate: 11, wantDPP: 4505, wantTax: 495},
		{name: "11 percent on a larger base", subtotal: 10000, rate: 11, wantDPP: 9009, wantTax: 991},
		{name: "zero subtotal splits to zero", subtotal: 0, rate: 11, wantDPP: 0, wantTax: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dpp, tax := SplitTaxInclusive(tc.subtotal, tc.rate)
			assert.Equal(t, tc.wantDPP, dpp, "DPP")
			assert.Equal(t, tc.wantTax, tax, "tax")
			assert.Equal(t, tc.subtotal, dpp+tax, "DPP + tax must equal the tax-inclusive subtotal exactly")
		})
	}
}
