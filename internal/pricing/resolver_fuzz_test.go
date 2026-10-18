package pricing

import (
	"math"
	"testing"
)

// computePriceFuzzMethods maps the fuzzed selector onto the four real pricing
// methods plus one unknown value so the default branch stays covered.
var computePriceFuzzMethods = []Method{
	PricingMethodFixedPrice,
	PricingMethodDiscountPct,
	PricingMethodDiscountAmt,
	PricingMethodMarkupPct,
	"unknown_method",
}

// FuzzComputePrice explores computePrice beyond the pinned table cases in
// TestComputePrice_RoundToNearestCent. It asserts the contracts that must hold
// for every input: no panic, determinism, clamp-at-zero, and half-up rounding
// to the nearest integer cent.
//
// Inputs whose intermediate float is infinite or large enough that the
// float -> int conversion leaves the int64-safe range are reduced to the
// determinism check alone: the Go spec leaves out-of-range float -> int
// conversion implementation-defined, so pinning a value there would encode
// platform behaviour as contract. NaN/Inf PricingValue inputs return early —
// PricingValue is validated gte=0 at the API boundary, so they can never
// reach this function in production, and t.Skip inside a fuzz target stops
// the whole engine instead of discarding one input.
func FuzzComputePrice(f *testing.F) {
	// Seeds mirror the documented rounding contract in
	// TestComputePrice_RoundToNearestCent so the engine always starts from the
	// pinned cases.
	f.Add(29999, 29999.0, uint8(0)) // fixed price, integer cents
	f.Add(20000, 19999.5, uint8(0)) // fixed price, exact half cent
	f.Add(100000, 10.0, uint8(1))   // discount percent, whole
	f.Add(5000, 9000.0, uint8(2))   // discount amount clamps at zero
	f.Add(99997, 15.0, uint8(3))    // markup percent, fractional
	f.Add(12345, 7.5, uint8(4))     // unknown method falls back to base price
	f.Add(-100, 10.0, uint8(2))     // negative base still clamps at zero
	f.Add(0, 0.0, uint8(0))         // zero everything

	f.Fuzz(func(t *testing.T, basePrice int, pricingValue float64, methodSel uint8) {
		if math.IsNaN(pricingValue) || math.IsInf(pricingValue, 0) {
			return
		}
		rule := &Rule{
			Method:       computePriceFuzzMethods[int(methodSel)%len(computePriceFuzzMethods)],
			PricingValue: pricingValue,
		}

		got := computePrice(basePrice, rule)
		if again := computePrice(basePrice, rule); again != got {
			t.Fatalf("computePrice not deterministic for base=%d value=%g method=%s: %d vs %d",
				basePrice, pricingValue, rule.Method, got, again)
		}

		// Reference intermediate, same arithmetic as the implementation; the
		// oracle pins dispatch, clamping, and rounding across inputs the table
		// test never enumerates (negative bases, extreme magnitudes).
		var unitPrice float64
		switch rule.Method {
		case PricingMethodFixedPrice:
			unitPrice = pricingValue
		case PricingMethodDiscountPct:
			unitPrice = float64(basePrice) * (1 - pricingValue/100)
		case PricingMethodDiscountAmt:
			unitPrice = float64(basePrice) - pricingValue
		case PricingMethodMarkupPct:
			unitPrice = float64(basePrice) * (1 + pricingValue/100)
		default:
			unitPrice = float64(basePrice)
		}
		if math.IsInf(unitPrice, 0) || math.Abs(unitPrice) >= 1<<62 {
			return // determinism is the only contract pinned at this magnitude
		}

		want := 0
		if unitPrice >= 0 {
			want = int(unitPrice + 0.5) // half-up to nearest integer cent
		}
		if got != want {
			t.Fatalf("computePrice(base=%d, method=%s, value=%g) = %d, want %d",
				basePrice, rule.Method, pricingValue, got, want)
		}
	})
}
