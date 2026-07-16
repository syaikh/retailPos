package supplier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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
		c.Set("permissions", []string{"pricing:read", "pricing:create", "pricing:update", "pricing:delete"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(_ string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func setupSupplierRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListSuppliers(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/suppliers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Supplier `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_CreateSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	t.Run("success", func(t *testing.T) {
		code := "HDL-SUP-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		body := `{"name":"Handler Supplier","code":"` + code + `","contact_name":"Jane","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Supplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Supplier", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		body := `{"contact_name":"no name or code"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/suppliers", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	repo := NewRepository(dbPool)
	s := &Supplier{
		Name:     "Before Update",
		Code:     "HDL-UPD-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("success", func(t *testing.T) {
		body := `{"name":"After Update","code":"` + s.Code + `","contact_name":"Updated","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/"+strconv.Itoa(s.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Supplier `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "After Update", resp.Data.Name)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/suppliers/abc", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteSupplier(t *testing.T) {
	skipIfNoDB(t)
	r := setupSupplierRouter()

	repo := NewRepository(dbPool)
	s := &Supplier{
		Name:     "Delete Handler",
		Code:     "HDL-DEL-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		IsActive: true,
	}
	require.NoError(t, repo.Create(t.Context(), s))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/"+strconv.Itoa(s.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/suppliers/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
