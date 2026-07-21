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

func TestSaleService_CreateSaleDeductsStock(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	initialStock := 100
	quantity := 7
	prodID := insertTestProduct(t, ctx, "SVC-STOCK-DED", "Stock Deduction Product", 5000, initialStock)

	var stockBefore int
	err := dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&stockBefore)
	require.NoError(t, err)
	assert.Equal(t, initialStock, stockBefore)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-STOCK-DED-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      5000 * quantity,
		TotalAmount:   5000 * quantity,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  quantity,
		UnitPrice: 5000,
		Subtotal:  5000 * quantity,
		DPPAmount: 5000 * quantity,
		TaxAmount: 0,
	}}

	err = svc.CreateSale(ctx, sale, items)
	require.NoError(t, err)

	var stockAfter int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&stockAfter)
	require.NoError(t, err)
	assert.Equal(t, initialStock-quantity, stockAfter, "stock should be reduced by sale quantity")
}

func TestSaleService_CreateSaleWithDiscount(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-DISC-PROD", "Discount Product", 10000, 50)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-DISC-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      30000,
		Discount:      5000,
		TotalAmount:   25000,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  3,
		UnitPrice: 10000,
		Subtotal:  30000,
		DPPAmount: 30000,
		TaxAmount: 0,
	}}

	err := svc.CreateSale(ctx, sale, items)
	require.NoError(t, err)
	require.Greater(t, sale.ID, 0)

	got, err := svc.GetSaleByID(ctx, sale.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 30000, got.Subtotal)
	assert.Equal(t, 5000, got.Discount)
	assert.Equal(t, 25000, got.TotalAmount)
	assert.Len(t, got.Items, 1)
	assert.Equal(t, 3, got.Items[0].Quantity)
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
		got, err := svc.GetSaleByID(ctx, sale.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, sale.InvoiceNumber, got.InvoiceNumber)
		assert.Equal(t, "QRIS", got.PaymentMethod)
	})

	t.Run("ListSales", func(t *testing.T) {
		sales, total, err := svc.ListSales(ctx, 10, 0, inv, "", "", "", "", "", nil, nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(sales), 1)
	})

	t.Run("GetSalesForExport", func(t *testing.T) {
		rows, err := svc.GetSalesForExport(ctx, "", "", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, rows)
	})

	t.Run("GetNextInvoiceNumber", func(t *testing.T) {
		next, err := svc.GetNextInvoiceNumber(ctx)
		require.NoError(t, err)
		assert.Contains(t, next, "INV-")
	})
}

type mockPriceStore struct {
	prices map[int]int
}

func (m *mockPriceStore) GetProductPrice(_ context.Context, productID int) (int, error) {
	if p, ok := m.prices[productID]; ok {
		return p, nil
	}
	return 0, assert.AnError
}

func (m *mockPriceStore) GetProductPrices(_ context.Context, productIDs []int) (map[int]int, error) {
	result := make(map[int]int, len(productIDs))
	for _, pid := range productIDs {
		if p, ok := m.prices[pid]; ok {
			result[pid] = p
		} else {
			return nil, assert.AnError
		}
	}
	return result, nil
}

func TestSaleService_CreateSalePriceValidation(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-PRICE-VAL", "Price Validation Product", 10000, 100)

	t.Run("price mismatch logs warning and uses server price", func(t *testing.T) {
		svc.SetPriceStore(&mockPriceStore{prices: map[int]int{prodID: 15000}})

		sale := &Sale{
			InvoiceNumber: "INV-SVC-PRICE-001",
			CashierID:     insertTestCashier(t, ctx),
			Subtotal:      15000,
			TotalAmount:   15000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  1,
			UnitPrice: 10000,
			Subtotal:  10000,
			DPPAmount: 10000,
			TaxAmount: 0,
		}}

		err := svc.CreateSale(ctx, sale, items)
		require.NoError(t, err)
		assert.Equal(t, 15000, sale.Subtotal)
		assert.Equal(t, 15000, items[0].UnitPrice)
	})

	t.Run("price match succeeds", func(t *testing.T) {
		svc.SetPriceStore(&mockPriceStore{prices: map[int]int{prodID: 10000}})

		sale := &Sale{
			InvoiceNumber: "INV-SVC-PRICE-002",
			CashierID:     insertTestCashier(t, ctx),
			Subtotal:      10000,
			TotalAmount:   10000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  1,
			UnitPrice: 10000,
			Subtotal:  10000,
			DPPAmount: 10000,
			TaxAmount: 0,
		}}

		err := svc.CreateSale(ctx, sale, items)
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
	})
}
