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
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-GETALL", Name: "Svc GetAll", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	locations, total, err := svc.GetAll(ctx, 10, 0, "", nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(locations), 1)
}

func TestService_GetByID_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-GETID", Name: "Svc GetByID", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	got, err := svc.GetByID(ctx, sl.ID)
	require.NoError(t, err)
	assert.Equal(t, sl.Name, got.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999999)
	assert.Error(t, err)
}

func TestService_GetAllActive(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-ACTIVE", Name: "Svc Active", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	locations, err := svc.GetAllActive(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(locations), 1)
}

func TestService_Create_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	code := "SVC-CREATE-" + t.Name()
	warehouseID := whID
	result, err := svc.Create(ctx, CreateRequest{Code: code, Name: "Svc Create", WarehouseID: &warehouseID})
	require.NoError(t, err)
	assert.Equal(t, code, result.Code)
	assert.Greater(t, result.ID, 0)
	defer func() { _ = repo.Delete(ctx, result.ID) }()
}

func TestService_Create_EmptyCode(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	warehouseID := whID
	_, err := svc.Create(ctx, CreateRequest{Code: "  ", Name: "No Code", WarehouseID: &warehouseID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code is required")
}

func TestService_Create_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	warehouseID := whID
	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-NONAME", Name: "  ", WarehouseID: &warehouseID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestService_Create_NoScope(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-NOSCOPE", Name: "No Scope"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse_id or store_id is required")
}

func TestService_Create_InvalidWarehouse(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	invalid := 999999
	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-BADWH", Name: "Bad WH", WarehouseID: &invalid})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse not found")
}

func TestService_Create_InvalidStore(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	invalid := 999999
	_, err := svc.Create(ctx, CreateRequest{Code: "SVC-BADSTORE", Name: "Bad Store", StoreID: &invalid})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store not found")
}

func TestService_Create_DuplicateCode(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	code := "SVC-DUP-" + t.Name()
	warehouseID := whID
	result, err := svc.Create(ctx, CreateRequest{Code: code, Name: "Dup", WarehouseID: &warehouseID})
	require.NoError(t, err)
	defer func() { _ = repo.Delete(ctx, result.ID) }()

	_, err = svc.Create(ctx, CreateRequest{Code: code, Name: "Dup2", WarehouseID: &warehouseID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_Update_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-UPD", Name: "Svc Upd", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))
	defer func() { _ = repo.Delete(ctx, sl.ID) }()

	newName := "Svc Upd Updated"
	updated, err := svc.Update(ctx, sl.ID, UpdateRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
}

func TestService_Update_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	code := "X"
	_, err := svc.Update(ctx, 999999, UpdateRequest{Code: &code})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_Update_DuplicateCode(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
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

	_, err := svc.Update(ctx, sl1.ID, UpdateRequest{Code: &sl2.Code})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_Delete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl := &StorageLocation{Code: "SVC-DEL", Name: "Svc Del", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl))

	err := svc.Delete(ctx, sl.ID)
	assert.NoError(t, err)

	_, err = repo.GetByID(ctx, sl.ID)
	assert.Error(t, err)
}

func TestService_Delete_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	err := svc.Delete(ctx, 999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_BulkUpdate_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
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

	updated, err := svc.BulkUpdate(ctx, []int{sl1.ID, sl2.ID}, false)
	require.NoError(t, err)
	assert.Equal(t, 2, updated)
}

func TestService_BulkUpdate_EmptyIDs(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.BulkUpdate(ctx, []int{}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no IDs provided")
}

func TestService_BulkDelete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_ = shared.TruncateTestData(dbPool)
	whID := createTestWarehouse(t, "SVC")
	sl1 := &StorageLocation{Code: "SVC-BD1", Name: "Svc Bulk Del 1", WarehouseID: &whID, IsActive: true}
	sl2 := &StorageLocation{Code: "SVC-BD2", Name: "Svc Bulk Del 2", WarehouseID: &whID, IsActive: true}
	require.NoError(t, repo.Create(ctx, sl1))
	require.NoError(t, repo.Create(ctx, sl2))

	deleted, err := svc.BulkDelete(ctx, []int{sl1.ID, sl2.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)
}

func TestService_BulkDelete_EmptyIDs(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.BulkDelete(ctx, []int{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no IDs provided")
}
