package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProductService struct {
	getAllFn            func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error)
	getByIDsFn          func(ctx context.Context, ids []int) ([]Product, error)
	getByIDFn           func(ctx context.Context, id, storeID int) (*Product, error)
	createFn            func(ctx context.Context, product *Product) error
	updateFn            func(ctx context.Context, product *Product) error
	deleteFn            func(ctx context.Context, id int, storeID *int) error
	bulkStatusFn        func(ctx context.Context, ids []int, isActive bool, storeID *int) error
	nextSKUFn           func(ctx context.Context) (string, error)
	getTaxClassesFn     func(ctx context.Context) ([]TaxClass, error)
	getWarehousesFn     func(ctx context.Context) ([]Warehouse, error)
	getProductOptionsFn func(ctx context.Context) ([]ProductOption, error)
}

func (m *mockProductService) GetAllProducts(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
	return m.getAllFn(ctx, limit, offset, search, sortBy, sortDir, category, storeID, isActive, maxStock, status, supplierID)
}
func (m *mockProductService) GetProductsByIDs(ctx context.Context, ids []int) ([]Product, error) {
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, ids)
	}
	return []Product{}, nil
}
func (m *mockProductService) GetProductByID(ctx context.Context, id, storeID int) (*Product, error) {
	return m.getByIDFn(ctx, id, storeID)
}
func (m *mockProductService) CreateProduct(ctx context.Context, product *Product) error {
	return m.createFn(ctx, product)
}
func (m *mockProductService) UpdateProduct(ctx context.Context, product *Product) error {
	return m.updateFn(ctx, product)
}
func (m *mockProductService) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	return m.deleteFn(ctx, id, storeID)
}
func (m *mockProductService) BulkUpdateProductStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	return m.bulkStatusFn(ctx, ids, isActive, storeID)
}
func (m *mockProductService) GetNextSKU(ctx context.Context) (string, error) {
	return m.nextSKUFn(ctx)
}
func (m *mockProductService) GetAllTaxClasses(ctx context.Context) ([]TaxClass, error) {
	return m.getTaxClassesFn(ctx)
}
func (m *mockProductService) GetAllWarehouses(ctx context.Context) ([]Warehouse, error) {
	if m.getWarehousesFn != nil {
		return m.getWarehousesFn(ctx)
	}
	return []Warehouse{}, nil
}
func (m *mockProductService) GetActiveProductOptions(ctx context.Context) ([]ProductOption, error) {
	if m.getProductOptionsFn != nil {
		return m.getProductOptionsFn(ctx)
	}
	return []ProductOption{}, nil
}

var _ ProductService = (*mockProductService)(nil)

func setupMockProductRouter(svc ProductService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPublicRoutes(r.Group("/"))
	return r
}

func TestParseIDs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []int
	}{
		{"empty string", "", nil},
		{"single id", "1", []int{1}},
		{"multiple ids", "1,2,3", []int{1, 2, 3}},
		{"with spaces", " 1 , 2 , 3 ", []int{1, 2, 3}},
		{"deduplication", "1,2,1,3,2", []int{1, 2, 3}},
		{"zero filtered out", "0,1,2", []int{1, 2}},
		{"negative filtered out", "-1,1,-2,2", []int{1, 2}},
		{"non-numeric filtered out", "abc,1,xyz", []int{1}},
		{"empty segments", "1,,2,,3", []int{1, 2, 3}},
		{"all invalid", "abc,xyz", nil},
		{"trailing comma", "1,2,", []int{1, 2}},
		{"single zero", "0", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIDs(tt.raw)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMockHandler_GetProducts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
				assert.Equal(t, 20, limit)
				return []Product{{ID: 1, Name: "Widget"}}, 1, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("minPrice rejection", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?minPrice=100", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "not yet implemented")
	})

	t.Run("maxPrice rejection", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?maxPrice=500", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("brand rejection", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?brand=Nike", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("isActive param", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
				require.NotNil(t, isActive)
				assert.True(t, *isActive)
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?isActive=true", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nil products become empty array", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
				return nil, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("maxStock param", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
				require.NotNil(t, maxStock)
				assert.Equal(t, 10, *maxStock)
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?maxStock=10", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetProducts_ByIDs(t *testing.T) {
	t.Run("valid ids batch", func(t *testing.T) {
		svc := &mockProductService{
			getByIDsFn: func(ctx context.Context, ids []int) ([]Product, error) {
				assert.Equal(t, []int{1, 2, 3}, ids)
				return []Product{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}, {ID: 3, Name: "C"}}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=1,2,3", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(3), resp["total"])
	})

	t.Run("all invalid ids returns empty", func(t *testing.T) {
		svc := &mockProductService{}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=abc,xyz", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(0), resp["total"])
		assert.Equal(t, "[]", fmt.Sprint(resp["data"]))
	})

	t.Run("deduplicated ids", func(t *testing.T) {
		svc := &mockProductService{
			getByIDsFn: func(ctx context.Context, ids []int) ([]Product, error) {
				assert.Equal(t, []int{5}, ids, "should deduplicate")
				return []Product{{ID: 5, Name: "Only"}}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=5,5,5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockProductService{
			getByIDsFn: func(ctx context.Context, ids []int) ([]Product, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=1,2", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("nil products become empty array", func(t *testing.T) {
		svc := &mockProductService{
			getByIDsFn: func(ctx context.Context, ids []int) ([]Product, error) {
				return nil, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("empty ids param falls through to normal listing", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error) {
				return []Product{{ID: 1}}, 1, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ids with negative and zero are filtered out", func(t *testing.T) {
		svc := &mockProductService{
			getByIDsFn: func(ctx context.Context, ids []int) ([]Product, error) {
				assert.Equal(t, []int{3, 7}, ids)
				return []Product{}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?ids=-1,0,3,7", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMockHandler_GetProductByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
				assert.Equal(t, 5, id)
				return &Product{ID: 5, Name: "Widget"}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products/5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockProductService{
			getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products/99", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMockHandler_CreateProduct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			createFn: func(ctx context.Context, product *Product) error {
				product.ID = 100
				return nil
			},
		}
		r := setupMockProductRouter(svc)
		body := `{"name":"New Product","price":1000,"cost":500,"stock":10}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty name", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		body := `{"name":"","price":1000}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "name is required")
	})

	t.Run("negative price", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		body := `{"name":"Product","price":-1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "price must not be negative")
	})

	t.Run("negative cost", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		body := `{"name":"Product","price":100,"cost":-1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "cost must not be negative")
	})

	t.Run("negative stock", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		body := `{"name":"Product","price":100,"stock":-1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "stock must not be negative")
	})
}

func TestMockHandler_UpdateProduct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			updateFn: func(ctx context.Context, product *Product) error {
				assert.Equal(t, 5, product.ID)
				return nil
			},
		}
		r := setupMockProductRouter(svc)
		body := `{"name":"Updated","price":2000,"cost":1000,"stock":20}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/products/5", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/products/abc", strings.NewReader(`{"name":"X","price":100}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/products/5", strings.NewReader(`{"name":"","price":100}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestMockHandler_DeleteProduct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			deleteFn: func(ctx context.Context, id int, storeID *int) error { return nil },
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/products/5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/products/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockProductService{
			deleteFn: func(ctx context.Context, id int, storeID *int) error { return errors.New("fail") },
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/products/5", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_BulkUpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			bulkStatusFn: func(ctx context.Context, ids []int, isActive bool, storeID *int) error {
				assert.Equal(t, []int{1, 2}, ids)
				assert.True(t, isActive)
				return nil
			},
		}
		r := setupMockProductRouter(svc)
		body := `{"ids":[1,2],"is_active":true}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products/bulk/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ids", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products/bulk/status", strings.NewReader(`{"ids":[],"is_active":true}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "no product IDs")
	})

	t.Run("too many ids", func(t *testing.T) {
		r := setupMockProductRouter(&mockProductService{})
		ids := strings.Repeat("1,", 200) + "1"
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/products/bulk/status", strings.NewReader(`{"ids":[`+ids+`],"is_active":true}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "too many")
	})
}

func TestMockHandler_GetNextSKU(t *testing.T) {
	svc := &mockProductService{
		nextSKUFn: func(ctx context.Context) (string, error) {
			return "PRD-00001", nil
		},
	}
	r := setupMockProductRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products/next-sku", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "PRD-00001", resp["data"])
}

func TestMockHandler_ListTaxClasses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			getTaxClassesFn: func(ctx context.Context) ([]TaxClass, error) {
				return []TaxClass{{ID: 1, Name: "PPN"}}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/tax-classes", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nil becomes empty array", func(t *testing.T) {
		svc := &mockProductService{
			getTaxClassesFn: func(ctx context.Context) ([]TaxClass, error) {
				return nil, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/tax-classes", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockProductService{
			getTaxClassesFn: func(ctx context.Context) ([]TaxClass, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/tax-classes", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetStockThresholds(t *testing.T) {
	r := setupMockProductRouter(&mockProductService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/stock-thresholds", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "warning")
	assert.Contains(t, w.Body.String(), "critical")
}

func TestMockHandler_ListWarehouses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			getWarehousesFn: func(ctx context.Context) ([]Warehouse, error) {
				return []Warehouse{{ID: 1, Name: "Main WH", Code: "WH-01"}}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/warehouses", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Main WH")
	})

	t.Run("nil becomes empty array", func(t *testing.T) {
		svc := &mockProductService{
			getWarehousesFn: func(ctx context.Context) ([]Warehouse, error) {
				return nil, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/warehouses", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockProductService{
			getWarehousesFn: func(ctx context.Context) ([]Warehouse, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/warehouses", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_ListProductOptions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockProductService{
			getProductOptionsFn: func(ctx context.Context) ([]ProductOption, error) {
				return []ProductOption{{ID: 1, SKU: "SKU-1", Name: "Apple"}}, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products/options", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Apple")
	})

	t.Run("nil becomes empty array", func(t *testing.T) {
		svc := &mockProductService{
			getProductOptionsFn: func(ctx context.Context) ([]ProductOption, error) {
				return nil, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products/options", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]")
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockProductService{
			getProductOptionsFn: func(ctx context.Context) ([]ProductOption, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products/options", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetNextSKU_ServiceError(t *testing.T) {
	svc := &mockProductService{
		nextSKUFn: func(ctx context.Context) (string, error) {
			return "", errors.New("gen error")
		},
	}
	r := setupMockProductRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products/next-sku", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMockHandler_BulkUpdateStatus_ServiceError(t *testing.T) {
	svc := &mockProductService{
		bulkStatusFn: func(ctx context.Context, ids []int, isActive bool, storeID *int) error {
			return errors.New("db error")
		},
	}
	r := setupMockProductRouter(svc)
	body := `{"ids":[1,2],"is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/products/bulk/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMockHandler_CreateProduct_ServiceError(t *testing.T) {
	svc := &mockProductService{
		createFn: func(ctx context.Context, product *Product) error {
			return errors.New("duplicate sku")
		},
	}
	r := setupMockProductRouter(svc)
	body := `{"name":"New Product","price":1000,"cost":500,"stock":10}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMockHandler_UpdateProduct_ServiceError(t *testing.T) {
	svc := &mockProductService{
		updateFn: func(ctx context.Context, product *Product) error {
			return errors.New("update failed")
		},
	}
	r := setupMockProductRouter(svc)
	body := `{"name":"Updated","price":2000,"cost":1000,"stock":20}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/products/5", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMockHandler_UpdateProduct_InvalidJSON(t *testing.T) {
	r := setupMockProductRouter(&mockProductService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/products/5", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
