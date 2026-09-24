package storagelocation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestService_GetAll(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-GETALL", Name: "Svc GetAll", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	locations, total, err := svc.GetAll(ctx, 10, 0, "", nil, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(locations), 1)
}

func TestService_GetByID_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-GETID", Name: "Svc GetByID", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	got, err := svc.GetByID(ctx, sl.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, sl.Name, got.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999999, nil)
	assert.Error(t, err)
}

func TestService_Create_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	code := "SVC-CREATE-" + t.Name()
	warehouseID := whID
	result, err := svc.Create(ctx, CreateRequest{Code: code, Name: "Svc Create", WarehouseID: &warehouseID}, nil)
	require.NoError(t, err)
	assert.Equal(t, code, result.Code)
	assert.Greater(t, result.ID, 0)
	defer func() { _ = repo.Delete(ctx, result.ID) }()
}

func TestService_Create_EmptyCode(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	warehouseID := whID
	_, err := svc.Create(ctx, CreateRequest{Code: "  ", Name: "No Code", WarehouseID: &warehouseID}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code is required")
}

func TestService_Create_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	warehouseID := whID
	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-NONAME", Name: "  ", WarehouseID: &warehouseID}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestService_Create_NoScope(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-NOSCOPE", Name: "No Scope"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse_id or store_id is required")
}

func TestService_Create_InvalidWarehouse(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	invalid := 999999
	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-BADWH", Name: "Bad WH", WarehouseID: &invalid}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse not found")
}

func TestService_Create_InvalidStore(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	invalid := 999999
	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-BADSTORE", Name: "Bad Store", StoreID: &invalid}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store not found")
}

func TestService_Create_DuplicateCode(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	code := "SVC-DUP-" + t.Name()
	warehouseID := whID
	result, err := svc.Create(ctx, CreateRequest{Code: code, Name: "Dup", WarehouseID: &warehouseID}, nil)
	require.NoError(t, err)
	defer func() { _ = repo.Delete(ctx, result.ID) }()

	_, err = svc.Create(ctx, CreateRequest{Code: code, Name: "Dup2", WarehouseID: &warehouseID}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_Update_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-UPD", Name: "Svc Upd", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	newName := "Svc Upd Updated"
	updated, err := svc.Update(ctx, sl.ID, UpdateRequest{Name: &newName}, nil)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
}

func TestService_Update_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	code := "X"
	_, err := svc.Update(ctx, 999999, UpdateRequest{Code: &code}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_Update_DuplicateCode(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl1 := &StorageLocation{Code: "SVC-UDP1", Name: "Svc Upd Dup 1", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl1))
	defer func() { _ = repo.Delete(ctx, sl1.ID) }()

	sl2 := &StorageLocation{Code: "SVC-UDP2", Name: "Svc Upd Dup 2", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl2))
	defer func() { _ = repo.Delete(ctx, sl2.ID) }()

	_, err := svc.Update(ctx, sl1.ID, UpdateRequest{Code: &sl2.Code}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_Delete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-DEL", Name: "Svc Del", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))

	err := svc.Delete(ctx, sl.ID, nil)
	assert.NoError(t, err)

	_, err = repo.GetByID(ctx, sl.ID)
	assert.Error(t, err)
}

func TestService_Delete_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	err := svc.Delete(ctx, 999999, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_BulkUpdate_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl1 := &StorageLocation{Code: "SVC-BU1", Name: "Svc Bulk Upd 1", WarehouseID: &whID, IsActive: true}
	sl2 := &StorageLocation{Code: "SVC-BU2", Name: "Svc Bulk Upd 2", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl1))
	require.NoError(t, repo.Create(ctx, sl2))
	defer func() { _ = repo.Delete(ctx, sl1.ID) }()
	defer func() { _ = repo.Delete(ctx, sl2.ID) }()

	updated, err := svc.BulkUpdate(ctx, []int{sl1.ID, sl2.ID}, false, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, updated)
}

func TestService_BulkUpdate_EmptyIDs(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.BulkUpdate(ctx, []int{}, true, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no IDs provided")
}

func TestService_BulkDelete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl1 := &StorageLocation{Code: "SVC-BD1", Name: "Svc Bulk Del 1", WarehouseID: &whID, IsActive: true}
	sl2 := &StorageLocation{Code: "SVC-BD2", Name: "Svc Bulk Del 2", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl1))
	require.NoError(t, repo.Create(ctx, sl2))

	deleted, err := svc.BulkDelete(ctx, []int{sl1.ID, sl2.ID}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)
}

func TestService_BulkDelete_EmptyIDs(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.BulkDelete(ctx, []int{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no IDs provided")
}

func TestService_StoreBoundary(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	storeA := createTestStore(t, "BOUND-A")
	storeB := createTestStore(t, "BOUND-B")
	whA := createTestWarehouseForStore(t, "BOUND-A-WH", storeA)
	whB := createTestWarehouseForStore(t, "BOUND-B-WH", storeB)
	centralWH := createTestWarehouse(t, "BOUND-CENTRAL")

	locA := &StorageLocation{Code: "BOUND-A-LOC", Name: "Loc A", WarehouseID: &whA, IsActive: true}
	require.NoError(t, repo.Create(ctx, locA))
	t.Cleanup(func() { _ = repo.Delete(ctx, locA.ID) })

	locCentral := &StorageLocation{Code: "BOUND-CENTRAL-LOC", Name: "Central Loc", WarehouseID: &centralWH, IsActive: true}
	require.NoError(t, repo.Create(ctx, locCentral))
	t.Cleanup(func() { _ = repo.Delete(ctx, locCentral.ID) })

	t.Run("GetByID own warehouse-scoped row succeeds", func(t *testing.T) {
		got, err := svc.GetByID(ctx, locA.ID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, locA.ID, got.ID)
	})

	t.Run("GetByID cross-store row forbidden", func(t *testing.T) {
		_, err := svc.GetByID(ctx, locA.ID, &storeB)
		assert.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("GetByID superadmin bypasses", func(t *testing.T) {
		got, err := svc.GetByID(ctx, locA.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, locA.ID, got.ID)
	})

	t.Run("GetByID central warehouse row forbidden for store caller", func(t *testing.T) {
		_, err := svc.GetByID(ctx, locCentral.ID, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)
		got, err := svc.GetByID(ctx, locCentral.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, locCentral.ID, got.ID)
	})

	t.Run("Create with foreign warehouse forbidden", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateRequest{Code: "BOUND-XWH", Name: "Foreign WH", WarehouseID: &whB}, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("Create with foreign store forbidden", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateRequest{Code: "BOUND-XSTORE", Name: "Foreign Store", StoreID: &storeB}, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("Create with central warehouse forbidden for store caller", func(t *testing.T) {
		_, err := svc.Create(ctx, CreateRequest{Code: "BOUND-CWH", Name: "Central WH", WarehouseID: &centralWH}, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("Create own warehouse succeeds", func(t *testing.T) {
		created, err := svc.Create(ctx, CreateRequest{Code: "BOUND-OWN", Name: "Own WH", WarehouseID: &whA}, &storeA)
		require.NoError(t, err)
		t.Cleanup(func() { _ = repo.Delete(ctx, created.ID) })

		got, err := svc.GetByID(ctx, created.ID, &storeA)
		require.NoError(t, err)
		assert.Equal(t, "BOUND-OWN", got.Code)
	})

	t.Run("Create own store succeeds", func(t *testing.T) {
		created, err := svc.Create(ctx, CreateRequest{Code: "BOUND-OWNSTORE", Name: "Own Store", StoreID: &storeA}, &storeA)
		require.NoError(t, err)
		t.Cleanup(func() { _ = repo.Delete(ctx, created.ID) })
		assert.Equal(t, storeA, *created.StoreID)
	})

	t.Run("Update moving to foreign warehouse forbidden", func(t *testing.T) {
		newName := "Renamed"
		_, err := svc.Update(ctx, locA.ID, UpdateRequest{Name: &newName, WarehouseID: &whB}, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)
		got, err := repo.GetByID(ctx, locA.ID)
		require.NoError(t, err)
		assert.Equal(t, locA.Name, got.Name)
	})

	t.Run("Update own row succeeds", func(t *testing.T) {
		newName := "Renamed Own"
		updated, err := svc.Update(ctx, locA.ID, UpdateRequest{Name: &newName}, &storeA)
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		locA.Name = newName
	})

	t.Run("Delete cross-store forbidden and row survives", func(t *testing.T) {
		err := svc.Delete(ctx, locA.ID, &storeB)
		assert.ErrorIs(t, err, ErrStoreForbidden)
		_, err = repo.GetByID(ctx, locA.ID)
		require.NoError(t, err)
	})

	t.Run("BulkUpdate with one foreign id aborts entire batch", func(t *testing.T) {
		locB := &StorageLocation{Code: "BOUND-B-LOC", Name: "Loc B", StoreID: &storeB, IsActive: true}
		require.NoError(t, repo.Create(ctx, locB))
		t.Cleanup(func() { _ = repo.Delete(ctx, locB.ID) })

		_, err := svc.BulkUpdate(ctx, []int{locA.ID, locB.ID}, false, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)

		got, err := repo.GetByID(ctx, locA.ID)
		require.NoError(t, err)
		assert.True(t, got.IsActive, "own row must not be partially written")
		gotB, err := repo.GetByID(ctx, locB.ID)
		require.NoError(t, err)
		assert.True(t, gotB.IsActive, "foreign row must not be written")
	})

	t.Run("BulkDelete with foreign id forbidden", func(t *testing.T) {
		locB := &StorageLocation{Code: "BOUND-B-LOC2", Name: "Loc B2", StoreID: &storeB, IsActive: true}
		require.NoError(t, repo.Create(ctx, locB))
		t.Cleanup(func() { _ = repo.Delete(ctx, locB.ID) })

		_, err := svc.BulkDelete(ctx, []int{locA.ID, locB.ID}, &storeA)
		assert.ErrorIs(t, err, ErrStoreForbidden)

		_, err = repo.GetByID(ctx, locA.ID)
		require.NoError(t, err)
		_, err = repo.GetByID(ctx, locB.ID)
		require.NoError(t, err)
	})

	t.Run("BulkUpdate own rows succeeds", func(t *testing.T) {
		updated, err := svc.BulkUpdate(ctx, []int{locA.ID}, false, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 1, updated)
		got, err := repo.GetByID(ctx, locA.ID)
		require.NoError(t, err)
		assert.False(t, got.IsActive)
		_, err = svc.BulkUpdate(ctx, []int{locA.ID}, true, &storeA)
		require.NoError(t, err)
	})

	t.Run("BulkDelete own rows succeeds", func(t *testing.T) {
		own := &StorageLocation{Code: "BOUND-OWN-DEL", Name: "Own Del", StoreID: &storeA, IsActive: true}
		require.NoError(t, repo.Create(ctx, own))

		deleted, err := svc.BulkDelete(ctx, []int{own.ID}, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 1, deleted)
		_, err = repo.GetByID(ctx, own.ID)
		assert.Error(t, err)
	})

	t.Run("Bulk with only nonexistent ids is a no-op", func(t *testing.T) {
		updated, err := svc.BulkUpdate(ctx, []int{999999}, false, &storeA)
		require.NoError(t, err)
		assert.Equal(t, 0, updated)
	})
}
