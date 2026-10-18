package sale

import (
	"math"
	"testing"
)

// FuzzComputeLineTotals exercises the per-line money math behind cart_items.
// Subtotal is quantity*unitPrice; with a positive tax rate the line splits into
// DPP and tax, dpp = round(subtotal*100/(100+rate)) and tax = subtotal - dpp.
// The API validates quantity >= 1 and unitPrice >= 0, so this target also
// probes the unvalidated arithmetic: int overflow wraps (never panics) and the
// identity dpp+tax == subtotal must survive every input, wrapped or not.
func FuzzComputeLineTotals(f *testing.F) {
	f.Add(1, 250000, 11.0, true)
	f.Add(3, 33333, 0.0, true)
	f.Add(2, 100000, 0.0, false) // no tax class on the line: nil tax rate
	f.Add(1, 0, 100.0, true)
	f.Add(0, 5000, 11.0, true)
	f.Add(-2, 1000, 11.0, true)

	f.Fuzz(func(t *testing.T, quantity, unitPrice int, taxRate float64, hasRate bool) {
		var ratePtr *float64
		if hasRate {
			ratePtr = &taxRate
		}
		subtotal, dpp, tax := computeLineTotals(quantity, unitPrice, ratePtr)

		if dpp+tax != subtotal {
			t.Fatalf("computeLineTotals(%d, %d, %g) split %d+%d, want subtotal %d",
				quantity, unitPrice, taxRate, dpp, tax, subtotal)
		}
		// int(math.Round(NaN)) is implementation-defined, so only the identity
		// above is meaningful; a numeric DB column cannot produce NaN anyway.
		if math.IsNaN(taxRate) {
			return
		}
		if ratePtr == nil || taxRate <= 0 {
			if dpp != subtotal || tax != 0 {
				t.Fatalf("computeLineTotals(%d, %d, %g, hasRate=%t) with non-positive rate = (%d, %d), want (%d, 0)",
					quantity, unitPrice, taxRate, hasRate, dpp, tax, subtotal)
			}
			return
		}
		// A wrapped (negative) subtotal has no meaningful DPP/tax bounds.
		if subtotal < 0 {
			return
		}
		if dpp < 0 || dpp > subtotal || tax < 0 || tax > subtotal {
			t.Fatalf("computeLineTotals(%d, %d, %g) = subtotal %d, dpp %d, tax %d out of bounds",
				quantity, unitPrice, taxRate, subtotal, dpp, tax)
		}
	})
}

// FuzzAggregateCartItems exercises the checkout-time line collapse. The API
// InsertCartItem is a plain insert with no unique constraint, so duplicate
// lines (same product, unit price, tax class, and pricing rule) can reach
// checkout; collapsing them must not lose quantity or money, and lines that
// differ in any key component must stay separate.
func FuzzAggregateCartItems(f *testing.F) {
	f.Add(1, 250000, 2, 100)
	f.Add(0, 0, 0, 0)
	f.Add(-5, 12345, -3, math.MaxInt)
	f.Add(7, 1, 1, math.MinInt)

	f.Fuzz(func(t *testing.T, productID, unitPrice, quantity, extra int) {
		taxClassID, ruleID := 7, 9
		items := []CartItem{
			{
				ProductID: productID, UnitPrice: unitPrice, Quantity: quantity,
				TaxClassID: &taxClassID, PricingRuleID: &ruleID,
				Subtotal: quantity * unitPrice, DPPAmount: extra, TaxAmount: extra + 1,
			},
			{
				ProductID: productID, UnitPrice: unitPrice, Quantity: quantity + 1,
				TaxClassID: &taxClassID, PricingRuleID: &ruleID,
				Subtotal: (quantity + 1) * unitPrice, DPPAmount: extra * 2, TaxAmount: extra*2 + 1,
			},
			{
				// Same product, different unit price: a distinct key, so it must
				// not be merged with the two above.
				ProductID: productID, UnitPrice: unitPrice + 1, Quantity: quantity + 2,
				Subtotal: (quantity + 2) * (unitPrice + 1), DPPAmount: extra + 3, TaxAmount: extra + 4,
			},
		}

		aggregated := aggregateCartItems(items)
		if len(aggregated) != 2 {
			t.Fatalf("aggregateCartItems collapsed %d lines into %d, want 2", len(items), len(aggregated))
		}

		wantMerged := items[0]
		wantMerged.Quantity += items[1].Quantity
		wantMerged.Subtotal += items[1].Subtotal
		wantMerged.DPPAmount += items[1].DPPAmount
		wantMerged.TaxAmount += items[1].TaxAmount
		if aggregated[0] != wantMerged {
			t.Fatalf("merged line = %+v, want %+v", aggregated[0], wantMerged)
		}
		if aggregated[1] != items[2] {
			t.Fatalf("distinct line = %+v, want unchanged %+v", aggregated[1], items[2])
		}

		// Money and quantity must be conserved across the collapse.
		var inQty, inSub, inDPP, inTax, outQty, outSub, outDPP, outTax int
		for _, item := range items {
			inQty += item.Quantity
			inSub += item.Subtotal
			inDPP += item.DPPAmount
			inTax += item.TaxAmount
		}
		for _, item := range aggregated {
			outQty += item.Quantity
			outSub += item.Subtotal
			outDPP += item.DPPAmount
			outTax += item.TaxAmount
		}
		if inQty != outQty || inSub != outSub || inDPP != outDPP || inTax != outTax {
			t.Fatalf("aggregateCartItems lost money/quantity: in (qty %d, sub %d, dpp %d, tax %d) vs out (qty %d, sub %d, dpp %d, tax %d)",
				inQty, inSub, inDPP, inTax, outQty, outSub, outDPP, outTax)
		}

		// Every emitted line must have a key no other emitted line shares.
		seen := make(map[cartItemKey]bool, len(aggregated))
		for _, item := range aggregated {
			taxClass, hasTax := optionalID(item.TaxClassID)
			rule, hasRule := optionalID(item.PricingRuleID)
			key := cartItemKey{
				ProductID: item.ProductID, UnitPrice: item.UnitPrice,
				TaxClassID: taxClass, HasTax: hasTax,
				PricingRuleID: rule, HasRule: hasRule,
			}
			if seen[key] {
				t.Fatalf("aggregateCartItems emitted two lines with key %+v", key)
			}
			seen[key] = true
		}
	})
}
