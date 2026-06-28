package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/sale"
)

func TestStockDeductListener(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	ctx := context.Background()

	productID := insertTestProduct(t, ctx, "LSTN-001")
	insertTestStock(t, ctx, productID, 100)

	bus.Subscribe(NewStockDeductListener(repo))

	saleEvent := &sale.Sale{
		ID:            999,
		InvoiceNumber: "INV-TEST-LSTN",
		CashierID:     1,
		Subtotal:      50000,
		TotalAmount:   50000,
		PaymentMethod: "cash",
		Status:        "completed",
		Items: []sale.SaleItem{
			{ProductID: productID, Quantity: 7, UnitPrice: 5000, Subtotal: 35000},
		},
	}

	err := bus.Publish(ctx, "sale.created", saleEvent)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	stock, err := repo.GetStockByProductID(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, 93, stock.Quantity, "stock should be reduced by sale quantity")
}
