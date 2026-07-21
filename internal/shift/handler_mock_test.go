package shift

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

type mockShiftService struct {
	openShiftFn        func(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error)
	closeShiftFn       func(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	getActiveShiftFn   func(ctx context.Context, userID int) (*Shift, error)
	listShiftsFn       func(ctx context.Context, userID *int, status string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error)
	getShiftByIDFn     func(ctx context.Context, shiftID int) (*Shift, error)
	reviewShiftFn      func(ctx context.Context, shiftID, reviewerID int) (*Shift, error)
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
func (m *mockShiftService) ListShifts(ctx context.Context, userID *int, status string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
	return m.listShiftsFn(ctx, userID, status, limit, offset, sortBy, sortDir)
}
func (m *mockShiftService) GetShiftByID(ctx context.Context, shiftID int) (*Shift, error) {
	return m.getShiftByIDFn(ctx, shiftID)
}
func (m *mockShiftService) ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
	return m.reviewShiftFn(ctx, shiftID, reviewerID)
}

type mockAudit struct {
	createAuditLogFn func(ctx context.Context, log *audit.AuditLog) error
}

func (m *mockAudit) CreateAuditLog(ctx context.Context, log *audit.AuditLog) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupShiftHandler(svc ShiftService, auditSvc audit.AuditCreator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
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
		createAuditLogFn: func(ctx context.Context, log *audit.AuditLog) error {
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
