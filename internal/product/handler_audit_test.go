package product

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/audit"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockProductRouterWithAudit(svc ProductService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	var auditSvc *audit.Service
	if dbPool != nil {
		auditSvc = audit.NewService(audit.NewRepository(dbPool))
	}
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPublicRoutes(r.Group("/"))
	return r
}

func TestAuditHandler_CreateProduct(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
	svc := &mockProductService{
		createFn: func(ctx context.Context, product *Product) error {
			product.ID = 42
			return nil
		},
	}
	r := setupMockProductRouterWithAudit(svc)
	body := `{"name":"Widget","sku":"WDG-001","price":10000,"cost":5000,"status":"active","category_id":1,"brand_id":1,"unit_of_measure_id":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuditHandler_UpdateProduct(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			return &Product{ID: 1, Name: "Old Widget", Price: 5000}, nil
		},
		updateFn: func(ctx context.Context, product *Product) error {
			return nil
		},
	}
	r := setupMockProductRouterWithAudit(svc)
	body := `{"name":"New Widget","sku":"WDG-001","price":10000,"cost":5000,"status":"active","category_id":1,"brand_id":1,"unit_of_measure_id":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/products/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteProduct(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			return &Product{ID: 1, Name: "Widget"}, nil
		},
		deleteFn: func(ctx context.Context, id int, storeID *int) error {
			return nil
		},
	}
	r := setupMockProductRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/products/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteProduct_OldProductNotFound(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			return nil, errors.New("not found")
		},
		deleteFn: func(ctx context.Context, id int, storeID *int) error {
			return nil
		},
	}
	r := setupMockProductRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/products/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateProduct_OldProductNotFound(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
	svc := &mockProductService{
		getByIDFn: func(ctx context.Context, id, storeID int) (*Product, error) {
			return nil, errors.New("not found")
		},
		updateFn: func(ctx context.Context, product *Product) error {
			return nil
		},
	}
	r := setupMockProductRouterWithAudit(svc)
	body := `{"name":"New Widget","sku":"WDG-001","price":10000,"cost":5000,"status":"active","category_id":1,"brand_id":1,"unit_of_measure_id":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/products/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
