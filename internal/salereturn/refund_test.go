package salereturn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// lineRefund must mirror sale.computeLineTotals: tax-INCLUSIVE subtotal,
// DPP the rounded base, tax the remainder — refunding never double-counts.
func TestLineRefund(t *testing.T) {
	cases := []struct {
		name      string
		qty       int
		unitPrice int
		taxRate   float64
		wantSub   int
		wantDPP   int
		wantTax   int
	}{
		{"zero rate: DPP equals the whole refund", 3, 1000, 0, 3000, 3000, 0},
		{"11 percent: 111000 refunds 100000 DPP + 11000 tax", 1, 111000, 11, 111000, 100000, 11000},
		{"10 percent with a rounding remainder", 2, 45000, 10, 90000, 81818, 8182},
		{"rounding down at 11 percent", 1, 9999, 11, 9999, 9008, 991},
		{"negative rate treated as no tax", 1, 5000, -1, 5000, 5000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub, dpp, tax := lineRefund(tc.qty, tc.unitPrice, tc.taxRate)
			require.Equal(t, tc.wantSub, sub)
			require.Equal(t, tc.wantDPP, dpp)
			require.Equal(t, tc.wantTax, tax)
			// Invariant: DPP + tax reconstructs the inclusive subtotal.
			require.Equal(t, sub, dpp+tax)
		})
	}
}

func pay(id int, code string, amount int) shared.ReturnSalePayment {
	return shared.ReturnSalePayment{PaymentMethodID: id, PaymentMethodCode: code, Amount: amount}
}

func TestAllocateRefund_DegenerateInputs(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(1, "CASH", 5000)}
	assert.Nil(t, allocateRefund(payments, 0, 0), "zero total allocates nothing")
	assert.Nil(t, allocateRefund(payments, 0, -100), "negative total allocates nothing")
	assert.Nil(t, allocateRefund(nil, 0, 1000), "no payments means no split")
}

// Cash nets the sale's cash payments by the change due (change never entered
// the drawer), and every method is refunded strictly proportionally to its net
// contribution.
func TestAllocateRefund_ProportionalWithChangeDue(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(1, "CASH", 60000), pay(2, "QRIS", 40000)}
	alloc := allocateRefund(payments, 5000, 19000)

	require.Len(t, alloc, 2)
	// Nets: cash 55000 (60000-5000 change), QRIS 40000; totalNet 95000.
	// 19000 * 55000/95000 = 11000 exact, 19000 * 40000/95000 = 8000 exact.
	assert.Equal(t, 11000, alloc[0].Amount)
	assert.Equal(t, "CASH", alloc[0].PaymentMethodCode)
	assert.Equal(t, 8000, alloc[1].Amount)
	assert.Equal(t, "QRIS", alloc[1].PaymentMethodCode)
	assert.Equal(t, 19000, alloc[0].Amount+alloc[1].Amount, "allocation must sum to the total")
}

// Largest-remainder distribution: 10 across nets 1:2 gives floor 3+6, the
// single leftover unit goes to the larger fractional part (QRIS), for 3+7.
func TestAllocateRefund_LargestRemainder(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(1, "CASH", 1), pay(2, "QRIS", 2)}
	alloc := allocateRefund(payments, 0, 10)

	require.Len(t, alloc, 2)
	assert.Equal(t, 3, alloc[0].Amount)
	assert.Equal(t, 7, alloc[1].Amount)
	assert.Equal(t, 10, alloc[0].Amount+alloc[1].Amount)
}

// A method whose computed share rounds to zero is dropped, not kept at 0.
func TestAllocateRefund_ZeroShareDropped(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(1, "CASH", 1000000), pay(2, "QRIS", 1)}
	alloc := allocateRefund(payments, 0, 1)

	require.Len(t, alloc, 1, "zero-share rows must not appear in the split")
	assert.Equal(t, "CASH", alloc[0].PaymentMethodCode)
	assert.Equal(t, 1, alloc[0].Amount)
}

// Change is only ever netted against CASH rows: a non-cash method's full
// contribution stays refundable.
func TestAllocateRefund_ChangeSkipsNonCashRows(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(2, "QRIS", 5000), pay(1, "CASH", 5000)}
	alloc := allocateRefund(payments, 5000, 5000)

	require.Len(t, alloc, 1)
	assert.Equal(t, "QRIS", alloc[0].PaymentMethodCode)
	assert.Equal(t, 5000, alloc[0].Amount, "cash nets to zero, QRIS refunds in full")
}

// Change walks cash rows in order, so a later cash row keeps its remainder.
func TestAllocateRefund_ChangeSpansCashRowsInOrder(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(1, "CASH", 3000), pay(2, "CASH", 4000)}
	alloc := allocateRefund(payments, 5000, 2000)

	require.Len(t, alloc, 1)
	assert.Equal(t, 2, alloc[0].PaymentMethodID, "first cash row is fully consumed by change")
	assert.Equal(t, 2000, alloc[0].Amount)
}

// Degenerate sale whose payments are fully consumed by change cannot lose the
// refund: it falls back to the first method.
func TestAllocateRefund_FullyConsumedByChangeFallsBack(t *testing.T) {
	payments := []shared.ReturnSalePayment{pay(1, "CASH", 100)}
	alloc := allocateRefund(payments, 100, 50)

	require.Len(t, alloc, 1)
	assert.Equal(t, 50, alloc[0].Amount)
}

func TestCashAndNonCash(t *testing.T) {
	cash, nonCash := cashAndNonCash([]refundAllocation{
		{PaymentMethodID: 1, PaymentMethodCode: "cash", Amount: 7000},
		{PaymentMethodID: 2, PaymentMethodCode: "CARD", Amount: 3000},
	})
	assert.Equal(t, 7000, cash, "cash detection is case-insensitive")
	assert.Equal(t, 3000, nonCash)

	cash, nonCash = cashAndNonCash(nil)
	assert.Equal(t, 0, cash)
	assert.Equal(t, 0, nonCash)
}
