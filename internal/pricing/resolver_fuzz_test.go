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

// FuzzSpecificityScore exercises the scope-precedence scoring that decides which
// rule wins when several match a product. The precedence is product (3) >
// category (2) > brand (1) > none (0), and every dereference must stay nil-safe:
// a rule scoped to a category must not panic when the caller passes a nil
// category id, and a query id must not be dereferenced when the rule leaves that
// scope unset.
func FuzzSpecificityScore(f *testing.F) {
	f.Add(uint8(0), uint8(0), 7, 7)
	f.Add(uint8(0), uint8(1), 7, 7)
	f.Add(uint8(1), uint8(3), 7, 7)
	f.Add(uint8(2), uint8(2), 7, 7)
	f.Add(uint8(3), uint8(3), 7, 7)
	f.Add(uint8(1), uint8(0), 7, 8)

	f.Fuzz(func(t *testing.T, scopeSel, querySel uint8, ruleID, queryID int) {
		rule := Rule{}
		switch scopeSel % 4 {
		case 0:
			rule.ProductID = &ruleID
		case 1:
			rule.CategoryID = &ruleID
		case 2:
			rule.BrandID = &ruleID
		case 3: // unscoped rule
		}

		productID := queryID
		var categoryID, brandID *int
		switch querySel % 4 {
		case 1:
			categoryID = &queryID
		case 2:
			brandID = &queryID
		case 3:
			categoryID, brandID = &queryID, &queryID
		}

		got := specificityScore(rule, productID, categoryID, brandID)

		want := 0
		switch {
		case rule.ProductID != nil && *rule.ProductID == productID:
			want = 3
		case rule.CategoryID != nil && categoryID != nil && *rule.CategoryID == *categoryID:
			want = 2
		case rule.BrandID != nil && brandID != nil && *rule.BrandID == *brandID:
			want = 1
		}
		if got != want {
			t.Fatalf("specificityScore(product=%d category=%v brand=%v, rule product=%v category=%v brand=%v) = %d, want %d",
				productID, categoryID, brandID, rule.ProductID, rule.CategoryID, rule.BrandID, got, want)
		}
		if got < 0 || got > 3 {
			t.Fatalf("specificityScore returned %d outside 0..3", got)
		}
	})
}
