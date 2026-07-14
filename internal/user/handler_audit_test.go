package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/audit"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupMockUserRouterWithAudit(svc UserService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	var auditSvc *audit.Service
	if dbPool != nil {
		auditSvc = audit.NewService(audit.NewRepository(dbPool))
	}
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
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
	var auditSvc *audit.Service
	if dbPool != nil {
		auditSvc = audit.NewService(audit.NewRepository(dbPool))
	}
	h := NewAuthHandler(svc, auditSvc)
	h.RegisterChangePasswordRoute(r.Group("/"), func(c *gin.Context) { c.Next() })
	h.RegisterRefreshRoute(r.Group("/"), func(c *gin.Context) { c.Next() })
	r.POST("/logout", h.Logout)
	return r
}

func TestAuditHandler_UpdateUser(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
	if dbPool == nil {
		t.Skip("requires DB")
	}
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

func TestAuditHandler_Logout_EmptyRefreshToken(t *testing.T) {
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
	if dbPool == nil {
		t.Skip("requires DB")
	}
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
