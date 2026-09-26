package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoreStaffCounts_CountsActiveStoreAssignedAccounts exercises the port
// internal/store consumes for onboarding readiness: only active, non-deleted
// accounts assigned to the store are counted, grouped by role name. Accounts
// without a store (HQ roles) and accounts of another store are excluded, and
// roles with no qualifying account are absent from the map.
func TestStoreStaffCounts_CountsActiveStoreAssignedAccounts(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	repo := NewRepository(dbPool)
	provider := StoreStaffCounts{}

	var storeID, otherStoreID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO stores (name, is_active) VALUES ($1, true) RETURNING id`,
		"Staff Provider Store").Scan(&storeID))
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO stores (name, is_active) VALUES ($1, true) RETURNING id`,
		"Staff Provider Other Store").Scan(&otherStoreID))

	var cashierRoleID, managerRoleID int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'cashier'`).Scan(&cashierRoleID))
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'manager'`).Scan(&managerRoleID))

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	create := func(tag string, roleID int, targetStore *int) int {
		t.Helper()
		username := fmt.Sprintf("sp_%s_%s", suffix, tag)
		user := &User{
			Username: username,
			Email:    username + "@test.local",
			Password: testPasswordHash(),
			RoleID:   roleID,
			StoreID:  targetStore,
			IsActive: true,
		}
		require.NoError(t, repo.CreateUser(ctx, user))
		return user.ID
	}

	inStore := func(n int) *int { return &n }

	// Counted: two active cashiers and one active manager on this store.
	create("c1", cashierRoleID, inStore(storeID))
	create("c2", cashierRoleID, inStore(storeID))
	create("m1", managerRoleID, inStore(storeID))

	// Excluded: inactive, soft-deleted, HQ (no store) and foreign store.
	inactiveID := create("c_inactive", cashierRoleID, inStore(storeID))
	_, err := dbPool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, inactiveID)
	require.NoError(t, err)

	deletedID := create("c_deleted", cashierRoleID, inStore(storeID))
	_, err = dbPool.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1`, deletedID)
	require.NoError(t, err)

	create("c_hq", cashierRoleID, nil)
	create("c_other", cashierRoleID, inStore(otherStoreID))

	counts, err := provider.StaffCountsByStore(ctx, dbPool, storeID)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"cashier": 2, "manager": 1}, counts)

	// The store nobody else is assigned to only reports its own staff.
	foreign, err := provider.StaffCountsByStore(ctx, dbPool, otherStoreID)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"cashier": 1}, foreign, "only the foreign-store cashier belongs there")
}
