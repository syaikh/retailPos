package sale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/shared"
)

// insertTestCashierNamed creates a distinct cashier so ownership-mismatch paths
// can be exercised (insertTestCashier always reuses the same user).
func insertTestCashierNamed(ctx context.Context, t *testing.T, username string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ($1, $2, 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`, username, username+"@test.com").Scan(&id)
	require.NoError(t, err)
	return id
}

// TestCartService_GetCartByID covers the ownership and not-found paths of
// GetCartByID.
func TestCartService_GetCartByID(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	t.Run("returns owned cart", func(t *testing.T) {
		got, err := svc.GetCartByID(ctx, cart.ID, cashierID)
		require.NoError(t, err)
		assert.Equal(t, cart.ID, got.ID)
	})

	t.Run("rejects other cashier", func(t *testing.T) {
		other := insertTestCashierNamed(ctx, t, "sale_test_other_cashier")
		_, err := svc.GetCartByID(ctx, cart.ID, other)
		assert.ErrorIs(t, err, ErrCartNotOwned)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetCartByID(ctx, 999999, cashierID)
		assert.ErrorContains(t, err, "cart session not found")
	})
}

// TestCartService_HoldCart_NotOpen asserts holding a non-open cart is rejected.
func TestCartService_HoldCart_NotOpen(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)

	_, err = svc.HoldCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartNotOpen)
}

// TestCartService_ResumeCart_NotHeld asserts resuming an open cart is rejected.
func TestCartService_ResumeCart_NotHeld(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartNotOpen)
}

// TestCartService_ResumeCart_Expired asserts resuming an expired held cart is
// rejected.
func TestCartService_ResumeCart_Expired(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	_, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `UPDATE cart_sessions SET expired_at = now() - interval '1 hour' WHERE id = $1`, cart.ID)
	require.NoError(t, err)

	_, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	assert.ErrorIs(t, err, ErrCartExpired)
}

// TestCartService_UpdateCartCustomer_Branches covers the not-owned and
// not-open paths of UpdateCartCustomer.
func TestCartService_UpdateCartCustomer_Branches(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	var customerID int
	err = dbPool.QueryRow(ctx, `INSERT INTO customers (name, email, phone) VALUES ('Cart Branch Customer', 'cartbranch@test.com', '08123') RETURNING id`).Scan(&customerID)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		got, err := svc.UpdateCartCustomer(ctx, cart.ID, &customerID, cashierID)
		require.NoError(t, err)
		require.NotNil(t, got.CustomerID)
		assert.Equal(t, customerID, *got.CustomerID)
	})

	t.Run("rejects other cashier", func(t *testing.T) {
		other := insertTestCashierNamed(ctx, t, "sale_test_customer_other")
		_, err := svc.UpdateCartCustomer(ctx, cart.ID, &customerID, other)
		assert.ErrorIs(t, err, ErrCartNotOwned)
	})

	t.Run("rejects non-open cart", func(t *testing.T) {
		held, err := svc.HoldCart(ctx, cart.ID, cashierID)
		require.NoError(t, err)
		_, err = svc.UpdateCartCustomer(ctx, held.ID, &customerID, cashierID)
		assert.ErrorIs(t, err, ErrCartNotOpen)
	})
}

// TestCartService_RemoveCartItem_Branches covers the success, not-owned and
// not-open paths of RemoveCartItem.
func TestCartService_RemoveCartItem_Branches(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-REMOVE-PROD", "Remove Item Product", 12000, 50, 11)

	t.Run("success removes item", func(t *testing.T) {
		cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
		require.NoError(t, err)
		cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 2, nil, cashierID)
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		itemID := cart.Items[0].ID

		got, err := svc.RemoveCartItem(ctx, cart.ID, itemID, cashierID)
		require.NoError(t, err)
		assert.Empty(t, got.Items)
	})

	t.Run("rejects other cashier", func(t *testing.T) {
		cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
		require.NoError(t, err)
		other := insertTestCashierNamed(ctx, t, "sale_test_remove_other")
		_, err = svc.RemoveCartItem(ctx, cart.ID, 1, other)
		assert.ErrorIs(t, err, ErrCartNotOwned)
	})

	t.Run("rejects non-open cart", func(t *testing.T) {
		cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
		require.NoError(t, err)
		_, err = svc.HoldCart(ctx, cart.ID, cashierID)
		require.NoError(t, err)
		_, err = svc.RemoveCartItem(ctx, cart.ID, 1, cashierID)
		assert.ErrorIs(t, err, ErrCartNotOpen)
	})
}

// TestCartService_QuantityValidation covers the zero-quantity guard in the
// add/update item service methods.
func TestCartService_QuantityValidation(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.AddCartItem(ctx, cart.ID, 1, 0, nil, cashierID)
	assert.ErrorIs(t, err, ErrCartItemQuantity)

	_, err = svc.UpdateCartItemQuantity(ctx, cart.ID, 1, 0, cashierID)
	assert.ErrorIs(t, err, ErrCartItemQuantity)
}

// TestCartService_CheckoutCart_WithShift exercises the shift totals path of
// finalizeSaleCreation by checking out a cart bound to an open shift.
func TestCartService_CheckoutCart_WithShift(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	var shiftID int
	err := dbPool.QueryRow(ctx, `INSERT INTO shifts (user_id, status, opening_balance, opened_at) VALUES ($1, 'open', 0, NOW()) RETURNING id`, cashierID).Scan(&shiftID)
	require.NoError(t, err)

	prodID := insertTestProductWithTax(ctx, t, "CART-SHIFT-PROD", "Shift Checkout Product", 3500, 100, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, &shiftID, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)

	sale, err := svc.CheckoutCart(ctx, cart.ID, []CreatePaymentRequest{{PaymentMethodCode: "CASH", Amount: cart.TotalAmount}}, cashierID)
	require.NoError(t, err)
	require.NotNil(t, sale.ShiftID)
	assert.Equal(t, shiftID, *sale.ShiftID)
}

// TestSaleRepository_CreateCartSession covers the direct cart-session insert.
func TestSaleRepository_CreateCartSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)

	cashierID := insertTestCashier(ctx, t)
	cart, err := repo.CreateCartSession(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, cashierID, cart.CashierID)
	assert.Equal(t, "open", cart.Status)
	assert.NotEmpty(t, cart.CreatedAt)
	assert.NotEmpty(t, cart.UpdatedAt)

	got, err := repo.GetCartSessionByID(ctx, cart.ID)
	require.NoError(t, err)
	assert.Equal(t, cart.ID, got.ID)
}

// TestCartService_HeldCart_WithShiftAndCustomer exercises the non-null
// optional-column scan branches in scanCartSession/ListHeldCarts and the
// item-loading path of ListHeldCarts.
func TestCartService_HeldCart_WithShiftAndCustomer(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	var shiftID int
	err := dbPool.QueryRow(ctx, `INSERT INTO shifts (user_id, status, opening_balance, opened_at) VALUES ($1, 'open', 0, NOW()) RETURNING id`, cashierID).Scan(&shiftID)
	require.NoError(t, err)
	var customerID int
	err = dbPool.QueryRow(ctx, `INSERT INTO customers (name, email, phone) VALUES ('Held Cart Customer', 'heldcart@test.com', '08123') RETURNING id`).Scan(&customerID)
	require.NoError(t, err)
	var storeID int
	err = dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Held Cart Store') RETURNING id`).Scan(&storeID)
	require.NoError(t, err)
	prodID := insertTestProductWithTax(ctx, t, "CART-HELD-PROD", "Held Cart Product", 12000, 50, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, &storeID, &shiftID, &customerID)
	require.NoError(t, err)
	require.NotNil(t, cart.ShiftID)
	require.NotNil(t, cart.CustomerID)
	require.NotNil(t, cart.StoreID)
	assert.Equal(t, shiftID, *cart.ShiftID)
	assert.Equal(t, customerID, *cart.CustomerID)
	assert.Equal(t, storeID, *cart.StoreID)

	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 2, nil, cashierID)
	require.NoError(t, err)

	_, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)

	held, err := svc.ListHeldCarts(ctx, cashierID)
	require.NoError(t, err)
	require.Len(t, held, 1)
	assert.Equal(t, shiftID, *held[0].ShiftID)
	assert.Equal(t, customerID, *held[0].CustomerID)
	assert.Equal(t, storeID, *held[0].StoreID)
	require.Len(t, held[0].Items, 1)

	got, err := svc.GetCartByID(ctx, cart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, shiftID, *got.ShiftID)
	assert.Equal(t, customerID, *got.CustomerID)
	assert.Equal(t, storeID, *got.StoreID)

	_, err = svc.ResumeCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)
}

// TestCartService_HoldCart_DefaultTTL covers the default-TTL fallback when no
// cart config is set.
func TestCartService_HoldCart_DefaultTTL(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)
	svc := NewService(NewRepository(dbPool), bus)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	held, err := svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)
	assert.Equal(t, "held", held.Status)
	require.NotNil(t, held.ExpiredAt)
}

// TestCartService_HoldCart_NotOwned asserts a cashier cannot hold another
// cashier's cart.
func TestCartService_HoldCart_NotOwned(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	other := insertTestCashierNamed(ctx, t, "sale_test_hold_other")
	_, err = svc.HoldCart(ctx, cart.ID, other)
	assert.ErrorIs(t, err, ErrCartNotOwned)
}

// TestCartService_ResumeCart_NotOwned asserts a cashier cannot resume another
// cashier's held cart.
func TestCartService_ResumeCart_NotOwned(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	_, err = svc.HoldCart(ctx, cart.ID, cashierID)
	require.NoError(t, err)

	other := insertTestCashierNamed(ctx, t, "sale_test_resume_other")
	_, err = svc.ResumeCart(ctx, cart.ID, other)
	assert.ErrorIs(t, err, ErrCartNotOwned)
}

// TestCartService_UpdateCartItemQuantity_Branches covers the not-owned and
// item-not-found paths of UpdateCartItemQuantity.
func TestCartService_UpdateCartItemQuantity_Branches(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	prodID := insertTestProductWithTax(ctx, t, "CART-UQ-PROD", "Update Qty Product", 12000, 50, 11)

	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)
	cart, err = svc.AddCartItem(ctx, cart.ID, prodID, 1, nil, cashierID)
	require.NoError(t, err)
	itemID := cart.Items[0].ID

	t.Run("rejects other cashier", func(t *testing.T) {
		other := insertTestCashierNamed(ctx, t, "sale_test_uq_other")
		_, err := svc.UpdateCartItemQuantity(ctx, cart.ID, itemID, 2, other)
		assert.ErrorIs(t, err, ErrCartNotOwned)
	})

	t.Run("rejects missing item", func(t *testing.T) {
		_, err := svc.UpdateCartItemQuantity(ctx, cart.ID, 999999, 2, cashierID)
		assert.ErrorIs(t, err, ErrCartItemNotFound)
	})

	t.Run("success updates quantity", func(t *testing.T) {
		got, err := svc.UpdateCartItemQuantity(ctx, cart.ID, itemID, 3, cashierID)
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, 3, got.Items[0].Quantity)
	})
}

// TestCartService_RemoveCartItem_ItemNotFound covers the repository
// no-rows-affected branch when removing a nonexistent item.
func TestCartService_RemoveCartItem_ItemNotFound(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	svc, _ := newCartTestService(ctx, t)

	cashierID := insertTestCashier(ctx, t)
	cart, err := svc.CreateOrGetOpenCart(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	_, err = svc.RemoveCartItem(ctx, cart.ID, 999999, cashierID)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

// TestSaleRepository_UpdateCartItemQuantity_MissingItem covers the repository
// no-rows-affected branch for quantity updates.
func TestSaleRepository_UpdateCartItemQuantity_MissingItem(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)

	cashierID := insertTestCashier(ctx, t)
	cart, err := repo.CreateCartSession(ctx, cashierID, nil, nil, nil)
	require.NoError(t, err)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = repo.UpdateCartItemQuantity(ctx, tx, cart.ID, 999999, 2, 24000, 24000, 0)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}
