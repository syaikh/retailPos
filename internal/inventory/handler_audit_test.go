package inventory

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
	"github.com/stretchr/testify/require"
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

func setupMockInventoryRouterWithAudit(svc Service) *gin.Engine {
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

func TestAuditHandler_AdjustStock(t *testing.T) {
	svc := &mockService{
		adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
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
	svc := &mockService{
		adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
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
	r := setupMockInventoryRouterWithAudit(&mockService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_AdjustStock_MissingNotes(t *testing.T) {
	r := setupMockInventoryRouterWithAudit(&mockService{})
	body := `{"product_id":1,"quantity_change":5,"notes":""}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_AdjustStock_ZeroQuantity(t *testing.T) {
	r := setupMockInventoryRouterWithAudit(&mockService{})
	body := `{"product_id":1,"quantity_change":0,"notes":"test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_AdjustStock_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockService{
		adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
		captured = log
		return nil
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"product_id":42,"quantity_change":10,"notes":"restock cycle"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for a stock adjustment")
	assert.Equal(t, "inventory_adjustment", captured.Action)
	assert.Equal(t, "inventory", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 42, *captured.EntityID)
	assert.Contains(t, captured.Description, "restock cycle")
}

func TestAuditHandler_TransferLocationStock_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockService{
		transferLocationStockFn: func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
		captured = log
		return nil
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"product_id":7,"from_location_id":1,"to_location_id":2,"quantity":5}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for a stock transfer")
	assert.Equal(t, "inventory_transfer", captured.Action)
	assert.Equal(t, "inventory", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 7, *captured.EntityID)
}
