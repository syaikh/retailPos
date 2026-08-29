package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuditCreator struct {
	createAuditLogFn func(ctx context.Context, log *audit.Log) error
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.Log) error {
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

func setupMockUserRouterWithAudit(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	auditSvc := &mockAuditCreator{}
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func setupMockAuthRouterWithAudit(svc AuthLoginService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	auditSvc := &mockAuditCreator{}
	h := NewAuthHandler(svc, auditSvc)
	h.RegisterChangePasswordRoute(r.Group("/"), func(c *gin.Context) { c.Next() })
	h.RegisterRefreshRoute(r.Group("/"), func(c *gin.Context) { c.Next() })
	r.POST("/logout", h.Logout)
	return r
}

func TestAuditHandler_UpdateUser(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 2, Username: "oldname", Email: "old@test.com", RoleID: 1, IsActive: true}, nil
		},
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return nil, errors.New("not found")
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"username":"newname","email":"new@test.com"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateUser_GetUserError(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return nil, errors.New("not found")
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"username":"newname"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuditHandler_UpdateRole(t *testing.T) {
	svc := &mockUserService{
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return &Role{ID: 1, Name: "admin", Description: "Admin role"}, nil
		},
		updateRoleFn: func(ctx context.Context, role *Role) error {
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"name":"superadmin","description":"Super Admin"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/roles/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteRole(t *testing.T) {
	svc := &mockUserService{
		countByRoleFn: func(ctx context.Context, roleID int) (int, error) {
			return 0, nil
		},
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return &Role{ID: 5, Name: "to-delete"}, nil
		},
		deleteRoleFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/5", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_DeleteRole_HasUsers(t *testing.T) {
	svc := &mockUserService{
		countByRoleFn: func(ctx context.Context, roleID int) (int, error) {
			return 3, nil
		},
	}
	r := setupMockUserRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/1", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "users are assigned")
}

func TestAuditHandler_DeleteRole_CountError(t *testing.T) {
	svc := &mockUserService{
		countByRoleFn: func(ctx context.Context, roleID int) (int, error) {
			return 0, errors.New("db error")
		},
	}
	r := setupMockUserRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_ChangePassword(t *testing.T) {
	svc := &mockAuthLoginService{
		changePasswordFn: func(ctx context.Context, userID int, currentPassword, newPassword string) error {
			return nil
		},
	}
	r := setupMockAuthRouterWithAudit(svc)
	body := `{"current_password":"old123","new_password":"newpass123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_Logout(t *testing.T) {
	svc := &mockAuthLoginService{
		logoutFn: func(ctx context.Context, userID int, refreshToken string) error {
			return nil
		},
	}
	r := setupMockAuthRouterWithAudit(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "test-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_RefreshToken_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockAuthLoginService{
		refreshTokenFn: func(ctx context.Context, oldRefreshToken string) (string, string, *User, error) {
			return "new-access-token", "new-refresh-token", &User{ID: 7, Username: "jdoe", Role: Role{Name: "cashier"}}, nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			captured = log
			return nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 7)
		c.Set("username", "jdoe")
		c.Set("role", "cashier")
		c.Next()
	})
	h := NewAuthHandler(svc, auditSvc)
	h.RegisterRefreshRoute(r.Group("/"), func(c *gin.Context) { c.Next() })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/refresh", nil)
	req.Header.Set("X-Refresh-Token", "old-refresh-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for a successful token refresh")
	assert.Equal(t, "token_refresh", captured.Action)
	assert.Equal(t, "auth", captured.EntityType)
	require.NotNil(t, captured.UserID)
	assert.Equal(t, 7, *captured.UserID)
	assert.Equal(t, "jdoe", captured.Username)
	assert.Equal(t, "cashier", captured.Role)
	assert.Contains(t, captured.Description, "refreshed access token")
}

func TestAuditHandler_Logout_EmptyRefreshToken(t *testing.T) {
	svc := &mockAuthLoginService{
		logoutFn: func(ctx context.Context, userID int, refreshToken string) error {
			t.Fatal("should not be called")
			return nil
		},
	}
	r := setupMockAuthRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/logout", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_ChangePassword_InvalidPassword(t *testing.T) {
	svc := &mockAuthLoginService{
		changePasswordFn: func(ctx context.Context, userID int, currentPassword, newPassword string) error {
			return ErrInvalidPassword
		},
	}
	r := setupMockAuthRouterWithAudit(svc)
	body := `{"current_password":"wrong","new_password":"newpass123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuditHandler_ChangePassword_InvalidPassword_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockAuthLoginService{
		changePasswordFn: func(ctx context.Context, userID int, currentPassword, newPassword string) error {
			return ErrInvalidPassword
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			captured = log
			return nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 7)
		c.Set("username", "jdoe")
		c.Set("role", "cashier")
		c.Next()
	})
	h := NewAuthHandler(svc, auditSvc)
	h.RegisterChangePasswordRoute(r.Group("/"), func(c *gin.Context) { c.Next() })
	body := `{"current_password":"wrong","new_password":"newpass123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	require.NotNil(t, captured, "an audit log should be written for a failed password change")
	assert.Equal(t, "password_change_failed", captured.Action)
	assert.Equal(t, "auth", captured.EntityType)
	require.NotNil(t, captured.UserID)
	assert.Equal(t, 7, *captured.UserID)
	assert.Equal(t, "jdoe", captured.Username)
	assert.Equal(t, "cashier", captured.Role)
	assert.Contains(t, captured.Description, "invalid current password")
}

func TestAuditHandler_CreateUser_IsActiveFalse(t *testing.T) {
	svc := &mockUserService{
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return nil, errors.New("not found")
		},
		createUserFn: func(ctx context.Context, user *User) error {
			assert.False(t, user.IsActive)
			user.ID = 10
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"username":"inactive","email":"in@test.com","password":"password123","role_id":1,"is_active":false}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuditHandler_UpdateUser_BindError(t *testing.T) {
	r := setupMockUserRouterWithAudit(&mockUserService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_UpdateUser_PasswordChange(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 2, Username: "old", RoleID: 1, IsActive: true}, nil
		},
		getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
			return nil, errors.New("not found")
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			assert.NotEmpty(t, user.Password)
			assert.NotEqual(t, "newpassword123", user.Password)
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"password":"newpassword123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateUser_StoreID(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 2, Username: "old", RoleID: 1, IsActive: true}, nil
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			require.NotNil(t, user.StoreID)
			assert.Equal(t, 5, *user.StoreID)
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"store_id":5}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateUser_IsActive(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 2, Username: "old", RoleID: 1, IsActive: true}, nil
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			assert.False(t, user.IsActive)
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"is_active":false}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateUser_ServiceError(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 2, Username: "old", RoleID: 1, IsActive: true}, nil
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			return errors.New("db error")
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"email":"updated@test.com"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_DeleteUser_GetUserError(t *testing.T) {
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return nil, errors.New("not found")
		},
		deleteUserFn: func(ctx context.Context, id int) error { return nil },
	}
	r := setupMockUserRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/users/99", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_CreateRole_ServiceError(t *testing.T) {
	svc := &mockUserService{
		createRoleFn: func(ctx context.Context, role *Role) error {
			return errors.New("db error")
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"name":"duplicate","description":"Dup role"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/roles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_UpdateRole_BindError(t *testing.T) {
	r := setupMockUserRouterWithAudit(&mockUserService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/roles/1", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_UpdateRole_ServiceError(t *testing.T) {
	svc := &mockUserService{
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return &Role{ID: 1, Name: "admin"}, nil
		},
		updateRoleFn: func(ctx context.Context, role *Role) error {
			return errors.New("db error")
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"name":"new name"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/roles/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_UpdateRolePermissions_BindError(t *testing.T) {
	r := setupMockUserRouterWithAudit(&mockUserService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/roles/1/permissions", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditHandler_UpdateRolePermissions_WritesAudit(t *testing.T) {
	var captured *audit.Log
	currentPerms := []Permission{{ID: 1, Code: "role.view"}, {ID: 2, Code: "role.create"}}
	svc := &mockUserService{
		getRolePermissionsFn: func(ctx context.Context, roleID int) ([]Permission, error) {
			return currentPerms, nil
		},
		updatePermsFn: func(ctx context.Context, roleID int, permissionIDs []int) error {
			assert.Equal(t, 1, roleID)
			assert.Equal(t, []int{2, 3}, permissionIDs) // keep b, add c
			// Simulate the persisted change so the second GetRolePermissions
			// reflects the new state.
			currentPerms = []Permission{{ID: 2, Code: "role.create"}, {ID: 3, Code: "role.update"}}
			return nil
		},
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return &Role{ID: 1, Name: "admin"}, nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			captured = log
			return nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	// Update to permissions [2,3] => keep role.create(b), add role.update(c), remove role.view(a).
	body := `{"permission_ids":[2,3]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/roles/1/permissions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for permission changes")
	assert.Equal(t, "update_permissions", captured.Action)
	assert.Equal(t, "role", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 1, *captured.EntityID)
	assert.Contains(t, captured.Description, "Updated permissions for role admin")
	assert.Contains(t, captured.Description, "added")
	assert.Contains(t, captured.Description, "removed")
	assert.Contains(t, captured.Description, "role.update") // added code
	assert.Contains(t, captured.Description, "role.view")   // removed code
}

func TestAuditHandler_UpdateRolePermissions_GetRoleError(t *testing.T) {
	svc := &mockUserService{
		updatePermsFn: func(ctx context.Context, roleID int, permissionIDs []int) error {
			return nil
		},
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return nil, errors.New("not found")
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	body := `{"permission_ids":[1,2]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/roles/1/permissions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_DeleteRole_ServiceError(t *testing.T) {
	svc := &mockUserService{
		countByRoleFn: func(ctx context.Context, roleID int) (int, error) {
			return 0, nil
		},
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return &Role{ID: 5, Name: "to-delete"}, nil
		},
		deleteRoleFn: func(ctx context.Context, id int) error {
			return errors.New("db error")
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/5", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditHandler_DeleteRole_GetRoleError(t *testing.T) {
	svc := &mockUserService{
		countByRoleFn: func(ctx context.Context, roleID int) (int, error) {
			return 0, nil
		},
		getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
			return nil, errors.New("not found")
		},
		deleteRoleFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	r := setupMockUserRouterWithAudit(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/5", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditHandler_UpdateUser_Deactivated_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 2, Username: "old", RoleID: 1, IsActive: true}, nil
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			captured = log
			return nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"is_active":false}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for deactivation")
	assert.Equal(t, "user_deactivated", captured.Action)
	assert.Equal(t, "user", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 2, *captured.EntityID)
	assert.Contains(t, captured.Description, "Deactivated user")
}

func TestAuditHandler_UpdateUser_Activated_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 3, Username: "inactive", RoleID: 1, IsActive: false}, nil
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			captured = log
			return nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/3", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for activation")
	assert.Equal(t, "user_activated", captured.Action)
	assert.Equal(t, "user", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 3, *captured.EntityID)
	assert.Contains(t, captured.Description, "Activated user")
}

func TestAuditHandler_UpdateUser_RoleChanged_WritesAudit(t *testing.T) {
	var captured *audit.Log
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id int) (*User, error) {
			return &User{ID: 4, Username: "someone", RoleID: 1, IsActive: true}, nil
		},
		updateUserFn: func(ctx context.Context, user *User) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{
		createAuditLogFn: func(ctx context.Context, log *audit.Log) error {
			captured = log
			return nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"role_id":5}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/users/4", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured, "an audit log should be written for role change")
	assert.Equal(t, "user_role_changed", captured.Action)
	assert.Equal(t, "user", captured.EntityType)
	require.NotNil(t, captured.EntityID)
	assert.Equal(t, 4, *captured.EntityID)
	assert.Contains(t, captured.Description, "Changed role of user")
}
