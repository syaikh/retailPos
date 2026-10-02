package pricing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
)

// errDenying is returned by the mock behind these gates. A request that clears
// authorization fails inside the handler with this, which is the point: the
// assertion is about the 403 the permission middleware produces on the way in,
// and a non-403 afterwards proves the gate opened.
var errDenying = errors.New("denying mock: call not expected in a permission test")

// authWith grants exactly the listed permission codes, so a test can impersonate
// one role's worth of access without a database.
func authWith(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "tester")
		c.Set("role", "manager")
		c.Set("permissions", perms)
		// *int, matching shared.GetStoreID. A store-scoped role also has to
		// clear authorizeStoreRule, or a 403 from there would mask the
		// permission assertion these tests are actually making.
		storeID := 1
		c.Set("storeID", &storeID)
		c.Next()
	}
}

// newGatedRouter registers the real pricing routes against the real
// RequirePermission middleware over a service whose every method fails. The
// package's other router tests use a pass-through perm middleware, so this is
// the only place that proves which permission each route actually demands, and
// in particular that approval is gated on pricing.approve rather than on
// pricing.update.
func newGatedRouter(t *testing.T, perms ...string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	svc := &mockPricingService{
		getByIDFn: func(_ context.Context, _ int) (*Rule, error) { return nil, errDenying },
		getAllFn: func(_ context.Context, _, _ int, _ string, _ *int, _, _ string, _, _, _, _ *int, _ *bool, _ string) ([]Rule, int, error) {
			return nil, 0, errDenying
		},
		getByProductIDFn: func(_ context.Context, _ int) ([]Rule, error) { return nil, errDenying },
		createFn:         func(_ context.Context, _ *Rule) error { return errDenying },
		updateFn:         func(_ context.Context, _ *Rule) error { return errDenying },
		deleteFn:         func(_ context.Context, _ int) error { return errDenying },
		findConflictsForRuleFn: func(_ context.Context, _ *Rule, _ int) ([]Rule, error) {
			return nil, errDenying
		},
		approveFn: func(_ context.Context, _ int) error { return errDenying },
		rejectFn:  func(_ context.Context, _ int) error { return errDenying },
	}

	h := NewHandler(svc, nil, nil)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), authWith(perms...), middleware.RequirePermission)
	return r
}

type routeCase struct {
	method      string
	path        string
	body        string
	needs       permissions.Code
	description string
}

func pricingRouteCases() []routeCase {
	return []routeCase{
		{http.MethodGet, "/pricing-rules", "", permissions.PricingView, "list pricing rules"},
		{http.MethodGet, "/pricing-rules/1", "", permissions.PricingView, "read one pricing rule"},
		{http.MethodPost, "/pricing-rules", `{"name":"R","pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":1,"minimum_quantity":1}`, permissions.PricingCreate, "create pricing rule"},
		{http.MethodPut, "/pricing-rules/1", `{"name":"R"}`, permissions.PricingUpdate, "update pricing rule"},
		{http.MethodDelete, "/pricing-rules/1", "", permissions.PricingDelete, "delete pricing rule"},
		{http.MethodPost, "/pricing-rules/check-conflicts", `{"name":"R"}`, permissions.PricingView, "check pricing conflicts"},
		{http.MethodPost, "/pricing-rules/1/approve", "", permissions.PricingApprove, "approve pricing rule"},
		{http.MethodPost, "/pricing-rules/1/reject", "", permissions.PricingApprove, "reject pricing rule"},
		{http.MethodPost, "/pricing/resolve", `{"items":[]}`, permissions.PricingView, "resolve prices"},
		{http.MethodGet, "/products/search", "", permissions.PricingView, "search products for pricing"},
	}
}

// allPricingCodes is every code the pricing module can demand, so a test can
// grant all-but-one and prove a route is gated on the one that is missing.
var allPricingCodes = []string{
	"pricing.view", "pricing.create", "pricing.update", "pricing.delete", "pricing.approve",
}

func TestPricingRoutes_DemandPricingPermissions(t *testing.T) {
	for _, tc := range pricingRouteCases() {
		t.Run(tc.description, func(t *testing.T) {
			// Grant every pricing code except the one this route needs. Anything
			// other than a 403 means the route is gated on the wrong code — the
			// defect this wave fixed was approval being reachable with the update
			// permission, and a JSON body being able to set status directly.
			without := make([]string, 0, len(allPricingCodes))
			for _, code := range allPricingCodes {
				if code != tc.needs.String() {
					without = append(without, code)
				}
			}
			r := newGatedRouter(t, without...)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("with every pricing code except %s the route returned %d, want 403",
					tc.needs, w.Code)
			}

			// Holding the exact code must get past authorization. The handler then
			// fails against the denying mock, so any non-403 proves the gate opened.
			r = newGatedRouter(t, tc.needs.String())
			w = httptest.NewRecorder()
			req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code == http.StatusForbidden {
				t.Errorf("with %s the route returned 403: %s", tc.needs, w.Body.String())
			}
		})
	}
}

// TestPricingApproveRoute_RequiresApprovePermission is the route-level half of
// the decision's "unauthorized approval" case: TestAuthorizeApproval covers the
// self-approval rule, this covers a caller who simply lacks pricing.approve.
func TestPricingApproveRoute_RequiresApprovePermission(t *testing.T) {
	for _, path := range []string{"/pricing-rules/1/approve", "/pricing-rules/1/reject"} {
		t.Run(path, func(t *testing.T) {
			for _, perms := range [][]string{
				{},
				{"pricing.view"},
				{"pricing.create"},
				{"pricing.update"},
				{"pricing.delete"},
				{"pricing.view", "pricing.create", "pricing.update", "pricing.delete"},
			} {
				r := newGatedRouter(t, perms...)
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, path, nil)
				r.ServeHTTP(w, req)
				if w.Code != http.StatusForbidden {
					t.Errorf("%s with %v returned %d, want 403", path, perms, w.Code)
				}
			}

			// pricing.approve alone must open the gate.
			r := newGatedRouter(t, "pricing.approve")
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, nil)
			r.ServeHTTP(w, req)
			if w.Code == http.StatusForbidden {
				t.Errorf("%s with pricing.approve returned 403: %s", path, w.Body.String())
			}
		})
	}
}
