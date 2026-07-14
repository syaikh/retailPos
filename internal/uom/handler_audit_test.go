package uom

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

func setupMockUOMRouterWithAudit(svc UOMService) *gin.Engine {
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
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	h.RegisterPublicRoutes(r.Group("/"))
	return r
}

func TestAuditHandler_CreateUOM(t *testing.T) {
	svc := &mockUOMService{
		createFn: func(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error) {
			return &UnitOfMeasure{ID: 42, Name: req.Name, Code: req.Code}, nil
		},
	}
	r := setupMockUOMRouterWithAudit(svc)
	body := `{"name":"Kilogram","code":"KG"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/units-of-measure", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuditHandler_UpdateUOM(t *testing.T) {
	svc := &mockUOMService{
		getByIDFn: func(ctx context.Context, id int) (*UnitOfMeasure, error) {
			return &UnitOfMeasure{ID: 1, Name: "Old", Code: "OLD"}, nil
		},
		updateFn: func(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
			return &UnitOfMeasure{ID: id, Name: req.Name, Code: req.Code}, nil
		},
	}
	r := setupMockUOMRouterWithAudit(svc)
	body := `{"name":"Gram","code":"GR"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/units-of-measure/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteUOM(t *testing.T) {
	svc := &mockUOMService{
		getByIDFn: func(ctx context.Context, id int) (*UnitOfMeasure, error) {
			return &UnitOfMeasure{ID: 1, Name: "ToDelete", Code: "DEL"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	r := setupMockUOMRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/units-of-measure/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteUOM_GetByIDError(t *testing.T) {
	svc := &mockUOMService{
		getByIDFn: func(ctx context.Context, id int) (*UnitOfMeasure, error) {
			return nil, errors.New("not found")
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	r := setupMockUOMRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/units-of-measure/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateUOM_GetByIDError(t *testing.T) {
	svc := &mockUOMService{
		getByIDFn: func(ctx context.Context, id int) (*UnitOfMeasure, error) {
			return nil, errors.New("not found")
		},
		updateFn: func(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
			return &UnitOfMeasure{ID: id, Name: req.Name, Code: req.Code}, nil
		},
	}
	r := setupMockUOMRouterWithAudit(svc)
	body := `{"name":"Gram","code":"GR"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/units-of-measure/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
