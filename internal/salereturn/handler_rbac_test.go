package salereturn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

// returnRBACAuthMiddleware authenticates the caller and grants exactly the
// given permissions, mirroring the real JWT auth middleware (gin keys + request
// context claims). Local copy of the RBAC test template in
// internal/consignment/handler_rbac_test.go — each package's tests define their
// own since the helper is unexported.
func returnRBACAuthMiddleware(perms []string, storeID *int, uid int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", uid)
		c.Set("username", "salereturn_rbac_user")
		c.Set("role", "rbac-tester")
		c.Set("permissions", perms)
		c.Set("storeID", storeID)

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, uid)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "salereturn_rbac_user")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "rbac-tester")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, storeID)
		ctx = context.WithValue(ctx, middleware.CtxKeyPermissions, perms)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// setupReturnRBACRouter wires the real routes with the real permission
// middleware, so a request is gated exactly as in production. Every port is
// stubbed (as the composition root wires implementations): the SaleReader stub
// reports "no such sale" and the repository reads see an empty table, so an
// authorized request falls through to a deterministic 404/200 instead of a
// happy path that would depend on seeded rows.
func setupReturnRBACRouter(t *testing.T, perms []string, storeID *int, uid int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	s := NewService(NewRepository(dbPool), &stubBus{})
	s.SetSaleReader(stubSaleReader{sale: nil})
	s.SetActiveShiftFinder(stubShiftFinder{shiftID: 0})
	s.SetShiftTotalAdjuster(&stubAdjuster{})
	s.SetStockChecker(stubStock{})
	s.SetStockDeducer(&stubDeduct{})
	s.SetConsignmentGuard(stubConsignment{})
	s.SetSettingsProvider(stubSettings{})

	r := gin.New()
	NewHandler(s).RegisterRoutes(r.Group("/api"), returnRBACAuthMiddleware(perms, storeID, uid), middleware.RequirePermission)
	return r
}

func doReturnRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

// TestHandler_SalesReturn_Authorization pins each route's permission gate to
// its own code (PRD §9 / 067_sales_return.sql): sale.return creates, sale.view
// reads, sale.return.approve works the queue. Denied callers get 403; callers
// who pass the gate fall through to a deterministic status (404 for a missing
// sale/return, 200 for the reads) proving the gate — not the body — decided.
func TestHandler_SalesReturn_Authorization(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	storeID := 1
	const missingSale = "/api/sales/999999/return"
	const validProcessBody = `{"reason":"barang rusak","condition":"damaged","items":[{"sale_item_id":1,"quantity":1}]}`

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		perms  []string
		want   int
	}{
		// sale.return creates a return.
		{"sale.return can process a return", http.MethodPost, missingSale, validProcessBody,
			[]string{string(permissions.SaleReturn)}, http.StatusNotFound},
		{"sale.view alone cannot process a return", http.MethodPost, missingSale, validProcessBody,
			[]string{string(permissions.SaleView)}, http.StatusForbidden},
		{"unrelated permission cannot process a return", http.MethodPost, missingSale, validProcessBody,
			[]string{"report.view"}, http.StatusForbidden},
		{"no permissions cannot process a return", http.MethodPost, missingSale, validProcessBody,
			nil, http.StatusForbidden},
		{"a malformed sale id falls through to 400", http.MethodPost, "/api/sales/abc/return", validProcessBody,
			[]string{string(permissions.SaleReturn)}, http.StatusBadRequest},

		// sale.view reads.
		{"sale.view can list a sale's returns", http.MethodGet, "/api/sales/999999/returns", "",
			[]string{string(permissions.SaleView)}, http.StatusOK},
		{"sale.return alone cannot list a sale's returns", http.MethodGet, "/api/sales/999999/returns", "",
			[]string{string(permissions.SaleReturn)}, http.StatusForbidden},
		{"sale.view can read a return detail", http.MethodGet, "/api/sales/returns/999999", "",
			[]string{string(permissions.SaleView)}, http.StatusNotFound},
		{"sale.return alone cannot read a return detail", http.MethodGet, "/api/sales/returns/999999", "",
			[]string{string(permissions.SaleReturn)}, http.StatusForbidden},

		// sale.return.approve works the approval queue.
		{"sale.return.approve can list the pending queue", http.MethodGet, "/api/sales/returns/pending", "",
			[]string{string(permissions.SaleReturnApprove)}, http.StatusOK},
		{"sale.view alone cannot list the pending queue", http.MethodGet, "/api/sales/returns/pending", "",
			[]string{string(permissions.SaleView)}, http.StatusForbidden},
		{"sale.return alone cannot list the pending queue", http.MethodGet, "/api/sales/returns/pending", "",
			[]string{string(permissions.SaleReturn)}, http.StatusForbidden},
		{"sale.return.approve can approve a return", http.MethodPost, "/api/sales/returns/999999/approve", "",
			[]string{string(permissions.SaleReturnApprove)}, http.StatusNotFound},
		{"sale.return alone cannot approve a return", http.MethodPost, "/api/sales/returns/999999/approve", "",
			[]string{string(permissions.SaleReturn)}, http.StatusForbidden},
		{"sale.view alone cannot approve a return", http.MethodPost, "/api/sales/returns/999999/approve", "",
			[]string{string(permissions.SaleView)}, http.StatusForbidden},
		{"sale.return.approve can reject a return", http.MethodPost, "/api/sales/returns/999999/reject",
			`{"reason":"salah beli"}`, []string{string(permissions.SaleReturnApprove)}, http.StatusNotFound},
		{"sale.view alone cannot reject a return", http.MethodPost, "/api/sales/returns/999999/reject",
			`{"reason":"salah beli"}`, []string{string(permissions.SaleView)}, http.StatusForbidden},
		{"a rejection without a reason falls through to 400", http.MethodPost, "/api/sales/returns/999999/reject",
			`{}`, []string{string(permissions.SaleReturnApprove)}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupReturnRBACRouter(t, tt.perms, &storeID, 1)
			w := doReturnRequest(r, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}
