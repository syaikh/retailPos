package supplier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the HTTP half of audit D3 Option C: who may change which
// product_suppliers row. The SQL half lives in internal/product
// (product_supplier_link_store_scope_test.go) — this file is about the claim the
// handler resolves and hands down, which the SQL cannot police.

const (
	scopeStoreA = 1
	scopeStoreB = 2
)

// scopeRouter builds the real routes behind an auth middleware whose role and
// store claim the test chooses, so a claim of nil is expressible — that is the
// fail-closed case, and a fixture that could not express it could not test it.
func scopeRouter(role string, storeID *int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := NewService(denyingRepo{})
	NewHandler(svc, nil).RegisterRoutes(r.Group("/"), func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "scoped")
		c.Set("role", role)
		c.Set("permissions", []string{
			"supplier.view", "supplier.create", "supplier.update", "supplier.delete",
			"product.cost.view",
		})
		c.Set("storeID", storeID)
		c.Next()
	}, testPermMiddleware)
	return r
}

// scopeMockRouter is scopeRouter against a mock service, so a test can see which
// store the handler pinned and whether it called the write at all.
func scopeMockRouter(role string, storeID *int, svc *mockSupplierServiceForAudit) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, nil).RegisterRoutes(r.Group("/"), func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "scoped")
		c.Set("role", role)
		c.Set("permissions", []string{
			"supplier.view", "supplier.create", "supplier.update", "supplier.delete",
			"product.cost.view",
		})
		c.Set("storeID", storeID)
		c.Next()
	}, testPermMiddleware)
	return r
}

func doScope(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// TestLinkRoutes_FailClosedWithoutStoreClaim is the trap every nil scope invites:
// below, nil means "every store". A manager holding no claim must therefore be
// refused rather than quietly handed the whole estate's negotiated costs.
func TestLinkRoutes_FailClosedWithoutStoreClaim(t *testing.T) {
	storeA := scopeStoreA
	cases := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/products/1/suppliers", ""},
		{http.MethodGet, "/suppliers/1/products", ""},
		{http.MethodPost, "/suppliers/1/products", `{"product_id":1}`},
		{http.MethodDelete, "/suppliers/1/products/1", ""},
		{http.MethodPut, "/suppliers/1/products/1", `{"unit_cost":1}`},
		{http.MethodPost, "/suppliers/1/products/1/preferred", ""},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := doScope(scopeRouter("manager", nil), tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), "store scope is required")

			// The same request with a claim gets past this gate. denyingRepo then
			// fails, which is the proof it reached the handler.
			w = doScope(scopeRouter("manager", &storeA), tc.method, tc.path, tc.body)
			assert.NotEqual(t, http.StatusForbidden, w.Code)
			assert.NotContains(t, w.Body.String(), "store scope is required")
		})
	}
}

// TestLinkRoutes_SupplierRoutesStayGlobal pins the other half of Option C: the
// supplier itself is a trading partner and is not store-scoped, so these routes
// must not demand a claim.
func TestLinkRoutes_SupplierRoutesStayGlobal(t *testing.T) {
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/suppliers", ""},
		{http.MethodGet, "/suppliers/1", ""},
		{http.MethodPost, "/suppliers", `{"name":"N","code":"C"}`},
	} {
		w := doScope(scopeRouter("manager", nil), tc.method, tc.path, tc.body)
		assert.NotContains(t, w.Body.String(), "store scope is required",
			"%s %s is a global route and must not demand a store claim", tc.method, tc.path)
	}
}

func TestLinkRoutes_RejectForeignBodyStore(t *testing.T) {
	storeA := scopeStoreA
	foreign := scopeStoreB

	t.Run("foreign store in the body is refused", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{
			linkProductFn: func(ctx context.Context, ps *ProductSupplier) error {
				t.Errorf("write must not run for a foreign store, got store %v", ps.StoreID)
				return nil
			},
		}
		body := `{"product_id":1,"unit_cost":100,"store_id":` + strconv.Itoa(foreign) + `}`
		w := doScope(scopeMockRouter("manager", &storeA, svc),
			http.MethodPost, "/suppliers/1/products", body)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "store is not in your store")
	})

	t.Run("own store in the body is pinned to the claim", func(t *testing.T) {
		var written *ProductSupplier
		svc := &mockSupplierServiceForAudit{
			linkProductFn: func(ctx context.Context, ps *ProductSupplier) error {
				written = ps
				return nil
			},
		}
		body := `{"product_id":1,"unit_cost":100,"store_id":` + strconv.Itoa(storeA) + `}`
		w := doScope(scopeMockRouter("manager", &storeA, svc),
			http.MethodPost, "/suppliers/1/products", body)

		assert.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, written)
		require.NotNil(t, written.StoreID)
		assert.Equal(t, storeA, *written.StoreID)
		assert.Equal(t, 1, written.SupplierID, "the path names the supplier, not the body")
	})

	t.Run("superadmin may target an explicit store", func(t *testing.T) {
		var written *ProductSupplier
		svc := &mockSupplierServiceForAudit{
			linkProductFn: func(ctx context.Context, ps *ProductSupplier) error {
				written = ps
				return nil
			},
		}
		body := `{"product_id":1,"unit_cost":100,"store_id":` + strconv.Itoa(foreign) + `}`
		w := doScope(scopeMockRouter("superadmin", nil, svc),
			http.MethodPost, "/suppliers/1/products", body)

		assert.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, written)
		require.NotNil(t, written.StoreID, "an explicit store from superadmin is honoured")
		assert.Equal(t, foreign, *written.StoreID)
	})
}

// TestLinkRoutes_GlobalTermsAreRefused covers the reason the write SQL is exact.
// The link is visible to the store — it is on screen — but it is the estate's,
// so editing it is refused with a message that says what to do instead.
func TestLinkRoutes_GlobalTermsAreRefused(t *testing.T) {
	storeA := scopeStoreA

	mutations := []struct {
		name, method, path, body string
		writes                   func(*mockSupplierServiceForAudit)
	}{
		{
			name: "unlink", method: http.MethodDelete, path: "/suppliers/1/products/1", body: "",
			writes: func(m *mockSupplierServiceForAudit) {
				m.unlinkProductFn = func(ctx context.Context, productID, supplierID int, storeID *int) error {
					t.Error("must not unlink a global link")
					return nil
				}
			},
		},
		{
			name: "update", method: http.MethodPut, path: "/suppliers/1/products/1", body: `{"unit_cost":7000}`,
			writes: func(m *mockSupplierServiceForAudit) {
				m.updateProductSupplierFn = func(ctx context.Context, ps *ProductSupplier, storeID *int) error {
					t.Error("must not update a global link")
					return nil
				}
			},
		},
		{
			name: "set preferred", method: http.MethodPost, path: "/suppliers/1/products/1/preferred", body: "",
			writes: func(m *mockSupplierServiceForAudit) {
				m.setPreferredSupplierFn = func(ctx context.Context, productID, supplierID int, storeID *int) error {
					t.Error("must not change the global preference")
					return nil
				}
			},
		},
	}

	for _, tc := range mutations {
		t.Run(tc.name+" refuses the inherited global link", func(t *testing.T) {
			svc := &mockSupplierServiceForAudit{
				getProductSupplierFn: func(ctx context.Context, productID, supplierID int, storeID *int) (*ProductSupplier, error) {
					return &ProductSupplier{ProductID: productID, SupplierID: supplierID, StoreID: nil}, nil
				},
			}
			tc.writes(svc)

			w := doScope(scopeMockRouter("manager", &storeA, svc), tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), "global default")
		})
	}

	t.Run("an own link is writable", func(t *testing.T) {
		own := scopeStoreA
		called := false
		svc := &mockSupplierServiceForAudit{
			getProductSupplierFn: func(ctx context.Context, productID, supplierID int, storeID *int) (*ProductSupplier, error) {
				return &ProductSupplier{ProductID: productID, SupplierID: supplierID, StoreID: &own}, nil
			},
			unlinkProductFn: func(ctx context.Context, productID, supplierID int, storeID *int) error {
				called = true
				assert.Equal(t, storeA, *storeID, "the write carries the claim's store, not the row's")
				return nil
			},
		}

		w := doScope(scopeMockRouter("manager", &storeA, svc),
			http.MethodDelete, "/suppliers/1/products/1", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("a missing link is a 404, not a 403", func(t *testing.T) {
		svc := &mockSupplierServiceForAudit{
			getProductSupplierFn: func(ctx context.Context, productID, supplierID int, storeID *int) (*ProductSupplier, error) {
				return nil, ErrProductSupplierNotFound
			},
			unlinkProductFn: func(ctx context.Context, productID, supplierID int, storeID *int) error {
				t.Error("must not write when the link does not exist")
				return nil
			},
		}

		w := doScope(scopeMockRouter("manager", &storeA, svc),
			http.MethodDelete, "/suppliers/1/products/1", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestLinkRoutes_ReadScopeReachesTheQuery is the assertion the SQL cannot make:
// the scope the handler resolves is the scope the query is asked for.
func TestLinkRoutes_ReadScopeReachesTheQuery(t *testing.T) {
	storeA := scopeStoreA

	var productScope *int
	svc := &mockSupplierServiceForAudit{
		getProductsBySupplierIDFn: func(ctx context.Context, supplierID int, storeID *int) ([]ProductSupplier, error) {
			productScope = storeID
			return nil, nil
		},
	}
	w := doScope(scopeMockRouter("manager", &storeA, svc), http.MethodGet, "/suppliers/1/products", "")
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, productScope)
	assert.Equal(t, storeA, *productScope, "a store's read is scoped to that store")

	// Same mock, so the same recorder answers both requests.
	svc.getProductsBySupplierIDFn = func(ctx context.Context, supplierID int, storeID *int) ([]ProductSupplier, error) {
		productScope = storeID
		return nil, nil
	}
	w = doScope(scopeMockRouter("superadmin", nil, svc), http.MethodGet, "/suppliers/1/products", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, productScope, "superadmin stays unrestricted")
}
