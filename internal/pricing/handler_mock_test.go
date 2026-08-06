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
	submitForApprovalFn    func(ctx context.Context, id int) error
	approveFn              func(ctx context.Context, id int) error
	rejectFn               func(ctx context.Context, id int) error
}

func (m *mockPricingService) GetByID(ctx context.Context, id int) (*Rule, error) {
	return m.getByIDFn(ctx, id)
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
			Type:   resolved.Type,
			Method: resolved.Method,
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
				Type:   r.Type,
				Method: r.Method,
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

func TestPricingHandler_DeleteRule_WithAuditGetByIDError(t *testing.T) {
	auditCalled := false
	svc := &mockPricingService{
		getByIDFn: func(ctx context.Context, id int) (*Rule, error) {
			return nil, assert.AnError
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
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
		HasConflicts bool          `json:"has_conflicts"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Data)
	assert.False(t, resp.HasConflicts)
}
