package storagelocation

import (
	"context"
	"encoding/json"
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
		c.Set("permissions", []string{"storage_location.create", "storage_location.update", "storage_location.delete", "storage_location.view"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func testAuthMiddlewareWithStore(storeID int) gin.HandlerFunc {
	sid := storeID
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "teststoreuser")
		c.Set("roleID", 2)
		c.Set("role", "manager")
		c.Set("permissions", []string{"storage_location.create", "storage_location.update", "storage_location.delete", "storage_location.view"})
		c.Set("storeID", &sid)
		c.Next()
	}
}

func setupRouterWithStore(storeID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := newTestRepository()
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddlewareWithStore(storeID), testPermMiddleware)
	return r
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := newTestRepository()
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func createLocationViaHandler(t *testing.T, r *gin.Engine, code, name string, warehouseID int) int {
	t.Helper()
	body := fmt.Sprintf(`{"code":"%s","name":"%s","warehouse_id":%d}`, code, name, warehouseID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data StorageLocation `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Data.ID
}

func TestHandler_ListStorageLocations(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/storage-locations", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []StorageLocation `json:"data"`
		Total int               `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, 0)
}

func TestHandler_CreateAndGetStorageLocation(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "HANDLER")
	r := setupRouter()

	body := fmt.Sprintf(`{"code":"HANDLER-1","name":"Handler Rack","warehouse_id":%d}`, whID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data StorageLocation `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "HANDLER-1", resp.Data.Code)
	assert.Equal(t, "Handler Rack", resp.Data.Name)
	assert.Greater(t, resp.Data.ID, 0)

	id := resp.Data.ID

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/storage-locations/%d", id), nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("PUT", fmt.Sprintf("/storage-locations/%d", id), strings.NewReader(`{"name":"Updated Rack"}`))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("DELETE", fmt.Sprintf("/storage-locations/%d", id), nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}

func TestHandler_CreateInvalid(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupRouter()

	body := `{"code":"","name":"No Code"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/storage-locations/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_BulkUpdate(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "HANDLER")
	r := setupRouter()

	id1 := createLocationViaHandler(t, r, "HANDLER-BU1", "Bulk Upd 1", whID)
	id2 := createLocationViaHandler(t, r, "HANDLER-BU2", "Bulk Upd 2", whID)

	w := httptest.NewRecorder()
	body := fmt.Sprintf(`{"ids":[%d,%d],"is_active":false}`, id1, id2)
	req, _ := http.NewRequest("PUT", "/storage-locations/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Updated int `json:"updated"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Updated)
}

func TestHandler_ErrorBranches(t *testing.T) {
	skipIfNoDB(t)
	r := setupRouter()

	t.Run("list with is_active filter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/storage-locations?is_active=true", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get by id not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/storage-locations/999999999", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("create bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create validation error", func(t *testing.T) {
		body := `{"code":"   ","name":"No Code","warehouse_id":1}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get by id invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/storage-locations/abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/storage-locations/abc", strings.NewReader(`{"name":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/storage-locations/abc", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/storage-locations/999999999", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update not found", func(t *testing.T) {
		body := `{"name":"Updated"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/storage-locations/999999999", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/storage-locations/999999999", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bulk update bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/storage-locations/bulk", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bulk delete bad json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/storage-locations/bulk", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

type mockAuditCreator struct {
	createFn func(ctx context.Context, log *audit.Log) error
	calls    int
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.Log) error {
	m.calls++
	if m.createFn != nil {
		return m.createFn(ctx, log)
	}
	return nil
}

func TestHandler_BulkDelete(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "HANDLER-BD")
	r := setupRouter()

	id1 := createLocationViaHandler(t, r, "HANDLER-BD1", "Bulk Del 1", whID)
	id2 := createLocationViaHandler(t, r, "HANDLER-BD2", "Bulk Del 2", whID)

	w := httptest.NewRecorder()
	body := fmt.Sprintf(`{"ids":[%d,%d]}`, id1, id2)
	req, _ := http.NewRequest("DELETE", "/storage-locations/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Deleted int `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Deleted)
}

func TestHandler_AuditBranches(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "HANDLER-AUD")

	gin.SetMode(gin.TestMode)
	repo := newTestRepository()
	svc := NewService(repo)
	auditMock := &mockAuditCreator{}
	h := NewHandler(svc, auditMock)
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)

	body := fmt.Sprintf(`{"code":"AUD-1","name":"Audit Loc","warehouse_id":%d}`, whID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data StorageLocation `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	id := resp.Data.ID
	assert.Greater(t, auditMock.calls, 0)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/storage-locations/%d", id), strings.NewReader(`{"name":"Renamed"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/storage-locations/%d", id), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	id2 := createLocationViaHandler(t, r, "AUD-2", "Audit Two", whID)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/storage-locations/bulk", strings.NewReader(fmt.Sprintf(`{"ids":[%d],"is_active":true}`, id2)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/storage-locations/bulk", strings.NewReader(fmt.Sprintf(`{"ids":[%d]}`, id2)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_StoreBoundary403(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	storeA := createTestStore(t, "HANDLER-A")
	storeB := createTestStore(t, "HANDLER-B")
	whA := createTestWarehouseForStore(t, "HANDLER-A-WH", storeA)
	whB := createTestWarehouseForStore(t, "HANDLER-B-WH", storeB)
	centralWH := createTestWarehouse(t, "HANDLER-CENTRAL")

	repo := newTestRepository()
	locB := &StorageLocation{Code: "H403-FOREIGN", Name: "Foreign Loc", StoreID: &storeB, IsActive: true}
	require.NoError(t, repo.Create(context.Background(), locB))
	t.Cleanup(func() { _ = repo.Delete(context.Background(), locB.ID) })

	locViaWhA := &StorageLocation{Code: "H403-VIAWH", Name: "Via WH A", WarehouseID: &whA, IsActive: true}
	require.NoError(t, repo.Create(context.Background(), locViaWhA))
	t.Cleanup(func() { _ = repo.Delete(context.Background(), locViaWhA.ID) })

	var ownID int

	rA := setupRouterWithStore(storeA)

	t.Run("create into foreign warehouse returns 403", func(t *testing.T) {
		body := fmt.Sprintf(`{"code":"H403-1","name":"Foreign WH","warehouse_id":%d}`, whB)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("create into foreign store returns 403", func(t *testing.T) {
		body := fmt.Sprintf(`{"code":"H403-2","name":"Foreign Store","store_id":%d}`, storeB)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("create into central warehouse returns 403", func(t *testing.T) {
		body := fmt.Sprintf(`{"code":"H403-CWH","name":"Central WH","warehouse_id":%d}`, centralWH)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("create into own store succeeds", func(t *testing.T) {
		body := fmt.Sprintf(`{"code":"H403-OWN","name":"Own Store","store_id":%d}`, storeA)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/storage-locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data StorageLocation `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		ownID = resp.Data.ID
	})

	t.Run("get foreign id returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/storage-locations/%d", locB.ID), nil)
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("update foreign id returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/storage-locations/%d", locB.ID), strings.NewReader(`{"name":"Hijacked"}`))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("update own row returns 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/storage-locations/%d", ownID), strings.NewReader(`{"name":"Own Renamed"}`))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete foreign id returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/storage-locations/%d", locB.ID), nil)
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		got, err := repo.GetByID(context.Background(), locB.ID)
		require.NoError(t, err)
		assert.Equal(t, "Foreign Loc", got.Name)
	})

	t.Run("bulk update with foreign id returns 403", func(t *testing.T) {
		own := &StorageLocation{Code: "H403-BULK-OWN", Name: "Bulk Own", StoreID: &storeA, IsActive: true}
		require.NoError(t, repo.Create(context.Background(), own))
		t.Cleanup(func() { _ = repo.Delete(context.Background(), own.ID) })

		w := httptest.NewRecorder()
		body := fmt.Sprintf(`{"ids":[%d,%d],"is_active":false}`, own.ID, locB.ID)
		req, _ := http.NewRequest("PUT", "/storage-locations/bulk", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		got, err := repo.GetByID(context.Background(), own.ID)
		require.NoError(t, err)
		assert.True(t, got.IsActive, "own row must not be partially written")
	})

	t.Run("bulk delete with foreign id returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := fmt.Sprintf(`{"ids":[%d,%d]}`, ownID, locB.ID)
		req, _ := http.NewRequest("DELETE", "/storage-locations/bulk", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		_, err := repo.GetByID(context.Background(), ownID)
		require.NoError(t, err, "own row must survive")
		_, err = repo.GetByID(context.Background(), locB.ID)
		require.NoError(t, err, "foreign row must survive")
	})

	t.Run("list is scoped to own store", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/storage-locations?limit=100", nil)
		rA.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []StorageLocation `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		ids := make([]int, 0, len(resp.Data))
		for _, l := range resp.Data {
			ids = append(ids, l.ID)
		}
		assert.Contains(t, ids, ownID, "own store-linked row must appear")
		assert.Contains(t, ids, locViaWhA.ID, "own warehouse-linked row must appear")
		assert.NotContains(t, ids, locB.ID, "foreign row must not appear in store-scoped list")
	})
}
