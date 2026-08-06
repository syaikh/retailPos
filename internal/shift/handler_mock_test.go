package shift

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"
)

func init() {
	_ = os.Setenv("JWT_SECRET", "test-secret-for-shift-mock-tests")
}

type mockShiftService struct {
	openShiftFn      func(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error)
	closeShiftFn     func(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	closeAllFn       func(ctx context.Context, userID int) ([]int, error)
	getActiveShiftFn func(ctx context.Context, userID int) (*Shift, error)
	listShiftsFn     func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error)
	getShiftByIDFn   func(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error)
	reviewShiftFn    func(ctx context.Context, shiftID, reviewerID int) (*Shift, error)
	auditShiftFn     func(ctx context.Context, shiftID int) (*Shift, int, error)
	exportShiftsFn   func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error)
}

func (m *mockShiftService) OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
	return m.openShiftFn(ctx, userID, storeID, openingBalance)
}
func (m *mockShiftService) CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	return m.closeShiftFn(ctx, shiftID, userID, closingBalance, notes)
}
func (m *mockShiftService) CloseAll(ctx context.Context, userID int) ([]int, error) {
	return m.closeAllFn(ctx, userID)
}
func (m *mockShiftService) GetActiveShift(ctx context.Context, userID int) (*Shift, error) {
	return m.getActiveShiftFn(ctx, userID)
}
func (m *mockShiftService) ListShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
	return m.listShiftsFn(ctx, scope, status, needsReview, discrepancyFilter, limit, offset, sortBy, sortDir)
}
func (m *mockShiftService) GetShiftByID(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
	return m.getShiftByIDFn(ctx, scope, shiftID)
}
func (m *mockShiftService) ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
	return m.reviewShiftFn(ctx, shiftID, reviewerID)
}
func (m *mockShiftService) AuditShift(ctx context.Context, shiftID int) (*Shift, int, error) {
	return m.auditShiftFn(ctx, shiftID)
}
func (m *mockShiftService) ExportShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
	if m.exportShiftsFn != nil {
		return m.exportShiftsFn(ctx, scope, status, needsReview, discrepancyFilter)
	}
	return nil, nil
}

type mockAudit struct {
	createAuditLogFn func(ctx context.Context, log *audit.Log) error
}

func (m *mockAudit) CreateAuditLog(ctx context.Context, log *audit.Log) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupShiftHandler(svc Service, auditSvc audit.Creator) *gin.Engine {
	return setupShiftHandlerWithCtx(svc, auditSvc, 1, "superadmin", nil)
}

func setupShiftHandlerWithCtx(svc Service, auditSvc audit.Creator, userID int, role string, perms []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", role)
		c.Set("storeID", nil)
		c.Set("permissions", perms)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestShiftHandler_ReviewShift_Success(t *testing.T) {
	svc := &mockShiftService{
		reviewShiftFn: func(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
			return &Shift{ID: 1, Status: "closed", NeedsReview: false, ReviewedBy: &reviewerID}, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/review", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Shift `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Data.ID)
	assert.False(t, resp.Data.NeedsReview)
	require.NotNil(t, resp.Data.ReviewedBy)
	assert.Equal(t, 1, *resp.Data.ReviewedBy)
}

func TestShiftHandler_ReviewShift_InvalidID(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/abc/review", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid shift id")
}

func TestShiftHandler_ReviewShift_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		reviewShiftFn: func(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/review", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShiftHandler_ReviewShift_CreatesAuditLog(t *testing.T) {
	auditCalled := false
	svc := &mockShiftService{
		reviewShiftFn: func(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
			return &Shift{ID: 1, NeedsReview: false}, nil
		},
	}
	auditSvc := &mockAudit{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "review", log.Action)
			assert.Equal(t, "shift", log.EntityType)
			return nil
		},
	}
	r := setupShiftHandler(svc, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/review", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestShiftHandler_OpenShift_InvalidJSON(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/open", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestShiftHandler_CloseShift_InvalidID(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/abc/close", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid shift id")
}

func TestShiftHandler_CloseAll_Success(t *testing.T) {
	svc := &mockShiftService{
		closeAllFn: func(ctx context.Context, userID int) ([]int, error) {
			return []int{1, 2}, nil
		},
	}
	auditCalled := false
	auditSvc := &mockAudit{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "update", log.Action)
			assert.Equal(t, "shift", log.EntityType)
			assert.Contains(t, log.Description, "1, 2")
			return nil
		},
	}
	r := setupShiftHandler(svc, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/close-all", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestShiftHandler_CloseAll_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		closeAllFn: func(ctx context.Context, userID int) ([]int, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/close-all", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestShiftHandler_AuditShift_Success(t *testing.T) {
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 50000, nil
		},
	}
	auditCalled := false
	auditSvc := &mockAudit{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "audit", log.Action)
			assert.Equal(t, "shift", log.EntityType)
			assert.NotNil(t, log.EntityID)
			assert.Equal(t, 1, *log.EntityID)
			return nil
		},
	}
	r := setupShiftHandler(svc, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":160000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
	var resp struct {
		Data struct {
			Shift        Shift `json:"shift"`
			ExpectedCash int   `json:"expected_cash"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Data.Shift.ID)
	assert.Equal(t, 150000, resp.Data.ExpectedCash)
}

func TestShiftHandler_AuditShift_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return nil, 0, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":160000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShiftHandler_GetActiveShift_Success(t *testing.T) {
	svc := &mockShiftService{
		getActiveShiftFn: func(ctx context.Context, userID int) (*Shift, error) {
			return &Shift{ID: 1, UserID: userID, Status: "open"}, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/active", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Shift `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "open", resp.Data.Status)
}

func TestShiftHandler_ListShifts_Success(t *testing.T) {
	svc := &mockShiftService{
		listShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
			return []Shift{{ID: 1, Status: "open"}}, 1, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []Shift `json:"data"`
		Total int     `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, 1, resp.Total)
}

func TestShiftHandler_GetShiftByID_Success(t *testing.T) {
	svc := &mockShiftService{
		getShiftByIDFn: func(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
			return &Shift{ID: shiftID, Status: "closed"}, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Shift `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Data.ID)
	assert.Equal(t, "closed", resp.Data.Status)
}

func TestShiftHandler_GetShiftByID_InvalidID(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/abc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid shift id")
}

func TestShiftHandler_GetShiftByID_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		getShiftByIDFn: func(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestShiftHandler_GetActiveShift_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		getActiveShiftFn: func(ctx context.Context, userID int) (*Shift, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/active", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestShiftHandler_ListShifts_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		listShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
			return nil, 0, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShiftHandler_OpenShift_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		openShiftFn: func(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/open", strings.NewReader(`{"opening_balance":100000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShiftHandler_OpenShift_CreatesAuditLog(t *testing.T) {
	auditCalled := false
	svc := &mockShiftService{
		openShiftFn: func(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
			return &Shift{ID: 1, OpeningBalance: openingBalance, Status: "open"}, nil
		},
	}
	auditSvc := &mockAudit{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "create", log.Action)
			assert.Equal(t, "shift", log.EntityType)
			return nil
		},
	}
	r := setupShiftHandler(svc, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/open", strings.NewReader(`{"opening_balance":100000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestShiftHandler_CloseShift_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		closeShiftFn: func(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/close", strings.NewReader(`{"closing_balance":200000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShiftHandler_CloseShift_CreatesAuditLog(t *testing.T) {
	auditCalled := false
	svc := &mockShiftService{
		closeShiftFn: func(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
			return &Shift{ID: shiftID, Status: "closed"}, nil
		},
	}
	auditSvc := &mockAudit{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			auditCalled = true
			assert.Equal(t, "update", log.Action)
			assert.Equal(t, "shift", log.EntityType)
			return nil
		},
	}
	r := setupShiftHandler(svc, auditSvc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/close", strings.NewReader(`{"closing_balance":200000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, auditCalled, "audit log should be created")
}

func TestShiftHandler_ExportShifts_Success(t *testing.T) {
	svc := &mockShiftService{
		exportShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
			return []Shift{{ID: 1, Status: "closed"}}, nil
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/export", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShiftHandler_ExportShifts_ServiceError(t *testing.T) {
	svc := &mockShiftService{
		exportShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
			return nil, assert.AnError
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/shifts/export", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShiftHandler_CloseShift_InvalidJSON(t *testing.T) {
	svc := &mockShiftService{}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/close", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShiftHandler_ListShifts_OwnershipScope(t *testing.T) {
	var gotScope ownership.Scope
	svc := &mockShiftService{
		listShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
			gotScope = scope
			return []Shift{}, 0, nil
		},
	}

	t.Run("cashier is scoped to own shifts", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 7, "cashier", []string{"shift.view", "shift.create"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ownerID, restricted := gotScope.OwnID()
		assert.True(t, restricted, "cashier must be ownership-restricted")
		assert.Equal(t, 7, ownerID)
	})

	t.Run("cashier user_id filter cannot widen scope", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 7, "cashier", []string{"shift.view", "shift.create"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts?user_id=99", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ownerID, restricted := gotScope.OwnID()
		assert.True(t, restricted, "cashier must stay ownership-restricted")
		assert.Equal(t, 7, ownerID)
	})

	t.Run("manager with shift.review sees all when no filter", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 7, "manager", []string{"shift.view", "shift.review"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, gotScope.CanAccess(12345), "manager without filter must have all-access")
	})

	t.Run("manager user_id filter is honored", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 7, "manager", []string{"shift.view", "shift.review"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts?user_id=42", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ownerID, restricted := gotScope.OwnID()
		assert.True(t, restricted)
		assert.Equal(t, 42, ownerID)
	})
}

func TestShiftHandler_GetShiftByID_OwnershipScope(t *testing.T) {
	var gotScope ownership.Scope
	svc := &mockShiftService{
		getShiftByIDFn: func(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
			gotScope = scope
			if !scope.CanAccess(2) {
				return nil, assert.AnError
			}
			return &Shift{ID: shiftID, UserID: 2, Status: "closed"}, nil
		},
	}

	t.Run("cashier accessing another user's shift gets 404", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 1, "cashier", []string{"shift.view", "shift.create"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts/5", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code, "ownership-miss must look like not found, not forbidden")
		ownerID, restricted := gotScope.OwnID()
		assert.True(t, restricted)
		assert.Equal(t, 1, ownerID)
	})

	t.Run("manager accessing another user's shift succeeds", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 1, "manager", []string{"shift.view", "shift.review"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts/5", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, gotScope.CanAccess(2), "manager must have all-access")
	})
}

func TestShiftHandler_ExportShifts_OwnershipScope(t *testing.T) {
	var gotScope ownership.Scope
	svc := &mockShiftService{
		exportShiftsFn: func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
			gotScope = scope
			return []Shift{}, nil
		},
	}

	t.Run("cashier export user_id filter cannot widen scope", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 7, "cashier", []string{"shift.view", "shift.create"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts/export?user_id=99", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ownerID, restricted := gotScope.OwnID()
		assert.True(t, restricted)
		assert.Equal(t, 7, ownerID)
	})

	t.Run("manager export honors user_id filter", func(t *testing.T) {
		r := setupShiftHandlerWithCtx(svc, nil, 7, "manager", []string{"shift.view", "shift.review"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shifts/export?user_id=42", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ownerID, restricted := gotScope.OwnID()
		assert.True(t, restricted)
		assert.Equal(t, 42, ownerID)
	})
}
