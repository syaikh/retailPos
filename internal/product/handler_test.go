package product

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"product.create", "product.update", "product.delete"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupProductRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, nil, nil, nil, bus)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_GetProducts(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	t.Run("returns empty list when no products match", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products?search=NONEXISTENT_XYZ", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Product `json:"data"`
			Total int       `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})

	t.Run("returns products with valid params", func(t *testing.T) {
		repo := NewRepository(dbPool)
		ctx := context.Background()
		p := &Product{
			SKU: "HDL-PROD-01", Name: "Handler Product",
			Price: 10000, Cost: 5000, Stock: 5, Status: "active",
		}
		require.NoError(t, repo.CreateProduct(ctx, p))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products?limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_GetProductByID(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	p := &Product{
		SKU: "HDL-BYID", Name: "By ID Test",
		Price: 5000, Cost: 2500, Stock: 3, Status: "active",
	}
	require.NoError(t, repo.CreateProduct(ctx, p))

	t.Run("found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/"+strconv.Itoa(p.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Product `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, p.Name, resp.Data.Name)
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/999999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_CreateProduct(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	t.Run("success", func(t *testing.T) {
		body := `{"sku":"HDL-CREATE","name":"Created Product","price":15000,"cost":10000,"stock":10,"status":"active"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Product `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Created Product", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateProduct(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	p := &Product{
		SKU: "HDL-UPD", Name: "Before Update",
		Price: 5000, Cost: 2500, Stock: 3, Status: "active",
	}
	require.NoError(t, repo.CreateProduct(ctx, p))

	t.Run("success", func(t *testing.T) {
		body := `{"sku":"HDL-UPD","name":"After Update","price":8000,"cost":4000,"stock":5,"status":"active"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/products/"+strconv.Itoa(p.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_DeleteProduct(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	p := &Product{
		SKU: "HDL-DEL", Name: "To Delete",
		Price: 3000, Cost: 1500, Stock: 2, Status: "active",
	}
	require.NoError(t, repo.CreateProduct(ctx, p))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/products/"+strconv.Itoa(p.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_NextSKU(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/products/next-sku", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data string `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Data, "SKU-")
}

func TestHandler_BulkUpdateProductStatus(t *testing.T) {
	skipIfNoDB(t)
	r := setupProductRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	p1 := &Product{SKU: "BULK-01", Name: "Bulk Product 1", Price: 5000, Cost: 2500, Stock: 10, Status: "active"}
	require.NoError(t, repo.CreateProduct(ctx, p1))
	p2 := &Product{SKU: "BULK-02", Name: "Bulk Product 2", Price: 5000, Cost: 2500, Stock: 10, Status: "active"}
	require.NoError(t, repo.CreateProduct(ctx, p2))

	t.Run("success deactivate", func(t *testing.T) {
		body := fmt.Sprintf(`{"ids":[%d,%d],"is_active":false}`, p1.ID, p2.ID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success activate", func(t *testing.T) {
		body := fmt.Sprintf(`{"ids":[%d],"is_active":true}`, p1.ID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ids", func(t *testing.T) {
		body := `{"ids":[],"is_active":false}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/products/bulk/status", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_PublicRoutes(t *testing.T) {
	skipIfNoDB(t)
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, nil, nil, nil, bus)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterPublicRoutes(r.Group("/"))

	t.Run("list tax classes", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/tax-classes", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get stock thresholds", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/stock-thresholds", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Warning  int `json:"warning"`
			Critical int `json:"critical"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 10, resp.Warning)
		assert.Equal(t, 5, resp.Critical)
	})
}
