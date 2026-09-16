package consignment

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
)

// rbacAuthMiddleware authenticates the caller and grants exactly the given
// permissions, mirroring the real JWT auth middleware (gin keys + request
// context claims).
func rbacAuthMiddleware(perms []string, storeID *int, uid int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", uid)
		c.Set("username", "consignment_rbac_user")
		c.Set("role", "rbac-tester")
		c.Set("permissions", perms)
		c.Set("storeID", storeID)

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, uid)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "consignment_rbac_user")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "rbac-tester")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, storeID)
		ctx = context.WithValue(ctx, middleware.CtxKeyPermissions, perms)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// setupConsignmentRBACRouter wires the real consignment routes with the real
// permission middleware, so a request is gated exactly as in production.
func setupConsignmentRBACRouter(t *testing.T, perms []string, storeID *int, uid int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(newTestService(t), nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/api"), rbacAuthMiddleware(perms, storeID, uid), middleware.RequirePermission)
	return r
}

func doConsignmentRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

// TestHandler_PaymentMethods_RequiresPayOrSettle pins the authorization gate of
// GET /consignment/payment-methods to (consignment.pay OR consignment.settle).
func TestHandler_PaymentMethods_RequiresPayOrSettle(t *testing.T) {
	ctx := context.Background()
	storeID := insertTestStore(ctx, t)
	testUserID := insertTestUser(ctx, t)

	tests := []struct {
		name  string
		perms []string
		want  int
	}{
		{"consignment.pay alone can open the cash picker", []string{string(permissions.ConsignmentPay)}, http.StatusOK},
		{"consignment.settle alone can open the cash picker", []string{string(permissions.ConsignmentSettle)}, http.StatusOK},
		{"finance (pay + view) can open the cash picker", []string{string(permissions.ConsignmentPay), string(permissions.ConsignmentView)}, http.StatusOK},
		{"consignment.view alone cannot open the cash picker", []string{string(permissions.ConsignmentView)}, http.StatusForbidden},
		{"unrelated permission is rejected", []string{"report.view"}, http.StatusForbidden},
		{"no permissions is rejected", []string{}, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupConsignmentRBACRouter(t, tt.perms, &storeID, testUserID)
			w := doConsignmentRequest(r, http.MethodGet, "/api/consignment/payment-methods", "")
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

// TestHandler_ConsignmentSettlementAuthorization locks the consignment.*
// separation of duties behind the settlements module (migration 047: finance
// gains view, still pay-and-view-only).
func TestHandler_ConsignmentSettlementAuthorization(t *testing.T) {
	ctx := context.Background()
	storeID := insertTestStore(ctx, t)
	testUserID := insertTestUser(ctx, t)

	t.Run("listing settlements requires consignment.view", func(t *testing.T) {
		rView := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentView)}, &storeID, testUserID)
		assert.Equal(t, http.StatusOK, doConsignmentRequest(rView, http.MethodGet, "/api/consignment/settlements?supplier_id=1", "").Code)

		rPay := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentPay)}, &storeID, testUserID)
		assert.Equal(t, http.StatusForbidden, doConsignmentRequest(rPay, http.MethodGet, "/api/consignment/settlements?supplier_id=1", "").Code)
	})

	t.Run("finance (pay + view) can list settlements", func(t *testing.T) {
		r := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentPay), string(permissions.ConsignmentView)}, &storeID, testUserID)
		assert.Equal(t, http.StatusOK, doConsignmentRequest(r, http.MethodGet, "/api/consignment/settlements?supplier_id=1", "").Code)
	})

	t.Run("creating a settlement requires consignment.settle", func(t *testing.T) {
		for _, perms := range [][]string{
			{string(permissions.ConsignmentPay)},
			{string(permissions.ConsignmentView)},
		} {
			r := setupConsignmentRBACRouter(t, perms, &storeID, testUserID)
			assert.Equal(t, http.StatusForbidden, doConsignmentRequest(r, http.MethodPost, "/api/consignment/settlements", `{"supplier_id":1}`).Code)
		}

		rSettle := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentSettle)}, &storeID, testUserID)
		w := doConsignmentRequest(rSettle, http.MethodPost, "/api/consignment/settlements", `{}`)
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})

	t.Run("recording a payout requires consignment.pay", func(t *testing.T) {
		_, stID, _, store := seedSettlement(t, "RBAC-PAY")
		for _, perms := range [][]string{
			{string(permissions.ConsignmentSettle)},
			{string(permissions.ConsignmentView)},
		} {
			r := setupConsignmentRBACRouter(t, perms, &store, testUserID)
			assert.Equal(t, http.StatusForbidden, doConsignmentRequest(
				r, http.MethodPost, fmt.Sprintf("/api/consignment/settlements/%d/payouts", stID), `{"amount":1000}`).Code)
		}

		// A consignment.pay holder passes authorization (not 403); the malformed
		// body is then rejected at the HTTP binding layer — the committed
		// CreatePayoutRequest pins Amount binding:"required,min=1" and
		// PaymentMethodID binding:"required", so `{}` fails ShouldBindJSON and the
		// handler returns 400 before service validation is ever reached. Pinned
		// deterministically against a settlement seeded for this test's own store.
		rPay := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentPay)}, &store, testUserID)
		w := doConsignmentRequest(rPay, http.MethodPost, fmt.Sprintf("/api/consignment/settlements/%d/payouts", stID), `{}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestHandler_CreateArrangement_StoreRequired verifies that a superadmin
// (nil storeID claim) must explicitly select a store when creating an
// arrangement. Omitting store_id returns 400 Bad Request (ErrStoreRequired).
func TestHandler_CreateArrangement_StoreRequired(t *testing.T) {
	ctx := context.Background()
	storeID := insertTestStore(ctx, t)
	supplierID := insertTestSupplier(ctx, t, "StoreReq Supplier", true)
	testUserID := insertTestUser(ctx, t)

	t.Run("superadmin without store_id gets 400", func(t *testing.T) {
		r := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentCreate)}, nil, testUserID)
		body := fmt.Sprintf(`{"supplier_id":%d}`, supplierID)
		w := doConsignmentRequest(r, http.MethodPost, "/api/consignment/arrangements", body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("superadmin with store_id succeeds", func(t *testing.T) {
		r := setupConsignmentRBACRouter(t, []string{string(permissions.ConsignmentCreate)}, nil, testUserID)
		body := fmt.Sprintf(`{"supplier_id":%d,"store_id":%d}`, supplierID, storeID)
		w := doConsignmentRequest(r, http.MethodPost, "/api/consignment/arrangements", body)
		assert.Contains(t, []int{http.StatusCreated, http.StatusConflict}, w.Code)
	})
}

// TestHandler_ListAddTermProductOptions_RequiresView pins the authorization of
// GET /consignment/arrangements/:id/available-products to consignment.view.
func TestHandler_ListAddTermProductOptions_RequiresView(t *testing.T) {
	ctx := context.Background()
	storeID := insertTestStore(ctx, t)

	svc := newTestService(t)
	userID := insertTestUser(ctx, t)
	supplierID := insertTestSupplier(ctx, t, "Konsinyasi Avail Test Supplier", true)
	arr, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: supplierID, StoreID: storeID}, userID, nil)
	require.NoError(t, err)

	tests := []struct {
		name  string
		perms []string
		want  int
	}{
		{"consignment.view can list available products", []string{string(permissions.ConsignmentView)}, http.StatusOK},
		{"consignment.update alone cannot list available products", []string{string(permissions.ConsignmentUpdate)}, http.StatusForbidden},
		{"unrelated permission is rejected", []string{"report.view"}, http.StatusForbidden},
		{"no permissions is rejected", []string{}, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupConsignmentRBACRouter(t, tt.perms, &storeID, userID)
			w := doConsignmentRequest(r, http.MethodGet, fmt.Sprintf("/api/consignment/arrangements/%d/available-products", arr.ID), "")
			assert.Equal(t, tt.want, w.Code)
		})
	}
}
