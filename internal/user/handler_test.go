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
	"retail-pos-system/internal/permissions"
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
			"user.view", "user.create", "user.update", "user.delete",
			"role.view", "role.create",
		})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
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

func TestHandler_CreateUser_WithReportsTo(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	// Create a manager first
	mgr := &User{
		Username: "hdl_mgr_reports",
		Email:    "hdl_mgr_reports@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, mgr))

	t.Run("create user with reports_to", func(t *testing.T) {
		body := fmt.Sprintf(`{"username":"hdlstaff1","email":"hdlstaff1@test.com","password":"secret123","role_id":3,"reports_to":%d}`, mgr.ID)
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
		require.NotNil(t, resp.Data.ReportsToID)
		assert.Equal(t, mgr.ID, *resp.Data.ReportsToID)
	})
}

func TestHandler_UpdateUser_WithReportsTo(t *testing.T) {
	skipIfNoDB(t)
	ensureTestUserExists(t)
	r := setupUserRouterWithAudit()
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	// Create manager and user
	mgr := &User{
		Username: "hdl_upd_mgr",
		Email:    "hdl_upd_mgr@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, mgr))

	user := &User{
		Username: "hdl_upd_user",
		Email:    "hdl_upd_user@test.com",
		Password: hash,
		RoleID:   2,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, user))

	// Create a subordinate
	sub := &User{
		Username:    "hdl_upd_sub",
		Email:       "hdl_upd_sub@test.com",
		Password:    hash,
		RoleID:      2,
		IsActive:    true,
		ReportsToID: &user.ID,
	}
	require.NoError(t, repo.CreateUser(ctx, sub))

	t.Run("update reports_to success", func(t *testing.T) {
		body := fmt.Sprintf(`{"reports_to":%d}`, mgr.ID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/users/"+strconv.Itoa(user.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("self reference rejected", func(t *testing.T) {
		body := fmt.Sprintf(`{"reports_to":%d}`, user.ID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/users/"+strconv.Itoa(user.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp["error"], "cannot set self as manager")
	})

	t.Run("circular reference rejected", func(t *testing.T) {
		// Try to set the subordinate as manager of the user
		body := fmt.Sprintf(`{"reports_to":%d}`, sub.ID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/users/"+strconv.Itoa(user.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp["error"], "circular reference")
	})
}

func TestHandler_GetSubordinates(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	mgr := &User{
		Username: "hdl_subs_mgr",
		Email:    "hdl_subs_mgr@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, mgr))

	sub := &User{
		Username:    "hdl_subs_sub",
		Email:       "hdl_subs_sub@test.com",
		Password:    hash,
		RoleID:      2,
		IsActive:    true,
		ReportsToID: &mgr.ID,
	}
	require.NoError(t, repo.CreateUser(ctx, sub))

	t.Run("returns subordinates", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/users/%d/subordinates", mgr.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, sub.ID, resp.Data[0].ID)
	})

	t.Run("empty list when no subordinates", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/users/%d/subordinates", sub.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/users/abc/subordinates", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_GetManager(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	mgr := &User{
		Username: "hdl_mgr_get",
		Email:    "hdl_mgr_get@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, mgr))

	sub := &User{
		Username:    "hdl_sub_get",
		Email:       "hdl_sub_get@test.com",
		Password:    hash,
		RoleID:      2,
		IsActive:    true,
		ReportsToID: &mgr.ID,
	}
	require.NoError(t, repo.CreateUser(ctx, sub))

	t.Run("returns manager", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/users/%d/manager", sub.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, mgr.ID, resp.Data.ID)
	})

	t.Run("not found for top-level user", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/users/%d/manager", mgr.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/users/abc/manager", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_GetOrgChart(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	mgr := &User{
		Username: "hdl_org_mgr",
		Email:    "hdl_org_mgr@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	require.NoError(t, repo.CreateUser(ctx, mgr))

	sub := &User{
		Username:    "hdl_org_sub",
		Email:       "hdl_org_sub@test.com",
		Password:    hash,
		RoleID:      2,
		IsActive:    true,
		ReportsToID: &mgr.ID,
	}
	require.NoError(t, repo.CreateUser(ctx, sub))

	t.Run("returns org chart", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/users/org-chart", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Data)
		foundIDs := make(map[int]bool)
		for _, u := range resp.Data {
			foundIDs[u.ID] = true
		}
		assert.True(t, foundIDs[mgr.ID], "manager should be in org chart")
		assert.True(t, foundIDs[sub.ID], "subordinate should be in org chart")
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
	_ = shared.TruncateTestData(dbPool)
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
	_ = shared.TruncateTestData(dbPool)
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
	_ = shared.TruncateTestData(dbPool)
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
	_ = shared.TruncateTestData(dbPool)
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

func TestHandler_UpdateRole(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	roleName := uniqueRoleName("upd_role")
	role := &Role{Name: roleName, Description: "To be updated"}
	require.NoError(t, repo.CreateRole(ctx, role))

	t.Run("success", func(t *testing.T) {
		newName := uniqueRoleName("upd_role_new")
		body := fmt.Sprintf(`{"name":"%s","description":"Updated description"}`, newName)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/roles/"+strconv.Itoa(role.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Role `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, newName, resp.Data.Name)
		assert.Equal(t, "Updated description", resp.Data.Description)
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"name":"nobody"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/roles/999999", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		body := `{"name":"bad"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/roles/abc", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_DeleteRole(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		roleName := uniqueRoleName("del_role")
		role := &Role{Name: roleName, Description: "To be deleted"}
		require.NoError(t, repo.CreateRole(ctx, role))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/admin/roles/"+strconv.Itoa(role.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("role with users cannot be deleted", func(t *testing.T) {
		roleName := uniqueRoleName("del_protected")
		role := &Role{Name: roleName, Description: "Protected role"}
		require.NoError(t, repo.CreateRole(ctx, role))

		hash := testPasswordHash()
		u := &User{
			Username: "delroleuser_" + roleName,
			Email:    "delrole_" + roleName + "@test.com",
			Password: hash,
			RoleID:   role.ID,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, u))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/admin/roles/"+strconv.Itoa(role.ID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found is idempotent", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/admin/roles/999999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/admin/roles/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_ListPermissions(t *testing.T) {
	skipIfNoDB(t)
	r := setupUserRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/permissions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []Permission `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}
