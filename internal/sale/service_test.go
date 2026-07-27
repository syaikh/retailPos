package sale

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/shared"
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

	payments := []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 5000}}

	err := svc.CreateSale(ctx, sale, items, payments)
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

	payments := []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 100000}}

	err := svc.CreateSale(ctx, sale, items, payments)
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
	err := svc.CreateSale(ctx, sale1, items1, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 10000}})
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

	err = svc.CreateSale(ctx, sale2, items2, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 20000}})
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

	err = svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 5000 * quantity}})
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

	err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 25000}})
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
	err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "QRIS", Amount: 15000}})
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

		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 15000}})
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

		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 10000}})
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
	})
}

func TestSaleService_ParkSale(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	prodID := insertTestProduct(t, ctx, "SVC-PARK-001", "Park Service Product", 10000, 50)
	cashierID := insertTestCashier(t, ctx)

	t.Run("success", func(t *testing.T) {
		sale := &Sale{
			InvoiceNumber: "INV-SVC-PARK-001",
			CashierID:     cashierID,
			Subtotal:      20000,
			TotalAmount:   20000,
			PaymentMethod: "CASH",
			Status:        "parked",
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  2,
			UnitPrice: 10000,
			Subtotal:  20000,
			DPPAmount: 20000,
			TaxAmount: 0,
		}}

		err := svc.ParkSale(ctx, sale, items, nil)
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
		assert.Equal(t, "parked", sale.Status)
		assert.NotEmpty(t, sale.InvoiceNumber)

		var status string
		err = dbPool.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1`, sale.ID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "parked", status)

		var stockAfter int
		err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&stockAfter)
		require.NoError(t, err)
		assert.Equal(t, 50, stockAfter, "stock should NOT be deducted when parking a sale")
	})

	t.Run("invalid quantity", func(t *testing.T) {
		sale := &Sale{
			InvoiceNumber: "INV-SVC-PARK-INV",
			CashierID:     cashierID,
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  0,
			UnitPrice: 10000,
			Subtotal:  10000,
		}}

		err := svc.ParkSale(ctx, sale, items, nil)
		assert.ErrorContains(t, err, "invalid quantity")
	})

	t.Run("with recalled sale ID", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-PARK-RECALL", "parked", prodID, 1, 10000)
		recalled, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)

		sale := &Sale{
			InvoiceNumber: "INV-SVC-PARK-RECALLED",
			CashierID:     cashierID,
			Subtotal:      10000,
			TotalAmount:   10000,
			PaymentMethod: "CASH",
			Status:        "parked",
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  1,
			UnitPrice: 10000,
			Subtotal:  10000,
			DPPAmount: 10000,
			TaxAmount: 0,
		}}

		err = svc.ParkSale(ctx, sale, items, &recalled.ID)
		require.NoError(t, err)
		assert.Equal(t, "parked", sale.Status)

		var cancelledStatus string
		err = dbPool.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1`, recalled.ID).Scan(&cancelledStatus)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", cancelledStatus, "previous recalled sale should be cancelled")
	})
}

func TestSaleService_RecallSale(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	prodID := insertTestProduct(t, ctx, "SVC-RECALL-001", "Recall Service Product", 10000, 50)
	cashierID := insertTestCashier(t, ctx)

	t.Run("recall parked sale", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-RECALL-001", "parked", prodID, 2, 10000)

		sale, err := svc.RecallSale(ctx, parked.ID)
		require.NoError(t, err)
		assert.Equal(t, "recalled", sale.Status)
		assert.NotEmpty(t, sale.Items)
	})

	t.Run("recall non-existent returns error", func(t *testing.T) {
		_, err := svc.RecallSale(ctx, -999)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})

	t.Run("recall completed returns error", func(t *testing.T) {
		completed := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-RECALL-CMP", "completed", prodID, 1, 10000)
		_, err := svc.RecallSale(ctx, completed.ID)
		assert.ErrorIs(t, err, ErrSaleNotFound)
	})
}

func TestSaleService_CancelParkedSale(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	prodID := insertTestProduct(t, ctx, "SVC-CANCEL-001", "Cancel Service Product", 10000, 50)
	cashierID := insertTestCashier(t, ctx)

	t.Run("cancel parked", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-CANCEL-001", "parked", prodID, 1, 10000)
		err := svc.CancelParkedSale(ctx, parked.ID)
		assert.NoError(t, err)
	})

	t.Run("cancel non-existent returns error", func(t *testing.T) {
		err := svc.CancelParkedSale(ctx, -999)
		assert.Error(t, err)
	})
}

func TestSaleService_ValidatePayments(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	prodID := insertTestProduct(t, ctx, "SVC-SPLIT-PROD", "Split Payment Product", 50000, 100)
	cashierID := insertTestCashier(t, ctx)

	makeSale := func(inv string, total int) (*Sale, []SaleItem) {
		return &Sale{
			InvoiceNumber: inv,
			CashierID:     cashierID,
			Subtotal:      total,
			TotalAmount:   total,
			PaymentMethod: "",
			Status:        "completed",
		}, []SaleItem{{
			ProductID: prodID,
			Quantity:  1,
			UnitPrice: total,
			Subtotal:  total,
			DPPAmount: total,
			TaxAmount: 0,
		}}
	}

	t.Run("success single cash payment", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-SGL-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 50000}})
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
		assert.Equal(t, "CASH", sale.PaymentMethod)
	})

	t.Run("success split payments CASH+QRIS", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-CQ-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{
			{PaymentMethodCode: "CASH", Amount: 30000},
			{PaymentMethodCode: "QRIS", Amount: 20000},
		})
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
		assert.Contains(t, sale.PaymentMethod, "CASH")
		assert.Contains(t, sale.PaymentMethod, "QRIS")

		got, err := svc.GetSaleByID(ctx, sale.ID, nil)
		require.NoError(t, err)
		assert.Len(t, got.Payments, 2)
	})

	t.Run("empty payments returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-EMP-001", 50000)
		err := svc.CreateSale(ctx, sale, items, nil)
		assert.ErrorIs(t, err, ErrZeroPaymentAmount)
	})

	t.Run("exceeds max payments", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-MAX-001", 100000)
		payments := make([]CreatePaymentRequest, 0, MaxPaymentsPerSale+1)
		for i := 0; i <= MaxPaymentsPerSale; i++ {
			payments = append(payments, CreatePaymentRequest{PaymentMethodCode: "CASH", Amount: 10000})
		}
		err := svc.CreateSale(ctx, sale, items, payments)
		assert.ErrorIs(t, err, ErrMaxPaymentsExceeded)
	})

	t.Run("zero amount payment returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-ZAM-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 0}})
		assert.ErrorIs(t, err, ErrZeroPaymentAmount)
	})

	t.Run("payment total mismatch returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-MIS-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 30000}})
		assert.ErrorIs(t, err, ErrPaymentTotalMismatch)
	})

	t.Run("duplicate non-cash method returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-DUP-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{
			{PaymentMethodCode: "QRIS", Amount: 25000},
			{PaymentMethodCode: "QRIS", Amount: 25000},
		})
		assert.ErrorIs(t, err, ErrDuplicatePaymentMethod)
	})

	t.Run("inactive payment method returns error", func(t *testing.T) {
		inactiveCode := fmt.Sprintf("INA-%d", time.Now().UnixNano())
		var inactiveID int
		err := dbPool.QueryRow(ctx, `INSERT INTO payment_methods (code, name, is_active, requires_reference) VALUES ($1, $2, false, false) RETURNING id`, inactiveCode, "Inactive Method").Scan(&inactiveID)
		require.NoError(t, err)

		sale, items := makeSale("INV-SPLIT-INA-001", 50000)
		err = svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: inactiveCode, Amount: 50000}})
		assert.ErrorIs(t, err, ErrPaymentMethodInactive)
	})

	t.Run("invalid payment method code returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-INV-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "NONEXISTENT", Amount: 50000}})
		assert.ErrorIs(t, err, ErrInvalidPaymentMethod)
	})

	t.Run("multiple cash payments returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-MC-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{
			{PaymentMethodCode: "CASH", Amount: 25000},
			{PaymentMethodCode: "CASH", Amount: 25000},
		})
		assert.ErrorIs(t, err, ErrMultipleCashPayments)
	})

	t.Run("reference required but missing returns error", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-REF-001", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CARD", Amount: 50000}})
		assert.ErrorIs(t, err, ErrPaymentReferenceRequired)
	})

	t.Run("reference provided for method that requires it", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-REF-002", 50000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CARD", Amount: 50000, ReferenceNumber: "REF123"}})
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
	})

	t.Run("cash plus card with reference", func(t *testing.T) {
		sale, items := makeSale("INV-SPLIT-CC-001", 100000)
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{
			{PaymentMethodCode: "CASH", Amount: 60000},
			{PaymentMethodCode: "CARD", Amount: 40000, ReferenceNumber: "TXN-ABC-123"},
		})
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)
		assert.Contains(t, sale.PaymentMethod, "CASH")
		assert.Contains(t, sale.PaymentMethod, "CARD")
	})
}

func TestSaleService_ListParkedSales(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	prodID := insertTestProduct(t, ctx, "SVC-LIST-PARK-001", "List Park Product", 10000, 50)
	cashierID := insertTestCashier(t, ctx)

	_ = createParkedSale(t, ctx, repo, cashierID, "INV-SVC-LP-001", "parked", prodID, 1, 10000)
	_ = createParkedSale(t, ctx, repo, cashierID, "INV-SVC-LP-002", "recalled", prodID, 2, 10000)
	_ = createParkedSale(t, ctx, repo, cashierID, "INV-SVC-LP-003", "completed", prodID, 3, 10000)

	sales, err := svc.ListParkedSales(ctx, cashierID)
	require.NoError(t, err)
	assert.Len(t, sales, 2)
	for _, s := range sales {
		assert.Contains(t, []string{"parked", "recalled"}, s.Status)
	}
}

func TestSaleService_CreateSaleWithParkedSaleID(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	prodID := insertTestProduct(t, ctx, "SVC-CHECKOUT-PARK", "Checkout Park Product", 10000, 100)
	cashierID := insertTestCashier(t, ctx)

	t.Run("checkout from recalled sale", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-COP-001", "parked", prodID, 2, 10000)
		_, err := repo.RecallSale(ctx, parked.ID)
		require.NoError(t, err)

		sale := &Sale{
			InvoiceNumber: "INV-SVC-COP-001-NEW",
			CashierID:     cashierID,
			Subtotal:      20000,
			TotalAmount:   20000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []SaleItem{{
			ProductID: prodID,
			Quantity:  2,
			UnitPrice: 10000,
			Subtotal:  20000,
			DPPAmount: 20000,
			TaxAmount: 0,
		}}

		err = svc.CreateSaleWithParkedSale(ctx, sale, items, &parked.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 20000}})
		require.NoError(t, err)
		assert.Greater(t, sale.ID, 0)

		var status string
		err = dbPool.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1`, parked.ID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", status)
	})

	t.Run("checkout with non-recalled parked sale fails", func(t *testing.T) {
		parked := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-COP-002", "parked", prodID, 1, 10000)

		sale := &Sale{
			InvoiceNumber: "INV-SVC-COP-002-NEW",
			CashierID:     cashierID,
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

		err := svc.CreateSaleWithParkedSale(ctx, sale, items, &parked.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 10000}})
		assert.ErrorIs(t, err, ErrParkedSaleNotRecalled)
	})
}

func TestSaleService_SetPriceResolver(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	priceRes := &mockPriceResolver{}
	svc.SetPriceResolver(priceRes)
	assert.NotNil(t, svc.resolver)
	assert.Equal(t, priceRes, svc.resolver)
}

type mockPriceResolver struct{}

func (m *mockPriceResolver) Resolve(ctx context.Context, rc pricing.ResolveContext) (*pricing.ResolvedPrice, error) {
	return &pricing.ResolvedPrice{UnitPrice: 10000, OriginalPrice: 10000, Discount: 0, PricingType: pricing.PricingTypeDefault, PricingMethod: pricing.PricingMethodFixedPrice}, nil
}

func (m *mockPriceResolver) ResolveBatch(ctx context.Context, items []pricing.ResolveItem) ([]pricing.ResolvedPrice, error) {
	result := make([]pricing.ResolvedPrice, len(items))
	for i := range items {
		result[i] = pricing.ResolvedPrice{UnitPrice: 10000, OriginalPrice: 10000, Discount: 0, PricingType: pricing.PricingTypeDefault, PricingMethod: pricing.PricingMethodFixedPrice}
	}
	return result, nil
}

type mockSimplePriceStore struct {
	prices map[int]int
}

func (m *mockSimplePriceStore) GetProductPrice(_ context.Context, productID int) (int, error) {
	if p, ok := m.prices[productID]; ok {
		return p, nil
	}
	return 0, assert.AnError
}

func TestSaleService_CreateSaleWithPriceResolver(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	svc.SetPriceResolver(&mockPriceResolver{})
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-RESOLV-PROD", "Resolver Product", 5000, 100)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-RESOLV-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      0,
		Tax:           0,
		TotalAmount:   0,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  2,
		UnitPrice: 9999,
		Subtotal:  19998,
		DPPAmount: 19998,
		TaxAmount: 0,
	}}

	err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 20000}})
	require.NoError(t, err)
	assert.Equal(t, 20000, sale.TotalAmount, "resolver overrides prices to 10000 each x2 items")
	assert.Equal(t, 10000, items[0].UnitPrice, "resolver should override unit price")
}

func TestSaleService_CreateSaleWithNonBatchPriceStore(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-NOBATCH-PROD", "No Batch Product", 7500, 100)
	svc.SetPriceStore(&mockSimplePriceStore{prices: map[int]int{prodID: 7500}})

	sale := &Sale{
		InvoiceNumber: "INV-SVC-NOBATCH-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      7500,
		TotalAmount:   7500,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Quantity:  1,
		UnitPrice: 7000,
		Subtotal:  7000,
		DPPAmount: 7000,
		TaxAmount: 0,
	}}

	err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 7500}})
	require.NoError(t, err)
	assert.Equal(t, 7500, sale.Subtotal, "non-batch price store should set server price")
	assert.Equal(t, 7500, items[0].UnitPrice, "server price should override client price")
}

func TestSaleService_CreateSaleStockRecordNotFound(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	var prodID int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, cost, status) VALUES ($1, $2, 5000, 2500, 'active') RETURNING id`,
		"SVC-NOSTOCK-PROD", "No Stock Record",
	).Scan(&prodID)
	require.NoError(t, err)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-NOSTOCK-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      5000,
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

	err = svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 5000}})
	assert.ErrorContains(t, err, "stock record not found")
}

func TestSaleService_CreateSaleTotalAmountClamp(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "SVC-CLAMP-PROD", "Clamp Product", 10000, 100)

	sale := &Sale{
		InvoiceNumber: "INV-SVC-CLAMP-001",
		CashierID:     insertTestCashier(t, ctx),
		Subtotal:      10000,
		Discount:      15000,
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

	err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 5000}})
	assert.ErrorIs(t, err, ErrPaymentTotalMismatch, "clamping should set TotalAmount=0, payment 5000 mismatches")
}

func TestSaleService_GetAllPaymentMethods(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	ctx := context.Background()

	methods, err := svc.GetAllPaymentMethods(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, methods)
}

func TestSaleService_GetParkedSaleByID(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProduct(t, ctx, "SVC-GETPID-001", "Get Parked ByID", 10000, 50)
	parked := createParkedSale(t, ctx, repo, cashierID, "INV-SVC-GETPID-001", "parked", prodID, 2, 10000)

	sale, err := svc.GetParkedSaleByID(ctx, parked.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, parked.ID, sale.ID)
	assert.Equal(t, "parked", sale.Status)
	assert.NotEmpty(t, sale.Items)
	assert.Len(t, sale.Items, 1)
}

func TestSaleService_GetParkedSaleByID_NotFound(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	ctx := context.Background()

	_, err := svc.GetParkedSaleByID(ctx, -999, 0)
	assert.ErrorIs(t, err, ErrSaleNotFound)
}
