package brand

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockAuditCreator struct {
	createAuditLogFn func(ctx context.Context, log *audit.AuditLog) error
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.AuditLog) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupMockBrandRouterWithAudit(svc BrandService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	auditSvc := &mockAuditCreator{}
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPublicRoutes(r.Group("/"))
	return r
}

func TestAuditHandler_CreateBrand(t *testing.T) {
	svc := &mockBrandService{
		createFn: func(ctx context.Context, req *BrandCreateRequest) (*Brand, error) {
			return &Brand{ID: 10, Name: req.Name}, nil
		},
	}
	r := setupMockBrandRouterWithAudit(svc)
	body := `{"name":"Audit Brand","description":"test","is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/brands", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuditHandler_UpdateBrand(t *testing.T) {
	svc := &mockBrandService{
		getByIDFn: func(ctx context.Context, id int) (*Brand, error) {
			return &Brand{ID: 1, Name: "Old Brand"}, nil
		},
		updateFn: func(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
			return &Brand{ID: id, Name: req.Name}, nil
		},
	}
	r := setupMockBrandRouterWithAudit(svc)
	body := `{"name":"Updated Brand","is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/brands/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteBrand(t *testing.T) {
	svc := &mockBrandService{
		getByIDFn: func(ctx context.Context, id int) (*Brand, error) {
			return &Brand{ID: 1, Name: "ToDelete"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error { return nil },
	}
	r := setupMockBrandRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/brands/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteBrand_GetByIDError(t *testing.T) {
	svc := &mockBrandService{
		getByIDFn: func(ctx context.Context, id int) (*Brand, error) {
			return nil, errors.New("not found")
		},
		deleteFn: func(ctx context.Context, id int) error { return nil },
	}
	r := setupMockBrandRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/brands/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateBrand_GetByIDError(t *testing.T) {
	svc := &mockBrandService{
		getByIDFn: func(ctx context.Context, id int) (*Brand, error) {
			return nil, errors.New("not found")
		},
		updateFn: func(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
			return &Brand{ID: id, Name: req.Name}, nil
		},
	}
	r := setupMockBrandRouterWithAudit(svc)
	body := `{"name":"Updated"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/brands/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_NilAuditSvc(t *testing.T) {
	svc := &mockBrandService{
		createFn: func(ctx context.Context, req *BrandCreateRequest) (*Brand, error) {
			return &Brand{ID: 20, Name: req.Name}, nil
		},
	}
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

	body := `{"name":"Nil Audit","description":"no audit","is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/brands", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}
