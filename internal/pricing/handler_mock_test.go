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

type mockPricingService struct {
	getByIDFn              func(ctx context.Context, id int) (*PricingRule, error)
	getByProductIDFn       func(ctx context.Context, productID int) ([]PricingRule, error)
	getAllFn               func(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]PricingRule, int, error)
	createFn               func(ctx context.Context, rule *PricingRule) error
	updateFn               func(ctx context.Context, rule *PricingRule) error
	deleteFn               func(ctx context.Context, id int) error
	findConflictsForRuleFn func(ctx context.Context, rule *PricingRule, excludeID int) ([]PricingRule, error)
	submitForApprovalFn    func(ctx context.Context, id int) error
	approveFn              func(ctx context.Context, id int) error
	rejectFn               func(ctx context.Context, id int) error
}

func (m *mockPricingService) GetByID(ctx context.Context, id int) (*PricingRule, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockPricingService) GetByProductID(ctx context.Context, productID int) ([]PricingRule, error) {
	return m.getByProductIDFn(ctx, productID)
}
func (m *mockPricingService) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]PricingRule, int, error) {
	return m.getAllFn(ctx, limit, offset, search, productID, pricingType, pricingMethod, categoryID, brandID, customerGroupID, storeID, isActive, status)
}
func (m *mockPricingService) Create(ctx context.Context, rule *PricingRule) error {
	return m.createFn(ctx, rule)
}
func (m *mockPricingService) Update(ctx context.Context, rule *PricingRule) error {
	return m.updateFn(ctx, rule)
}
func (m *mockPricingService) Delete(ctx context.Context, id int) error {
	return m.deleteFn(ctx, id)
}
func (m *mockPricingService) FindConflictsForRule(ctx context.Context, rule *PricingRule, excludeID int) ([]PricingRule, error) {
	return m.findConflictsForRuleFn(ctx, rule, excludeID)
}
func (m *mockPricingService) SubmitForApproval(ctx context.Context, id int) error {
	return m.submitForApprovalFn(ctx, id)
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

type mockProductSearcher struct {
	searchProductsFn func(ctx context.Context, query string, limit int) ([]ProductSearchResult, error)
}

func (m *mockProductSearcher) SearchProducts(ctx context.Context, query string, limit int) ([]ProductSearchResult, error) {
	return m.searchProductsFn(ctx, query, limit)
}

func setupPricingMockRouter(svc PricingService, resolver PriceResolver, searcher ProductSearcher, auditSvc audit.AuditCreator) *gin.Engine {
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
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
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
		getAllFn: func(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]PricingRule, int, error) {
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
		findConflictsForRuleFn: func(ctx context.Context, rule *PricingRule, excludeID int) ([]PricingRule, error) {
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
		updateFn: func(ctx context.Context, rule *PricingRule) error {
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
		getByIDFn: func(ctx context.Context, id int) (*PricingRule, error) {
			return &PricingRule{ID: id, Name: "Rule To Delete"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
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

func TestPricingHandler_DeleteRule_WithAuditGetByIDError(t *testing.T) {
	auditCalled := false
	svc := &mockPricingService{
		getByIDFn: func(ctx context.Context, id int) (*PricingRule, error) {
			return nil, assert.AnError
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
			auditCalled = true
			assert.Equal(t, "delete", log.Action)
			assert.Contains(t, log.Description, "Deleted pricing rule #1")
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

func TestPricingHandler_CreateRule_WithAudit(t *testing.T) {
	auditCalled := false
	svc := &mockPricingService{
		createFn: func(ctx context.Context, rule *PricingRule) error {
			rule.ID = 42
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
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
		getByIDFn: func(ctx context.Context, id int) (*PricingRule, error) {
			return &PricingRule{ID: id, Name: "Old Name"}, nil
		},
		updateFn: func(ctx context.Context, rule *PricingRule) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
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
		findConflictsForRuleFn: func(ctx context.Context, rule *PricingRule, excludeID int) ([]PricingRule, error) {
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
		Data         []PricingRule `json:"data"`
		HasConflicts bool          `json:"has_conflicts"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Data)
	assert.False(t, resp.HasConflicts)
}
