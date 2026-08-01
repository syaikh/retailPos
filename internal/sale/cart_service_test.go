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

// newCartTestService builds a service wired with a real pricing resolver so
// snapshots carry actual unit prices, cost, and tax rates from the test DB.
func newCartTestService(t *testing.T, ctx context.Context) (*Service, *eventbus.Bus) {
	t.Helper()
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)

	svc := NewService(repo, bus)
	svc.SetCartConfig(CartConfig{HoldTTLHours: 24})
	svc.SetPriceResolver(pricing.NewResolver(pricing.NewRepository(dbPool)))
	return svc, bus
}

func insertTestProductWithTax(t *testing.T, ctx context.Context, sku, name string, price, stock int, taxRate float64) int {
	t.Helper()
	var taxClassID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO tax_classes (name, rate_percent) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET rate_percent = EXCLUDED.rate_percent
		RETURNING id
	`, fmt.Sprintf("tax_%g", taxRate), taxRate).Scan(&taxClassID)
	require.NoError(t, err)

	var id int
	err = dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, tax_class_id, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		RETURNING id
	`, sku, name, price, price/2, taxClassID).Scan(&id)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)
	`, id, stock)
	require.NoError(t, err)
	return id
}

func TestCartService_IT01_PriceChangeDuringHold(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT01-PROD", "IT01 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	oldItem := cart.Items[0]
	assert.Equal(t, 3500, oldItem.UnitPrice, "snapshot price at add time")

	cart, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, "held", cart.Status)

	_, err = dbPool.Exec(ctx, `UPDATE products SET price = 3000 WHERE id = $1`, prodID)
	require.NoError(t, err)

	cart, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, "open", cart.Status)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, 3500, cart.Items[0].UnitPrice, "held snapshot must survive master data change")
	assert.Equal(t, oldItem.SnapshotCreatedAt, cart.Items[0].SnapshotCreatedAt, "snapshot_created_at must not change")
}

func TestCartService_IT02_NewItemAfterResumeUsesLatestPrice(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT02-PROD", "IT02 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `UPDATE products SET price = 3000 WHERE id = $1`, prodID)
	require.NoError(t, err)

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 2)

	assert.Equal(t, 3500, cart.Items[0].UnitPrice, "old snapshot")
	assert.Equal(t, 3000, cart.Items[1].UnitPrice, "new snapshot uses latest price")
	t1, err := time.Parse(time.RFC3339, cart.Items[0].SnapshotCreatedAt)
	require.NoError(t, err)
	t2, err := time.Parse(time.RFC3339, cart.Items[1].SnapshotCreatedAt)
	require.NoError(t, err)
	assert.False(t, t2.Before(t1), "second snapshot must not be earlier than the first")
	assert.Equal(t, 6500, cart.Subtotal, "subtotal = 3500 + 3000")
}

func TestCartService_IT03_QuantityUpdateKeepsPrice(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT03-PROD", "IT03 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	snapshotCreatedAt := cart.Items[0].SnapshotCreatedAt
	cart, err = svc.UpdateCartItemQuantity(ctx, cart.ID, cart.Items[0].ID, 3, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, 3500, cart.Items[0].UnitPrice, "unit price unchanged")
	assert.Equal(t, 10500, cart.Items[0].Subtotal, "subtotal scales with qty")
	assert.Equal(t, snapshotCreatedAt, cart.Items[0].SnapshotCreatedAt, "snapshot_created_at unchanged")
}

func TestCartService_IT04_VoidThenRescanCreatesNewSnapshot(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT04-PROD", "IT04 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	itemID := cart.Items[0].ID

	cart, err = svc.RemoveCartItem(ctx, cart.ID, itemID, cashierID)
	require.NoError(t, err)
	assert.Empty(t, cart.Items, "item removed")

	_, err = dbPool.Exec(ctx, `UPDATE products SET price = 3000 WHERE id = $1`, prodID)
	require.NoError(t, err)

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, 3000, cart.Items[0].UnitPrice, "rescan uses latest price")
}

func TestCartService_IT05_PromoAfterItemKeepsOldSnapshot(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT05-PROD", "IT05 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	assert.Equal(t, 3500, cart.Items[0].UnitPrice, "no promo yet")

	_, err = dbPool.Exec(ctx, `
		INSERT INTO pricing_rules (product_id, pricing_type, pricing_method, pricing_value, name, priority, is_active, status)
		VALUES ($1, 'promotion', 'discount_percent', 10, 'IT05 Promo', 5, true, 'approved')
	`, prodID)
	require.NoError(t, err)

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 2)
	assert.Equal(t, 3500, cart.Items[0].UnitPrice, "first item keeps old snapshot")
	assert.Equal(t, 3150, cart.Items[1].UnitPrice, "second item uses promo price 3500*0.9")
}

func TestCartService_IT06_PromoBeforeScanIsUsed(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT06-PROD", "IT06 Product", 3500, 100, 11)

	_, err := dbPool.Exec(ctx, `
		INSERT INTO pricing_rules (product_id, pricing_type, pricing_method, pricing_value, name, priority, is_active, status)
		VALUES ($1, 'promotion', 'discount_percent', 10, 'IT06 Promo', 5, true, 'approved')
	`, prodID)
	require.NoError(t, err)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	assert.Equal(t, 3150, cart.Items[0].UnitPrice, "promo applies when active before scan")

	_, err = dbPool.Exec(ctx, `UPDATE pricing_rules SET is_active = false WHERE name = 'IT06 Promo'`)
	require.NoError(t, err)

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 2)
	assert.Equal(t, 3150, cart.Items[0].UnitPrice)
	assert.Equal(t, 3500, cart.Items[1].UnitPrice, "promo disabled before second scan")
}

func TestCartService_IT07_MultiplePriceChanges(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT07-PROD", "IT07 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	for _, price := range []int{3000, 4000, 3200} {
		_, err = dbPool.Exec(ctx, `UPDATE products SET price = $1 WHERE id = $2`, price, prodID)
		require.NoError(t, err)
	}

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 2)
	assert.Equal(t, 3500, cart.Items[0].UnitPrice, "first scan price")
	assert.Equal(t, 3200, cart.Items[1].UnitPrice, "last price at second scan")
}

func TestCartService_IT08_CheckoutDeductsStockAndPublishes(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, bus := newCartTestService(t, ctx)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.SaleCreated},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT08-PROD", "IT08 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 2, nil, cashierID)
	require.NoError(t, err)
	total := cart.TotalAmount

	sale, err := svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: total}}, cashierID)
	require.NoError(t, err)
	assert.Equal(t, total, sale.TotalAmount)
	assert.Equal(t, "completed", sale.Status)
	assert.Len(t, sale.Items, 1)
	assert.Equal(t, 3500, sale.Items[0].UnitPrice, "sale item uses snapshot")

	var stockQty int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1`, prodID).Scan(&stockQty)
	require.NoError(t, err)
	assert.Equal(t, 98, stockQty, "stock reduced by 2")

	var cartStatus string
	err = dbPool.QueryRow(ctx, `SELECT status FROM cart_sessions WHERE id = $1`, cart.ID).Scan(&cartStatus)
	require.NoError(t, err)
	assert.Equal(t, "checked_out", cartStatus)

	select {
	case <-published:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for sale.created event")
	}
}

func TestCartService_CheckoutCartWithPaymentMethod_DerivesAmountFromTotal(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-PM-PROD", "Legacy PM Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 2, nil, cashierID)
	require.NoError(t, err)
	total := cart.TotalAmount

	sale, err := svc.CheckoutCartWithPaymentMethod(ctx, cart.ID, "CASH", cashierID)
	require.NoError(t, err)
	assert.Equal(t, total, sale.TotalAmount, "amount derived from recomputed sale total")
	assert.Equal(t, "CASH", sale.PaymentMethod)
	assert.Len(t, sale.Payments, 1)
	assert.Equal(t, total, sale.Payments[0].Amount)
}

func TestCartService_CheckoutCartWithPaymentMethod_EmptyCartRejected(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.CheckoutCartWithPaymentMethod(ctx, cart.ID, "CASH", cashierID)
	assert.ErrorIs(t, err, ErrCartEmpty)
}

func TestCartService_IT09_CheckoutTwiceRejected(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT09-PROD", "IT09 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	_, err = svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: cart.TotalAmount}}, cashierID)
	require.NoError(t, err)

	_, err = svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: cart.TotalAmount}}, cashierID)
	assert.ErrorIs(t, err, ErrCartAlreadyCheckedOut)

	_, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	assert.ErrorIs(t, err, ErrCartNotOpen)
}

func TestCartService_IT10_ExpiredCartRejected(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-IT10-PROD", "IT10 Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	cart, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `UPDATE cart_sessions SET expired_at = NOW() - interval '1 hour' WHERE id = $1`, cart.ID)
	require.NoError(t, err)

	_, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartExpired)

	_, err = svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: cart.TotalAmount}}, cashierID)
	assert.ErrorIs(t, err, ErrCartExpired)
}

func TestCartService_OpenCartUniquePerCashier(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	cart1, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "open", cart1.Status)

	cart2, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, cart1.ID, cart2.ID, "same open cart returned for cashier")

	cart, err := svc.GetOpenCart(ctx, cashierID)
	require.NoError(t, err)
	assert.Equal(t, cart1.ID, cart.ID)
}

func TestCartService_CheckoutPaymentMismatch(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	prodID := insertTestProductWithTax(t, ctx, "CART-PAY-PROD", "Payment Mismatch", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	_, err = svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 1}}, cashierID)
	assert.ErrorIs(t, err, ErrPaymentTotalMismatch)
}

func TestCartService_ResumeHeldCartIdempotentForOpen(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartNotOpen, "open cart is not in held state")
}

func TestCartService_UpdateCartCustomer(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(t, ctx)

	cashierID := insertTestCashier(t, ctx)
	var customerID int
	err := dbPool.QueryRow(ctx, `INSERT INTO customers (name, email, phone) VALUES ('Cart Customer', 'cart@test.com', '08123') RETURNING id`).Scan(&customerID)
	require.NoError(t, err)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	cart, err = svc.UpdateCartCustomer(ctx, cart.ID, &customerID, cashierID)
	require.NoError(t, err)
	require.NotNil(t, cart.CustomerID)
	assert.Equal(t, customerID, *cart.CustomerID)
}
