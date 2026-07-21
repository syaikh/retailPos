package uom

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
		c.Set("permissions", []string{"product.create", "product.update", "product.delete", "product:export", "product:import"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupUOMRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)

	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	h.RegisterPublicRoutes(r.Group("/"))
	return r
}

func TestHandler_ListUnitsOfMeasure(t *testing.T) {
	skipIfNoDB(t)
	r := setupUOMRouter()

	t.Run("returns list", func(t *testing.T) {
		repo := NewRepository(dbPool)
		_ = repo.Create(context.Background(), &UnitOfMeasure{Code: "HDLST", Name: "Handler List UOM", IsActive: true})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/units-of-measure", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []UnitOfMeasure `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})
}

func TestHandler_CreateUnitOfMeasure(t *testing.T) {
	skipIfNoDB(t)
	r := setupUOMRouter()

	t.Run("success", func(t *testing.T) {
		body := `{"code":"HDLCREATE","name":"Handler Create UOM","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/units-of-measure", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data UnitOfMeasure `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler Create UOM", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/units-of-measure", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		body := `{"description":"no code or name"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/units-of-measure", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_UpdateUnitOfMeasure(t *testing.T) {
	skipIfNoDB(t)
	r := setupUOMRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	u := &UnitOfMeasure{Code: "HDLUP", Name: "Handler Before UOM", IsActive: true}
	require.NoError(t, repo.Create(ctx, u))

	t.Run("success", func(t *testing.T) {
		body := `{"code":"HDLUP","name":"Handler After UOM","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/units-of-measure/"+strconv.Itoa(u.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data UnitOfMeasure `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Handler After UOM", resp.Data.Name)
		assert.Equal(t, "HDLUP", resp.Data.Code)
		assert.True(t, resp.Data.IsActive)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/units-of-measure/abc", strings.NewReader(`{"code":"X","name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteUnitOfMeasure(t *testing.T) {
	skipIfNoDB(t)
	r := setupUOMRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	u := &UnitOfMeasure{Code: "HDLDL", Name: "Handler Delete UOM", IsActive: true}
	require.NoError(t, repo.Create(ctx, u))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/units-of-measure/"+strconv.Itoa(u.ID), nil)
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
		req, _ := http.NewRequest("DELETE", "/units-of-measure/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
