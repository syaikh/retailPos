package sale

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
)

// recordingConsignmentCheckout is a scripted ConsignmentCheckout test double. It
// treats a fixed set of products as consignment-owned (returning a checkout
// record for them) and records every ResolveAndDeductConsignment /
// RecordConsignmentSaleItems call so tests can assert that (a) consignment lines
// are ALSO deducted from product_stock (Model A: product_stock is the sellable
// total, consignment_stock only tracks ownership), (b) records carry the sale
// id/invoice, and (c) the whole checkout rolls back when resolution fails.
type recordingConsignmentCheckout struct {
	consignmentProducts map[int]bool
	resolveErr          error

	resolved       []shared.ConsignmentCheckoutItem
	recordedSaleID int
	recorded       []shared.ConsignmentSaleRecord
}

func (c *recordingConsignmentCheckout) ResolveAndDeductConsignment(ctx context.Context, tx pgx.Tx, items []shared.ConsignmentCheckoutItem) ([]shared.ConsignmentSaleRecord, error) {
	c.resolved = append(c.resolved, items...)
	if c.resolveErr != nil {
		return nil, c.resolveErr
	}
	records := make([]shared.ConsignmentSaleRecord, 0, len(items))
	for _, it := range items {
		if c.consignmentProducts[it.ProductID] {
			records = append(records, shared.ConsignmentSaleRecord{
				ProductID:       it.ProductID,
				SupplierID:      99,
				ArrangementID:   7,
				StoreID:         1,
				Quantity:        it.Quantity,
				UnitPrice:       it.UnitPrice,
				Subtotal:        it.UnitPrice * it.Quantity,
				StoreShareType:  "percentage",
				StoreShareValue: 20,
			})
		}
	}
	return records, nil
}

func (c *recordingConsignmentCheckout) RecordConsignmentSaleItems(ctx context.Context, tx pgx.Tx, saleID int, records []shared.ConsignmentSaleRecord) error {
	c.recordedSaleID = saleID
	c.recorded = records
	return nil
}

// newConsignmentSaleService builds a sale service wired with a real stock
// deducer and a scripted consignment checkout.
func newConsignmentSaleService(t *testing.T, cc *recordingConsignmentCheckout) (Service, *eventbus.Bus) {
	t.Helper()
	repo := newTestRepo(t)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)

	svc := NewService(repo, bus)
	svc.SetStockDeducer(inventory.StockDeducer{})
	svc.SetConsignmentCheckout(cc)
	svc.SetShiftTotalUpdater(shift.TotalUpdater{})
	return svc, bus
}

func stockQty(ctx context.Context, t *testing.T, productID int) int {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`,
		productID).Scan(&qty)
	require.NoError(t, err)
	return qty
}

// TestSaleService_MixedCartConsignment covers the ownership-aware checkout
// (BR-19): a cart with store-owned + consignment-owned lines deducts BOTH from
// product_stock (Model A — product_stock is the sellable total, so the
// consignment line's receipt-added quantity is removed on sale), persists the
// consignment records with the sale id/invoice, and commits atomically.
func TestSaleService_MixedCartConsignment(t *testing.T) {
	ctx := context.Background()

	t.Run("consignment lines deduct product_stock and records carry sale id", func(t *testing.T) {
		storeProd := insertTestProduct(ctx, t, "MIX-STORE", "Store Owned", 10000, 50)
		consignProd := insertTestProduct(ctx, t, "MIX-CONSIGN", "Consignment", 10000, 50)

		cc := &recordingConsignmentCheckout{consignmentProducts: map[int]bool{consignProd: true}}
		svc, _ := newConsignmentSaleService(t, cc)

		sale := &Sale{
			InvoiceNumber: "INV-MIX-001",
			CashierID:     insertTestCashier(ctx, t),
			Subtotal:      20000,
			Tax:           0,
			TotalAmount:   20000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []Item{
			{ProductID: storeProd, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000},
			{ProductID: consignProd, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000},
		}
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 20000}})
		require.NoError(t, err)

		// Both lines deducted from product_stock (Model A: sellable total).
		require.Equal(t, 49, stockQty(ctx, t, storeProd))
		require.Equal(t, 49, stockQty(ctx, t, consignProd))

		// Resolver saw both lines; records captured and persisted with sale id+invoice.
		require.Len(t, cc.resolved, 2)
		require.Len(t, cc.recorded, 1)
		require.Equal(t, consignProd, cc.recorded[0].ProductID)
		require.Equal(t, sale.ID, cc.recordedSaleID)
		require.Equal(t, sale.ID, cc.recorded[0].SaleID)
		require.Equal(t, "INV-MIX-001", cc.recorded[0].InvoiceNumber)
		require.Greater(t, sale.ID, 0)
	})

	t.Run("all-consignment cart deducts from product_stock too", func(t *testing.T) {
		consignProd := insertTestProduct(ctx, t, "MIX-ALL-CONSIGN", "All Consignment", 8000, 20)

		cc := &recordingConsignmentCheckout{consignmentProducts: map[int]bool{consignProd: true}}
		svc, _ := newConsignmentSaleService(t, cc)

		sale := &Sale{
			InvoiceNumber: "INV-MIX-002",
			CashierID:     insertTestCashier(ctx, t),
			Subtotal:      8000,
			Tax:           0,
			TotalAmount:   8000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []Item{{ProductID: consignProd, Quantity: 1, UnitPrice: 8000, Subtotal: 8000, DPPAmount: 8000}}
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 8000}})
		require.NoError(t, err)

		require.Equal(t, 19, stockQty(ctx, t, consignProd))
		require.Len(t, cc.recorded, 1)
	})

	t.Run("resolver failure rolls back the whole sale", func(t *testing.T) {
		storeProd := insertTestProduct(ctx, t, "MIX-ROLLBACK", "Store Owned", 10000, 50)
		consignProd := insertTestProduct(ctx, t, "MIX-ROLLBACK-CONSIGN", "Consignment", 10000, 20)

		cc := &recordingConsignmentCheckout{
			consignmentProducts: map[int]bool{consignProd: true},
			resolveErr:          errors.New("insufficient consignment stock"),
		}
		svc, _ := newConsignmentSaleService(t, cc)

		sale := &Sale{
			InvoiceNumber: "INV-MIX-ROLLBACK",
			CashierID:     insertTestCashier(ctx, t),
			Subtotal:      20000,
			Tax:           0,
			TotalAmount:   20000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []Item{
			{ProductID: storeProd, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000},
			{ProductID: consignProd, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000},
		}
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 20000}})
		require.Error(t, err)

		// Neither product_stock row moved and no sale row exists.
		require.Equal(t, 50, stockQty(ctx, t, storeProd))
		require.Equal(t, 20, stockQty(ctx, t, consignProd))

		var n int
		err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM sales WHERE invoice_number = 'INV-MIX-ROLLBACK'`).Scan(&n)
		require.NoError(t, err)
		require.Zero(t, n)
	})

	t.Run("insufficient store stock still aborts", func(t *testing.T) {
		storeProd := insertTestProduct(ctx, t, "MIX-LOW", "Store Owned", 10000, 2)
		consignProd := insertTestProduct(ctx, t, "MIX-LOW-CONSIGN", "Consignment", 10000, 20)

		cc := &recordingConsignmentCheckout{consignmentProducts: map[int]bool{consignProd: true}}
		svc, _ := newConsignmentSaleService(t, cc)

		sale := &Sale{
			InvoiceNumber: "INV-MIX-LOW",
			CashierID:     insertTestCashier(ctx, t),
			Subtotal:      20000,
			Tax:           0,
			TotalAmount:   20000,
			PaymentMethod: "CASH",
			Status:        "completed",
		}
		items := []Item{
			{ProductID: storeProd, Quantity: 3, UnitPrice: 10000, Subtotal: 30000, DPPAmount: 30000},
			{ProductID: consignProd, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000},
		}
		err := svc.CreateSale(ctx, sale, items, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 40000}})
		require.Error(t, err)
		require.ErrorIs(t, err, shared.ErrInsufficientStock)

		require.Equal(t, 2, stockQty(ctx, t, storeProd))
		require.Equal(t, 20, stockQty(ctx, t, consignProd))
	})
}

// TestSaleService_ParkedSaleCompletesConsignment covers the parked-sale
// completion path: a consignment line recalled from a parked sale persists its
// record through CreateSaleWithParkedSale.
func TestSaleService_ParkedSaleCompletesConsignment(t *testing.T) {
	ctx := context.Background()
	consignProd := insertTestProduct(ctx, t, "PARK-CONSIGN", "Consignment", 10000, 10)

	cc := &recordingConsignmentCheckout{consignmentProducts: map[int]bool{consignProd: true}}
	svc, _ := newConsignmentSaleService(t, cc)

	repo := newTestRepo(t)
	cashierID := insertTestCashier(ctx, t)
	caller := Caller{UserID: cashierID}

	parked := createParkedSale(ctx, t, repo, cashierID, "INV-PARK-PARKED", "parked", consignProd, 1, 10000)
	_, err := repo.RecallSale(ctx, parked.ID, &cashierID, nil)
	require.NoError(t, err)

	completed := &Sale{
		InvoiceNumber: "INV-PARK-DONE",
		CashierID:     cashierID,
		Subtotal:      10000,
		Tax:           0,
		TotalAmount:   10000,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	completedItems := []Item{{ProductID: consignProd, Quantity: 1, UnitPrice: 10000, Subtotal: 10000, DPPAmount: 10000}}
	err = svc.CreateSaleWithParkedSale(ctx, completed, completedItems, &parked.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 10000}}, caller)
	require.NoError(t, err)

	require.Len(t, cc.recorded, 1)
	require.Equal(t, consignProd, cc.recorded[0].ProductID)
	require.Equal(t, "INV-PARK-DONE", cc.recorded[0].InvoiceNumber)
	assert.NotZero(t, cc.recordedSaleID)
}
