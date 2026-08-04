package customer

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

func setupMockCustomerRouterWithAudit(svc CustomerService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Set("storeID", nil)
		c.Next()
	})
	auditSvc := &mockAuditCreator{}
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestAuditHandler_CreateCustomer(t *testing.T) {
	svc := &mockCustomerService{
		createFn: func(ctx context.Context, customer *Customer, storeID *int) error {
			customer.ID = 42
			return nil
		},
	}
	r := setupMockCustomerRouterWithAudit(svc)
	body := `{"name":"Test Customer","phone":"0812345678","email":"test@test.com"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuditHandler_UpdateCustomer(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return &Customer{ID: 1, Name: "Old Customer"}, nil
		},
		updateFn: func(ctx context.Context, customer *Customer, id int, storeID *int) error {
			return nil
		},
	}
	r := setupMockCustomerRouterWithAudit(svc)
	body := `{"name":"Updated Customer","phone":"0812345678"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteCustomer(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return &Customer{ID: 1, Name: "ToDelete"}, nil
		},
		deleteFn: func(ctx context.Context, id int, storeID *int) error { return nil },
	}
	r := setupMockCustomerRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteCustomer_GetByIDError(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return nil, errors.New("not found")
		},
		deleteFn: func(ctx context.Context, id int, storeID *int) error { return nil },
	}
	r := setupMockCustomerRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/customers/1", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuditHandler_UpdateCustomer_GetByIDError(t *testing.T) {
	svc := &mockCustomerService{
		getByIDFn: func(ctx context.Context, id int, storeID *int) (*Customer, error) {
			return nil, errors.New("not found")
		},
		updateFn: func(ctx context.Context, customer *Customer, id int, storeID *int) error {
			return nil
		},
	}
	r := setupMockCustomerRouterWithAudit(svc)
	body := `{"name":"Updated Customer","phone":"0812345678","email":"test@test.com"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/customers/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
