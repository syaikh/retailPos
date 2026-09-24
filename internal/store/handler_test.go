package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"store.create", "store.update", "store.delete", "store.view"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// testAuthMiddlewareWithStore mimics a store-scoped (non-superadmin) caller:
// typed *int store claim, exactly as AuthMiddleware sets it.
func testAuthMiddlewareWithStore(storeID int) gin.HandlerFunc {
	sid := storeID
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "scopeduser")
		c.Set("roleID", 2)
		c.Set("role", "manager")
		c.Set("permissions", []string{"store.view"})
		c.Set("storeID", &sid)
		c.Next()
	}
}

func setupStoreRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func setupStoreRouterWithAuth(auth gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), auth, testPermMiddleware)
	return r
}

func TestHandler_ListWarehouses(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	var whID int
	err := dbPool.QueryRow(ctx, `INSERT INTO warehouses (name, code, is_active) VALUES ('Handler WH', 'HWH01', true) RETURNING id`).Scan(&whID)
	require.NoError(t, err)

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/warehouses", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Warehouse `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
	found := false
	for _, wh := range resp.Data {
		if wh.ID == whID {
			found = true
		}
	}
	assert.True(t, found)
}

func TestHandler_ListWarehouses_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	var storeA, storeB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeWh Store A', true) RETURNING id`).Scan(&storeA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeWh Store B', true) RETURNING id`).Scan(&storeB))
	var whA, whB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO warehouses (name, code, store_id, is_active) VALUES ('Scope WH A', 'SWHA01', $1, true) RETURNING id`, storeA).Scan(&whA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO warehouses (name, code, store_id, is_active) VALUES ('Scope WH B', 'SWHB01', $1, true) RETURNING id`, storeB).Scan(&whB))

	// Store-scoped caller only sees their own store's warehouses.
	rScoped := setupStoreRouterWithAuth(testAuthMiddlewareWithStore(storeA))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/warehouses", nil)
	rScoped.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Warehouse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	ids := make([]int, 0, len(resp.Data))
	for _, wh := range resp.Data {
		ids = append(ids, wh.ID)
	}
	assert.Contains(t, ids, whA)
	assert.NotContains(t, ids, whB)

	// Superadmin (nil store) sees every warehouse.
	rSuper := setupStoreRouter()
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/warehouses", nil)
	rSuper.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	var resp2 struct {
		Data []Warehouse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	ids2 := make([]int, 0, len(resp2.Data))
	for _, wh := range resp2.Data {
		ids2 = append(ids2, wh.ID)
	}
	assert.Contains(t, ids2, whA)
	assert.Contains(t, ids2, whB)
}

func TestHandler_ListStores_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	var storeA, storeB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeList Store A', true) RETURNING id`).Scan(&storeA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeList Store B', true) RETURNING id`).Scan(&storeB))

	r := setupStoreRouterWithAuth(testAuthMiddlewareWithStore(storeA))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var listResp struct {
		Data  []Store `json:"data"`
		Total int     `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	assert.Equal(t, 1, listResp.Total)
	require.Len(t, listResp.Data, 1)
	assert.Equal(t, storeA, listResp.Data[0].ID)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/stores/active", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	var activeResp struct {
		Data []Store `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &activeResp))
	require.Len(t, activeResp.Data, 1)
	assert.Equal(t, storeA, activeResp.Data[0].ID)
}

func TestHandler_ListStores(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []Store `json:"data"`
		Total int     `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, 0)
}

func TestHandler_CreateAndGetStore(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	body := `{"name":"Handler Test Store","address":"456 Oak Ave","phone":"08987654321"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stores", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data Store `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Handler Test Store", resp.Data.Name)
	assert.Greater(t, resp.Data.ID, 0)

	id := resp.Data.ID

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/stores/%d", id), nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("PUT", fmt.Sprintf("/stores/%d", id), strings.NewReader(`{"name":"Updated Store"}`))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("DELETE", fmt.Sprintf("/stores/%d", id), nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}

func TestHandler_ListActive(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/active", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Store `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_GetByID_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	s := &Store{Name: "Handler GetByID", Address: "Addr", Phone: "111", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/stores/%d", s.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Store `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Handler GetByID", resp.Data.Name)

	_ = repo.Delete(ctx, s.ID)
}

func TestHandler_GetByID_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid id")
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "store not found")
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stores", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Create_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stores", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/stores/abc", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid id")
}

func TestHandler_Update_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/stores/999999", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "store not found")
}

func TestHandler_Update_InvalidJSON(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()
	s := &Store{Name: "Handler Update Invalid", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/stores/%d", s.ID), strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_ = repo.Delete(ctx, s.ID)
}

func TestHandler_Update_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()
	s := &Store{Name: "Handler Update OK", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	body := `{"name":"Handler Updated"}`
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/stores/%d", s.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Store `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Handler Updated", resp.Data.Name)

	_ = repo.Delete(ctx, s.ID)
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/stores/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid id")
}

func TestHandler_Delete_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/stores/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "store not found")
}

func TestHandler_Delete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()
	s := &Store{Name: "Handler Delete OK", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/stores/%d", s.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "deleted", resp["status"])
}

func TestHandler_List_WithIsActiveFilter(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	s := &Store{Name: "Filter Active Handler", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores?is_active=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []Store `json:"data"`
		Total int     `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, 1)

	_ = repo.Delete(ctx, s.ID)
}

func TestHandler_List_IsActiveFalse(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	s := &Store{Name: "Filter Inactive Handler", IsActive: false}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores?is_active=false", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []Store `json:"data"`
		Total int     `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	for _, st := range resp.Data {
		assert.False(t, st.IsActive)
	}

	_ = repo.Delete(ctx, s.ID)
}

type mockStoreAudit struct {
	called bool
}

func (m *mockStoreAudit) CreateAuditLog(_ context.Context, _ *audit.Log) error {
	m.called = true
	return nil
}

func setupStoreRouterWithAudit() (*gin.Engine, *mockStoreAudit) {
	gin.SetMode(gin.TestMode)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	mockAudit := &mockStoreAudit{}
	h := NewHandler(svc, mockAudit)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r, mockAudit
}

func TestHandler_Create_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	r, mockAudit := setupStoreRouterWithAudit()

	body := `{"name":"Audit Store","address":"Addr","phone":"123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stores", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, mockAudit.called)

	var resp struct {
		Data Store `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	repo := NewRepository(dbPool)
	_ = repo.Delete(context.Background(), resp.Data.ID)
}

func TestHandler_Update_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()
	s := &Store{Name: "Audit Update Store", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r, mockAudit := setupStoreRouterWithAudit()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/stores/%d", s.ID), strings.NewReader(`{"name":"Audit Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockAudit.called)

	_ = repo.Delete(ctx, s.ID)
}

func TestHandler_Delete_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()
	s := &Store{Name: "Audit Delete Store", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r, mockAudit := setupStoreRouterWithAudit()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/stores/%d", s.ID), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockAudit.called)
}

func TestHandler_List_WithSearch(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	s := &Store{Name: "SearchableUniqueXyz", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores?search=SearchableUniqueXyz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []Store `json:"data"`
		Total int     `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)

	_ = repo.Delete(ctx, s.ID)
}

func TestHandler_ListActive_ActiveOnly(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	sActive := &Store{Name: "ListActive Yes", IsActive: true}
	sInactive := &Store{Name: "ListActive No", IsActive: false}
	require.NoError(t, repo.Create(ctx, sActive))
	require.NoError(t, repo.Create(ctx, sInactive))

	r := setupStoreRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/active", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Store `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	for _, st := range resp.Data {
		assert.True(t, st.IsActive)
	}

	_ = repo.Delete(ctx, sActive.ID)
	_ = repo.Delete(ctx, sInactive.ID)
}

func TestHandler_Create_Success(t *testing.T) {
	skipIfNoDB(t)
	r := setupStoreRouter()

	body := `{"name":"Handler Create Solo","address":"Solo Addr","phone":"777"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stores", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data Store `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Handler Create Solo", resp.Data.Name)
	assert.True(t, resp.Data.IsActive)

	repo := NewRepository(dbPool)
	_ = repo.Delete(context.Background(), resp.Data.ID)
}

func TestHandler_GetByID_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	var storeA, storeB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeByID Store A', true) RETURNING id`).Scan(&storeA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeByID Store B', true) RETURNING id`).Scan(&storeB))

	r := setupStoreRouterWithAuth(testAuthMiddlewareWithStore(storeA))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/stores/%d", storeA), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Store `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, storeA, resp.Data.ID)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/stores/%d", storeB), nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code,
		"a cross-store read must be 403, matching the scoped list")

	// Superadmin (nil store) reads any store; a missing id is a real 404.
	rSuper := setupStoreRouter()
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", fmt.Sprintf("/stores/%d", storeB), nil)
	rSuper.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/stores/999999999", nil)
	rSuper.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusNotFound, w4.Code)
}

func TestHandler_Mutations_StoreScoped(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	var storeA, storeB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeMut Store A', true) RETURNING id`).Scan(&storeA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('ScopeMut Store B', true) RETURNING id`).Scan(&storeB))

	r := setupStoreRouterWithAuth(testAuthMiddlewareWithStore(storeA))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/stores/%d", storeB), strings.NewReader(`{"name":"Hijacked"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code,
		"a cross-store update must be 403 before any write")

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/stores/%d", storeB), nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code,
		"a cross-store delete must be 403")

	repo := NewRepository(dbPool)
	got, err := repo.GetByID(ctx, storeB)
	require.NoError(t, err, "foreign store must survive a scoped delete attempt")
	assert.Equal(t, "ScopeMut Store B", got.Name)
}

// failingRepo fails every call it overrides; used to prove repository
// outages surface as 500 (ErrInternal), never as 400/404. Unoverridden
// methods come from the embedded nil interface and must not be reached.
type failingRepo struct {
	Repo
}

func (failingRepo) GetByID(context.Context, int) (*Store, error) {
	return nil, errors.New("db down")
}

func (failingRepo) Create(context.Context, *Store) error {
	return errors.New("db down")
}

func newFailingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(failingRepo{}), nil)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_GetByID_DBErrorReturns500(t *testing.T) {
	r := newFailingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stores/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code,
		"a repository failure during get must surface as 500, not 400/404")
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestHandler_Create_DBErrorReturns500(t *testing.T) {
	r := newFailingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/stores", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code,
		"a repository failure during create must surface as 500, not 400")
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestHandler_Update_DBErrorReturns500(t *testing.T) {
	r := newFailingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/stores/1", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code,
		"a repository failure during update must surface as 500, not 400")
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestHandler_Delete_DBErrorReturns500(t *testing.T) {
	r := newFailingRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/stores/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code,
		"a repository failure during delete must surface as 500, not 400")
	assert.Contains(t, w.Body.String(), "internal server error")
}
