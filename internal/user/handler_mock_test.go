package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserService struct {
	getByIDFn            func(ctx context.Context, id int) (*User, error)
	getByUsernameFn      func(ctx context.Context, username string) (*User, error)
	getAllUsersFn        func(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error)
	createUserFn         func(ctx context.Context, user *User) error
	updateUserFn         func(ctx context.Context, user *User) error
	deleteUserFn         func(ctx context.Context, id int) error
	getSubordinatesFn    func(ctx context.Context, managerID int) ([]User, error)
	getManagerFn         func(ctx context.Context, userID int) (*User, error)
	getOrgChartFn        func(ctx context.Context) ([]User, error)
	isSubordinateFn      func(ctx context.Context, managerID, userID int) (bool, error)
	getAllRolesFn        func(ctx context.Context) ([]Role, error)
	getRoleByIDFn        func(ctx context.Context, id int) (*Role, error)
	createRoleFn         func(ctx context.Context, role *Role) error
	updateRoleFn         func(ctx context.Context, role *Role) error
	deleteRoleFn         func(ctx context.Context, id int) error
	countByRoleFn        func(ctx context.Context, roleID int) (int, error)
	getAllPermsFn        func(ctx context.Context) ([]Permission, error)
	getRolePermissionsFn func(ctx context.Context, roleID int) ([]Permission, error)
	updatePermsFn        func(ctx context.Context, roleID int, permissionIDs []int) error
	updatePreferencesFn  func(ctx context.Context, userID int, language, theme string) error
	inTxFn               func(ctx context.Context, fn func(tx pgx.Tx) error) error
	updateUserTxFn       func(ctx context.Context, tx pgx.Tx, user *User) error
	updateRolePermsTxFn  func(ctx context.Context, tx pgx.Tx, roleID int, permissionIDs []int) error
}

func (m *mockUserService) GetUserByID(ctx context.Context, id int) (*User, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockUserService) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return m.getByUsernameFn(ctx, username)
}
func (m *mockUserService) GetAllUsers(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
	return m.getAllUsersFn(ctx, limit, offset, search, sortBy, sortDir, roleID, isActive)
}
func (m *mockUserService) CreateUser(ctx context.Context, user *User) error {
	return m.createUserFn(ctx, user)
}
func (m *mockUserService) UpdateUser(ctx context.Context, user *User) error {
	return m.updateUserFn(ctx, user)
}
func (m *mockUserService) DeleteUser(ctx context.Context, id int) error {
	return m.deleteUserFn(ctx, id)
}
func (m *mockUserService) GetSubordinates(ctx context.Context, managerID int) ([]User, error) {
	return m.getSubordinatesFn(ctx, managerID)
}
func (m *mockUserService) GetManager(ctx context.Context, userID int) (*User, error) {
	return m.getManagerFn(ctx, userID)
}
func (m *mockUserService) GetOrgChart(ctx context.Context) ([]User, error) {
	return m.getOrgChartFn(ctx)
}
func (m *mockUserService) IsSubordinate(ctx context.Context, managerID, userID int) (bool, error) {
	if m.isSubordinateFn != nil {
		return m.isSubordinateFn(ctx, managerID, userID)
	}
	return false, nil
}
func (m *mockUserService) GetAllRoles(ctx context.Context) ([]Role, error) {
	return m.getAllRolesFn(ctx)
}
func (m *mockUserService) GetRoleByID(ctx context.Context, id int) (*Role, error) {
	if m.getRoleByIDFn != nil {
		return m.getRoleByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockUserService) CreateRole(ctx context.Context, role *Role) error {
	return m.createRoleFn(ctx, role)
}
func (m *mockUserService) UpdateRole(ctx context.Context, role *Role) error {
	return m.updateRoleFn(ctx, role)
}
func (m *mockUserService) DeleteRole(ctx context.Context, id int) error {
	return m.deleteRoleFn(ctx, id)
}
func (m *mockUserService) CountUsersByRole(ctx context.Context, roleID int) (int, error) {
	return m.countByRoleFn(ctx, roleID)
}
func (m *mockUserService) GetAllPermissions(ctx context.Context) ([]Permission, error) {
	if m.getAllPermsFn != nil {
		return m.getAllPermsFn(ctx)
	}
	return nil, nil
}
func (m *mockUserService) GetRolePermissions(ctx context.Context, roleID int) ([]Permission, error) {
	if m.getRolePermissionsFn != nil {
		return m.getRolePermissionsFn(ctx, roleID)
	}
	return nil, nil
}
func (m *mockUserService) UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error {
	return m.updatePermsFn(ctx, roleID, permissionIDs)
}
func (m *mockUserService) UpdatePreferences(ctx context.Context, userID int, language, theme string) error {
	if m.updatePreferencesFn != nil {
		return m.updatePreferencesFn(ctx, userID, language, theme)
	}
	return nil
}

// InTx runs fn (defaulting to a nil transaction, adequate for unit tests).
func (m *mockUserService) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if m.inTxFn != nil {
		return m.inTxFn(ctx, fn)
	}
	return fn(nil)
}

// UpdateUserTx runs within an existing transaction; fall back to the plain
// UpdateUser mock so existing tests that only wire updateUserFn still pass.
func (m *mockUserService) UpdateUserTx(ctx context.Context, tx pgx.Tx, user *User) error {
	if m.updateUserTxFn != nil {
		return m.updateUserTxFn(ctx, tx, user)
	}
	if m.updateUserFn != nil {
		return m.updateUserFn(ctx, user)
	}
	return nil
}

// UpdateRolePermissionsTx runs within an existing transaction; fall back to
// the plain UpdateRolePermissions mock.
func (m *mockUserService) UpdateRolePermissionsTx(ctx context.Context, tx pgx.Tx, roleID int, permissionIDs []int) error {
	if m.updateRolePermsTxFn != nil {
		return m.updateRolePermsTxFn(ctx, tx, roleID, permissionIDs)
	}
	if m.updatePermsFn != nil {
		return m.updatePermsFn(ctx, roleID, permissionIDs)
	}
	return nil
}

var _ Service = (*mockUserService)(nil)

func setupMockUserRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "superadmin")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestMockHandler_ListUsers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getAllUsersFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
				return []User{{ID: 1, Username: "admin"}}, 1, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["total"])
	})

	t.Run("role_id param", func(t *testing.T) {
		svc := &mockUserService{
			getAllUsersFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
				require.NotNil(t, roleID)
				assert.Equal(t, 3, *roleID)
				return []User{}, 0, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users?role_id=3", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("is_active param", func(t *testing.T) {
		svc := &mockUserService{
			getAllUsersFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
				require.NotNil(t, isActive)
				assert.True(t, *isActive)
				return []User{}, 0, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users?is_active=true", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			getAllUsersFn: func(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
				return nil, 0, errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_CreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return nil, errors.New("not found")
			},
			createUserFn: func(ctx context.Context, user *User) error {
				user.ID = 100
				return nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"newuser","email":"new@test.com","password":"password123","role_id":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid username format", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return nil, errors.New("not found")
			},
		})
		body := `{"username":"Invalid User!","email":"e@test.com","password":"password123","role_id":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "lowercase alphanumeric")
	})

	t.Run("duplicate username", func(t *testing.T) {
		svc := &mockUserService{
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return &User{ID: 1, Username: "existing"}, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"existing","email":"e@test.com","password":"password123","role_id":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already exists")
	})

	t.Run("short password", func(t *testing.T) {
		svc := &mockUserService{
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"newuser","email":"e@test.com","password":"short","role_id":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "at least 8 characters")
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return nil, errors.New("not found")
			},
			createUserFn: func(ctx context.Context, user *User) error {
				return errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"newuser","email":"e@test.com","password":"password123","role_id":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_UpdateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 1, Username: "old", RoleID: 1}, nil
			},
			updateUserFn: func(ctx context.Context, user *User) error { return nil },
		}
		r := setupMockUserRouter(svc)
		body := `{"email":"updated@test.com"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/abc", strings.NewReader(`{"email":"x@test.com"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/99", strings.NewReader(`{"email":"x@test.com"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid username in update", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 1, Username: "old"}, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"Invalid!"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate username on update", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 1, Username: "olduser", RoleID: 1}, nil
			},
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return &User{ID: 2, Username: "taken"}, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"taken"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("self role modify forbidden", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 1, Username: "admin", RoleID: 1}, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"role_id":2}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestMockHandler_DeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			deleteUserFn: func(ctx context.Context, id int) error { return nil },
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/users/5", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/users/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			deleteUserFn: func(ctx context.Context, id int) error { return errors.New("fail") },
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/users/5", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_ListRoles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getAllRolesFn: func(ctx context.Context) ([]Role, error) {
				return []Role{{ID: 1, Name: "admin"}}, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/roles", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("error", func(t *testing.T) {
		svc := &mockUserService{
			getAllRolesFn: func(ctx context.Context) ([]Role, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/roles", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_CreateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			createRoleFn: func(ctx context.Context, role *Role) error {
				role.ID = 10
				return nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"name":"manager","description":"Manager role"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/roles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/roles", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestMockHandler_UpdateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
				return &Role{ID: 1, Name: "old"}, nil
			},
			updateRoleFn: func(ctx context.Context, role *Role) error { return nil },
		}
		r := setupMockUserRouter(svc)
		body := `{"name":"new name"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/roles/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/roles/abc", strings.NewReader(`{"name":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockUserService{
			getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
				return nil, errors.New("not found")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/roles/99", strings.NewReader(`{"name":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMockHandler_UpdateRolePermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			updatePermsFn: func(ctx context.Context, roleID int, permissionIDs []int) error {
				assert.Equal(t, 1, roleID)
				assert.Equal(t, []int{1, 2}, permissionIDs)
				return nil
			},
			getRoleByIDFn: func(ctx context.Context, id int) (*Role, error) {
				return &Role{ID: 1, Name: "admin"}, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"permission_ids":[1,2]}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/roles/1/permissions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/roles/abc/permissions", strings.NewReader(`{"permission_ids":[]}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			updatePermsFn: func(ctx context.Context, roleID int, permissionIDs []int) error {
				return errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/roles/1/permissions", strings.NewReader(`{"permission_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_DeleteRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			countByRoleFn: func(ctx context.Context, roleID int) (int, error) { return 0, nil },
			deleteRoleFn:  func(ctx context.Context, id int) error { return nil },
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/abc", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("role in use", func(t *testing.T) {
		svc := &mockUserService{
			countByRoleFn: func(ctx context.Context, roleID int) (int, error) { return 5, nil },
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/roles/1", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "users are assigned")
	})
}

func TestMockHandler_ListPermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getAllPermsFn: func(ctx context.Context) ([]Permission, error) {
				return []Permission{{ID: 1, Code: "user.view"}}, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/permissions", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("error", func(t *testing.T) {
		svc := &mockUserService{
			getAllPermsFn: func(ctx context.Context) ([]Permission, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/permissions", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetSubordinates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getSubordinatesFn: func(ctx context.Context, managerID int) ([]User, error) {
				assert.Equal(t, 1, managerID)
				return []User{{ID: 2, Username: "staff1", ReportsToID: intPtr(1)}}, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/1/subordinates", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/abc/subordinates", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			getSubordinatesFn: func(ctx context.Context, managerID int) ([]User, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/1/subordinates", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getManagerFn: func(ctx context.Context, userID int) (*User, error) {
				assert.Equal(t, 2, userID)
				return &User{ID: 1, Username: "manager"}, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/2/manager", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("no manager", func(t *testing.T) {
		svc := &mockUserService{
			getManagerFn: func(ctx context.Context, userID int) (*User, error) {
				return nil, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/1/manager", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/abc/manager", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			getManagerFn: func(ctx context.Context, userID int) (*User, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/2/manager", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_GetOrgChart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockUserService{
			getOrgChartFn: func(ctx context.Context) ([]User, error) {
				return []User{
					{ID: 1, Username: "superadmin"},
					{ID: 2, Username: "manager1", ReportsToID: intPtr(1)},
				}, nil
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/org-chart", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockUserService{
			getOrgChartFn: func(ctx context.Context) ([]User, error) {
				return nil, errors.New("db error")
			},
		}
		r := setupMockUserRouter(svc)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/admin/users/org-chart", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_CreateUser_WithReportsTo(t *testing.T) {
	t.Run("create with reports_to_id", func(t *testing.T) {
		svc := &mockUserService{
			getByUsernameFn: func(ctx context.Context, username string) (*User, error) {
				return nil, errors.New("not found")
			},
			createUserFn: func(ctx context.Context, user *User) error {
				assert.NotNil(t, user.ReportsToID)
				assert.Equal(t, 1, *user.ReportsToID)
				user.ID = 100
				return nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"username":"staff1","email":"staff1@test.com","password":"password123","role_id":3,"reports_to":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestMockHandler_UpdateUser_WithReportsTo(t *testing.T) {
	t.Run("self reference reports_to rejected", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 1, Username: "admin", RoleID: 1}, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"reports_to":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("circular reference rejected", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 3, Username: "staff3", RoleID: 4}, nil
			},
			isSubordinateFn: func(ctx context.Context, managerID, userID int) (bool, error) {
				return true, nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"reports_to":2}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/3", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "circular")
	})

	t.Run("update reports_to success", func(t *testing.T) {
		svc := &mockUserService{
			getByIDFn: func(ctx context.Context, id int) (*User, error) {
				return &User{ID: 2, Username: "staff", RoleID: 3}, nil
			},
			updateUserFn: func(ctx context.Context, user *User) error {
				assert.NotNil(t, user.ReportsToID)
				assert.Equal(t, 1, *user.ReportsToID)
				return nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"reports_to":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/admin/users/2", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func intPtr(i int) *int {
	return &i
}

// ──────────────────────────────────────────────────────────────────────
// PUT /api/users/me/preferences
// ──────────────────────────────────────────────────────────────────────

func TestMockHandler_UpdatePreferences(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capturedUserID int
		var capturedLang, capturedTheme string
		svc := &mockUserService{
			updatePreferencesFn: func(ctx context.Context, userID int, language, theme string) error {
				capturedUserID = userID
				capturedLang = language
				capturedTheme = theme
				return nil
			},
		}
		r := setupMockUserRouter(svc)
		body := `{"language":"en","theme":"dark"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/users/me/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, capturedUserID)
		assert.Equal(t, "en", capturedLang)
		assert.Equal(t, "dark", capturedTheme)

		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "en", resp["language"])
		assert.Equal(t, "dark", resp["theme"])
	})

	t.Run("invalid language", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		body := `{"language":"fr","theme":"light"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/users/me/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid theme", func(t *testing.T) {
		r := setupMockUserRouter(&mockUserService{})
		body := `{"language":"id","theme":"neon"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/users/me/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
