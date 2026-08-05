package shift

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"
)

func setupShiftHandlerNoUser(svc ShiftService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "cashier")
		c.Set("storeID", nil)
		c.Set("permissions", []string{})
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestShiftHandler_InvalidUserBranches(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandlerNoUser(svc)

	t.Run("open shift", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/shifts/open", strings.NewReader(`{"opening_balance":100000}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("close shift", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/shifts/1/close", strings.NewReader(`{"closing_balance":200000}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("close all", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/shifts/close-all", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("get active shift", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts/active", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestShiftHandler_ExportShifts_XLSX(t *testing.T) {
	auditCalled := false
	disc := 10000
	closed := "2026-01-01T00:00:00+07:00"
	svc := &mockShiftService{
		exportShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
			assert.Equal(t, "closed", status)
			assert.NotNil(t, needsReview)
			assert.True(t, *needsReview)
			assert.Equal(t, "surplus", discrepancyFilter)
			return []Shift{
				{ID: 1, Username: "cashier1", StoreName: "Store A", Status: "closed", OpeningBalance: 100000, CashSales: 120000, NonCashSales: 30000, TotalSales: 150000, TransactionCount: 4, Discrepancy: &disc, NeedsReview: true, OpenedAt: "2026-01-01T08:00:00+07:00", ClosedAt: closed},
				{ID: 2, Username: "cashier2", StoreName: "Store B", Status: "closed", OpeningBalance: 50000, CashSales: 40000, TotalSales: 40000, TransactionCount: 1, NeedsReview: false, OpenedAt: "2026-01-01T09:00:00+07:00", ClosedAt: closed},
			}, nil
		},
	}
	auditSvc := &mockAudit{
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
			auditCalled = true
			assert.Equal(t, "export", log.Action)
			assert.Equal(t, "shift", log.EntityType)
			assert.Equal(t, "Exported 2 shifts as xlsx", log.Description)
			return nil
		},
	}
	r := setupShiftHandler(svc, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/export?format=xlsx&status=closed&needs_review=true&discrepancy=surplus", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
	assert.NotEmpty(t, w.Body.Bytes())
	assert.True(t, auditCalled, "audit log should be created")
}

func TestShiftHandler_ExportShifts_CSVFullData(t *testing.T) {
	disc := -5000
	closed := "2026-01-01T00:00:00+07:00"
	svc := &mockShiftService{
		exportShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
			return []Shift{
				{ID: 1, Username: "cashier1", StoreName: "Store A", Status: "closed", OpeningBalance: 100000, CashSales: 95000, TotalSales: 95000, TransactionCount: 1, Discrepancy: &disc, OpenedAt: "2026-01-01T08:00:00+07:00", ClosedAt: closed},
			}, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/export?discrepancy=shortage", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	body := w.Body.String()
	assert.Contains(t, body, "Store A")
	assert.Contains(t, body, "-5000")
	assert.Contains(t, body, closed)
}

func TestShiftHandler_ExportShifts_NilShifts(t *testing.T) {
	svc := &mockShiftService{
		exportShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
			return nil, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/export", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
}

func TestShiftHandler_AuditShift_InvalidID(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/abc/audit", strings.NewReader(`{"actual_balance":160000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid shift id")
}

func TestShiftHandler_AuditShift_InvalidJSON(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}
