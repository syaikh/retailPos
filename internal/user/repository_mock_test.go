package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_SetCache(t *testing.T) {
	repo := NewRepository(nil)
	c := cache.New(5*time.Minute, 10*time.Minute)
	repo.SetCache(c)
	assert.NotNil(t, repo.cache)
}

func TestRepository_GetByUsername_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	c.Set("user:username:admin", User{ID: 1, Username: "admin"})

	u, err := repo.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByUsername_CacheSet(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "username", "email", "password_hash", "role_id", "store_id", "is_active", "created_at", "updated_at", "last_login"}).
		AddRow(1, "admin", "admin@test.com", "hash", 1, nil, true, now, now, nil)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE username").WithArgs("admin").WillReturnRows(rows)

	roleRows := pgxmock.NewRows([]string{"id", "name", "description", "is_system", "created_at"}).
		AddRow(1, "admin", "Admin", true, now)
	mock.ExpectQuery("SELECT (.+) FROM roles WHERE id").WithArgs(1).WillReturnRows(roleRows)

	u, err := repo.GetByUsername(context.Background(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Username)

	v, ok := c.Get("user:username:admin")
	assert.True(t, ok)
	assert.NotNil(t, v)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByUsername_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE username").WithArgs("nobody").WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetByUsername(context.Background(), "nobody")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByUsername_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE username").WithArgs("admin").WillReturnError(fmt.Errorf("db lost"))

	repo := NewRepository(mock)
	_, err = repo.GetByUsername(context.Background(), "admin")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdatePassword(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET password_hash").WithArgs("hashed", 1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	err = repo.UpdatePassword(context.Background(), 1, "hashed")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUserRefreshTokens(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM refresh_tokens").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 2))

	repo := NewRepository(mock)
	err = repo.DeleteUserRefreshTokens(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRoleByID_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	c.Set("role:1", Role{ID: 1, Name: "admin"})

	role, err := repo.GetRoleByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRoleByID_CacheSet(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_system", "created_at"}).
		AddRow(1, "admin", "Admin role", true, now)
	mock.ExpectQuery("SELECT (.+) FROM roles WHERE id").WithArgs(1).WillReturnRows(rows)

	role, err := repo.GetRoleByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)

	v, ok := c.Get("role:1")
	assert.True(t, ok)
	assert.NotNil(t, v)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRoleByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM roles WHERE id").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetRoleByID(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRoleByID_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM roles WHERE id").WithArgs(1).WillReturnError(fmt.Errorf("db lost"))

	repo := NewRepository(mock)
	_, err = repo.GetRoleByID(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRole_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("role:1", Role{ID: 1, Name: "old"})
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("UPDATE roles SET").WithArgs("new", "", 1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateRole(context.Background(), &Role{ID: 1, Name: "new"})
	assert.NoError(t, err)

	_, ok := c.Get("role:1")
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteRole_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("role:1", Role{ID: 1})
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("DELETE FROM roles").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeleteRole(context.Background(), 1)
	assert.NoError(t, err)

	_, ok := c.Get("role:1")
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRolePermissions_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("role:1", Role{ID: 1})
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 3))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs(1, 10).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs(1, 20).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	err = repo.UpdateRolePermissions(context.Background(), 1, []int{10, 20})
	assert.NoError(t, err)

	_, ok := c.Get("role:1")
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRolePermissions_BeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))

	repo := NewRepository(mock)
	err = repo.UpdateRolePermissions(context.Background(), 1, []int{10})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRolePermissions_DeleteError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(1).WillReturnError(fmt.Errorf("delete failed"))

	repo := NewRepository(mock)
	err = repo.UpdateRolePermissions(context.Background(), 1, []int{10})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRolePermissions_InsertError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs(1, 10).WillReturnError(fmt.Errorf("insert failed"))

	repo := NewRepository(mock)
	err = repo.UpdateRolePermissions(context.Background(), 1, []int{10})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRolePermissions_CommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs(1, 10).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("commit failed"))

	repo := NewRepository(mock)
	err = repo.UpdateRolePermissions(context.Background(), 1, []int{10})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByID_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	c.Set("user:1", User{ID: 1, Username: "admin"})

	now := time.Now()
	userRows := pgxmock.NewRows([]string{"id", "username", "email", "password_hash", "role_id", "store_id", "is_active", "created_at", "updated_at", "last_login"}).
		AddRow(1, "admin", "admin@test.com", "hash", 1, nil, true, now, now, nil)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").WithArgs(1).WillReturnRows(userRows)

	roleRows := pgxmock.NewRows([]string{"id", "name", "description", "is_system", "created_at"}).
		AddRow(1, "admin", "Admin", true, now)
	mock.ExpectQuery("SELECT (.+) FROM roles WHERE id").WithArgs(1).WillReturnRows(roleRows)

	u, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetByID(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByID_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").WithArgs(1).WillReturnError(fmt.Errorf("db lost"))

	repo := NewRepository(mock)
	_, err = repo.GetByID(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllUsers_NoFilters(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WithArgs().WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "username", "email", "password_hash", "role_id", "store_id", "is_active",
		"created_at", "updated_at", "last_login",
		"role_id_2", "role_name", "role_description", "role_is_system", "role_created_at",
	}).AddRow(1, "admin", "admin@test.com", "hash", 1, nil, true, now, now, nil,
		1, "admin", "Admin", true, now)
	mock.ExpectQuery("SELECT u.id, u.username").WithArgs(10, 0).WillReturnRows(rows)

	repo := NewRepository(mock)
	users, total, err := repo.GetAllUsers(context.Background(), 10, 0, "", "id", "asc", 0, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "admin", users[0].Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllUsers_CountError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WillReturnError(fmt.Errorf("count error"))

	repo := NewRepository(mock)
	_, _, err = repo.GetAllUsers(context.Background(), 10, 0, "", "id", "asc", 0, nil)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllUsers_InvalidSort(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WithArgs().WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT u.id, u.username").WithArgs(10, 0).WillReturnRows(pgxmock.NewRows([]string{
		"id", "username", "email", "password_hash", "role_id", "store_id", "is_active",
		"created_at", "updated_at", "last_login",
		"role_id_2", "role_name", "role_description", "role_is_system", "role_created_at",
	}))

	repo := NewRepository(mock)
	_, _, err = repo.GetAllUsers(context.Background(), 10, 0, "", "evil_col", "evil_dir", 0, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(10, now, now)
	mock.ExpectQuery("INSERT INTO users").WithArgs("new", "new@test.com", "hash", 1, pgxmock.AnyArg(), true).WillReturnRows(rows)

	repo := NewRepository(mock)
	u := &User{Username: "new", Email: "new@test.com", Password: "hash", RoleID: 1, IsActive: true}
	err = repo.CreateUser(context.Background(), u)
	require.NoError(t, err)
	assert.Equal(t, 10, u.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateUser_WithPassword(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET username = .+password_hash").WithArgs("admin", "admin@test.com", "newhash", 1, pgxmock.AnyArg(), true, 1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	u := &User{ID: 1, Username: "admin", Email: "admin@test.com", Password: "newhash", RoleID: 1, IsActive: true}
	err = repo.UpdateUser(context.Background(), u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateUser_WithoutPassword(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET username = .+\\$6").WithArgs("admin", "admin@test.com", 1, pgxmock.AnyArg(), true, 1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	u := &User{ID: 1, Username: "admin", Email: "admin@test.com", RoleID: 1, IsActive: true}
	err = repo.UpdateUser(context.Background(), u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET deleted_at").WithArgs(1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	err = repo.DeleteUser(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateLastLogin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET last_login").WithArgs(1).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	err = repo.UpdateLastLogin(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateRole(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at"}).AddRow(10, now)
	mock.ExpectQuery("INSERT INTO roles").WithArgs("editor", "Editor", false).WillReturnRows(rows)

	repo := NewRepository(mock)
	r := &Role{Name: "editor", Description: "Editor"}
	err = repo.CreateRole(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, 10, r.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRolePermissions_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "created_at"}).
		AddRow(1, "products.view", "View Products", now)
	mock.ExpectQuery("SELECT p.id, p.code").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	perms, err := repo.GetRolePermissions(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, perms, 1)
	assert.Equal(t, "products.view", perms[0].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRolePermissions_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT p.id, p.code").WithArgs(1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetRolePermissions(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllRoles_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_system", "created_at", "permissions"}).
		AddRow(1, "admin", "Admin", true, now, []string{"*"})
	mock.ExpectQuery("SELECT r.id, r.name").WillReturnRows(rows)

	repo := NewRepository(mock)
	roles, err := repo.GetAllRoles(context.Background())
	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, "admin", roles[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllRoles_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT r.id, r.name").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetAllRoles(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllPermissions_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "created_at"}).
		AddRow(1, "users.view", "View Users", now)
	mock.ExpectQuery("SELECT id, code, name, created_at FROM permissions").WillReturnRows(rows)

	repo := NewRepository(mock)
	perms, err := repo.GetAllPermissions(context.Background())
	require.NoError(t, err)
	assert.Len(t, perms, 1)
	assert.Equal(t, "users.view", perms[0].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllPermissions_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, code, name, created_at FROM permissions").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetAllPermissions(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CountUsersByRole(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	count, err := repo.CountUsersByRole(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRole_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE roles SET").WithArgs("x", "", 1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	err = repo.UpdateRole(context.Background(), &Role{ID: 1, Name: "x"})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteRole_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM roles").WithArgs(1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	err = repo.DeleteRole(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUser_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET deleted_at").WithArgs(1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	err = repo.DeleteUser(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateLastLogin_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET last_login").WithArgs(1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	err = repo.UpdateLastLogin(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdatePassword_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE users SET password_hash").WithArgs("hash", 1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	err = repo.UpdatePassword(context.Background(), 1, "hash")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUserRefreshTokens_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM refresh_tokens").WithArgs(1).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	err = repo.DeleteUserRefreshTokens(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
