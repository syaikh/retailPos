package pricing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"
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

type mockPricingService struct {
	getByIDFn              func(ctx context.Context, id int) (*Rule, error)
	getByProductIDFn       func(ctx context.Context, productID int) ([]Rule, error)
	getAllFn               func(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]Rule, int, error)
	createFn               func(ctx context.Context, rule *Rule) error
	updateFn               func(ctx context.Context, rule *Rule) error
	deleteFn               func(ctx context.Context, id int) error
	findConflictsForRuleFn func(ctx context.Context, rule *Rule, excludeID int) ([]Rule, error)
	approveFn              func(ctx context.Context, id int) error
	rejectFn               func(ctx context.Context, id int) error
}

// GetByID defaults to not-found. The mutating handlers now load the target rule
// to authorize it, and treat a missing rule as a tolerated no-op, so a test that
// only cares about the downstream call can leave getByIDFn nil.
//
// The error must be ErrRuleNotFound, which is what Repository.GetByID returns
// for a missing row. Returning pgx.ErrNoRows here (as this mock used to) is not
// interchangeable: ruleForAction tolerates only the domain sentinel and turns
// anything else into a 500, so a mock that disagreed with production would make
// the not-found paths untestable.
func (m *mockPricingService) GetByID(ctx context.Context, id int) (*Rule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, ErrRuleNotFound
}
func (m *mockPricingService) GetByProductID(ctx context.Context, productID int) ([]Rule, error) {
	return m.getByProductIDFn(ctx, productID)
}
func (m *mockPricingService) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]Rule, int, error) {
	return m.getAllFn(ctx, limit, offset, search, productID, pricingType, pricingMethod, categoryID, brandID, customerGroupID, storeID, isActive, status)
}
func (m *mockPricingService) Create(ctx context.Context, rule *Rule) error {
	return m.createFn(ctx, rule)
}
func (m *mockPricingService) Update(ctx context.Context, rule *Rule) error {
	return m.updateFn(ctx, rule)
}
func (m *mockPricingService) Delete(ctx context.Context, id int) error {
	return m.deleteFn(ctx, id)
}
func (m *mockPricingService) FindConflictsForRule(ctx context.Context, rule *Rule, excludeID int) ([]Rule, error) {
	return m.findConflictsForRuleFn(ctx, rule, excludeID)
}
func (m *mockPricingService) Approve(ctx context.Context, id int) error {
	return m.approveFn(ctx, id)
}
func (m *mockPricingService) Reject(ctx context.Context, id int) error {
	return m.rejectFn(ctx, id)
}

type mockPriceResolver struct {
	resolveFn      func(ctx context.Context, rc ResolveContext) (*ResolvedPrice, error)
	resolveBatchFn func(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error)
}

func (m *mockPriceResolver) Resolve(ctx context.Context, rc ResolveContext) (*ResolvedPrice, error) {
	return m.resolveFn(ctx, rc)
}

func (m *mockPriceResolver) ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error) {
	return m.resolveBatchFn(ctx, items)
}

func (m *mockPriceResolver) ResolveSnapshot(ctx context.Context, rc ResolveContext) (*PriceSnapshot, error) {
	if m.resolveFn != nil {
		resolved, err := m.resolveFn(ctx, rc)
		if err != nil {
			return nil, err
		}
		return &PriceSnapshot{
			ProductID:     rc.ProductID,
			UnitPrice:     resolved.UnitPrice,
			OriginalPrice: resolved.OriginalPrice,
			Discount:      resolved.Discount,
			Type:          resolved.Type,
			Method:        resolved.Method,
			Rule:          resolved.Rule,
		}, nil
	}
	return nil, nil
}

func (m *mockPriceResolver) ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error) {
	if m.resolveBatchFn != nil {
		resolved, err := m.resolveBatchFn(ctx, items)
		if err != nil {
			return nil, err
		}
		snapshots := make([]PriceSnapshot, len(resolved))
		for i, r := range resolved {
			snapshots[i] = PriceSnapshot{
				ProductID:     items[i].ProductID,
				UnitPrice:     r.UnitPrice,
				OriginalPrice: r.OriginalPrice,
				Discount:      r.Discount,
				Type:          r.Type,
				Method:        r.Method,
				Rule:          r.Rule,
			}
		}
		return snapshots, nil
	}
	return nil, nil
}

type mockProductSearcher struct {
	searchProductsFn func(ctx context.Context, query string, limit int) ([]ProductSearchResult, error)
}

func (m *mockProductSearcher) SearchProducts(ctx context.Context, query string, limit int) ([]ProductSearchResult, error) {
	return m.searchProductsFn(ctx, query, limit)
}

func setupPricingMockRouter(svc Service, resolver PriceResolver, searcher ProductSearcher, auditSvc audit.Creator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"pricing.view", "pricing.create", "pricing.update", "pricing.delete"})
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(svc, resolver, auditSvc)
	h.SetProductSearcher(searcher)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestPricingHandler_DeleteRule_ServiceError(t *testing.T) {
	svc := &mockPricingService{
		deleteFn: func(ctx context.Context, id int) error {
			return assert.AnError
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/pricing-rules/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPricingHandler_ListRules_ServiceError(t *testing.T) {
	svc := &mockPricingService{
		getAllFn: func(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]Rule, int, error) {
			return nil, 0, assert.AnError
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/pricing-rules", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPricingHandler_CheckConflicts_ServiceError(t *testing.T) {
	svc := &mockPricingService{
		findConflictsForRuleFn: func(ctx context.Context, rule *Rule, excludeID int) ([]Rule, error) {
			return nil, assert.AnError
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, nil)
	w := httptest.NewRecorder()
	body := `{"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":9999,"minimum_quantity":1}`
	req := httptest.NewRequest("POST", "/pricing-rules/check-conflicts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPricingHandler_ResolvePrices_ResolverError(t *testing.T) {
	svc := &mockPricingService{}
	resolver := &mockPriceResolver{
		resolveBatchFn: func(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error) {
			return nil, assert.AnError
		},
	}
	r := setupPricingMockRouter(svc, resolver, nil, nil)
	w := httptest.NewRecorder()
	body := `{"items":[{"product_id":1,"quantity":1}]}`
	req := httptest.NewRequest("POST", "/pricing/resolve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPricingHandler_SearchProducts_SearcherError(t *testing.T) {
	svc := &mockPricingService{}
	searcher := &mockProductSearcher{
		searchProductsFn: func(ctx context.Context, query string, limit int) ([]ProductSearchResult, error) {
			return nil, assert.AnError
		},
	}
	r := setupPricingMockRouter(svc, nil, searcher, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/products/search?q=test&limit=10", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPricingHandler_UpdateRule_ServiceError(t *testing.T) {
	svc := &mockPricingService{
		// UpdateRule loads the target to authorize it before writing, so the rule
		// must resolve for the update error to be the one under test.
		getByIDFn: func(ctx context.Context, id int) (*Rule, error) {
			return &Rule{ID: id, Name: "Existing"}, nil
		},
		updateFn: func(ctx context.Context, rule *Rule) error {
			return assert.AnError
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, nil)
	w := httptest.NewRecorder()
	body := `{"name":"Test","pricing_type":"promotion","pricing_method":"fixed_price"}`
	req := httptest.NewRequest("PUT", "/pricing-rules/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPricingHandler_DeleteRule_WithAudit(t *testing.T) {
	auditCalled := false
	svc := &mockPricingService{
		getByIDFn: func(ctx context.Context, id int) (*Rule, error) {
			return &Rule{ID: id, Name: "Rule To Delete"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "delete", log.Action)
			assert.Equal(t, "pricing_rule", log.EntityType)
			return nil
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/pricing-rules/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

// GetByID is now the store authorization gate, not just the source of an audit
// description. A load failure must therefore fail closed: reporting 500 and
// skipping the delete, rather than treating the error as "no such rule" and
// executing an unauthorized write.
func TestPricingHandler_DeleteRule_WithAuditGetByIDError(t *testing.T) {
	auditCalled := false
	deleted := false
	svc := &mockPricingService{
		getByIDFn: func(ctx context.Context, id int) (*Rule, error) {
			return nil, assert.AnError
		},
		deleteFn: func(ctx context.Context, id int) error {
			deleted = true
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			return nil
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/pricing-rules/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.False(t, deleted, "delete must not run when the store check could not be made")
	assert.False(t, auditCalled, "no audit log for a request that did not mutate")
}

func TestPricingHandler_CreateRule_WithAudit(t *testing.T) {
	auditCalled := false
	svc := &mockPricingService{
		createFn: func(ctx context.Context, rule *Rule) error {
			rule.ID = 42
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "create", log.Action)
			assert.Equal(t, "pricing_rule", log.EntityType)
			return nil
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, auditSvc)
	w := httptest.NewRecorder()
	body := `{"name":"Audit Create","pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":10000}`
	req := httptest.NewRequest("POST", "/pricing-rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestPricingHandler_UpdateRule_WithAudit(t *testing.T) {
	auditCalled := false
	svc := &mockPricingService{
		getByIDFn: func(ctx context.Context, id int) (*Rule, error) {
			return &Rule{ID: id, Name: "Old Name"}, nil
		},
		updateFn: func(ctx context.Context, rule *Rule) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "update", log.Action)
			assert.Equal(t, "pricing_rule", log.EntityType)
			return nil
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, auditSvc)
	w := httptest.NewRecorder()
	body := `{"name":"Updated Name","pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":10000}`
	req := httptest.NewRequest("PUT", "/pricing-rules/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestPricingHandler_CheckConflicts_Success(t *testing.T) {
	svc := &mockPricingService{
		findConflictsForRuleFn: func(ctx context.Context, rule *Rule, excludeID int) ([]Rule, error) {
			return nil, nil
		},
	}
	r := setupPricingMockRouter(svc, nil, nil, nil)
	w := httptest.NewRecorder()
	body := `{"pricing_type":"promotion","pricing_method":"fixed_price","pricing_value":9999,"minimum_quantity":1}`
	req := httptest.NewRequest("POST", "/pricing-rules/check-conflicts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data         []Rule `json:"data"`
		HasConflicts bool   `json:"has_conflicts"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Data)
	assert.False(t, resp.HasConflicts)
}

// Wave 2f: GetRule and UpdateRule now share ruleForAction with the mutating
// routes, so a load failure is no longer reported as a missing rule. The two
// cases look alike from outside — both mean "you get no rule" — but they are
// different events: a 404 is a normal outcome a client may cache or branch on,
// while a 500 says the boundary could not be evaluated at all, and answering
// 404 hides a database problem from both the user and the logs.
func TestPricingHandler_LoadFailureIsNotReportedAsNotFound(t *testing.T) {
	// The mock router runs as superadmin, so the store boundary is not what is
	// under test here; the load outcome is.
	for _, tc := range []struct {
		name     string
		err      error
		wantCode int
	}{
		{"missing rule is a 404", ErrRuleNotFound, http.StatusNotFound},
		{"load failure is a 500", assert.AnError, http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPricingService{
				getByIDFn: func(_ context.Context, _ int) (*Rule, error) { return nil, tc.err },
				updateFn:  func(_ context.Context, _ *Rule) error { return nil },
			}
			r := setupPricingMockRouter(svc, nil, nil, nil)

			for _, req := range []*http.Request{
				httptest.NewRequest(http.MethodGet, "/pricing-rules/1", nil),
				httptest.NewRequest(http.MethodPut, "/pricing-rules/1", strings.NewReader(`{"name":"R"}`)),
			} {
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, tc.wantCode, w.Code, "%s %s", req.Method, req.URL.Path)
			}
		})
	}
}
