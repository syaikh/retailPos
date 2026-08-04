package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

func init() {
	os.Setenv("JWT_SECRET", "test-secret-for-audit-tests")
}

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
		c.Set("permissions", []string{"audit.view"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupAuditRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListAuditLogs(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
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
		require.NoError(t, repo.CreateAuditLog(ctx, &AuditLog{
			UserID:     intPtr(999),
			Role:       "admin",
			Action:     "handler_user_filter",
			EntityType: "user",
		}))
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

	t.Run("filters by entity_id", func(t *testing.T) {
		eid := 42
		require.NoError(t, repo.CreateAuditLog(ctx, &AuditLog{
			Role:       "admin",
			Action:     "handler_eid_filter",
			EntityType: "order",
			EntityID:   &eid,
		}))
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?entity_id=42&entity_type=order", nil)
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
			require.NotNil(t, l.EntityID)
			assert.Equal(t, 42, *l.EntityID)
		}
	})

	t.Run("filters by invalid entity_id ignores gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?entity_id=abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_ExportAuditLogs(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
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

func TestHandler_GetAuditLog(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &AuditLog{
		Role:       "admin",
		Action:     "handler_getbyid_test",
		EntityType: "product",
		IPAddress:  "10.0.0.3",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))

	t.Run("found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/"+strconv.Itoa(al.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data AuditLog `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "handler_getbyid_test", resp.Data.Action)
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/999999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_ListEntityTypes(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	require.NoError(t, repo.CreateAuditLog(ctx, &AuditLog{
		Role:       "admin",
		Action:     "entity_type_test",
		EntityType: "widget",
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs/entity-types", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []string `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Data, "widget")
}

func TestHandler_ListAuditLogs_CreatedAtJakartaTimezone(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &AuditLog{
		Role:       "admin",
		Action:     "handler_tz_test_" + time.Now().Format("0102150405"),
		EntityType: "product",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs?limit=10&action="+al.Action, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []AuditLogListItem `json:"data"`
		Total int                `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)

	jakartaFormat := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+07:00$`)
	assert.Regexp(t, jakartaFormat, resp.Data[0].CreatedAt, "CreatedAt should be in Jakarta timezone format")
}

func TestHandler_ExportCSV_CreatedAtJakartaTimezone(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &AuditLog{
		Role:       "admin",
		Action:     "export_tz_test_" + time.Now().Format("0102150405"),
		EntityType: "product",
		IPAddress:  "10.0.0.99",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs/export?format=csv&action="+al.Action, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	jakartaTimestamp := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	assert.Regexp(t, jakartaTimestamp, body, "CSV export should contain Jakarta timezone formatted timestamp")
}
