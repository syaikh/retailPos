package sale

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func TestSaleService_CreateSalePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.SaleCreated},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	prodID := insertTestProduct(t, ctx, "SVC-EVT-PROD", "Service Event Product", 5000, 100)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-EVT-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      5000,
		Tax:           0,
		TotalAmount:   5000,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  1,
		UnitPrice: 5000,
		Subtotal:  5000,
		DPPAmount: 5000,
		TaxAmount: 0,
	}}

	err := svc.CreateSale(ctx, sale, items)
	require.NoError(t, err)

	select {
	case <-published:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for sale.created event")
	}
}

func TestSaleService_CreateSaleInsufficientStock(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-LOW-STOCK", "Low Stock Product", 10000, 2)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-LOW-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      100000,
		TotalAmount:   100000,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  10,
		UnitPrice: 10000,
		Subtotal:  100000,
		DPPAmount: 100000,
		TaxAmount: 0,
	}}

	err := svc.CreateSale(ctx, sale, items)
	assert.ErrorIs(t, err, ErrInsufficientStock)
}

func TestSaleService_CreateSaleDuplicateInvoice(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-DUP-PROD", "Duplicate Svc Prod", 10000, 20)

	sale1 := &Sale{
		InvoiceNumber: "INV-SVC-DUP-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      10000,
		TotalAmount:   10000,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items1 := []SaleItem{{
		ProductID: prodID,
		Quantity:  1,
		UnitPrice: 10000,
		Subtotal:  10000,
		DPPAmount: 10000,
		TaxAmount: 0,
	}}
	err := svc.CreateSale(ctx, sale1, items1)
	require.NoError(t, err)

	prodID2 := insertTestProduct(t, ctx, "SVC-DUP-PROD2", "Duplicate Svc Prod 2", 10000, 20)

	sale2 := &Sale{
		InvoiceNumber: "INV-SVC-DUP-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      20000,
		TotalAmount:   20000,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items2 := []SaleItem{{
		ProductID: prodID2,
		Quantity:  2,
		UnitPrice: 10000,
		Subtotal:  20000,
		DPPAmount: 20000,
		TaxAmount: 0,
	}}

	err = svc.CreateSale(ctx, sale2, items2)
	assert.Error(t, err)
}

func TestSaleService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-READ-PROD", "Service Read Product", 15000, 50)
	inv := "INV-SVC-READ-001"
	sale := &Sale{
		InvoiceNumber: inv,
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      15000,
		TotalAmount:   15000,
		PaymentMethod: "QRIS",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  1,
		UnitPrice: 15000,
		Subtotal:  15000,
		DPPAmount: 15000,
		TaxAmount: 0,
	}}
	err := svc.CreateSale(ctx, sale, items)
	require.NoError(t, err)

	t.Run("GetSaleByID", func(t *testing.T) {
		got, err := svc.GetSaleByID(ctx, sale.ID)
		require.NoError(t, err)
		assert.Equal(t, sale.InvoiceNumber, got.InvoiceNumber)
		assert.Equal(t, "QRIS", got.PaymentMethod)
	})

	t.Run("ListSales", func(t *testing.T) {
		sales, total, err := svc.ListSales(ctx, 10, 0, inv, "", "", "", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(sales), 1)
	})

	t.Run("GetSalesForExport", func(t *testing.T) {
		rows, err := svc.GetSalesForExport(ctx, "", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, rows)
	})

	t.Run("GetNextInvoiceNumber", func(t *testing.T) {
		next, err := svc.GetNextInvoiceNumber(ctx)
		require.NoError(t, err)
		assert.Contains(t, next, "INV-")
	})
}
