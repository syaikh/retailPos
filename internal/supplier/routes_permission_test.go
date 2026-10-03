package supplier

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
)

// authWith builds an auth middleware that grants exactly the listed codes, so a
// test can impersonate one role's worth of access without a database.
func authWith(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "tester")
		c.Set("role", "manager")
		c.Set("permissions", perms)
		// A *int, as middleware/auth.go sets from the JWT claim. An int here
		// would be silently read as "no store" by shared.GetStoreID, which the
		// link routes refuse — a manager with no claim is exactly the fail-closed
		// case, so the fixture would be asserting the wrong thing.
		storeID := 1
		c.Set("storeID", &storeID)
		c.Next()
	}
}

// newGatedRouter registers the real supplier routes against the real
// RequirePermission middleware. The package's other router tests use a
// pass-through perm middleware, so this is the only place that proves which
// permission each route actually asks for — the defect that started this wave was
// every route asking for a pricing.* code.
func newGatedRouter(t *testing.T, perms ...string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	svc := NewService(denyingRepo{})
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), authWith(perms...), middleware.RequirePermission)
	return r
}

// denyingRepo satisfies the Service port but fails every call, so a request that
// clears authorization fails inside the handler. That is the point: the
// assertion is about the 403 that permission middleware produces on the way in,
// and a non-403 afterwards proves the gate opened.
type denyingRepo struct{}

var errDenying = errors.New("denyingRepo: call not expected in a permission test")

func (denyingRepo) GetByID(context.Context, int) (*Supplier, error) { return nil, errDenying }
func (denyingRepo) GetByCode(context.Context, string) (*Supplier, error) {
	return nil, errDenying
}
func (denyingRepo) GetAll(context.Context, int, int, string, *bool, *bool) ([]Supplier, int, error) {
	return nil, 0, errDenying
}
func (denyingRepo) Create(context.Context, *Supplier) error             { return errDenying }
func (denyingRepo) Update(context.Context, *Supplier) error             { return errDenying }
func (denyingRepo) Delete(context.Context, int) error                   { return errDenying }
func (denyingRepo) LinkProduct(context.Context, *ProductSupplier) error { return errDenying }
func (denyingRepo) UnlinkProduct(context.Context, int, int, *int) error { return errDenying }
func (denyingRepo) GetProductSupplier(context.Context, int, int, *int) (*ProductSupplier, error) {
	return nil, errDenying
}
func (denyingRepo) GetPreferredSupplier(context.Context, int, *int) (*ProductSupplier, error) {
	return nil, errDenying
}
func (denyingRepo) SetPreferredSupplier(context.Context, int, int, *int) error { return errDenying }
func (denyingRepo) UpdateProductSupplier(context.Context, *ProductSupplier, *int) error {
	return errDenying
}
func (denyingRepo) GetSuppliersByProductID(context.Context, int, *int) ([]ProductSupplier, error) {
	return nil, errDenying
}
func (denyingRepo) GetProductsBySupplierID(context.Context, int, *int) ([]ProductSupplier, error) {
	return nil, errDenying
}
func (denyingRepo) BulkUpdate(context.Context, []int, bool, *int) (int, error) {
	return 0, errDenying
}
func (denyingRepo) BulkDelete(context.Context, []int) (int, error)      { return 0, errDenying }
func (denyingRepo) GetNextSupplierCode(context.Context) (string, error) { return "", errDenying }
func (denyingRepo) CountUsage(context.Context, int) (Usage, error) {
	return Usage{}, errDenying
}

// routeCase is one (method, path) pair plus the permission it must demand.
type routeCase struct {
	method      string
	path        string
	body        string
	needs       permissions.Code
	description string
}

func supplierRouteCases() []routeCase {
	return []routeCase{
		{http.MethodGet, "/suppliers", "", permissions.SupplierView, "list suppliers"},
		{http.MethodGet, "/suppliers/1", "", permissions.SupplierView, "read one supplier"},
		{http.MethodGet, "/suppliers/1/usage", "", permissions.SupplierView, "read a supplier's reference usage"},
		{http.MethodPost, "/suppliers", `{"name":"N","code":"C"}`, permissions.SupplierCreate, "create supplier"},
		{http.MethodPut, "/suppliers/1", `{"name":"N","code":"C"}`, permissions.SupplierUpdate, "update supplier"},
		{http.MethodDelete, "/suppliers/1", "", permissions.SupplierDelete, "delete supplier"},
		{http.MethodPut, "/suppliers/bulk", `{"ids":[1]}`, permissions.SupplierUpdate, "bulk update suppliers"},
		{http.MethodDelete, "/suppliers/bulk", `{"ids":[1]}`, permissions.SupplierDelete, "bulk delete suppliers"},
		{http.MethodGet, "/suppliers/1/products", "", permissions.SupplierView, "list a supplier's products"},
		{http.MethodPost, "/suppliers/1/products", `{"product_id":1}`, permissions.SupplierUpdate, "link product"},
		{http.MethodDelete, "/suppliers/1/products/1", "", permissions.SupplierUpdate, "unlink product"},
		{http.MethodPut, "/suppliers/1/products/1", `{"unit_cost":1}`, permissions.SupplierUpdate, "update product link"},
		{http.MethodPost, "/suppliers/1/products/1/preferred", "", permissions.SupplierUpdate, "set preferred supplier"},
		{http.MethodGet, "/products/1/suppliers", "", permissions.SupplierView, "list a product's suppliers"},
	}
}

// TestSupplierRoutes_DemandSupplierPermissions is the regression test for the
// audit finding: every route used to ask for a pricing.* permission, so the
// pricing grant list silently governed the supplier module.
func TestSupplierRoutes_DemandSupplierPermissions(t *testing.T) {
	for _, tc := range supplierRouteCases() {
		t.Run(tc.description, func(t *testing.T) {
			// Holding only the pricing codes must no longer be enough.
			r := newGatedRouter(t, "pricing.view", "pricing.create", "pricing.update", "pricing.delete")
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("with only pricing.* the route returned %d, want 403 — it is not gated on %s",
					w.Code, tc.needs)
			}

			// Holding the right code must get past authorization. The handler
			// then fails against nilRepo, so any non-403 means the gate opened.
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

// TestSupplierRoutes_CashierHasNoSupplierAccess pins the decision that
// front-of-house staff get no supplier permissions at all.
func TestSupplierRoutes_CashierHasNoSupplierAccess(t *testing.T) {
	cashier := []string{"sale.create", "sale.view", "product.view", "shift.view", "dashboard.view"}
	for _, tc := range supplierRouteCases() {
		t.Run(tc.description, func(t *testing.T) {
			r := newGatedRouter(t, cashier...)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("cashier reached %s %s: got %d, want 403", tc.method, tc.path, w.Code)
			}
		})
	}
}
