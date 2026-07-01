package brand

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
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
		c.Set("permissions", []string{"product:create", "product:update", "product:delete", "product:export", "product:import"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupBrandRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, bus)
	h := NewHandler(svc)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListBrands(t *testing.T) {
	skipIfNoDB(t)
	r := setupBrandRouter()

	t.Run("returns list", func(t *testing.T) {
		repo := NewRepository(dbPool)
		_ = repo.Create(context.Background(), &Brand{Name: "HandlerListBrand", IsActive: true})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/brands", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []Brand `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})
}

func TestHandler_CreateBrand(t *testing.T) {
	skipIfNoDB(t)
	r := setupBrandRouter()

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler Create Brand","description":"test","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/brands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Brand `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Create Brand", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/brands", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing name", func(t *testing.T) {
		body := `{"description":"no name"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/brands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateBrand(t *testing.T) {
	skipIfNoDB(t)
	r := setupBrandRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	b := &Brand{Name: "Handler Before Brand", Description: "Before", IsActive: true}
	require.NoError(t, repo.Create(ctx, b))

	t.Run("success", func(t *testing.T) {
		body := `{"name":"Handler After Brand","description":"After","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/brands/"+strconv.Itoa(b.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Brand `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler After Brand", resp.Data.Name)
		assert.Equal(t, "After", resp.Data.Description)
		assert.True(t, resp.Data.IsActive)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/brands/abc", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteBrand(t *testing.T) {
	skipIfNoDB(t)
	r := setupBrandRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	b := &Brand{Name: "Handler Delete Brand", IsActive: true}
	require.NoError(t, repo.Create(ctx, b))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/brands/"+strconv.Itoa(b.ID), nil)
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
		req, _ := http.NewRequest("DELETE", "/brands/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}


