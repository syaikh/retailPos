package inventory

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

func setupMockInventoryRouterWithAudit(svc InventoryService) *gin.Engine {
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
	return r
}

func TestAuditHandler_AdjustStock(t *testing.T) {
	svc := &mockInventoryService{
		adjustStockFn: func(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
			return nil
		},
	}
	r := setupMockInventoryRouterWithAudit(svc)
	body := `{"product_id":42,"quantity_change":10,"notes":"restock"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_AdjustStock_ServiceError(t *testing.T) {
	svc := &mockInventoryService{
		adjustStockFn: func(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
			return errors.New("product not found")
		},
	}
	r := setupMockInventoryRouterWithAudit(svc)
	body := `{"product_id":999,"quantity_change":5,"notes":"test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_AdjustStock_BindError(t *testing.T) {
	r := setupMockInventoryRouterWithAudit(&mockInventoryService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_AdjustStock_MissingNotes(t *testing.T) {
	r := setupMockInventoryRouterWithAudit(&mockInventoryService{})
	body := `{"product_id":1,"quantity_change":5,"notes":""}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_AdjustStock_ZeroQuantity(t *testing.T) {
	r := setupMockInventoryRouterWithAudit(&mockInventoryService{})
	body := `{"product_id":1,"quantity_change":0,"notes":"test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
