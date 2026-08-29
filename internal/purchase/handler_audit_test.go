package purchase

import (
	"context"
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

func TestAuditHandler_ConfirmPO_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockPurchaseService{}
	auditSvc := &mockAuditCreator{createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
		captured = log
		return nil
	}}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Set("storeID", intPtr(1))
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/api"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/purchase-orders/1/confirm", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for a confirmed purchase order")
	assert.Equal(t, "purchase_order_confirmed", captured.Action)
	assert.Equal(t, "purchase_order", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 1, *captured.EntityID)
}

func TestAuditHandler_CancelPO_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockPurchaseService{}
	auditSvc := &mockAuditCreator{createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
		captured = log
		return nil
	}}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Set("storeID", intPtr(1))
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/api"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/purchase-orders/1/cancel", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for a cancelled purchase order")
	assert.Equal(t, "purchase_order_cancelled", captured.Action)
	assert.Equal(t, "purchase_order", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 1, *captured.EntityID)
}
