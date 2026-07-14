package product

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockHandler_GetProducts_StatusParam(t *testing.T) {
	t.Run("status=active", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Equal(t, "active", status)
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?status=active", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("status overrides isActive", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Equal(t, "inactive", status)
				assert.Nil(t, isActive, "isActive should be nil when status is set")
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?status=inactive&isActive=true", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("isActive=false", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				require.NotNil(t, isActive)
				assert.False(t, *isActive)
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?isActive=false", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("isActive=1", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				require.NotNil(t, isActive)
				assert.True(t, *isActive)
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?isActive=1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("custom limit and offset", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Equal(t, 25, limit)
				assert.Equal(t, 50, offset)
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?limit=25&offset=50", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid limit ignored", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Equal(t, 50, limit, "invalid limit should default to 50")
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?limit=abc", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("limit exceeds max ignored", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Equal(t, 50, limit, "limit >200 should default to 50")
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?limit=300", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid offset ignored", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Equal(t, 0, offset, "invalid offset should default to 0")
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?offset=-5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid maxStock ignored", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Nil(t, maxStock, "invalid maxStock should be nil")
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?maxStock=abc", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative maxStock ignored", func(t *testing.T) {
		svc := &mockProductService{
			getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
				assert.Nil(t, maxStock, "negative maxStock should be nil")
				return []Product{}, 0, nil
			},
		}
		r := setupMockProductRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/products?maxStock=-1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMockHandler_GetProductByID_WithStoreID(t *testing.T) {
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			assert.Equal(t, 5, id)
			assert.Equal(t, 7, storeID, "storeID from context should be 7")
			return &Product{ID: 5, Name: "Widget"}, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		// Set storeID as *int (like the real middleware does)
		sid := 7
		c.Set("storeID", &sid)
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products/5", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_GetProductByID_WithInvalidStoreID(t *testing.T) {
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			assert.Equal(t, 0, storeID, "invalid storeID type => 0")
			return &Product{ID: 5}, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", "not-an-int-ptr") // wrong type
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products/5", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_UpdateProduct_WithStoreID(t *testing.T) {
	svc := &mockProductService{
		updateFn: func(ctx context.Context, product *Product) error {
			assert.NotNil(t, product.StoreID, "storeID should be set")
			assert.Equal(t, 7, *product.StoreID)
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		sid := 7
		c.Set("storeID", &sid)
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/products/5", strings.NewReader(`{"name":"Updated","price":2000,"cost":1000,"stock":20}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_DeleteProduct_WithStoreID(t *testing.T) {
	svc := &mockProductService{
		deleteFn: func(ctx context.Context, id int, storeID *int) error {
			assert.NotNil(t, storeID)
			assert.Equal(t, 7, *storeID)
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		sid := 7
		c.Set("storeID", &sid)
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/products/5", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_BulkUpdateStatus_WithStoreID(t *testing.T) {
	svc := &mockProductService{
		bulkStatusFn: func(ctx context.Context, ids []int, isActive bool, storeID *int) error {
			assert.NotNil(t, storeID)
			assert.Equal(t, 3, *storeID)
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 3
		c.Set("storeID", &sid)
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/products/bulk/status", strings.NewReader(`{"ids":[1,2],"is_active":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_GetProducts_WithStoreID(t *testing.T) {
	svc := &mockProductService{
		getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
			assert.NotNil(t, storeID)
			assert.Equal(t, 5, *storeID)
			return []Product{}, 0, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 5
		c.Set("storeID", &sid)
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_GetProducts_WithInvalidStoreIDType(t *testing.T) {
	svc := &mockProductService{
		getAllFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string) ([]Product, int, error) {
			assert.Nil(t, storeID, "invalid storeID type should result in nil")
			return []Product{}, 0, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", "bad-type")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMockHandler_GetProductByID_NotFound(t *testing.T) {
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			return nil, errors.New("product not found")
		},
	}
	r := setupMockProductRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/products/999", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMockHandler_UpdateProduct_NotFound(t *testing.T) {
	svc := &mockProductService{
		updateFn: func(ctx context.Context, product *Product) error {
			return errors.New("product not found")
		},
	}
	r := setupMockProductRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/products/999", strings.NewReader(`{"name":"X","price":100,"cost":50,"stock":10}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
