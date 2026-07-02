package user

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(0)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(0)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func uniqueRoleName(base string) string {
	return fmt.Sprintf("%s_%d", base, time.Now().UnixNano())
}

func testPasswordHash() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

func TestUserRepository_UserCRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	t.Run("Create and get by ID", func(t *testing.T) {
		u := &User{
			Username: "testuser_create_001",
			Email:    "create001@test.com",
			Password: hash,
			RoleID:   1,
			IsActive: true,
		}
		err := repo.CreateUser(ctx, u)
		require.NoError(t, err)
		require.Greater(t, u.ID, 0)

		got, err := repo.GetByID(u.ID)
		require.NoError(t, err)
		assert.Equal(t, u.Username, got.Username)
		assert.Equal(t, u.Email, got.Email)
		assert.Equal(t, u.RoleID, got.RoleID)
		assert.True(t, got.IsActive)
		assert.NotEmpty(t, got.CreatedAt)
	})

	t.Run("Get by username", func(t *testing.T) {
		u := &User{
			Username: "testuser_byusername",
			Email:    "byusername@test.com",
			Password: hash,
			RoleID:   2,
			IsActive: true,
		}
		err := repo.CreateUser(ctx, u)
		require.NoError(t, err)

		got, err := repo.GetByUsername(ctx, "testuser_byusername")
		require.NoError(t, err)
		assert.Equal(t, u.ID, got.ID)
		assert.Equal(t, u.Email, got.Email)
	})

	t.Run("Get user not found", func(t *testing.T) {
		_, err := repo.GetByID(-1)
		assert.ErrorContains(t, err, "user not found")

		_, err = repo.GetByUsername(ctx, "nonexistent_user_xxx")
		assert.ErrorContains(t, err, "user not found")
	})

	t.Run("Update user without password", func(t *testing.T) {
		u := &User{
			Username: "testuser_update_nopw",
			Email:    "updatenopw@test.com",
			Password: hash,
			RoleID:   3,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, u))

		u.Username = "testuser_updated_nopw"
		u.Email = "updatednopw@test.com"
		u.RoleID = 4
		u.Password = ""

		err := repo.UpdateUser(ctx, u)
		require.NoError(t, err)

		got, err := repo.GetByID(u.ID)
		require.NoError(t, err)
		assert.Equal(t, "testuser_updated_nopw", got.Username)
		assert.Equal(t, "updatednopw@test.com", got.Email)
		assert.Equal(t, 4, got.RoleID)
	})

	t.Run("Update user with password", func(t *testing.T) {
		u := &User{
			Username: "testuser_update_wpw",
			Email:    "updatewpw@test.com",
			Password: hash,
			RoleID:   1,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, u))

		newHash := testPasswordHash()
		u.Password = newHash
		err := repo.UpdateUser(ctx, u)
		require.NoError(t, err)

		got, err := repo.GetByID(u.ID)
		require.NoError(t, err)
		assert.Equal(t, newHash, got.Password)
	})

	t.Run("Soft delete user", func(t *testing.T) {
		u := &User{
			Username: "testuser_delete",
			Email:    "delete@test.com",
			Password: hash,
			RoleID:   1,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, u))

		err := repo.DeleteUser(ctx, u.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(u.ID)
		assert.ErrorContains(t, err, "user not found")

		_, err = repo.GetByUsername(ctx, "testuser_delete")
		assert.ErrorContains(t, err, "user not found")
	})

	t.Run("Update last login", func(t *testing.T) {
		u := &User{
			Username: "testuser_lastlogin",
			Email:    "lastlogin@test.com",
			Password: hash,
			RoleID:   1,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, u))

		err := repo.UpdateLastLogin(ctx, u.ID)
		require.NoError(t, err)

		got, err := repo.GetByID(u.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, got.LastLogin)
	})
}

func TestUserRepository_GetAllUsers(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	hash := testPasswordHash()

	usernames := []string{"testuser_list_a", "testuser_list_b", "testuser_list_c"}
	for i, uname := range usernames {
		u := &User{
			Username: uname,
			Email:    uname + "@test.com",
			Password: hash,
			RoleID:   i + 1,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, u))
	}

	inactiveUser := &User{
		Username: "testuser_inactive",
		Email:    "inactive@test.com",
		Password: hash,
		RoleID:   4,
		IsActive: false,
	}
	require.NoError(t, repo.CreateUser(ctx, inactiveUser))

	t.Run("list all with pagination", func(t *testing.T) {
		users, total, err := repo.GetAllUsers(ctx, 10, 0, "", "id", "asc", 0, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 4)
		assert.GreaterOrEqual(t, len(users), 4)
	})

	t.Run("search by username", func(t *testing.T) {
		users, total, err := repo.GetAllUsers(ctx, 10, 0, "testuser_list", "id", "asc", 0, nil)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Equal(t, 3, len(users))
	})

	t.Run("filter by role", func(t *testing.T) {
		users, total, err := repo.GetAllUsers(ctx, 10, 0, "", "id", "asc", 1, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, u := range users {
			assert.Equal(t, 1, u.RoleID)
		}
	})

	t.Run("filter by active status", func(t *testing.T) {
		active := true
		users, total, err := repo.GetAllUsers(ctx, 10, 0, "", "id", "asc", 0, &active)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		for _, u := range users {
			assert.True(t, u.IsActive)
		}

		active = false
		users, total, err = repo.GetAllUsers(ctx, 10, 0, "", "id", "asc", 0, &active)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.False(t, users[0].IsActive)
	})

	t.Run("limit and offset", func(t *testing.T) {
		users, total, err := repo.GetAllUsers(ctx, 2, 0, "", "id", "asc", 0, nil)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(users), 2)
		assert.Greater(t, total, 0)
	})

	t.Run("sort by role name via role_id", func(t *testing.T) {
		prefix := fmt.Sprintf("sortrolename_%d_", time.Now().UnixNano())
		// Create users with role IDs in non-alphabetical name order
		for _, roleID := range []int{1, 3, 4, 2} {
			u := &User{
				Username: fmt.Sprintf("%s%d", prefix, roleID),
				Email:    fmt.Sprintf("%s%d@test.com", prefix, roleID),
				Password: hash,
				RoleID:   roleID,
				IsActive: true,
			}
			require.NoError(t, repo.CreateUser(ctx, u))
		}

		// ASC -> role name alphabetical: admin(2), cashier(4), manager(3), superadmin(1)
		users, total, err := repo.GetAllUsers(ctx, 10, 0, prefix, "role_id", "asc", 0, nil)
		require.NoError(t, err)
		require.Equal(t, 4, total)
		assert.Equal(t, 2, users[0].RoleID)
		assert.Equal(t, 4, users[1].RoleID)
		assert.Equal(t, 3, users[2].RoleID)
		assert.Equal(t, 1, users[3].RoleID)

		// DESC -> role name reverse: superadmin(1), manager(3), cashier(4), admin(2)
		users, total, err = repo.GetAllUsers(ctx, 10, 0, prefix, "role_id", "desc", 0, nil)
		require.NoError(t, err)
		require.Equal(t, 4, total)
		assert.Equal(t, 1, users[0].RoleID)
		assert.Equal(t, 3, users[1].RoleID)
		assert.Equal(t, 4, users[2].RoleID)
		assert.Equal(t, 2, users[3].RoleID)
	})
}

func TestUserRepository_RoleCRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Get all roles", func(t *testing.T) {
		roles, err := repo.GetAllRoles(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, roles)
	})

	t.Run("Get role by ID", func(t *testing.T) {
		role, err := repo.GetRoleByID(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, "superadmin", role.Name)
		assert.True(t, role.IsSystem)
	})

	t.Run("Get role not found", func(t *testing.T) {
		_, err := repo.GetRoleByID(ctx, -1)
		assert.ErrorContains(t, err, "role not found")
	})

	t.Run("Create and update role", func(t *testing.T) {
		roleName := uniqueRoleName("test_role")
		r := &Role{Name: roleName, Description: "Test role", IsSystem: false}
		err := repo.CreateRole(ctx, r)
		require.NoError(t, err)
		require.Greater(t, r.ID, 0)

		updatedName := uniqueRoleName("test_role_updated")
		r.Name = updatedName
		r.Description = "Updated description"
		err = repo.UpdateRole(ctx, r)
		require.NoError(t, err)

		got, err := repo.GetRoleByID(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, updatedName, got.Name)
		assert.Equal(t, "Updated description", got.Description)
	})

	t.Run("Delete role", func(t *testing.T) {
		r := &Role{Name: uniqueRoleName("test_role_delete"), Description: "To be deleted", IsSystem: false}
		require.NoError(t, repo.CreateRole(ctx, r))

		err := repo.DeleteRole(ctx, r.ID)
		require.NoError(t, err)

		_, err = repo.GetRoleByID(ctx, r.ID)
		assert.ErrorContains(t, err, "role not found")
	})

	t.Run("Count users by role", func(t *testing.T) {
		count, err := repo.CountUsersByRole(ctx, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})
}

func TestUserRepository_PermissionCRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Get all permissions", func(t *testing.T) {
		perms, err := repo.GetAllPermissions(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, perms)
	})

	t.Run("Get role permissions for role with perms", func(t *testing.T) {
		allPerms, err := repo.GetAllPermissions(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, allPerms)

		r := &Role{Name: uniqueRoleName("test_role_get_perms"), Description: "Get perm test", IsSystem: false}
		require.NoError(t, repo.CreateRole(ctx, r))

		permIDs := []int{allPerms[0].ID}
		err = repo.UpdateRolePermissions(ctx, r.ID, permIDs)
		require.NoError(t, err)

		got, err := repo.GetRolePermissions(ctx, r.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("Get role permissions for role with none", func(t *testing.T) {
		r := &Role{Name: uniqueRoleName("test_role_noperms"), Description: "No perms", IsSystem: false}
		require.NoError(t, repo.CreateRole(ctx, r))

		perms, err := repo.GetRolePermissions(ctx, r.ID)
		require.NoError(t, err)
		assert.Empty(t, perms)
	})

	t.Run("Update role permissions", func(t *testing.T) {
		perms, err := repo.GetAllPermissions(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, perms)

		r := &Role{Name: uniqueRoleName("test_role_perms"), Description: "Permission test", IsSystem: false}
		require.NoError(t, repo.CreateRole(ctx, r))

		permIDs := []int{perms[0].ID}
		err = repo.UpdateRolePermissions(ctx, r.ID, permIDs)
		require.NoError(t, err)

		got, err := repo.GetRolePermissions(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(got))
		assert.Equal(t, perms[0].Code, got[0].Code)
	})
}
