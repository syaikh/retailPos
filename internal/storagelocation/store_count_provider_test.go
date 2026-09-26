package storagelocation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoreLocationCount_CountsActiveLocationsOfStore covers the storage half
// of the store onboarding readiness check: only active rows assigned to the
// store itself count. Inactive rows, rows owned by another store and rows
// owned by a warehouse are excluded.
func TestStoreLocationCount_CountsActiveLocationsOfStore(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	provider := StoreLocationCount{}

	storeA := createTestStore(t, "Provider A")
	storeB := createTestStore(t, "Provider B")
	// warehouses.code is varchar(20) and unique, so derive a run-unique one.
	warehouseCode := fmt.Sprintf("P%019d", time.Now().UnixNano())
	whID := createTestWarehouse(t, warehouseCode)

	locCode := fmt.Sprintf("PROV%d", time.Now().UnixNano())
	ptr := func(v int) *int { return &v }
	insert := func(suffix string, storeID, warehouseID *int, isActive bool) {
		t.Helper()
		_, err := dbPool.Exec(ctx,
			`INSERT INTO storage_locations (code, name, store_id, warehouse_id, is_active)
			 VALUES ($1, $2, $3, $4, $5)`,
			locCode+suffix, "Provider Loc "+suffix, storeID, warehouseID, isActive)
		require.NoError(t, err)
	}

	insert("_A1", ptr(storeA), nil, true)
	insert("_A2", ptr(storeA), nil, true)
	insert("_A_OFF", ptr(storeA), nil, false)
	insert("_B", ptr(storeB), nil, true)
	insert("_WH", nil, ptr(whID), true)

	defer func() {
		_, _ = dbPool.Exec(ctx, `DELETE FROM storage_locations WHERE code LIKE $1`, locCode+"%")
		_, _ = dbPool.Exec(ctx, `DELETE FROM warehouses WHERE code = $1`, warehouseCode)
	}()

	count, err := provider.StorageLocationCountByStore(ctx, dbPool, storeA)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "only the two active locations of this store count")

	countB, err := provider.StorageLocationCountByStore(ctx, dbPool, storeB)
	require.NoError(t, err)
	assert.Equal(t, 1, countB, "a foreign store only sees its own location")

	// A store with no locations of its own reports zero — the readiness blocker.
	emptyStore := createTestStore(t, "Provider Empty")
	zero, err := provider.StorageLocationCountByStore(ctx, dbPool, emptyStore)
	require.NoError(t, err)
	assert.Equal(t, 0, zero)
}
