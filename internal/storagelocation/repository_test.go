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
		locations, total, err := repo.GetAll(ctx, 10, 0, "", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(locations), 1)
	})

	t.Run("Get all with search", func(t *testing.T) {
		locations, total, err := repo.GetAll(ctx, 10, 0, "Rack CRUD", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(locations), 1)
	})

	t.Run("Get all with is_active filter", func(t *testing.T) {
		active := true
		locations, _, err := repo.GetAll(ctx, 10, 0, "", &active)
		require.NoError(t, err)
		for _, l := range locations {
			assert.True(t, l.IsActive)
		}
	})

	t.Run("Get all active", func(t *testing.T) {
		locations, err := repo.GetAllActive(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(locations), 1)
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
}
