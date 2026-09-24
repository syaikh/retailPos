package storagelocation

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func createTestWarehouse(t *testing.T, code string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(context.Background(),
		`INSERT INTO warehouses (name, code) VALUES ($1, $2) RETURNING id`,
		"Test Warehouse "+code, code).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestWarehouseForStore(t *testing.T, code string, storeID int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(context.Background(),
		`INSERT INTO warehouses (name, code, store_id) VALUES ($1, $2, $3) RETURNING id`,
		"Test Warehouse "+code, code, storeID).Scan(&id)
	require.NoError(t, err)
	return id
}

func createTestStore(t *testing.T, name string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(context.Background(),
		`INSERT INTO stores (name) VALUES ($1) RETURNING id`,
		"Test Store "+name).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestStorageLocationRepository_CRUD(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepository()
	ctx := context.Background()

	whID := createTestWarehouse(t, "CRUD-WH")

	t.Run("Create and get by ID", func(t *testing.T) {
		sl := &StorageLocation{
			Code:        "CRUD-RACK-1",
			Name:        "Rack CRUD",
			WarehouseID: &whID,
			IsActive:    true,
		}
		err := repo.Create(ctx, sl)
		require.NoError(t, err)
		require.Greater(t, sl.ID, 0)

		fetched, err := repo.GetByID(ctx, sl.ID)
		require.NoError(t, err)
		assert.Equal(t, "CRUD-RACK-1", fetched.Code)
		assert.Equal(t, "Rack CRUD", fetched.Name)
		assert.True(t, fetched.IsActive)
		assert.Equal(t, whID, *fetched.WarehouseID)
	})

	t.Run("Get all with pagination", func(t *testing.T) {
		locations, total, err := repo.GetAll(ctx, 10, 0, "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(locations), 1)
	})

	t.Run("Get all with search", func(t *testing.T) {
		locations, total, err := repo.GetAll(ctx, 10, 0, "Rack CRUD", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(locations), 1)
	})

	t.Run("Get all with is_active filter", func(t *testing.T) {
		active := true
		locations, _, err := repo.GetAll(ctx, 10, 0, "", &active, nil)
		require.NoError(t, err)
		for _, l := range locations {
			assert.True(t, l.IsActive)
		}
	})

	t.Run("CodeExists", func(t *testing.T) {
		exists, err := repo.CodeExists(ctx, "CRUD-RACK-1", 0)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("GetByCode case insensitive", func(t *testing.T) {
		found, err := repo.GetByCode(ctx, "crud-rack-1")
		require.NoError(t, err)
		assert.Equal(t, "CRUD-RACK-1", found.Code)
	})

	t.Run("Update", func(t *testing.T) {
		sl := &StorageLocation{Code: "CRUD-UPD", Name: "To Update", WarehouseID: &whID, IsActive: true}
		require.NoError(t, repo.Create(ctx, sl))
		defer func() { _ = repo.Delete(ctx, sl.ID) }()

		fetched, err := repo.GetByID(ctx, sl.ID)
		require.NoError(t, err)
		fetched.Name = "Updated Location"
		fetched.Notes = "moved"
		require.NoError(t, repo.Update(ctx, fetched))

		updated, err := repo.GetByID(ctx, sl.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Location", updated.Name)
		assert.Equal(t, "moved", updated.Notes)
	})

	t.Run("Delete", func(t *testing.T) {
		sl := &StorageLocation{Code: "CRUD-DEL", Name: "To Delete", WarehouseID: &whID, IsActive: true}
		require.NoError(t, repo.Create(ctx, sl))

		require.NoError(t, repo.Delete(ctx, sl.ID))

		_, err := repo.GetByID(ctx, sl.ID)
		assert.Error(t, err)
	})

	t.Run("Create with store scope", func(t *testing.T) {
		storeID := createTestStore(t, "CRUD")
		sl := &StorageLocation{Code: "CRUD-STORE", Name: "Store Rack", StoreID: &storeID, IsActive: true}
		require.NoError(t, repo.Create(ctx, sl))
		defer func() { _ = repo.Delete(ctx, sl.ID) }()

		fetched, err := repo.GetByID(ctx, sl.ID)
		require.NoError(t, err)
		assert.Equal(t, storeID, *fetched.StoreID)
	})

	t.Run("BulkUpdate sets is_active", func(t *testing.T) {
		sl1 := &StorageLocation{Code: "BULK-UPD-1", Name: "Bulk Upd 1", WarehouseID: &whID, IsActive: true}
		sl2 := &StorageLocation{Code: "BULK-UPD-2", Name: "Bulk Upd 2", WarehouseID: &whID, IsActive: true}
		require.NoError(t, repo.Create(ctx, sl1))
		require.NoError(t, repo.Create(ctx, sl2))
		defer func() { _ = repo.Delete(ctx, sl1.ID) }()
		defer func() { _ = repo.Delete(ctx, sl2.ID) }()

		updated, err := repo.BulkUpdate(ctx, []int{sl1.ID, sl2.ID}, false)
		require.NoError(t, err)
		assert.Equal(t, 2, updated)

		f1, err := repo.GetByID(ctx, sl1.ID)
		require.NoError(t, err)
		assert.False(t, f1.IsActive)
	})

	t.Run("BulkDelete removes locations", func(t *testing.T) {
		sl1 := &StorageLocation{Code: "BULK-DEL-1", Name: "Bulk Del 1", WarehouseID: &whID, IsActive: true}
		sl2 := &StorageLocation{Code: "BULK-DEL-2", Name: "Bulk Del 2", WarehouseID: &whID, IsActive: true}
		require.NoError(t, repo.Create(ctx, sl1))
		require.NoError(t, repo.Create(ctx, sl2))

		deleted, err := repo.BulkDelete(ctx, []int{sl1.ID, sl2.ID})
		require.NoError(t, err)
		assert.Equal(t, 2, deleted)

		_, err = repo.GetByID(ctx, sl1.ID)
		assert.Error(t, err)
	})

	t.Run("Bulk operations with empty IDs return 0", func(t *testing.T) {
		updated, err := repo.BulkUpdate(ctx, []int{}, true)
		require.NoError(t, err)
		assert.Equal(t, 0, updated)

		deleted, err := repo.BulkDelete(ctx, []int{})
		require.NoError(t, err)
		assert.Equal(t, 0, deleted)
	})
}

func TestStorageLocationRepository_GetByID_NotFound(t *testing.T) {
	repo := newTestRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999999)
	assert.Error(t, err)
}

func TestStorageLocationRepository_ScopeChecks(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepository()
	ctx := context.Background()

	t.Run("WarehouseExists true for existing", func(t *testing.T) {
		whID := createTestWarehouse(t, "EXIST")
		exists, err := repo.WarehouseExists(ctx, whID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("WarehouseExists false for missing", func(t *testing.T) {
		exists, err := repo.WarehouseExists(ctx, 999999)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("StoreExists true for existing", func(t *testing.T) {
		storeID := createTestStore(t, "EXIST")
		exists, err := repo.StoreExists(ctx, storeID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("StoreExists false for missing", func(t *testing.T) {
		exists, err := repo.StoreExists(ctx, 999999)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("WarehouseStoreID resolves linked warehouse", func(t *testing.T) {
		storeID := createTestStore(t, "SC-WHSTORE")
		whID := createTestWarehouseForStore(t, "SC-WH-LINKED", storeID)
		got, err := repo.WarehouseStoreID(ctx, whID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, storeID, *got)
	})

	t.Run("WarehouseStoreID nil for central warehouse", func(t *testing.T) {
		whID := createTestWarehouse(t, "SC-WH-CENTRAL")
		got, err := repo.WarehouseStoreID(ctx, whID)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("WarehouseStoreID nil for missing warehouse", func(t *testing.T) {
		got, err := repo.WarehouseStoreID(ctx, 999999)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("WarehouseIDsByStoreID returns linked ids", func(t *testing.T) {
		storeID := createTestStore(t, "SC-WHLIST")
		wh1 := createTestWarehouseForStore(t, "SC-WHL-1", storeID)
		wh2 := createTestWarehouseForStore(t, "SC-WHL-2", storeID)
		ids, err := repo.WarehouseIDsByStoreID(ctx, storeID)
		require.NoError(t, err)
		assert.Contains(t, ids, wh1)
		assert.Contains(t, ids, wh2)
	})

	t.Run("WarehouseIDsByStoreID empty for unknown store", func(t *testing.T) {
		ids, err := repo.WarehouseIDsByStoreID(ctx, 999999)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})
}

func TestStorageLocationRepository_GetByIDs(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepository()
	ctx := context.Background()

	whID := createTestWarehouse(t, "GETBYIDS-WH")
	sl1 := &StorageLocation{Code: "GBI-1", Name: "GetByIDs 1", WarehouseID: &whID, IsActive: true}
	sl2 := &StorageLocation{Code: "GBI-2", Name: "GetByIDs 2", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl1))
	require.NoError(t, repo.Create(ctx, sl2))
	t.Cleanup(func() { _ = repo.Delete(ctx, sl1.ID) })
	t.Cleanup(func() { _ = repo.Delete(ctx, sl2.ID) })

	got, err := repo.GetByIDs(ctx, []int{sl1.ID, sl2.ID, 999999})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, sl1.ID, got[0].ID)
	assert.Equal(t, sl2.ID, got[1].ID)

	empty, err := repo.GetByIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestStorageLocationRepository_GetAllStoreScope(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepository()
	ctx := context.Background()

	storeA := createTestStore(t, "LIST-A")
	storeB := createTestStore(t, "LIST-B")
	whA := createTestWarehouseForStore(t, "LIST-A-WH", storeA)
	centralWH := createTestWarehouse(t, "LIST-CENTRAL")

	locDirect := &StorageLocation{Code: "LIST-A-DIRECT", Name: "Direct A", StoreID: &storeA, IsActive: true}
	locViaWH := &StorageLocation{Code: "LIST-A-VIAWH", Name: "Via WH A", WarehouseID: &whA, IsActive: true}
	locB := &StorageLocation{Code: "LIST-B-1", Name: "Loc B", StoreID: &storeB, IsActive: true}
	locCentral := &StorageLocation{Code: "LIST-CENTRAL-1", Name: "Central", WarehouseID: &centralWH, IsActive: true}
	for _, sl := range []*StorageLocation{locDirect, locViaWH, locB, locCentral} {
		require.NoError(t, repo.Create(ctx, sl))
		t.Cleanup(func() { _ = repo.Delete(ctx, sl.ID) })
	}

	locations, _, err := repo.GetAll(ctx, 50, 0, "", nil, &storeA)
	require.NoError(t, err)
	codes := make([]string, 0, len(locations))
	for _, l := range locations {
		codes = append(codes, l.Code)
	}
	assert.Contains(t, codes, "LIST-A-DIRECT")
	assert.Contains(t, codes, "LIST-A-VIAWH")
	assert.NotContains(t, codes, "LIST-B-1")
	assert.NotContains(t, codes, "LIST-CENTRAL-1")

	all, _, err := repo.GetAll(ctx, 50, 0, "", nil, nil)
	require.NoError(t, err)
	allCodes := make([]string, 0, len(all))
	for _, l := range all {
		allCodes = append(allCodes, l.Code)
	}
	assert.Contains(t, allCodes, "LIST-A-DIRECT")
	assert.Contains(t, allCodes, "LIST-B-1")
	assert.Contains(t, allCodes, "LIST-CENTRAL-1")
}
