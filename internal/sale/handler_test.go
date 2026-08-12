package sale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

var testCashierID int32

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", int(testCashierID))
		c.Set("username", "sale_handler_user")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"sale.create", "sale.view", "report.view"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupSaleRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := newTestRepo(t)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, bus)
	svc.SetStockDeducer(inventory.StockDeducer{})
	svc.SetShiftTotalUpdater(shift.TotalUpdater{})
	svc.SetPriceResolver(newPricingTestResolver())
	h := NewHandler(svc, nil)

	ctx := context.Background()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('sale_handler_user', 'sale_handler@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&id)
	if err != nil {
		panic(err)
	}
	testCashierID = int32(id)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_GetSalesHistory(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupSaleRouter(t)

	t.Run("returns empty list when no sales match", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales?search=NONEXISTENT_SALE_XYZ", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Sale `json:"data"`
			Total int    `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})

	t.Run("returns sales with valid params", func(t *testing.T) {
		repo := newTestRepo(t)
		ctx := context.Background()
		prodID := insertTestProduct(ctx, t, "HDL-LIST-PROD", "Handler List Product", 10000, 50)
		_ = createAndCommitSale(ctx, t, repo, "INV-HDL-LIST-001", prodID, 2, 10000, 20000, 20000, 0)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales?limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Sale `json:"data"`
			Total int    `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Data)
	})

	t.Run("filters by min_total and max_total", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales?min_total=0&max_total=50000000", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid min_total", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales?min_total=-1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_CreateSale(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupSaleRouter(t)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		prodID := insertTestProduct(ctx, t, "HDL-CREATE-PROD", "Handler Create Product", 15000, 100)

		body := `{"items":[{"product_id":` + strconv.Itoa(prodID) + `,"quantity":2}],"payment_method":"CASH"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Sale `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Data.InvoiceNumber, "invoice number generated server-side")
		assert.Equal(t, 30000, resp.Data.Subtotal, "server price (2 x 15000)")
		assert.Equal(t, 30000, resp.Data.TotalAmount)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("legacy pricing fields rejected", func(t *testing.T) {
		ctx := context.Background()
		prodID := insertTestProduct(ctx, t, "HDL-DISC-PROD", "Handler Discount Product", 15000, 100)

		body := `{"items":[{"product_id":` + strconv.Itoa(prodID) + `,"quantity":1,"unit_price":1}],"discount":14000,"tax":1,"payment_method":"CASH"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error.Message, "discount is not accepted")
	})

	t.Run("product not found", func(t *testing.T) {
		body := `{"items":[{"product_id":999999,"quantity":1}],"payment_method":"CASH"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error.Message, "product not found")
	})

	t.Run("insufficient stock", func(t *testing.T) {
		ctx := context.Background()
		prodID := insertTestProduct(ctx, t, "HDL-LOW-STOCK", "Handler Low Stock", 5000, 1)

		body := `{"items":[{"product_id":` + strconv.Itoa(prodID) + `,"quantity":10}],"payment_method":"CASH"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing items field", func(t *testing.T) {
		body := `{"invoice_number":"INV-HDL-NOITEMS-001","payment_method":"CASH"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// insertOpenCartWithItem creates a server-side cart session with a single snapshot
// item directly in the DB, returning the cart id.
func insertOpenCartWithItem(ctx context.Context, t *testing.T, cashierID, prodID int, qty, unitPrice int) int {
	t.Helper()
	var cartID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO cart_sessions (cashier_id, status, subtotal, discount, tax, total_amount)
		VALUES ($1, 'open', $2, 0, 0, $2)
		RETURNING id
	`, cashierID, qty*unitPrice).Scan(&cartID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO cart_items (cart_session_id, product_id, product_name, quantity, unit_price, original_price, discount, cost, subtotal, dpp_amount, tax_amount, snapshot_created_at)
		VALUES ($1, $2, $3, $4, $5, $5, 0, $6, $7, $7, 0, NOW())
	`, cartID, prodID, "Cart Snapshot Item", qty, unitPrice, unitPrice/2, qty*unitPrice)
	require.NoError(t, err)
	return cartID
}

func TestHandler_CreateSale_WithCartSessionID_Integration(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupSaleRouter(t)
	ctx := context.Background()

	prodID := insertTestProduct(ctx, t, "HDL-CART-CHECKOUT", "Cart Checkout Product", 10000, 100)
	cashierID := int(testCashierID)

	t.Run("cart_session_id behaves like CheckoutCart", func(t *testing.T) {
		cartID := insertOpenCartWithItem(ctx, t, cashierID, prodID, 2, 10000)

		body := `{"cart_session_id":` + strconv.Itoa(cartID) + `,"payments":[{"payment_method_code":"CASH","amount":20000}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Sale `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Greater(t, resp.Data.ID, 0)
		assert.Equal(t, 20000, resp.Data.Subtotal)
		assert.Equal(t, 20000, resp.Data.TotalAmount)
		require.Len(t, resp.Data.Items, 1)
		assert.Equal(t, 10000, resp.Data.Items[0].UnitPrice, "sale item should use the cart snapshot price")
		assert.Equal(t, 2, resp.Data.Items[0].Quantity)

		// Cart must be marked checked_out and stock deducted.
		var status string
		err = dbPool.QueryRow(ctx, `SELECT status FROM cart_sessions WHERE id = $1`, cartID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "checked_out", status)

		var stock int
		err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&stock)
		require.NoError(t, err)
		assert.Equal(t, 98, stock)
	})

	t.Run("cart_session_id falls back to legacy payment_method", func(t *testing.T) {
		cartID := insertOpenCartWithItem(ctx, t, cashierID, prodID, 1, 10000)

		body := `{"cart_session_id":` + strconv.Itoa(cartID) + `,"payment_method":"CASH"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Sale `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Greater(t, resp.Data.ID, 0)
		assert.Equal(t, 10000, resp.Data.TotalAmount)
		assert.Equal(t, "CASH", resp.Data.PaymentMethod)
	})

	t.Run("cart_session_id not found returns 404", func(t *testing.T) {
		body := `{"cart_session_id":999999,"payments":[{"payment_method_code":"CASH","amount":10000}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("cart_session_id with no payments returns 400", func(t *testing.T) {
		cartID := insertOpenCartWithItem(ctx, t, cashierID, prodID, 1, 10000)

		body := `{"cart_session_id":` + strconv.Itoa(cartID) + `}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("cart_session_id of another cashier returns 403", func(t *testing.T) {
		var otherID int
		err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('sale_handler_other', 'sale_handler_other@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&otherID)
		require.NoError(t, err)

		cartID := insertOpenCartWithItem(ctx, t, otherID, prodID, 1, 10000)

		body := `{"cart_session_id":` + strconv.Itoa(cartID) + `,"payments":[{"payment_method_code":"CASH","amount":10000}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("cart_session_id with empty cart returns 409", func(t *testing.T) {
		// Earlier subtests may have left an open cart for the same cashier, which
		// would violate uq_cart_sessions_open_cashier.
		_, err := dbPool.Exec(ctx, `
			DELETE FROM cart_items WHERE cart_session_id IN (SELECT id FROM cart_sessions WHERE cashier_id = $1 AND status = 'open')
		`, int(testCashierID))
		require.NoError(t, err)
		_, err = dbPool.Exec(ctx, `
			DELETE FROM cart_sessions WHERE cashier_id = $1 AND status = 'open'
		`, int(testCashierID))
		require.NoError(t, err)

		var cartID int
		err = dbPool.QueryRow(ctx, `
			INSERT INTO cart_sessions (cashier_id, status, subtotal, discount, tax, total_amount)
			VALUES ($1, 'open', 0, 0, 0, 0)
			RETURNING id
		`, int(testCashierID)).Scan(&cartID)
		require.NoError(t, err)

		body := `{"cart_session_id":` + strconv.Itoa(cartID) + `,"payments":[{"payment_method_code":"CASH","amount":10000}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestHandler_GetSaleByID(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupSaleRouter(t)

	repo := newTestRepo(t)
	ctx := context.Background()
	prodID := insertTestProduct(ctx, t, "HDL-GETBYID", "Handler ByID", 8000, 30)
	sale := createAndCommitSale(ctx, t, repo, "INV-HDL-GETBYID-001", prodID, 3, 8000, 24000, 24000, 0)

	t.Run("found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales/"+strconv.Itoa(sale.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Sale `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, sale.InvoiceNumber, resp.Data.InvoiceNumber)
		assert.Equal(t, sale.TotalAmount, resp.Data.TotalAmount)
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales/999999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_ExportSales(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupSaleRouter(t)

	t.Run("exports CSV", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales/export", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	})

	t.Run("exports XLSX", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/sales/export?format=xlsx", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "openxmlformats")
	})
}

func TestHandler_GetPaymentMethodByCode(t *testing.T) {
	skipIfNoDB(t)
	r := setupSaleRouter(t)

	t.Run("found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/payment-methods/CASH", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data PaymentMethod `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "CASH", resp.Data.Code)
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/payment-methods/NONEXISTENT", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
