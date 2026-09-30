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
	_ = os.Setenv("JWT_SECRET", "test-secret-for-audit-tests")
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func testAuthMiddleware() gin.HandlerFunc {
	return testAuthMiddlewareWithStore(nil, "superadmin")
}

// testAuthMiddlewareWithStore builds a store-scoped auth context. storeID nil
// models superadmin (no store claim); a non-nil id models a manager pinned to
// one store, which is the only way a caller can be denied a foreign audit log.
func testAuthMiddlewareWithStore(storeID *int, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", role)
		c.Set("permissions", []string{"audit.view"})
		c.Set("storeID", storeID)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupAuditRouter() *gin.Engine {
	return setupAuditRouterWithStore(nil, "superadmin")
}

// setupAuditRouterWithStore wires the real repository behind either a superadmin
// context (no store claim) or a store-scoped one, so the read-by-id store filter
// is exercised through the handler rather than only at the repository layer.
func setupAuditRouterWithStore(storeID *int, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// testAuthMiddleware acts as user id=1; ensure that row exists so the
	// fail-closed audit_exported write (which references users.id) succeeds.
	if dbPool != nil {
		_, _ = dbPool.Exec(context.Background(), `INSERT INTO users (id, username, email, password_hash, role_id) VALUES (1, 'testuser', 'testuser@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`)
	}

	repo := NewRepository(dbPool)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddlewareWithStore(storeID, role), testPermMiddleware)
	return r
}

func TestHandler_ListAuditLogs(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &Log{
		Role:       "manager",
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
			Data  []Log `json:"data"`
			Total int   `json:"total"`
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
			Data  []Log `json:"data"`
			Total int   `json:"total"`
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
			Data  []Log `json:"data"`
			Total int   `json:"total"`
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
		require.NoError(t, repo.CreateAuditLog(ctx, &Log{
			UserID:     intPtr(999),
			Role:       "manager",
			Action:     "handler_user_filter",
			EntityType: "user",
		}))
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?user_id="+strconv.Itoa(999), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Log `json:"data"`
			Total int   `json:"total"`
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
		require.NoError(t, repo.CreateAuditLog(ctx, &Log{
			Role:       "manager",
			Action:     "handler_eid_filter",
			EntityType: "order",
			EntityID:   &eid,
		}))
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs?entity_id=42&entity_type=order", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []Log `json:"data"`
			Total int   `json:"total"`
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

	require.NoError(t, repo.CreateAuditLog(ctx, &Log{
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

	t.Run("exports with id filters", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/export?format=csv&user_id=1&entity_id=2", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_GetAuditLog(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &Log{
		Role:       "manager",
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
			Data Log `json:"data"`
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

// TestHandler_GetAuditLog_StoreBoundary proves the boundary holds through the
// handler, not just in SQL. A store-scoped caller gets 404 (not 403) for another
// store's row: 403 would confirm the id exists and turn this endpoint into an
// existence oracle for audit-log ids across the whole chain.
func TestHandler_GetAuditLog_StoreBoundary(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()

	newStore := func(t *testing.T, name string) int {
		t.Helper()
		var id int
		require.NoError(t, dbPool.QueryRow(ctx,
			`INSERT INTO stores (name, is_active) VALUES ($1, true) RETURNING id`, name).Scan(&id))
		return id
	}
	ownStore := newStore(t, "audit_handler_own")
	otherStore := newStore(t, "audit_handler_other")

	var userID int
	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ('audit_handler_user', 'audit_handler@test.com', 'hash', 1)
		ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&userID))

	repo := NewRepository(dbPool)
	mkLog := func(t *testing.T, action string, storeID *int) *Log {
		t.Helper()
		al := &Log{UserID: &userID, StoreID: storeID, Role: "manager", Action: action, EntityType: "product"}
		require.NoError(t, repo.CreateAuditLog(ctx, al))
		return al
	}
	own := mkLog(t, "test_action_handler_own", &ownStore)
	other := mkLog(t, "test_action_handler_other", &otherStore)
	global := mkLog(t, "test_action_handler_global", nil)

	managerRouter := setupAuditRouterWithStore(&ownStore, "manager")
	superRouter := setupAuditRouter()

	get := func(r *gin.Engine, id int) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/audit-logs/"+strconv.Itoa(id), nil)
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("own store row is 200", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get(managerRouter, own.ID).Code)
	})

	t.Run("foreign store row is 404 not 403", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(managerRouter, other.ID).Code)
	})

	t.Run("global row is visible to store scoped caller", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get(managerRouter, global.ID).Code)
	})

	t.Run("superadmin reads every store", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get(superRouter, own.ID).Code)
		assert.Equal(t, http.StatusOK, get(superRouter, other.ID).Code)
		assert.Equal(t, http.StatusOK, get(superRouter, global.ID).Code)
	})
}

func TestHandler_ListEntityTypes(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	require.NoError(t, repo.CreateAuditLog(ctx, &Log{
		Role:       "manager",
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

	al := &Log{
		Role:       "manager",
		Action:     "handler_tz_test_" + time.Now().Format("0102150405"),
		EntityType: "product",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs?limit=10&action="+al.Action, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  []LogListItem `json:"data"`
		Total int           `json:"total"`
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

	al := &Log{
		Role:       "manager",
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

func TestHandler_ExportAuditLogs_EmitsExportedEvent(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	r := setupAuditRouter()
	ctx := context.Background()

	// The acting user (id=1 from testAuthMiddleware) must exist for the
	// fail-closed audit_exported write to satisfy the FK constraint.
	_, err := dbPool.Exec(ctx, `INSERT INTO users (id, username, email, password_hash, role_id) VALUES (1, 'export_test_user', 'export_test@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING`)
	require.NoError(t, err)

	repo := NewRepository(dbPool)
	require.NoError(t, repo.CreateAuditLog(ctx, &Log{
		Role:       "manager",
		Action:     "export_event_seed_" + time.Now().Format("0102150405"),
		EntityType: "product",
		IPAddress:  "10.0.0.99",
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs/export?format=csv", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cnt int
	err = dbPool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action = 'audit_exported' AND user_id = 1`).Scan(&cnt)
	require.NoError(t, err)
	assert.Equal(t, 1, cnt, "export must write exactly one audit_exported row for the acting user")

	var entityType string
	err = dbPool.QueryRow(ctx, `SELECT entity_type FROM audit_logs WHERE action = 'audit_exported' AND user_id = 1`).Scan(&entityType)
	require.NoError(t, err)
	assert.Equal(t, "audit", entityType)
}
