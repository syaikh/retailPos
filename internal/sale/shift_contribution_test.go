package sale

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShiftContributionChangeDue(t *testing.T) {
	cash := func(code string, amount int) Payment {
		return Payment{PaymentMethodCode: code, Amount: amount}
	}

	// Exact cash sale: no change, cash retained == tendered.
	c := shiftContribution(1, 2, 5000, 0, []Payment{cash("CASH", 5000)})
	assert.Equal(t, 5000, c.CashSales)
	assert.Equal(t, 0, c.NonCashSales)

	// Cash over-tender: 8000 tendered for 5000, 3000 change returned.
	// The drawer physically holds 5000, so expected cash must be 5000, not 8000.
	c = shiftContribution(1, 2, 5000, 3000, []Payment{cash("CASH", 8000)})
	assert.Equal(t, 5000, c.CashSales)
	assert.Equal(t, 0, c.NonCashSales)

	// Mixed with cash change: cash 8000 + QRIS 2000 for total 9000 -> 1000 change.
	c = shiftContribution(1, 2, 9000, 1000, []Payment{cash("CASH", 8000), cash("QRIS", 2000)})
	assert.Equal(t, 7000, c.CashSales) // 8000 - 1000
	assert.Equal(t, 2000, c.NonCashSales)

	// Mixed exact: cash 5000 + QRIS 5000, no change.
	c = shiftContribution(1, 2, 10000, 0, []Payment{cash("CASH", 5000), cash("QRIS", 5000)})
	assert.Equal(t, 5000, c.CashSales)
	assert.Equal(t, 5000, c.NonCashSales)
}
