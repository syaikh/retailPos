package category

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
		c.Set("permissions", []string{"category:create", "category:update", "category:delete", "category:read"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupCategoryRouter() *gin.Engine {
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

func TestHandler_ListCategories(t *testing.T) {
	skipIfNoDB(t)
	r := setupCategoryRouter()

	t.Run("public list returns data", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/categories", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []Category `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})
}

func TestHandler_ListCategoriesManagement(t *testing.T) {
	skipIfNoDB(t)
	r := setupCategoryRouter()

	t.Run("manage list returns data", func(t *testing.T) {
		// Create a category so list is non-empty
		repo := NewRepository(dbPool)
		_ = repo.CreateCategory(context.Background(), &Category{Name: "MgmtListCat", IsActive: true})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/categories/manage", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Category `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
		assert.Greater(t, resp.Total, 0)
	})
}

func TestHandler_CreateCategory(t *testing.T) {
	skipIfNoDB(t)
	r := setupCategoryRouter()

	t.Run("success", func(t *testing.T) {
		body := `{"name":"HandlerTestCat","description":"created via handler"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/categories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Category `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "HandlerTestCat", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/categories", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateCategory(t *testing.T) {
	skipIfNoDB(t)
	r := setupCategoryRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	cat := &Category{Name: "HandlerUpdCat", Slug: "handlerupdcat"}
	err := repo.CreateCategory(ctx, cat)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		body := `{"name":"HandlerUpdCatUpdated"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/categories/"+strconv.Itoa(cat.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_DeleteCategory(t *testing.T) {
	skipIfNoDB(t)
	r := setupCategoryRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	cat := &Category{Name: "HandlerDelCat", Slug: "handlerdelcat"}
	err := repo.CreateCategory(ctx, cat)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/categories/"+strconv.Itoa(cat.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
