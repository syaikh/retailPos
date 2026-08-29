package storagelocation

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/store"
)

func newMockRepo(t *testing.T) (pgxmock.PgxPoolIface, *Repository, context.Context) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	repo := NewRepository(mock)
	repo.SetStoreExistenceProvider(store.ExistenceProvider{})
	return mock, repo, context.Background()
}

func TestRepositoryMock_ErrorBranches(t *testing.T) {
	boom := errors.New("boom")

	t.Run("getall count error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnError(boom)
		_, _, err := repo.GetAll(ctx, 10, 0, "", nil)
		assert.ErrorContains(t, err, "count storage locations")
	})

	t.Run("getall rows error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("SELECT sl.id").WillReturnError(boom)
		_, _, err := repo.GetAll(ctx, 10, 0, "", nil)
		assert.ErrorContains(t, err, "list storage locations")
	})

	t.Run("getall scan error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT sl.id").WithArgs(10, 0).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))
		_, _, err := repo.GetAll(ctx, 10, 0, "", nil)
		assert.ErrorContains(t, err, "scan storage location")
	})

	t.Run("getbyid scan error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT sl.id").WithArgs(1).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))
		_, err := repo.GetByID(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("getbycode error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT sl.id").WithArgs("X").WillReturnError(boom)
		_, err := repo.GetByCode(ctx, "X")
		assert.Error(t, err)
	})

	t.Run("create error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("INSERT INTO storage_locations").WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(boom)
		err := repo.Create(ctx, &StorageLocation{Code: "C", Name: "N"})
		assert.ErrorContains(t, err, "create storage location")
	})

	t.Run("update error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("UPDATE storage_locations").WillReturnError(boom)
		err := repo.Update(ctx, &StorageLocation{ID: 1, Code: "C", Name: "N"})
		assert.ErrorContains(t, err, "update storage location")
	})

	t.Run("delete error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("DELETE FROM storage_locations").WithArgs(1).WillReturnError(boom)
		err := repo.Delete(ctx, 1)
		assert.ErrorContains(t, err, "delete storage location")
	})

	t.Run("getallactive error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT sl.id").WillReturnError(boom)
		_, err := repo.GetAllActive(ctx)
		assert.ErrorContains(t, err, "list active storage locations")
	})

	t.Run("codeexists error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WithArgs("C", 0).WillReturnError(boom)
		_, err := repo.CodeExists(ctx, "C", 0)
		assert.ErrorContains(t, err, "check code")
	})

	t.Run("warehouseexists error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT EXISTS.*warehouses").WithArgs(1).WillReturnError(boom)
		_, err := repo.WarehouseExists(ctx, 1)
		assert.ErrorContains(t, err, "check warehouse exists")
	})

	t.Run("storeexists error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectQuery("SELECT EXISTS.*stores").WithArgs(1).WillReturnError(boom)
		_, err := repo.StoreExists(ctx, 1)
		assert.ErrorContains(t, err, "check store exists")
	})

	t.Run("bulkupdate error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("UPDATE storage_locations SET is_active").WithArgs(true, []int{1}).WillReturnError(boom)
		_, err := repo.BulkUpdate(ctx, []int{1}, true)
		assert.ErrorContains(t, err, "bulk update")
	})

	t.Run("bulkdelete error", func(t *testing.T) {
		mock, repo, ctx := newMockRepo(t)
		mock.ExpectExec("DELETE FROM storage_locations WHERE id").WithArgs([]int{1}).WillReturnError(boom)
		_, err := repo.BulkDelete(ctx, []int{1})
		assert.ErrorContains(t, err, "bulk delete")
	})
}
