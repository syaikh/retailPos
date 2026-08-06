package sale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/secregtest"
	"retail-pos-system/internal/shared"
)

// regressionAuthMiddleware authenticates the caller and grants exactly the
// given permissions, letting tests compare sensitive-field visibility between
// roles (cashier vs manager).
func regressionAuthMiddleware(perms []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", int(testCashierID))
		c.Set("username", "sale_handler_user")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", perms)
		c.Set("storeID", nil)
		c.Next()
	}
}

// setupSaleRouterWithPerms builds the full sale + cart router authenticated as
// testCashierID holding exactly the given permissions.
func setupSaleRouterWithPerms(t *testing.T, perms []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)

	svc := NewService(repo, bus)
	svc.SetCartConfig(CartConfig{HoldTTLHours: 24})
	svc.SetPriceResolver(newPricingTestResolver())
	h := NewHandler(svc, nil)

	ctx := context.Background()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('sale_handler_user', 'sale_handler@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&id)
	require.NoError(t, err)
	testCashierID = int32(id)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), regressionAuthMiddleware(perms), testPermMiddleware)
	h.RegisterCartRoutes(r.Group("/"), regressionAuthMiddleware(perms), testPermMiddleware)
	return r
}

// insertRegressionProduct creates a product with a distinct cost so tests can
// assert the cost field is either present or fully absent.
func insertRegressionProduct(t *testing.T, ctx context.Context, sku, name string, price, cost, stock int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id
	`, sku, name, price, cost).Scan(&id)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity)
		VALUES ($1, $2)
	`, id, stock)
	require.NoError(t, err)
	return id
}

// createRegressionSale inserts a completed sale whose item carries a real cost.
func createRegressionSale(t *testing.T, ctx context.Context, repo *Repository, invoice string, prodID, qty, price, cost int) *Sale {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	cashierID := insertTestCashier(t, ctx)
	sale := &Sale{
		InvoiceNumber: invoice,
		CashierID:     cashierID,
		Subtotal:      price * qty,
		Discount:      0,
		Tax:           0,
		TotalAmount:   price * qty,
		PaymentMethod: "CASH",
		Status:        "completed",
	}
	items := []SaleItem{{
		ProductID: prodID,
		Name:      "Regression Sale Product",
		Quantity:  qty,
		UnitPrice: price,
		Subtotal:  price * qty,
		Cost:      cost,
	}}

	err = repo.CreateSale(ctx, tx, sale, items)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	return sale
}

// createCartWithItem opens a cart and adds one item via the HTTP API so the
// cart belongs to the authenticated caller, returning the cart ID.
func createCartWithItem(t *testing.T, r *gin.Engine, prodID int) int {
	t.Helper()
	req, _ := http.NewRequest("POST", "/pos/cart", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var createResp struct {
		Data CartSession `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))

	req2, _ := http.NewRequest("POST", "/pos/cart/items", bytes.NewBufferString(fmt.Sprintf(`{"product_id":%d,"quantity":2}`, prodID)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	return createResp.Data.ID
}

// TestRegression_SaleDetail_CostVisibility asserts the product cost embedded in
// sale items is only exposed to holders of product.cost.view.
func TestRegression_SaleDetail_CostVisibility(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	prodID := insertRegressionProduct(t, ctx, "REG-SALE-PROD", "Regression Sale Product", 10000, 6000, 10)
	sale := createRegressionSale(t, ctx, repo, "INV-REG-SALE-001", prodID, 2, 10000, 6000)

	cases := []struct {
		name   string
		perms  []string
		fields []secregtest.Field
	}{
		{
			name:  "cashier sale detail omits cost",
			perms: []string{permissions.SaleView.String()},
			fields: []secregtest.Field{
				secregtest.Visible("data.invoice_number"),
				secregtest.Visible("data.items.0.product_name"),
				secregtest.Visible("data.items.0.quantity"),
				secregtest.Visible("data.items.0.unit_price"),
				secregtest.Absent("data.items.0.cost"),
			},
		},
		{
			name:  "manager sale detail includes cost",
			perms: []string{permissions.SaleView.String(), permissions.ProductCostView.String()},
			fields: []secregtest.Field{
				secregtest.Visible("data.invoice_number"),
				secregtest.Visible("data.items.0.cost"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupSaleRouterWithPerms(t, tc.perms)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/sales/"+strconv.Itoa(sale.ID), nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			secregtest.Check(t, w.Body.Bytes(), tc.fields...)
		})
	}
}

// TestRegression_Cart_CostVisibility asserts the product cost on open cart
// items is only exposed to holders of product.cost.view.
func TestRegression_Cart_CostVisibility(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	prodID := insertRegressionProduct(t, ctx, "REG-CART-PROD", "Regression Cart Product", 10000, 5000, 50)
	createCartWithItem(t, setupSaleRouterWithPerms(t, []string{permissions.SaleCreate.String()}), prodID)

	cases := []struct {
		name   string
		perms  []string
		fields []secregtest.Field
	}{
		{
			name:  "cashier open cart omits cost",
			perms: []string{permissions.SaleCreate.String()},
			fields: []secregtest.Field{
				secregtest.Visible("data.items.0.product_name"),
				secregtest.Visible("data.items.0.quantity"),
				secregtest.Visible("data.items.0.unit_price"),
				secregtest.Absent("data.items.0.cost"),
			},
		},
		{
			name:  "manager open cart includes cost",
			perms: []string{permissions.SaleCreate.String(), permissions.ProductCostView.String()},
			fields: []secregtest.Field{
				secregtest.Visible("data.items.0.product_name"),
				secregtest.Visible("data.items.0.cost"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupSaleRouterWithPerms(t, tc.perms)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/pos/cart", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			secregtest.Check(t, w.Body.Bytes(), tc.fields...)
		})
	}
}

// TestRegression_HeldCarts_CostVisibility asserts held-cart items follow the
// same cost visibility rule.
func TestRegression_HeldCarts_CostVisibility(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	prodID := insertRegressionProduct(t, ctx, "REG-HELD-PROD", "Regression Held Product", 10000, 5000, 50)
	r := setupSaleRouterWithPerms(t, []string{permissions.SaleCreate.String()})
	cartID := createCartWithItem(t, r, prodID)

	req2, _ := http.NewRequest("POST", "/pos/cart/"+strconv.Itoa(cartID)+"/hold", bytes.NewBufferString(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	req3, _ := http.NewRequest("GET", "/pos/cart/held", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)
	secregtest.Check(t, w3.Body.Bytes(),
		secregtest.Visible("data.0.items.0.product_name"),
		secregtest.Visible("data.0.items.0.quantity"),
		secregtest.Absent("data.0.items.0.cost"),
	)
}
