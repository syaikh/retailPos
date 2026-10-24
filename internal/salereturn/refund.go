package salereturn

import (
	"sort"
	"strings"

	"retail-pos-system/internal/shared"
)

// CashMethodCode is the payment method code whose refunds leave the physical
// drawer. It aliases the shared canonical constant so the sale checkout, the
// salereturn refund split, and the shift/return SQL all agree on one value.
const CashMethodCode = shared.PaymentMethodCash

func isCash(code string) bool {
	return strings.EqualFold(code, CashMethodCode)
}

// lineRefund computes the refunded slice of one original sale line, using the
// same shared tax-inclusive split as sale.computeLineTotals: the line subtotal
// is tax-INCLUSIVE (subtotal = qty*unit_price), DPP is the rounded tax base and
// the tax portion is the remainder. Refunding qty units therefore returns
// qty*unit_price — NOT subtotal+tax, which would double-count the tax. taxRate
// comes from the snapshot on sale_items, so later tax changes never rewrite
// history.
func lineRefund(qty, unitPrice int, taxRate float64) (subtotal, dpp, tax int) {
	subtotal = qty * unitPrice
	dpp, tax = shared.SplitTaxInclusive(subtotal, taxRate)
	return subtotal, dpp, tax
}

// refundAllocation is one row of sale_return_payments.
type refundAllocation struct {
	PaymentMethodID   int
	PaymentMethodCode string
	Amount            int
}

// allocateRefund splits a total refund across the original sale's payment
// methods proportionally to each method's NET contribution, so a method is
// never refunded more than it actually took:
//
//   - cash nets the sale's cash payments minus the sale's change due (the
//     change never entered the drawer; sale_payments has no change column,
//     sales.change_due does),
//   - non-cash methods contribute their raw amounts.
//
// Each method gets floor(total*net/totalNet); the rounding remainder (always
// >= 0 and strictly less than the number of methods) is handed out one unit
// at a time by descending fractional part, so the allocation sums exactly to
// the total. Methods whose final share is zero are dropped.
func allocateRefund(payments []shared.ReturnSalePayment, changeDue, total int) []refundAllocation {
	if total <= 0 || len(payments) == 0 {
		return nil
	}

	nets := make([]int, len(payments))
	totalNet := 0
	changeLeft := changeDue
	for i, p := range payments {
		amt := p.Amount
		if isCash(p.PaymentMethodCode) && changeLeft > 0 {
			deduct := min(amt, changeLeft)
			amt -= deduct
			changeLeft -= deduct
		}
		nets[i] = amt
		totalNet += amt
	}
	if totalNet <= 0 {
		// Degenerate sale whose payments are fully consumed by change (cannot
		// occur for a completed sale, where Σpayments = total + change): fall
		// back to charging everything to the first method rather than losing
		// the refund entirely.
		p := payments[0]
		return []refundAllocation{{
			PaymentMethodID:   p.PaymentMethodID,
			PaymentMethodCode: p.PaymentMethodCode,
			Amount:            total,
		}}
	}

	type row struct {
		payment shared.ReturnSalePayment
		base    int
		frac    int
	}
	rows := make([]row, len(payments))
	remaining := total
	for i, p := range payments {
		prod := total * nets[i]
		rows[i] = row{payment: p, base: prod / totalNet, frac: prod % totalNet}
		remaining -= rows[i].base
	}

	// Largest remainder first; ties keep the original payment order (stable).
	order := make([]int, len(rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return rows[order[a]].frac > rows[order[b]].frac
	})
	for k := 0; k < remaining; k++ {
		rows[order[k]].base++
	}

	out := make([]refundAllocation, 0, len(rows))
	for _, r := range rows {
		if r.base > 0 {
			out = append(out, refundAllocation{
				PaymentMethodID:   r.payment.PaymentMethodID,
				PaymentMethodCode: r.payment.PaymentMethodCode,
				Amount:            r.base,
			})
		}
	}
	return out
}

// cashAndNonCash splits an allocation into the drawer payout (cash) and the
// non-cash remainder — the two deltas a shift's running totals need.
func cashAndNonCash(alloc []refundAllocation) (cash, nonCash int) {
	for _, a := range alloc {
		if isCash(a.PaymentMethodCode) {
			cash += a.Amount
		} else {
			nonCash += a.Amount
		}
	}
	return cash, nonCash
}
