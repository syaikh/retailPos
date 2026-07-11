package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()
	hash := testPasswordHash()

	u := &User{
		Username: "svc_read_ops_001",
		Email:    "svc_read_ops_001@test.com",
		Password: hash,
		RoleID:   3,
		IsActive: true,
	}
	require.NoError(t, svc.CreateUser(ctx, u))

	t.Run("GetUserByID", func(t *testing.T) {
		got, err := svc.GetUserByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, u.Username, got.Username)
		assert.Equal(t, u.Email, got.Email)
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		got, err := svc.GetUserByUsername(ctx, "svc_read_ops_001")
		require.NoError(t, err)
		assert.Equal(t, u.ID, got.ID)
	})

	t.Run("GetAllUsers", func(t *testing.T) {
		users, total, err := svc.GetAllUsers(ctx, 10, 0, "svc_read_ops_001", "id", "asc", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(users), 1)
	})
}

func TestUserService_RoleOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	t.Run("GetAllRoles", func(t *testing.T) {
		roles, err := svc.GetAllRoles(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, roles)
	})

	t.Run("GetRoleByID", func(t *testing.T) {
		role, err := svc.GetRoleByID(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, "superadmin", role.Name)
	})

	t.Run("CreateRole", func(t *testing.T) {
		r, err := svc.GetAllRoles(ctx)
		require.NoError(t, err)

		role := &Role{Name: uniqueRoleName("svc_test_role"), Description: "Service test", IsSystem: false}
		err2 := svc.CreateRole(ctx, role)
		require.NoError(t, err2)
		assert.Greater(t, role.ID, 0)
		assert.NotEqual(t, r, role)
	})

	t.Run("UpdateRole", func(t *testing.T) {
		roleName := uniqueRoleName("svc_role_update")
		role := &Role{Name: roleName, Description: "Before", IsSystem: false}
		require.NoError(t, svc.CreateRole(ctx, role))

		updatedName := uniqueRoleName("svc_role_updated")
		role.Name = updatedName
		role.Description = "After"
		err := svc.UpdateRole(ctx, role)
		require.NoError(t, err)

		got, err := svc.GetRoleByID(ctx, role.ID)
		require.NoError(t, err)
		assert.Equal(t, updatedName, got.Name)
		assert.Equal(t, "After", got.Description)
	})

	t.Run("DeleteRole", func(t *testing.T) {
		role := &Role{Name: uniqueRoleName("svc_role_delete"), Description: "Delete me", IsSystem: false}
		require.NoError(t, svc.CreateRole(ctx, role))

		err := svc.DeleteRole(ctx, role.ID)
		require.NoError(t, err)

		_, err = svc.GetRoleByID(ctx, role.ID)
		assert.Error(t, err)
	})

	t.Run("CountUsersByRole", func(t *testing.T) {
		count, err := svc.CountUsersByRole(ctx, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})
}

func TestUserService_PermissionOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	t.Run("GetAllPermissions", func(t *testing.T) {
		perms, err := svc.GetAllPermissions(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, perms)
	})

	t.Run("GetRolePermissions", func(t *testing.T) {
		allPerms, err := svc.GetAllPermissions(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, allPerms)

		role := &Role{Name: uniqueRoleName("svc_role_get_perms"), Description: "Get perm test", IsSystem: false}
		require.NoError(t, svc.CreateRole(ctx, role))

		err = svc.UpdateRolePermissions(ctx, role.ID, []int{allPerms[0].ID})
		require.NoError(t, err)

		got, err := svc.GetRolePermissions(ctx, role.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("UpdateRolePermissions", func(t *testing.T) {
		allPerms, err := svc.GetAllPermissions(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, allPerms)

		role := &Role{Name: uniqueRoleName("svc_role_perms"), Description: "Perm test", IsSystem: false}
		require.NoError(t, svc.CreateRole(ctx, role))

		err = svc.UpdateRolePermissions(ctx, role.ID, []int{allPerms[0].ID})
		require.NoError(t, err)

		got, err := svc.GetRolePermissions(ctx, role.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(got))
		assert.Equal(t, allPerms[0].Code, got[0].Code)
	})
}
