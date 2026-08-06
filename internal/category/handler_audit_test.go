package category

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
	createAuditLogFn func(ctx context.Context, log *audit.Log) error
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.Log) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupMockRouterWithAudit(svc Service) *gin.Engine {
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
	return r
}

func TestAuditHandler_CreateCategory(t *testing.T) {
	svc := &mockCategoryService{
		createFn: func(ctx context.Context, req *CreateRequest) (*Category, error) {
			return &Category{ID: 42, Name: req.Name}, nil
		},
	}
	r := setupMockRouterWithAudit(svc)
	body := `{"name":"New Category","description":"Test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/categories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuditHandler_UpdateCategory(t *testing.T) {
	svc := &mockCategoryService{
		getByIDFn: func(ctx context.Context, id int) (*Category, error) {
			return &Category{ID: 1, Name: "Old"}, nil
		},
		updateFn: func(ctx context.Context, id int, req *UpdateRequest) (*Category, error) {
			return &Category{ID: id, Name: req.Name}, nil
		},
	}
	r := setupMockRouterWithAudit(svc)
	body := `{"name":"Updated Category"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/categories/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteCategory(t *testing.T) {
	svc := &mockCategoryService{
		getByIDFn: func(ctx context.Context, id int) (*Category, error) {
			return &Category{ID: 1, Name: "ToDelete"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	r := setupMockRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/categories/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteCategory_GetByIDError(t *testing.T) {
	svc := &mockCategoryService{
		getByIDFn: func(ctx context.Context, id int) (*Category, error) {
			return nil, errors.New("not found")
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	r := setupMockRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/categories/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateCategory_GetByIDError(t *testing.T) {
	svc := &mockCategoryService{
		getByIDFn: func(ctx context.Context, id int) (*Category, error) {
			return nil, errors.New("not found")
		},
		updateFn: func(ctx context.Context, id int, req *UpdateRequest) (*Category, error) {
			return &Category{ID: id, Name: req.Name}, nil
		},
	}
	r := setupMockRouterWithAudit(svc)
	body := `{"name":"Updated Category"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/categories/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
