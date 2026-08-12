package pricing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/permissions"

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

func insertTestStore(ctx context.Context, t *testing.T, name string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ($1) RETURNING id`, name).Scan(&id)
	require.NoError(t, err)
	return id
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"pricing.view", "pricing.create", "pricing.update", "pricing.delete"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(_ permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func setupPricingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := newWiredRepo()
	svc := NewService(repo)
	resolver := NewResolver(repo)
	h := NewHandler(svc, resolver, nil)
	h.SetProductSearcher(repo)

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
		Data []Rule `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_CreateRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t.Context(), t, "HDL-CR-"+time.Now().Format("0102150405"), "Handler Create Product", 15000)

	t.Run("success", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":12000,"name":"Handler Discount","minimum_quantity":1,"priority":0,"is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Rule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Discount", resp.Data.Name)
		assert.Equal(t, PricingMethodFixedPrice, resp.Data.Method)
		assert.Equal(t, 12000.0, resp.Data.PricingValue)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate name", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":12000,"name":"Handler Discount","minimum_quantity":1,"priority":0,"is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "nama rule sudah digunakan")
	})
}

func TestHandler_UpdateRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t.Context(), t, "HDL-UPD-"+time.Now().Format("0102150405"), "Handler Update Product", 15000)
	repo := newWiredRepo()
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    12000,
		Name:            "Before Update",
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(t.Context(), rule))

	t.Run("success", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"name":"After Update","pricing_type":"special_price","pricing_method":"fixed_price","pricing_value":10000,"minimum_quantity":3,"priority":1,"is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/pricing-rules/"+strconv.Itoa(rule.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Rule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "After Update", resp.Data.Name)
		assert.Equal(t, PricingTypeSpecialPrice, resp.Data.Type)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/pricing-rules/"+strconv.Itoa(rule.ID), strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/pricing-rules/abc", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate name", func(t *testing.T) {
		secondRule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    8000,
			Name:            "Unique Name For Update Test",
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(t.Context(), secondRule))

		body := `{"product_id":` + strconv.Itoa(productID) + `,"name":"After Update","pricing_type":"special_price","pricing_method":"fixed_price","pricing_value":10000,"minimum_quantity":1,"priority":0,"is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/pricing-rules/"+strconv.Itoa(secondRule.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["error"], "nama rule sudah digunakan")
	})
}

func TestHandler_DeleteRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t.Context(), t, "HDL-DEL-"+time.Now().Format("0102150405"), "Handler Delete Product", 15000)
	repo := newWiredRepo()
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    5000,
		Name:            "Delete Me",
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

	productID := insertTestProduct(t.Context(), t, "HDL-RES-"+time.Now().Format("0102150405"), "Handler Resolve Product", 15000)

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

	t.Run("product not found", func(t *testing.T) {
		body := `{"items":[{"product_id":999999,"quantity":1}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("empty items", func(t *testing.T) {
		body := `{"items":[]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing/resolve", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_SearchProducts(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	insertTestProduct(t.Context(), t, "HDL-SRC-"+time.Now().Format("0102150405"), "Searchable Handler Product", 10000)

	t.Run("search by name", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/search?q=Searchable&limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Data)
	})

	t.Run("empty query", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/search?q=&limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("non-existent product", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/search?q=ZZZZNONEXISTENT&limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})
}

func TestHandler_SubmitForApproval(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-SUB-"+time.Now().Format("0102150405"), "Submit Test Product", 15000)

	t.Run("submit draft rule", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    12000,
			Name:            "Submit Test Rule " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusDraft,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/"+strconv.Itoa(rule.ID)+"/submit", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "pending", resp["status"])
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/abc/submit", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("submit non-draft rule fails", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    11000,
			Name:            "Non-Draft Submit " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusApproved,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/"+strconv.Itoa(rule.ID)+"/submit", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_ApproveRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-APR-"+time.Now().Format("0102150405"), "Approve Test Product", 15000)

	t.Run("approve pending rule", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    12000,
			Name:            "Approve Test Rule " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusPending,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/"+strconv.Itoa(rule.ID)+"/approve", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "approved", resp["status"])
	})

	t.Run("approve non-pending rule fails", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    11000,
			Name:            "Draft Approve " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusDraft,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/"+strconv.Itoa(rule.ID)+"/approve", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_RejectRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-REJ-"+time.Now().Format("0102150405"), "Reject Test Product", 15000)

	t.Run("reject pending rule", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    12000,
			Name:            "Reject Test Rule " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusPending,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/"+strconv.Itoa(rule.ID)+"/reject", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "rejected", resp["status"])
	})

	t.Run("reject non-pending rule fails", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    11000,
			Name:            "Draft Reject " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusDraft,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/"+strconv.Itoa(rule.ID)+"/reject", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_GetRule(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-GR-"+time.Now().Format("0102150405"), "GetRule Test Product", 15000)

	t.Run("success", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    12000,
			Name:            "GetRule Success " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(t.Context(), rule))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules/"+strconv.Itoa(rule.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Rule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, rule.ID, resp.Data.ID)
		assert.Equal(t, rule.Name, resp.Data.Name)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found still returns ok", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/pricing-rules/999999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_GetRule_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pricing-rules/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ListRules_WithFilters(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-LF-"+time.Now().Format("0102150405"), "Filter Test Product", 15000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            "Filter Test Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusDraft,
	}
	require.NoError(t, repo.Create(t.Context(), rule))

	t.Run("filter by product_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?product_id="+strconv.Itoa(productID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by pricing_method", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?pricing_method=fixed_price", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by is_active true", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?is_active=true", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by is_active false", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?is_active=false", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?status=draft", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by invalid product_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?product_id=abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by invalid category_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?category_id=abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by invalid brand_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?brand_id=abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by invalid customer_group_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?customer_group_id=abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by invalid store_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?store_id=abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by valid category_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?category_id=1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by valid brand_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?brand_id=1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by valid customer_group_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?customer_group_id=1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by valid store_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?store_id=1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by search", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?search=Filter", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_ListRules_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-STORE-"+time.Now().Format("0102150405"), "Store Scoped Product", 15000)
	storeA := insertTestStore(t.Context(), t, "HDL-STORE-A")
	storeB := insertTestStore(t.Context(), t, "HDL-STORE-B")
	ruleStoreA := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            "Store A Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusDraft,
		StoreID:         &storeA,
	}
	ruleStoreB := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    9000,
		Name:            "Store B Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusDraft,
		StoreID:         &storeB,
	}
	require.NoError(t, repo.Create(t.Context(), ruleStoreA))
	require.NoError(t, repo.Create(t.Context(), ruleStoreB))

	t.Run("store-scoped user can view own store rules", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("username", "store_user")
			c.Set("roleID", 2)
			c.Set("role", "manager")
			c.Set("permissions", []string{"pricing.view"})
			c.Set("storeID", &storeA)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?store_id="+strconv.Itoa(storeA), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("store-scoped user cannot view another store rules", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("username", "store_user")
			c.Set("roleID", 2)
			c.Set("role", "manager")
			c.Set("permissions", []string{"pricing.view"})
			c.Set("storeID", &storeA)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules?store_id="+strconv.Itoa(storeB), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("store-scoped user defaults to own store when no store_id provided", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("username", "store_user")
			c.Set("roleID", 2)
			c.Set("role", "manager")
			c.Set("permissions", []string{"pricing.view"})
			c.Set("storeID", &storeA)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_DeleteRule_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/pricing-rules/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SearchProducts_NilSearcher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"pricing.view"})
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(nil, nil, nil)
	r.GET("/products/search", testPermMiddleware("pricing.view"), h.SearchProducts)

	t.Run("nil searcher returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/search?q=test&limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})

	t.Run("nil searcher with invalid limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/search?q=test&limit=abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nil searcher with out-of-range limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/products/search?q=test&limit=100", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_ApproveRule_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pricing-rules/abc/approve", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RejectRule_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pricing-rules/abc/reject", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CheckConflicts(t *testing.T) {
	skipIfNoDB(t)
	r := setupPricingRouter()

	productID := insertTestProduct(t.Context(), t, "HDL-CHK-"+time.Now().Format("0102150405"), "Conflict Test Product", 15000)

	t.Run("no conflicts", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":9999,"minimum_quantity":1,"priority":99}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/check-conflicts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data         []Rule `json:"data"`
			HasConflicts bool   `json:"has_conflicts"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.HasConflicts)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/check-conflicts", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("minimum quantity zero", func(t *testing.T) {
		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":9999,"minimum_quantity":0,"priority":99}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules/check-conflicts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
