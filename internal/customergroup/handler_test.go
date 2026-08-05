package customergroup

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
		c.Set("permissions", []string{"customer_group.create", "customer_group.update", "customer_group.delete", "customer_group.view"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupCGRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListCustomerGroups(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/customer-groups", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []CustomerGroup `json:"data"`
		Total int             `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, 0)
}

func TestHandler_CreateAndGetCustomerGroup(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	body := `{"name":"Handler Test CG","description":"test desc"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data CustomerGroup `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Handler Test CG", resp.Data.Name)
	assert.Greater(t, resp.Data.ID, 0)

	id := resp.Data.ID

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/customer-groups/%d", id), nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("PUT", fmt.Sprintf("/customer-groups/%d", id), strings.NewReader(`{"name":"Updated CG"}`))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("DELETE", fmt.Sprintf("/customer-groups/%d", id), nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}

func TestHandler_ListWithHasCustomersFilter(t *testing.T) {
	skipIfNoDB(t)

	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/customer-groups?has_customers=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []CustomerGroup `json:"data"`
		Total int             `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, 0)
}

func TestHandler_CreateWithColor(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	body := `{"name":"Handler Color CG","description":"with color","color":"#FF5733"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data CustomerGroup `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Handler Color CG", resp.Data.Name)
	assert.Equal(t, "#FF5733", resp.Data.Color)

	// Cleanup
	_ = deleteGroupByID(r, resp.Data.ID)
}

func TestHandler_BulkUpdate(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	// Create groups to bulk-update
	id1 := createGroupViaHandler(t, r, "Bulk Upd 1")
	id2 := createGroupViaHandler(t, r, "Bulk Upd 2")
	defer func() {
		_ = deleteGroupByID(r, id1)
		_ = deleteGroupByID(r, id2)
	}()

	body := fmt.Sprintf(`{"ids":[%d,%d],"is_active":false}`, id1, id2)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/customer-groups/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Updated int `json:"updated"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Updated)
}

func TestHandler_BulkUpdateEmptyIds(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	body := `{"ids":[],"is_active":false}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/customer-groups/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BulkDelete(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	id1 := createGroupViaHandler(t, r, "Bulk Del 1")
	id2 := createGroupViaHandler(t, r, "Bulk Del 2")

	body := fmt.Sprintf(`{"ids":[%d,%d]}`, id1, id2)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/customer-groups/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Deleted int `json:"deleted"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Deleted)
}

func TestHandler_BulkDeleteEmptyIds(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	body := `{"ids":[]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/customer-groups/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetByID_Success(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	id := createGroupViaHandler(t, r, "GetByID Success")
	defer func() { _ = deleteGroupByID(r, id) }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/customer-groups/%d", id), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data CustomerGroup `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, id, resp.Data.ID)
}

func TestHandler_GetByID_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/customer-groups/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/customer-groups/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Create_MissingName(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader(`{"description":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_Success(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	id := createGroupViaHandler(t, r, "Update Success")
	defer func() { _ = deleteGroupByID(r, id) }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/customer-groups/%d", id), strings.NewReader(`{"name":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/customer-groups/abc", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/customer-groups/999999", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_InvalidJSON(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	id := createGroupViaHandler(t, r, "Update InvalidJSON")
	defer func() { _ = deleteGroupByID(r, id) }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/customer-groups/%d", id), strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/customer-groups/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/customer-groups/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListWithIsActiveFilter(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/customer-groups?is_active=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []CustomerGroup `json:"data"`
		Total int             `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, 0)
}

// mockAuditCreator is a test spy that records audit calls
type mockAuditCreator struct {
	called bool
}

func (m *mockAuditCreator) CreateAuditLog(_ context.Context, _ *audit.AuditLog) error {
	m.called = true
	return nil
}

func setupCGRouterWithAudit() (*gin.Engine, *mockAuditCreator) {
	gin.SetMode(gin.TestMode)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	mockAudit := &mockAuditCreator{}
	h := NewHandler(svc, mockAudit)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r, mockAudit
}

func TestHandler_Create_DuplicateName(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	body := `{"name":"Duplicate Test CG"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data CustomerGroup `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	id := resp.Data.ID
	defer func() { _ = deleteGroupByID(r, id) }()

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestHandler_Create_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	r, mockAudit := setupCGRouterWithAudit()

	id := createGroupViaHandler(t, r, "Audit Create Group")
	defer func() { _ = deleteGroupByID(r, id) }()

	assert.True(t, mockAudit.called)
}

func TestHandler_Update_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	r, mockAudit := setupCGRouterWithAudit()

	id := createGroupViaHandler(t, r, "Audit Update Group")
	defer func() { _ = deleteGroupByID(r, id) }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/customer-groups/%d", id), strings.NewReader(`{"name":"Audit Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockAudit.called)
}

func TestHandler_Delete_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	r, mockAudit := setupCGRouterWithAudit()

	id := createGroupViaHandler(t, r, "Audit Delete Group")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/customer-groups/%d", id), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockAudit.called)
}

func TestHandler_BulkUpdate_InvalidJSON(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/customer-groups/bulk", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BulkDelete_InvalidJSON(t *testing.T) {
	skipIfNoDB(t)
	r := setupCGRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/customer-groups/bulk", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BulkUpdate_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	r, mockAudit := setupCGRouterWithAudit()

	id1 := createGroupViaHandler(t, r, "BulkAudit1")
	id2 := createGroupViaHandler(t, r, "BulkAudit2")
	defer func() {
		_ = deleteGroupByID(r, id1)
		_ = deleteGroupByID(r, id2)
	}()

	body := fmt.Sprintf(`{"ids":[%d,%d],"is_active":false}`, id1, id2)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/customer-groups/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockAudit.called)
}

func TestHandler_BulkDelete_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	r, mockAudit := setupCGRouterWithAudit()

	id1 := createGroupViaHandler(t, r, "BulkDelAudit1")
	id2 := createGroupViaHandler(t, r, "BulkDelAudit2")

	body := fmt.Sprintf(`{"ids":[%d,%d]}`, id1, id2)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/customer-groups/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockAudit.called)
}

func createGroupViaHandler(t *testing.T, r *gin.Engine, name string) int {
	t.Helper()
	body := fmt.Sprintf(`{"name":"%s","description":"test"}`, name)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/customer-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data CustomerGroup `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Data.ID
}

func deleteGroupByID(r *gin.Engine, id int) error {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/customer-groups/%d", id), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		return fmt.Errorf("delete failed with status %d", w.Code)
	}
	return nil
}
