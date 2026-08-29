package sale

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
)

// newCartTestService builds a service wired with a real pricing resolver so
// snapshots carry actual unit prices, cost, and tax rates from the test DB.
func newCartTestService(ctx context.Context, t *testing.T) (Service, *eventbus.Bus) {
	t.Helper()
	repo := newTestRepo(t)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)

	svc := NewService(repo, bus)
	svc.SetStockDeducer(inventory.StockDeducer{})
	svc.SetConsignmentCheckout(noopConsignmentCheckout{})
	svc.SetShiftTotalUpdater(shift.TotalUpdater{})
	svc.SetCartConfig(CartConfig{HoldTTLHours: 24})
	svc.SetPriceResolver(newPricingTestResolver())
	return svc, bus
}

// pricingTestResolver adapts the real pricing subsystem to the consumer-side
// sale.PriceResolver port so cart tests exercise the wiring boundary.
type pricingTestResolver struct {
	resolver *pricing.Resolver
}

func newPricingTestResolver() *pricingTestResolver {
	repo := pricing.NewRepository(dbPool)
	repo.SetProductPricingProvider(product.PricingLookup{})
	repo.SetCategorySearchProvider(category.NamesProvider{})
	repo.SetBrandSearchProvider(brand.NamesProvider{})
	return &pricingTestResolver{resolver: pricing.NewResolver(repo)}
}

func (a *pricingTestResolver) ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error) {
	pricingItems := make([]pricing.ResolveItem, len(items))
	for i, it := range items {
		pricingItems[i] = pricing.ResolveItem{
			ProductID:       it.ProductID,
			Quantity:        it.Quantity,
			CustomerGroupID: it.CustomerGroupID,
			StoreID:         it.StoreID,
		}
	}
	snaps, err := a.resolver.ResolveSnapshotsBatch(ctx, pricingItems)
	if err != nil {
		return nil, err
	}
	result := make([]PriceSnapshot, len(snaps))
	for i, snap := range snaps {
		result[i] = PriceSnapshot{
			ProductID:     snap.ProductID,
			ProductName:   snap.ProductName,
			UnitPrice:     snap.UnitPrice,
			OriginalPrice: snap.OriginalPrice,
			Discount:      snap.Discount,
			Type:          Type(snap.Type),
			Cost:          snap.Cost,
			TaxClassID:    snap.TaxClassID,
			TaxRate:       snap.TaxRate,
			SnapshotAt:    snap.SnapshotAt,
		}
		if snap.Rule != nil {
			result[i].Rule = &Rule{
				ID:   snap.Rule.ID,
				Name: snap.Rule.Name,
				Type: Type(snap.Rule.Type),
			}
		}
	}
	return result, nil
}

func insertTestProductWithTax(ctx context.Context, t *testing.T, sku, name string, price, stock int, taxRate float64) int {
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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT01-PROD", "IT01 Product", 3500, 100, 11)

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

func TestCartService_CancelHeldCart(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashierNamed(ctx, t, "cancel_owner")
	otherCashier := insertTestCashierNamed(ctx, t, "cancel_other")
	prodID := insertTestProductWithTax(ctx, t, "CART-CANCEL-PROD", "Cancel Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	cart, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, "held", cart.Status)

	// Cancelling discards the held cart and removes it from the held list.
	cancelled, err := svc.CancelCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", cancelled.Status)

	held, err := svc.ListHeldCarts(ctx, cashierID)
	require.NoError(t, err)
	assert.Empty(t, held, "cancelled cart must not appear in the held list")

	// Ownership is enforced: another cashier cannot cancel a still-held cart.
	otherCart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	otherCart, err = svc.AddCartItem(ctx, otherCart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	otherCart, err = svc.HoldCart(ctx, otherCart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, "held", otherCart.Status)
	_, err = svc.CancelCart(ctx, otherCart.ID, otherCashier)
	assert.ErrorIs(t, err, ErrCartNotOwned)

	// Re-cancelling a cancelled cart is rejected.
	_, err = svc.CancelCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartNotOpen)
}

func TestCartService_IT02_NewItemAfterResumeUsesLatestPrice(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT02-PROD", "IT02 Product", 3500, 100, 11)

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
}

func TestCartService_StoreScopedPricingAppliedOnCartPath(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-STORE-PROD", "Store Scoped", 5000, 100, 11)

	var storeID int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Store A') RETURNING id`).Scan(&storeID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO pricing_rules (product_id, pricing_type, name, minimum_quantity, priority, pricing_method, pricing_value, store_id, status)
		VALUES ($1, 'special_price', 'store-a-promo', 1, 0, 'fixed_price', 3000, $2, 'approved')
	`, prodID, storeID)
	require.NoError(t, err)

	storeCart, err := svc.CreateOrGetOpenCart(ctx, cashierID, &storeID, nil, nil)
	require.NoError(t, err)
	storeCart, err = svc.AddCartItem(ctx, storeCart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, storeCart.Items, 1)
	assert.Equal(t, 3000, storeCart.Items[0].UnitPrice, "store-scoped price must apply on the cart path")

	plainCashierID := insertTestCashierNamed(ctx, t, "store_scope_plain_cashier")
	plainCart, err := svc.CreateOrGetOpenCart(ctx, plainCashierID, nil, nil, nil)
	require.NoError(t, err)
	plainCart, err = svc.AddCartItem(ctx, plainCart.ID, prodID, 1, nil, plainCashierID)
	require.NoError(t, err)
	require.Len(t, plainCart.Items, 1)
	assert.Equal(t, 5000, plainCart.Items[0].UnitPrice, "base price applies without a store scope")
}

func TestCartService_IT03_QuantityUpdateKeepsPrice(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT03-PROD", "IT03 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT04-PROD", "IT04 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT05-PROD", "IT05 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT06-PROD", "IT06 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT07-PROD", "IT07 Product", 3500, 100, 11)

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
	svc, bus := newCartTestService(ctx, t)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicSaleCreated},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT08-PROD", "IT08 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-PM-PROD", "Legacy PM Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.CheckoutCartWithPaymentMethod(ctx, cart.ID, "CASH", cashierID)
	assert.ErrorIs(t, err, ErrCartEmpty)
}

func TestCartService_IT09_CheckoutTwiceRejected(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT09-PROD", "IT09 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-IT10-PROD", "IT10 Product", 3500, 100, 11)

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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
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
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-PAY-PROD", "Payment Mismatch", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	_, err = svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: 1}}, cashierID)
	assert.ErrorIs(t, err, ErrPaymentTotalMismatch)
}

// P2-1 D2 CartCheckout regression: a cart that ends up with duplicate rows for
// the same product (reachable via direct API calls, since cart_items has no
// unique constraint) must be aggregated at checkout — stock deducted once for
// the combined quantity, no duplicate sale-item rows, and no oversell.
func TestCartService_CheckoutAggregatesDuplicateCartItems(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-DUP-PROD", "Cart Dup Product", 3500, 10, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	// First line adds 3 of the product; a second line adds 4 more of the same
	// product. The UI merges, but the API does not enforce uniqueness, so the
	// cart now holds two rows for prodID.
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 3, nil, cashierID)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 4, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 2, "cart holds two rows for the same product")

	total := cart.TotalAmount

	sale, err := svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: total}}, cashierID)
	require.NoError(t, err)
	require.Len(t, sale.Items, 1, "duplicate cart rows collapse into a single sale item")
	assert.Equal(t, prodID, sale.Items[0].ProductID)
	assert.Equal(t, 7, sale.Items[0].Quantity, "quantities are aggregated (3+4)")

	var stockAfter int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&stockAfter)
	require.NoError(t, err)
	assert.Equal(t, 3, stockAfter, "combined quantity 7 deducted from 10, never oversold")
}

// P2-1 D2 CartCheckout regression: a cart whose duplicate product rows exceed
// available stock must fail cleanly with ErrInsufficientStock and leave stock
// untouched (no negative stock). Each line individually fits, but the combined
// quantity does not — without aggregation this would have gone negative.
func TestCartService_CheckoutAggregatedOversellRejected(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-DUP-OVER", "Cart Dup Over", 3500, 8, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 5, nil, cashierID)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 5, nil, cashierID)
	require.NoError(t, err)
	require.Len(t, cart.Items, 2, "cart holds two rows for the same product")

	total := cart.TotalAmount

	_, err = svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: total}}, cashierID)
	require.ErrorIs(t, err, ErrInsufficientStock)

	var stockAfter int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&stockAfter)
	require.NoError(t, err)
	assert.Equal(t, 8, stockAfter, "stock untouched after rejected checkout")
}

// P2-1 D2 unit test: aggregateCartItems must only merge lines that share the
// full pricing snapshot. Two duplicate rows for the same product at the same
// unit price but resolved under different pricing rules (reachable when the
// same product is priced at different times via direct API calls) must NOT
// merge — a merged line would silently keep the first row's pricing-rule
// metadata while summing quantity.
func TestAggregateCartItemsKeepsDistinctPricingSnapshots(t *testing.T) {
	mkItem := func(qty int, ruleID *int, taxID *int) CartItem {
		return CartItem{ProductID: 1, UnitPrice: 3500, Quantity: qty, PricingRuleID: ruleID, TaxClassID: taxID}
	}

	t.Run("same product, price, tax, and rule merge", func(t *testing.T) {
		merged := aggregateCartItems([]CartItem{mkItem(2, intPtr(5), intPtr(11)), mkItem(3, intPtr(5), intPtr(11))})
		require.Len(t, merged, 1, "identical snapshots collapse")
		assert.Equal(t, 5, merged[0].Quantity)
	})

	t.Run("same product and price but different pricing rule do not merge", func(t *testing.T) {
		merged := aggregateCartItems([]CartItem{mkItem(2, intPtr(5), intPtr(11)), mkItem(3, intPtr(7), intPtr(11))})
		require.Len(t, merged, 2, "different pricing rules stay separate")
	})

	t.Run("same product and price but different tax class do not merge", func(t *testing.T) {
		merged := aggregateCartItems([]CartItem{mkItem(2, intPtr(5), intPtr(11)), mkItem(3, intPtr(5), intPtr(12))})
		require.Len(t, merged, 2, "different tax classes stay separate")
	})

	t.Run("tax-less lines merge with each other only", func(t *testing.T) {
		merged := aggregateCartItems([]CartItem{mkItem(2, nil, nil), mkItem(3, nil, nil)})
		require.Len(t, merged, 1, "two nil-tax lines collapse")
		assert.Equal(t, 5, merged[0].Quantity)
	})
}

func TestCartService_ResumeHeldCartIdempotentForOpen(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartNotOpen, "open cart is not in held state")
}

func TestCartService_UpdateCartCustomer(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
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
