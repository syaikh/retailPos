package pricing

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestHandler_GetRule_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-GR-STORE-"+time.Now().Format("0102150405"), "GetRule Store Scoped Product", 15000)
	storeA := insertTestStore(t.Context(), t, "HDL-GR-STORE-A")
	storeB := insertTestStore(t.Context(), t, "HDL-GR-STORE-B")
	globalRule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    12000,
		Name:            "Global Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusDraft,
	}
	ruleStoreA := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    11000,
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
		PricingValue:    10000,
		Name:            "Store B Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusDraft,
		StoreID:         &storeB,
	}
	require.NoError(t, repo.Create(t.Context(), globalRule))
	require.NoError(t, repo.Create(t.Context(), ruleStoreA))
	require.NoError(t, repo.Create(t.Context(), ruleStoreB))

	t.Run("store-scoped user can view own store rule", func(t *testing.T) {
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
		req, _ := http.NewRequest("GET", "/pricing-rules/"+strconv.Itoa(ruleStoreA.ID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("store-scoped user cannot view another store rule", func(t *testing.T) {
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
		req, _ := http.NewRequest("GET", "/pricing-rules/"+strconv.Itoa(ruleStoreB.ID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("store-scoped user cannot view global rule", func(t *testing.T) {
		// A global (store_id IS NULL) rule applies to every store at the point of
		// sale, so it must be superadmin-only for management. Otherwise any store
		// manager could read or edit a chain-wide price.
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
		req, _ := http.NewRequest("GET", "/pricing-rules/"+strconv.Itoa(globalRule.ID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("superadmin can view global rule", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("username", "root")
			c.Set("roleID", 1)
			c.Set("role", "superadmin")
			c.Set("permissions", []string{"pricing.view"})
			c.Set("storeID", nil)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pricing-rules/"+strconv.Itoa(globalRule.ID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
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

func TestHandler_CheckConflicts_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()

	productID := insertTestProduct(t.Context(), t, "HDL-CHK-STORE-"+time.Now().Format("0102150405"), "Conflict Store Scoped Product", 15000)
	storeA := insertTestStore(t.Context(), t, "HDL-CHK-STORE-A")
	storeB := insertTestStore(t.Context(), t, "HDL-CHK-STORE-B")
	globalRule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            "Global Conflict Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusApproved,
		Priority:        10,
	}
	ruleStoreA := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    9000,
		Name:            "Store A Conflict Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusApproved,
		Priority:        10,
		StoreID:         &storeA,
	}
	ruleStoreB := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    8000,
		Name:            "Store B Conflict Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		Status:          StatusApproved,
		Priority:        10,
		StoreID:         &storeB,
	}
	require.NoError(t, repo.Create(t.Context(), globalRule))
	require.NoError(t, repo.Create(t.Context(), ruleStoreA))
	require.NoError(t, repo.Create(t.Context(), ruleStoreB))

	t.Run("store-scoped user sees only own store and global conflicts", func(t *testing.T) {
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

		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":9999,"minimum_quantity":1,"priority":10}`
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
		assert.True(t, resp.HasConflicts)
		for _, conflict := range resp.Data {
			assert.True(t, conflict.StoreID == nil || *conflict.StoreID == storeA, "conflict rule %d has store_id=%v, expected nil or %d", conflict.ID, conflict.StoreID, storeA)
		}
	})

	t.Run("admin sees all conflicts", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("username", "manager")
			c.Set("roleID", 1)
			c.Set("role", "superadmin")
			c.Set("permissions", []string{"pricing.view"})
			c.Set("storeID", nil)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})

		body := `{"product_id":` + strconv.Itoa(productID) + `,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":9999,"minimum_quantity":1,"priority":10}`
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
		assert.True(t, resp.HasConflicts)
		assert.GreaterOrEqual(t, len(resp.Data), 3)
	})
}

// TestHandler_MutationStoreBoundary covers the store boundary on the six
// mutating pricing endpoints, which previously ran entirely unscoped: a
// store-scoped manager could update, delete, submit, approve, or reject another
// store's rule, and could create or move a rule into the global (store_id IS
// NULL) scope that only superadmin may manage.
func TestHandler_MutationStoreBoundary(t *testing.T) {
	skipIfNoDB(t)
	gin.SetMode(gin.TestMode)

	repo := newWiredRepo()
	productID := insertTestProduct(t.Context(), t, "MUT-SB-"+time.Now().Format("0102150405"), "Mutation Boundary Product", 15000)
	storeA := insertTestStore(t.Context(), t, "MutSB A "+time.Now().Format("0102150405.000"))
	storeB := insertTestStore(t.Context(), t, "MutSB B "+time.Now().Format("0102150405.000"))

	seq := 0
	mkRule := func(storeID *int, name string) *Rule {
		seq++
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            fmt.Sprintf("%s %d %d", name, time.Now().UnixNano(), seq),
			MinimumQuantity: 1,
			IsActive:        true,
			Status:          StatusDraft,
			StoreID:         storeID,
		}
		require.NoError(t, repo.Create(t.Context(), rule))
		return rule
	}

	// router builds a request context as a manager of claimsStore.
	router := func(claimsStore *int) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 2)
			c.Set("username", "mgr")
			c.Set("roleID", 2)
			c.Set("role", "manager")
			c.Set("permissions", []string{"pricing.view", "pricing.create", "pricing.update", "pricing.delete"})
			c.Set("storeID", claimsStore)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})
		return r
	}

	do := func(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		var req *http.Request
		if body == "" {
			req, _ = http.NewRequest(method, path, nil)
		} else {
			req, _ = http.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		return w
	}

	nameSeq := 0
	updateBody := func(value int, storeID *int) string {
		nameSeq++
		s := fmt.Sprintf(`{"product_id":%d,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":%d,"minimum_quantity":1,"name":"MutSB upd %d %d"`, productID, value, time.Now().UnixNano(), nameSeq)
		if storeID != nil {
			s += fmt.Sprintf(`,"store_id":%d`, *storeID)
		}
		return s + "}"
	}

	t.Run("update: foreign store rule is forbidden", func(t *testing.T) {
		ruleB := mkRule(&storeB, "MutSB foreign update")
		w := do(router(&storeA), "PUT", "/pricing-rules/"+strconv.Itoa(ruleB.ID), updateBody(5555, &storeB))
		assert.Equal(t, http.StatusForbidden, w.Code)
		after, err := repo.GetByID(t.Context(), ruleB.ID)
		require.NoError(t, err)
		assert.InDelta(t, 10000, after.PricingValue, 0.001, "value must be unchanged")
	})

	t.Run("update: global rule is forbidden", func(t *testing.T) {
		global := mkRule(nil, "MutSB global update")
		w := do(router(&storeA), "PUT", "/pricing-rules/"+strconv.Itoa(global.ID), updateBody(5555, nil))
		assert.Equal(t, http.StatusForbidden, w.Code)
		after, err := repo.GetByID(t.Context(), global.ID)
		require.NoError(t, err)
		assert.InDelta(t, 10000, after.PricingValue, 0.001)
	})

	t.Run("update: own store rule succeeds and cannot be moved to store B", func(t *testing.T) {
		ruleA := mkRule(&storeA, "MutSB own update")
		r := router(&storeA)

		w := do(r, "PUT", "/pricing-rules/"+strconv.Itoa(ruleA.ID), updateBody(7777, &storeA))
		assert.Equal(t, http.StatusOK, w.Code)
		after, err := repo.GetByID(t.Context(), ruleA.ID)
		require.NoError(t, err)
		assert.InDelta(t, 7777, after.PricingValue, 0.001)

		// Same rule, same caller, but the body now targets store B: rejected.
		w = do(r, "PUT", "/pricing-rules/"+strconv.Itoa(ruleA.ID), updateBody(8888, &storeB))
		assert.Equal(t, http.StatusForbidden, w.Code)
		after, err = repo.GetByID(t.Context(), ruleA.ID)
		require.NoError(t, err)
		assert.InDelta(t, 7777, after.PricingValue, 0.001)
	})

	t.Run("update: omitting store_id keeps the rule in its own store", func(t *testing.T) {
		ruleA := mkRule(&storeA, "MutSB inherit store")
		w := do(router(&storeA), "PUT", "/pricing-rules/"+strconv.Itoa(ruleA.ID), updateBody(9999, nil))
		assert.Equal(t, http.StatusOK, w.Code)
		after, err := repo.GetByID(t.Context(), ruleA.ID)
		require.NoError(t, err)
		require.NotNil(t, after.StoreID, "an omitted store_id must not clear the scope")
		assert.Equal(t, storeA, *after.StoreID)
		assert.InDelta(t, 9999, after.PricingValue, 0.001)
	})

	t.Run("delete: foreign and global rules are forbidden", func(t *testing.T) {
		ruleB := mkRule(&storeB, "MutSB foreign delete")
		global := mkRule(nil, "MutSB global delete")
		r := router(&storeA)

		assert.Equal(t, http.StatusForbidden, do(r, "DELETE", "/pricing-rules/"+strconv.Itoa(ruleB.ID), "").Code)
		assert.Equal(t, http.StatusForbidden, do(r, "DELETE", "/pricing-rules/"+strconv.Itoa(global.ID), "").Code)

		_, err := repo.GetByID(t.Context(), ruleB.ID)
		assert.NoError(t, err, "foreign rule must survive")
		_, err = repo.GetByID(t.Context(), global.ID)
		assert.NoError(t, err, "global rule must survive")
	})

	t.Run("delete: own store rule succeeds", func(t *testing.T) {
		ruleA := mkRule(&storeA, "MutSB own delete")
		assert.Equal(t, http.StatusOK, do(router(&storeA), "DELETE", "/pricing-rules/"+strconv.Itoa(ruleA.ID), "").Code)
		_, err := repo.GetByID(t.Context(), ruleA.ID)
		assert.Error(t, err)
	})

	t.Run("submit/approve/reject: foreign and global rules are forbidden", func(t *testing.T) {
		ruleB := mkRule(&storeB, "MutSB foreign approve")
		global := mkRule(nil, "MutSB global approve")
		r := router(&storeA)
		for _, id := range []int{ruleB.ID, global.ID} {
			for _, action := range []string{"submit", "approve", "reject"} {
				assert.Equal(t, http.StatusForbidden,
					do(r, "POST", "/pricing-rules/"+strconv.Itoa(id)+"/"+action, "").Code,
					"id=%d action=%s", id, action)
			}
			after, err := repo.GetByID(t.Context(), id)
			require.NoError(t, err)
			assert.Equal(t, StatusDraft, after.Status, "id=%d status must not change", id)
		}
	})

	t.Run("submit/approve/reject: own store rule is allowed", func(t *testing.T) {
		// Store scope only. This still lets one role author, submit, and approve
		// its own rule, which Wave 2d removes by splitting pricing.approve out of
		// pricing.update and blocking self-approval; do not read the 200s below
		// as endorsing that, they only assert the store check passes.
		ruleA := mkRule(&storeA, "MutSB own approve")
		r := router(&storeA)
		p := "/pricing-rules/" + strconv.Itoa(ruleA.ID)
		assert.Equal(t, http.StatusOK, do(r, "POST", p+"/submit", "").Code)
		assert.Equal(t, http.StatusOK, do(r, "POST", p+"/approve", "").Code)
		after, err := repo.GetByID(t.Context(), ruleA.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusApproved, after.Status)
	})

	t.Run("create: store-scoped role is pinned to its own store", func(t *testing.T) {
		// A fresh product per successful create: the service rejects a second rule
		// covering the same product/type/method as a conflict.
		freshProduct := func(tag string) int {
			return insertTestProduct(t.Context(), t, "MUT-CREATE-"+tag+"-"+time.Now().Format("0102150405.000000"), "Create "+tag, 15000)
		}

		// Omitted store_id -> caller's store.
		w := do(router(&storeA), "POST", "/pricing-rules", fmt.Sprintf(
			`{"product_id":%d,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":1234,"minimum_quantity":1,"name":"MutSB create implicit %d"}`, freshProduct("implicit"), time.Now().UnixNano()))
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: status=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Data Rule `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Data.StoreID)
		assert.Equal(t, storeA, *resp.Data.StoreID)

		// Explicit foreign store_id -> rejected outright.
		w = do(router(&storeA), "POST", "/pricing-rules", fmt.Sprintf(
			`{"product_id":%d,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":1234,"minimum_quantity":1,"name":"MutSB create cross %d","store_id":%d}`, freshProduct("cross"), time.Now().UnixNano(), storeB))
		assert.Equal(t, http.StatusForbidden, w.Code)

		// An explicit "store_id": null is indistinguishable from omitting it, so
		// the rule is accepted but silently scoped to the caller's store. The
		// security property under test is that a store-scoped role can never
		// *produce* a global (store_id IS NULL) rule.
		w = do(router(&storeA), "POST", "/pricing-rules", fmt.Sprintf(
			`{"product_id":%d,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":1234,"minimum_quantity":1,"name":"MutSB create null %d","store_id":null}`, freshProduct("null"), time.Now().UnixNano()))
		if w.Code != http.StatusCreated {
			t.Fatalf("null-store create failed: status=%d body=%s", w.Code, w.Body.String())
		}
		var nullResp struct {
			Data Rule `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &nullResp))
		require.NotNil(t, nullResp.Data.StoreID, "a store-scoped role must not create a global rule")
		assert.Equal(t, storeA, *nullResp.Data.StoreID)
	})

	t.Run("create: superadmin may create a global rule", func(t *testing.T) {
		superProduct := insertTestProduct(t.Context(), t, "MUT-CREATE-super-"+time.Now().Format("0102150405.000000"), "Create super", 15000)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("username", "root")
			c.Set("roleID", 1)
			c.Set("role", "superadmin")
			c.Set("permissions", []string{"pricing.view", "pricing.create"})
			c.Set("storeID", nil)
			c.Next()
		})
		h := NewHandler(NewService(repo), nil, nil)
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pricing-rules", strings.NewReader(fmt.Sprintf(
			`{"product_id":%d,"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":1234,"minimum_quantity":1,"name":"MutSB superadmin global %d"}`, superProduct, time.Now().UnixNano())))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("superadmin global create failed: status=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Data Rule `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Nil(t, resp.Data.StoreID, "superadmin must still be able to create a global rule")
	})
}
