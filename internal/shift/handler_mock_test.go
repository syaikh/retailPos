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
	"github.com/jackc/pgx/v5"
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
	openShiftFn               func(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error)
	closeShiftFn              func(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	getActiveShiftFn          func(ctx context.Context, userID int) (*Shift, error)
	listShiftsFn              func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error)
	getShiftByIDFn            func(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error)
	reviewShiftFn             func(ctx context.Context, shiftID, reviewerID int) (*Shift, error)
	flagForReviewFn           func(ctx context.Context, shiftID int) error
	getDiscrepancyThresholdFn func(ctx context.Context) int
	auditShiftFn              func(ctx context.Context, shiftID int) (*Shift, int, error)
	exportShiftsFn            func(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error)

	// Scope capture. These paths exist only to mutate a shift the caller names
	// by id, so "did the handler pass the caller's store down" is the whole
	// question; recording the scope answers it without a second signature per
	// mock method.
	lastReviewScope         ownership.Scope
	lastAuditScope          ownership.Scope
	lastReportScope         ownership.Scope
	lastListMovementScope   ownership.Scope
	lastCreateMovementScope ownership.Scope
}

func (m *mockShiftService) OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
	return m.openShiftFn(ctx, userID, storeID, openingBalance)
}
func (m *mockShiftService) CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	return m.closeShiftFn(ctx, shiftID, userID, closingBalance, notes)
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
func (m *mockShiftService) ReviewShift(ctx context.Context, scope ownership.Scope, shiftID, reviewerID int) (*Shift, error) {
	m.lastReviewScope = scope
	return m.reviewShiftFn(ctx, shiftID, reviewerID)
}
func (m *mockShiftService) FlagForReview(ctx context.Context, shiftID int) error {
	if m.flagForReviewFn != nil {
		return m.flagForReviewFn(ctx, shiftID)
	}
	return nil
}
func (m *mockShiftService) GetDiscrepancyThreshold(ctx context.Context) int {
	if m.getDiscrepancyThresholdFn != nil {
		return m.getDiscrepancyThresholdFn(ctx)
	}
	return defaultDiscrepancyThreshold
}
func (m *mockShiftService) SetSettingsProvider(p SettingsProvider) {}
func (m *mockShiftService) AuditShift(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, int, error) {
	m.lastAuditScope = scope
	return m.auditShiftFn(ctx, shiftID)
}
func (m *mockShiftService) ExportShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
	if m.exportShiftsFn != nil {
		return m.exportShiftsFn(ctx, scope, status, needsReview, discrepancyFilter)
	}
	return nil, nil
}

func (m *mockShiftService) CreateCashMovement(ctx context.Context, scope ownership.Scope, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error) {
	m.lastCreateMovementScope = scope
	return nil, nil
}
func (m *mockShiftService) ListCashMovements(ctx context.Context, scope ownership.Scope, shiftID int) ([]CashMovement, error) {
	m.lastListMovementScope = scope
	return nil, nil
}
func (m *mockShiftService) ShiftCashMovementSummary(ctx context.Context, shiftID int) (CashMovementSummary, error) {
	return CashMovementSummary{}, nil
}

func (m *mockShiftService) OpenShiftTx(ctx context.Context, tx pgx.Tx, userID int, storeID *int, openingBalance int) (*Shift, error) {
	return m.openShiftFn(ctx, userID, storeID, openingBalance)
}

func (m *mockShiftService) CloseShiftTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	return m.closeShiftFn(ctx, shiftID, userID, closingBalance, notes)
}

func (m *mockShiftService) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func (m *mockShiftService) GetShiftReportData(ctx context.Context, scope ownership.Scope, shiftID int) (*ReportData, error) {
	m.lastReportScope = scope
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

func (m *mockAudit) CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *audit.Log) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupShiftHandler(svc Service, auditSvc audit.TxCreator) *gin.Engine {
	return setupShiftHandlerWithCtx(svc, auditSvc, 1, "superadmin", nil)
}

// setupStoreScopedShiftHandler mirrors setupShiftHandlerWithCtx but puts a store
// claim in the context, the way RequireStoreID does for every role but
// superadmin. Wave 4's whole question is what a store-scoped caller can reach.
func setupStoreScopedShiftHandler(svc Service, storeID int, role string, perms []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	// shared.GetStoreID type-asserts *int, so the claim has to be a pointer.
	// A bare int in the context silently reads back as no claim at all, which
	// is exactly the kind of mistake that makes a store test pass for the
	// wrong reason.
	claim := storeID
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 7)
		c.Set("username", "storeuser")
		c.Set("roleID", 1)
		c.Set("role", role)
		c.Set("storeID", &claim)
		c.Set("permissions", perms)
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func setupShiftHandlerWithCtx(svc Service, auditSvc audit.TxCreator, userID int, role string, perms []string) *gin.Engine {
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

	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestShiftHandler_AuditShift_Success(t *testing.T) {
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 50000, nil
		},
		getDiscrepancyThresholdFn: func(ctx context.Context) int {
			return 50000
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
			Shift            Shift `json:"shift"`
			ExpectedCash     int   `json:"expected_cash"`
			OffBy            int   `json:"off_by"`
			FlaggedForReview bool  `json:"flagged_for_review"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Data.Shift.ID)
	assert.Equal(t, 150000, resp.Data.ExpectedCash)
	assert.Equal(t, 10000, resp.Data.OffBy)
	assert.False(t, resp.Data.FlaggedForReview, "offBy=10000 must not exceed 50000 threshold")
}

func TestShiftHandler_AuditShift_FlaggedForReview(t *testing.T) {
	var flagCalled bool
	var flagShiftID int
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 0, nil
		},
		flagForReviewFn: func(ctx context.Context, shiftID int) error {
			flagCalled = true
			flagShiftID = shiftID
			return nil
		},
		getDiscrepancyThresholdFn: func(ctx context.Context) int {
			return 50000
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":160000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, flagCalled, "FlagForReview must be called when offBy > 50000")
	assert.Equal(t, 1, flagShiftID)
	var resp struct {
		Data struct {
			FlaggedForReview bool `json:"flagged_for_review"`
			OffBy            int  `json:"off_by"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Data.FlaggedForReview)
	assert.Equal(t, 60000, resp.Data.OffBy)
}

func TestShiftHandler_AuditShift_NegativeOffByFlagged(t *testing.T) {
	var flagCalled bool
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 0, nil
		},
		flagForReviewFn: func(ctx context.Context, shiftID int) error {
			flagCalled = true
			return nil
		},
		getDiscrepancyThresholdFn: func(ctx context.Context) int {
			return 50000
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":40000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, flagCalled, "FlagForReview must be called when |offBy| > 50000")
	var resp struct {
		Data struct {
			FlaggedForReview bool `json:"flagged_for_review"`
			OffBy            int  `json:"off_by"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Data.FlaggedForReview)
	assert.Equal(t, -60000, resp.Data.OffBy)
}

func TestShiftHandler_AuditShift_ExactThreshold(t *testing.T) {
	var flagCalled bool
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 0, nil
		},
		flagForReviewFn: func(ctx context.Context, shiftID int) error {
			flagCalled = true
			return nil
		},
		getDiscrepancyThresholdFn: func(ctx context.Context) int {
			return 50000
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":150000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, flagCalled, "FlagForReview must NOT be called when offBy == 50000")
	var resp struct {
		Data struct {
			FlaggedForReview bool `json:"flagged_for_review"`
			OffBy            int  `json:"off_by"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Data.FlaggedForReview)
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

func TestShiftHandler_AuditShift_CustomThreshold(t *testing.T) {
	var flagCalled bool
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 0, nil
		},
		flagForReviewFn: func(ctx context.Context, shiftID int) error {
			flagCalled = true
			return nil
		},
		getDiscrepancyThresholdFn: func(ctx context.Context) int {
			return 20000
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":125000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, flagCalled, "FlagForReview must be called when offBy > 20000 custom threshold")
	var resp struct {
		Data struct {
			FlaggedForReview bool `json:"flagged_for_review"`
			OffBy            int  `json:"off_by"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Data.FlaggedForReview)
	assert.Equal(t, 25000, resp.Data.OffBy)
}

func TestShiftHandler_AuditShift_CustomThresholdNotExceeded(t *testing.T) {
	var flagCalled bool
	svc := &mockShiftService{
		auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
			return &Shift{ID: shiftID, Status: "closed", OpeningBalance: 100000}, 0, nil
		},
		flagForReviewFn: func(ctx context.Context, shiftID int) error {
			flagCalled = true
			return nil
		},
		getDiscrepancyThresholdFn: func(ctx context.Context) int {
			return 20000
		},
	}
	r := setupShiftHandler(svc, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/shifts/1/audit", strings.NewReader(`{"actual_balance":115000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, flagCalled, "FlagForReview must NOT be called when offBy <= 20000 custom threshold")
	var resp struct {
		Data struct {
			FlaggedForReview bool `json:"flagged_for_review"`
			OffBy            int  `json:"off_by"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Data.FlaggedForReview)
	assert.Equal(t, 15000, resp.Data.OffBy)
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
			assert.Equal(t, "shift_opened", log.Action)
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
			assert.Equal(t, "shift_closed", log.Action)
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

// --- Wave 4: shift store boundary ---

func TestShiftHandler_OpenShift_StoreBoundary(t *testing.T) {
	storeA, storeB := 1, 2

	cases := []struct {
		name      string
		role      string
		claims    *int
		body      string
		wantCode  int
		wantStore *int
	}{
		{
			name:     "a cashier cannot open a shift in another store",
			role:     "cashier",
			claims:   &storeA,
			body:     `{"store_id":2,"opening_balance":0}`,
			wantCode: http.StatusForbidden,
		},
		{
			name:      "omitting the store opens in the caller's own store",
			role:      "cashier",
			claims:    &storeA,
			body:      `{"opening_balance":0}`,
			wantCode:  http.StatusOK,
			wantStore: &storeA,
		},
		{
			name:      "naming the caller's own store is allowed",
			role:      "cashier",
			claims:    &storeA,
			body:      `{"store_id":1,"opening_balance":0}`,
			wantCode:  http.StatusOK,
			wantStore: &storeA,
		},
		{
			name:      "superadmin provisions in another store",
			role:      "superadmin",
			claims:    nil,
			body:      `{"store_id":2,"opening_balance":0}`,
			wantCode:  http.StatusOK,
			wantStore: &storeB,
		},
		{
			// RequireStoreID rejects this upstream in production; the handler
			// must not depend on its mounting to avoid minting a global shift.
			name:     "a non-superadmin with no store claim is refused",
			role:     "cashier",
			claims:   nil,
			body:     `{"opening_balance":0}`,
			wantCode: http.StatusForbidden,
		},
		{
			name:      "superadmin may still create a global shift",
			role:      "superadmin",
			claims:    nil,
			body:      `{"opening_balance":0}`,
			wantCode:  http.StatusOK,
			wantStore: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotStore *int
			called := false
			svc := &mockShiftService{
				openShiftFn: func(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
					called = true
					gotStore = storeID
					return &Shift{ID: 1, Status: "open"}, nil
				},
			}
			r := setupStoreScopedShiftHandler(svc, storeA, tc.role, []string{string(permissions.ShiftCreate)})
			if tc.claims == nil {
				// superadmin: no store claim in context
				r = setupShiftHandlerWithCtx(svc, nil, 7, tc.role, []string{string(permissions.ShiftCreate)})
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/shifts/open", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code, "body: %s", w.Body.String())
			if tc.wantCode == http.StatusOK {
				assert.True(t, called, "service must be reached")
				if tc.wantStore == nil {
					assert.Nil(t, gotStore, "a global shift keeps a nil store")
				} else {
					require.NotNil(t, gotStore)
					assert.Equal(t, *tc.wantStore, *gotStore)
				}
			} else {
				assert.False(t, called, "a refused open must not reach the service")
			}
		})
	}
}

// A store-scoped caller holding shift.review has all-access across users but
// not across stores. Every path that acts on a shift named by id must receive
// that store claim, or the id is unguarded.
func TestShiftHandler_ReadPaths_PassStoreClaim(t *testing.T) {
	storeA := 1

	t.Run("review", func(t *testing.T) {
		svc := &mockShiftService{
			reviewShiftFn: func(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
				return &Shift{ID: shiftID, Status: "closed"}, nil
			},
		}
		r := setupStoreScopedShiftHandler(svc, storeA, "supervisor", []string{string(permissions.ShiftReview)})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/shifts/5/review", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, svc.lastReviewScope.StoreID)
		assert.Equal(t, storeA, *svc.lastReviewScope.StoreID)
		// the user dimension stays open: all-access is about users only
		assert.Nil(t, svc.lastReviewScope.UserID)
	})

	t.Run("audit", func(t *testing.T) {
		svc := &mockShiftService{
			auditShiftFn: func(ctx context.Context, shiftID int) (*Shift, int, error) {
				return &Shift{ID: shiftID}, 0, nil
			},
		}
		r := setupStoreScopedShiftHandler(svc, storeA, "supervisor", []string{string(permissions.ShiftAudit)})
		w := httptest.NewRecorder()
		auditReq := httptest.NewRequest(http.MethodPost, "/shifts/5/audit",
			strings.NewReader(`{"actual_balance":500000}`))
		auditReq.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, auditReq)
		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		require.NotNil(t, svc.lastAuditScope.StoreID)
		assert.Equal(t, storeA, *svc.lastAuditScope.StoreID)
	})

	t.Run("get by id", func(t *testing.T) {
		var got ownership.Scope
		svc := &mockShiftService{
			getShiftByIDFn: func(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
				got = scope
				return &Shift{ID: shiftID}, nil
			},
		}
		r := setupStoreScopedShiftHandler(svc, storeA, "cashier", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/shifts/5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, got.StoreID)
		assert.Equal(t, storeA, *got.StoreID)
	})

	t.Run("cash movements list and create", func(t *testing.T) {
		svc := &mockShiftService{}
		r := setupStoreScopedShiftHandler(svc, storeA, "supervisor", []string{string(permissions.ShiftCashMovement)})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/shifts/5/cash-movements", nil))
		require.NotNil(t, svc.lastListMovementScope.StoreID)
		assert.Equal(t, storeA, *svc.lastListMovementScope.StoreID)

		w = httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/shifts/5/cash-movements",
			strings.NewReader(`{"type":"paid_in","amount":1000}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.NotNil(t, svc.lastCreateMovementScope.StoreID)
		assert.Equal(t, storeA, *svc.lastCreateMovementScope.StoreID)
	})
}

// Superadmin carries no store claim, so the scope's store dimension must stay
// nil rather than becoming a filter that matches nothing.
func TestShiftHandler_SuperadminScopeIsUnrestricted(t *testing.T) {
	svc := &mockShiftService{
		reviewShiftFn: func(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
			return &Shift{ID: shiftID}, nil
		},
	}
	r := setupShiftHandlerWithCtx(svc, nil, 7, "superadmin", []string{string(permissions.ShiftReview)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/shifts/5/review", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, svc.lastReviewScope.StoreID)
}
