package pricing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		c.Set("permissions", []string{"pricing:read", "pricing:create", "pricing:update", "pricing:delete"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(_ string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func setupPricingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	svc := NewService(repo)
	resolver := NewResolver(repo)
	h := NewHandler(svc, resolver, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListRules(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pricing-rules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []PricingRule `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_CreateRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t, t.Context(), "HDL-CR-"+time.Now().Format("0102150405"), "Handler Create Product", 15000)

	t.Run("success", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"discount","name":"Handler Discount","price":12000,"minimum_quantity":1,"priority":0,"is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data PricingRule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Discount", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t, t.Context(), "HDL-UPD-"+time.Now().Format("0102150405"), "Handler Update Product", 15000)
	repo := NewRepository(dbPool)
	rule := &PricingRule{
		ProductID:       productID,
		PricingType:     PricingTypeDiscount,
		Name:            "Before Update",
		Price:           12000,
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(t.Context(), rule))

	t.Run("success", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"name":"After Update","pricing_type":"wholesale","price":10000,"minimum_quantity":3,"priority":1,"is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/pricing-rules/"+strconv.Itoa(rule.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data PricingRule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "After Update", resp.Data.Name)
		assert.Equal(t, PricingTypeWholesale, resp.Data.PricingType)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/pricing-rules/abc", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t, t.Context(), "HDL-DEL-"+time.Now().Format("0102150405"), "Handler Delete Product", 15000)
	repo := NewRepository(dbPool)
	rule := &PricingRule{
		ProductID:       productID,
		PricingType:     PricingTypePromotion,
		Name:            "Delete Me",
		Price:           5000,
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(t.Context(), rule))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pricing-rules/"+strconv.Itoa(rule.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pricing-rules/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_ResolvePrices(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t, t.Context(), "HDL-RES-"+time.Now().Format("0102150405"), "Handler Resolve Product", 15000)

	t.Run("success", func(t *testing.T) {
		body := `{"items":[{"product_id":` + strconv.Itoa(productID) + `,"quantity":1}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []ResolvedPrice `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Data, 1)
		assert.Equal(t, 15000, resp.Data[0].UnitPrice)
		assert.Equal(t, 15000, resp.Data[0].OriginalPrice)
	})

	t.Run("empty items", func(t *testing.T) {
		body := `{"items":[]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing/resolve", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
