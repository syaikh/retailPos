package store

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
		c.Set("permissions", []string{"store:create", "store:update", "store:delete", "store:read"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
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

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	body := fmt.Sprintf(`{"name":"Handler Updated"}`)
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

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
