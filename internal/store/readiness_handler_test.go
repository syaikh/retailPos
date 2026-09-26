package store

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// Package-local port stubs so the readiness route can be exercised without
// importing the owning modules (clean-module-boundary applies to test files
// too) or a database.

type stubStaffCounts map[string]int

func (s stubStaffCounts) StaffCountsByStore(context.Context, shared.DBPool, int) (map[string]int, error) {
	return map[string]int(s), nil
}

type stubStockStats struct {
	active int
	zero   int
}

func (s stubStockStats) SellableStockStats(context.Context, shared.DBPool) (int, int, error) {
	return s.active, s.zero, nil
}

type stubLocationCount int

func (s stubLocationCount) StorageLocationCountByStore(context.Context, shared.DBPool, int) (int, error) {
	return int(s), nil
}

// readinessStubRepo is a store.Repo test double covering only the two calls
// Readiness makes; the embedded Repo keeps the rest of the interface satisfied
// and panics if reached.
type readinessStubRepo struct {
	Repo
	store   *Store
	details *ReadinessDetails
	detErr  error
}

func (r *readinessStubRepo) GetByID(context.Context, int) (*Store, error) {
	return r.store, nil
}

func (r *readinessStubRepo) ReadinessDetails(context.Context, int) (*ReadinessDetails, error) {
	if r.detErr != nil {
		return nil, r.detErr
	}
	return r.details, nil
}

func readinessRouter(auth gin.HandlerFunc, repo Repo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(repo), nil)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), auth, testPermMiddleware)
	return r
}

func TestHandler_Readiness_InvalidIDIsBadRequest(t *testing.T) {
	r := readinessRouter(testAuthMiddleware(),
		&readinessStubRepo{store: readyStore(), details: readyDetails()})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/not-a-number/readiness", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Readiness_CrossStoreIsForbidden(t *testing.T) {
	r := readinessRouter(testAuthMiddlewareWithStore(1),
		&readinessStubRepo{store: readyStore(), details: readyDetails()})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/2/readiness", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "outside your scope")
}

func TestHandler_Readiness_DetailFailureIsInternal(t *testing.T) {
	r := readinessRouter(testAuthMiddleware(),
		&readinessStubRepo{store: readyStore(), detErr: errStub{}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/1/readiness", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

type errStub struct{}

func (errStub) Error() string { return "provider unavailable" }

func TestHandler_Readiness_SuperadminReadsOwnStore(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	repo := NewRepository(dbPool)
	require.NoError(t, repo.Create(ctx, &Store{Name: "Ready Candidate", Address: "Addr", Phone: "0812", IsActive: true}))
	repo.SetStaffCountsProvider(stubStaffCounts{
		"manager": 1, "supervisor": 1, "cashier": 1, "inventory_staff": 1, "finance": 1,
	})
	repo.SetSellableStockProvider(stubStockStats{active: 10, zero: 3})
	repo.SetStorageLocationCountProvider(stubLocationCount(1))

	r := readinessRouter(testAuthMiddleware(), repo)

	var id int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT id FROM stores WHERE name = 'Ready Candidate'`).Scan(&id))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/"+strconv.Itoa(id)+"/readiness", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data Readiness `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, id, resp.Data.StoreID)
	assert.True(t, resp.Data.Ready)
	assert.Empty(t, resp.Data.Blockers)
	assert.Equal(t, 10, resp.Data.Catalog.ActiveProducts)
	assert.Equal(t, 1, resp.Data.StorageLocations)
	assert.Equal(t, RequiredRoles, resp.Data.RequiredRoles)
}
