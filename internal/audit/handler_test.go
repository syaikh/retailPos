package audit

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
		c.Set("permissions", []string{"audit:read"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupAuditRouter() *gin.Engine {
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

func TestHandler_ListAuditLogs(t *testing.T) {
	skipIfNoDB(t)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &AuditLog{
		Role:       "admin",
		Action:     "handler_list_test",
		EntityType: "order",
		IPAddress:  "10.0.0.1",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))

	t.Run("returns audit logs", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []AuditLog `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, resp.Total, 1)
		assert.GreaterOrEqual(t, len(resp.Data), 1)
	})

	t.Run("filters by action", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?action=handler_list_test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []AuditLog `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, "handler_list_test", resp.Data[0].Action)
	})

	t.Run("filters by entity_type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?entity_type=order", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []AuditLog `json:"data"`
			Total int        `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, resp.Total, 1)
		for _, l := range resp.Data {
			assert.Equal(t, "order", l.EntityType)
		}
	})

	t.Run("filters by user_id", func(t *testing.T) {
		var err error
		_, err = dbPool.Exec(ctx, `INSERT INTO users (id, username, email, password_hash, role_id) VALUES (999, 'audit_user', 'audit_user@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`)
		require.NoError(t, err)
		repo.CreateAuditLog(ctx, &AuditLog{
			UserID:     intPtr(999),
			Role:       "admin",
			Action:     "handler_user_filter",
			EntityType: "user",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?user_id="+strconv.Itoa(999), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []AuditLog `json:"data"`
			Total int        `json:"total"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, resp.Total, 1)
		for _, l := range resp.Data {
			require.NotNil(t, l.UserID)
			assert.Equal(t, 999, *l.UserID)
		}
	})
}

func TestHandler_ExportAuditLogs(t *testing.T) {
	skipIfNoDB(t)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	require.NoError(t, repo.CreateAuditLog(ctx, &AuditLog{
		Action:     "handler_export_test",
		EntityType: "report",
		IPAddress:  "10.0.0.2",
	}))

	t.Run("exports as csv", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/export?format=csv", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, w.Body.String(), "handler_export_test")
	})

	t.Run("exports as xlsx", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/export?format=xlsx", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
	})

	t.Run("exports with filters", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/export?format=csv&action=handler_export_test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "handler_export_test")
	})
}
