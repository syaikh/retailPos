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

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := NewRepository(dbPool)
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
	createFn func(ctx context.Context, log *audit.AuditLog) error
	calls    int
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.AuditLog) error {
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
	repo := NewRepository(dbPool)
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
