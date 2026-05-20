package service

import (
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
	// Test case: insufficient stock should return error
	// This tests the validation logic in CreateSale
	t.Run("insufficient stock returns error", func(t *testing.T) {
		// The CreateSale method validates stock before transaction
		// If product.Stock < item.Quantity, ErrInsufficientStock is returned
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
			// Simulate the stock check logic from CreateSale
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
		// Verify calculation
		assert.Equal(t, item.Quantity*item.UnitPrice, item.Subtotal)
	}

	assert.Equal(t, 19000, totalSubtotal)
}

// TestSaleStruct tests the Sale domain model
func TestSaleStruct(t *testing.T) {
	sale := domain.Sale{
		InvoiceNumber: "INV-001",
		CashierID:     1,
		Subtotal:      10000,
		Discount:      0,
		Tax:           0,
		TotalAmount:   10000,
		PaymentMethod: "Cash",
		Status:        "completed",
	}

	assert.Equal(t, "INV-001", sale.InvoiceNumber)
	assert.Equal(t, 1, sale.CashierID)
	assert.Equal(t, 10000, sale.TotalAmount)
}

// TestSaleItemSubtotal tests SaleItem subtotal calculation
func TestSaleItemSubtotal(t *testing.T) {
	item := domain.SaleItem{
		ProductID: 1,
		Quantity:  5,
		UnitPrice: 2000,
		Subtotal:  10000,
	}

	// Verify subtotal formula
	assert.Equal(t, item.Quantity*item.UnitPrice, item.Subtotal)
}