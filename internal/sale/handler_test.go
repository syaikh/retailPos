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
	"retail-pos-system/internal/shared"
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
		c.Set("permissions", []string{"sale:create", "sale:read", "report:read"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupSaleRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, bus)
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
	shared.TruncateTestData(dbPool)
	r := setupSaleRouter()

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
		repo := NewRepository(dbPool)
		ctx := context.Background()
		prodID := insertTestProduct(t, ctx, "HDL-LIST-PROD", "Handler List Product", 10000, 50)
		_ = createAndCommitSale(t, ctx, repo, "INV-HDL-LIST-001", prodID, 2, 10000, 20000, 20000, 0)

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
	shared.TruncateTestData(dbPool)
	r := setupSaleRouter()

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		prodID := insertTestProduct(t, ctx, "HDL-CREATE-PROD", "Handler Create Product", 15000, 100)

		body := `{"invoice_number":"INV-HDL-CREATE-001","items":[{"product_id":` + strconv.Itoa(prodID) + `,"quantity":2,"subtotal":30000}],"payment_method":"CASH"}`
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
		assert.Equal(t, "INV-HDL-CREATE-001", resp.Data.InvoiceNumber)
		assert.Equal(t, 30000, resp.Data.Subtotal)
		assert.Equal(t, 30000, resp.Data.TotalAmount)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("insufficient stock", func(t *testing.T) {
		ctx := context.Background()
		prodID := insertTestProduct(t, ctx, "HDL-LOW-STOCK", "Handler Low Stock", 5000, 1)

		body := `{"invoice_number":"INV-HDL-LOW-001","items":[{"product_id":` + strconv.Itoa(prodID) + `,"quantity":10,"subtotal":50000}],"payment_method":"CASH"}`
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

func TestHandler_GetSaleByID(t *testing.T) {
	skipIfNoDB(t)
	shared.TruncateTestData(dbPool)
	r := setupSaleRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	prodID := insertTestProduct(t, ctx, "HDL-GETBYID", "Handler ByID", 8000, 30)
	sale := createAndCommitSale(t, ctx, repo, "INV-HDL-GETBYID-001", prodID, 3, 8000, 24000, 24000, 0)

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
	shared.TruncateTestData(dbPool)
	r := setupSaleRouter()

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
