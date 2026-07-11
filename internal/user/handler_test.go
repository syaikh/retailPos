package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/shared"

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
		c.Set("permissions", []string{
			"user:read", "user:create", "user:update", "user:delete",
			"role:read", "role:create",
		})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupUserRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)

	svc := NewService(repo)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_ListUsers(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()
	u := &User{
		Username: "hdllistuser",
		Email:    "hdllistuser@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("returns users list", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/users?limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []User `json:"data"`
			Total int    `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Greater(t, resp.Total, 0)
	})

	t.Run("returns empty when no match", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/users?search=NONEXISTENT_XYZ", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []User `json:"data"`
			Total int    `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})
}

func TestHandler_CreateUser(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	t.Run("success", func(t *testing.T) {
		body := `{"username":"hdlcreate1","email":"hdlcreate1@test.com","password":"secret123","role_id":1}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "hdlcreate1", resp.Data.Username)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/users", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		body := `{"username":"hdlcreatemiss"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate username", func(t *testing.T) {
		body := `{"username":"hldup","email":"hldup@test.com","password":"secret123","role_id":1}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusConflict, w2.Code)
	})
}

func TestHandler_UpdateUser(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()
	u := &User{
		Username: "hdlupdatebefore",
		Email:    "hdlupdate@test.com",
		Password: hash,
		RoleID:   2,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("success", func(t *testing.T) {
		body := `{"username":"hdlupdateafter","email":"hdlupdate_new@test.com","role_id":3}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/users/"+strconv.Itoa(u.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"username":"nobody"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/users/999999", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		body := `{"username":"nobody"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/users/abc", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteUser(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()
	u := &User{
		Username: "hlddelete",
		Email:    "hlddelete@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, u))

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/admin/users/"+strconv.Itoa(u.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})
}

func TestHandler_ListRoles(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/roles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Role `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func setupUserRouterWithAudit() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	auditRepo := audit.NewRepository(dbPool)

	svc := NewService(repo)
	auditSvc := audit.NewService(auditRepo)
	h := NewHandler(svc, auditSvc)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func countAuditLogs(t *testing.T, entityType string) int {
	t.Helper()
	var count int
	err := dbPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = $1`, entityType).Scan(&count)
	require.NoError(t, err)
	return count
}

func ensureTestUserExists(t *testing.T) {
	t.Helper()
	_, err := dbPool.Exec(context.Background(),
		`INSERT INTO users (id, username, email, password_hash, role_id, is_active)
		 VALUES (1, 'test_actor', 'test_actor@test.com', 'hash', 1, true)
		 ON CONFLICT (id) DO NOTHING`)
	require.NoError(t, err)
}

func TestHandler_CreateUser_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	shared.TruncateTestData(dbPool)
	ensureTestUserExists(t)
	r := setupUserRouterWithAudit()

	before := countAuditLogs(t, "user")

	body := `{"username":"auditcreate1","email":"auditcreate1@test.com","password":"secret123","role_id":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	after := countAuditLogs(t, "user")
	assert.Equal(t, before+1, after, "audit log should be created for user create")
}

func TestHandler_DeleteUser_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	shared.TruncateTestData(dbPool)
	ensureTestUserExists(t)
	r := setupUserRouterWithAudit()

	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()
	u := &User{
		Username: "auditdelete1",
		Email:    "auditdelete1@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, u))

	before := countAuditLogs(t, "user")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/admin/users/"+strconv.Itoa(u.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	after := countAuditLogs(t, "user")
	assert.Equal(t, before+1, after, "audit log should be created for user delete")
}

func TestHandler_CreateRole_WithAudit(t *testing.T) {
	skipIfNoDB(t)
	shared.TruncateTestData(dbPool)
	ensureTestUserExists(t)
	r := setupUserRouterWithAudit()

	before := countAuditLogs(t, "role")

	roleName := uniqueRoleName("audit_role")
	body := fmt.Sprintf(`{"name":"%s","description":"Audit test role"}`, roleName)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/roles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	after := countAuditLogs(t, "role")
	assert.Equal(t, before+1, after, "audit log should be created for role create")
}

func TestHandler_CreateUser_NilAuditSvc(t *testing.T) {
	skipIfNoDB(t)
	shared.TruncateTestData(dbPool)
	ensureTestUserExists(t)
	r := setupUserRouter()

	body := `{"username":"nilaudit1","email":"nilaudit1@test.com","password":"secret123","role_id":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateRole(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	t.Run("success", func(t *testing.T) {
		roleName := uniqueRoleName("hdl_test_role")
		body := fmt.Sprintf(`{"name":"%s","description":"Handler test role"}`, roleName)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/roles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Role `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, roleName, resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/roles", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing name", func(t *testing.T) {
		body := `{"description":"no name"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/roles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
