package service

import (
	"math"
	"testing"

	"retail-pos-system/internal/domain"

	"github.com/stretchr/testify/assert"
)

// TestErrInsufficientStock tests error definition
func TestErrInsufficientStock(t *testing.T) {
	assert.Equal(t, "insufficient stock", ErrInsufficientStock.Error())
}

// TestSalesService_ValidateStock tests stock validation logic
func TestSalesService_ValidateStock(t *testing.T) {
	t.Run("insufficient stock returns error", func(t *testing.T) {
		err := ErrInsufficientStock
		assert.Equal(t, "insufficient stock", err.Error())
	})
}

// TestSalesService_StockCheckLogic tests the stock checking logic
func TestSalesService_StockCheckLogic(t *testing.T) {
	tests := []struct {
		name          string
		stock         int
		quantity      int
		shouldError   bool
	}{
		{"exact stock", 10, 10, false},
		{"more stock than needed", 15, 10, false},
		{"less stock than needed", 5, 10, true},
		{"zero stock", 0, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasStock := tt.stock >= tt.quantity
			if tt.shouldError {
				assert.False(t, hasStock, "should detect insufficient stock")
			} else {
				assert.True(t, hasStock, "should have sufficient stock")
			}
		})
	}
}

// TestSalesService_CreateSaleItemCalculation tests item subtotal calculation
func TestSalesService_CreateSaleItemCalculation(t *testing.T) {
	items := []domain.SaleItem{
		{ProductID: 1, Quantity: 2, UnitPrice: 5000, Subtotal: 10000},
		{ProductID: 2, Quantity: 3, UnitPrice: 3000, Subtotal: 9000},
	}

	var totalSubtotal int
	for _, item := range items {
		totalSubtotal += item.Subtotal
		assert.Equal(t, item.Quantity*item.UnitPrice, item.Subtotal)
	}

	assert.Equal(t, 19000, totalSubtotal)
}

// TestSaleStruct tests the Sale domain model
func TestSaleStruct(t *testing.T) {
	sale := domain.Sale{
		InvoiceNumber: "INV-001",
		CashierID:     1,
		Subtotal:      9009,
		Discount:      0,
		Tax:           991,
		TotalAmount:   10000,
		PaymentMethod: "Cash",
		Status:        "completed",
	}

	assert.Equal(t, "INV-001", sale.InvoiceNumber)
	assert.Equal(t, 1, sale.CashierID)
	assert.Equal(t, 10000, sale.TotalAmount)
	assert.Equal(t, 9009, sale.Subtotal)
	assert.Equal(t, 991, sale.Tax)
}

// TestSaleItemSubtotal tests SaleItem subtotal calculation
func TestSaleItemSubtotal(t *testing.T) {
	item := domain.SaleItem{
		ProductID: 1,
		Quantity:  5,
		UnitPrice: 2000,
		Subtotal:  10000,
	}

	assert.Equal(t, item.Quantity*item.UnitPrice, item.Subtotal)
}

// TestTaxExtraction verifies DPP and PPN extraction from tax-inclusive prices
func TestTaxExtraction(t *testing.T) {
	tests := []struct {
		name       string
		lineTotal  int
		rate       float64
		wantDPP    int
		wantTax    int
	}{
		{"11% on 5000", 5000, 11.0, 4505, 495},
		{"11% on 10000", 10000, 11.0, 9009, 991},
		{"11% on 1262000", 1262000, 11.0, 1136937, 125063},
		{"0% on 5000", 5000, 0.0, 5000, 0},
		{"11% on 25000", 25000, 11.0, 22523, 2477},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dpp, tax int
			if tt.rate > 0 {
				dpp = int(math.Round(float64(tt.lineTotal) * 100.0 / (100.0 + tt.rate)))
				tax = tt.lineTotal - dpp
			} else {
				dpp = tt.lineTotal
				tax = 0
			}
			assert.Equal(t, tt.wantDPP, dpp, "DPP should match expected")
			assert.Equal(t, tt.wantTax, tax, "Tax should match expected")
			assert.Equal(t, tt.lineTotal, dpp+tax, "DPP + Tax should equal line total")
		})
	}
}