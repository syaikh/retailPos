package uom

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

func TestUOMRepository_CRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		u := &UnitOfMeasure{Code: "XUNIT", Name: "Test Unit", IsActive: true}
		err := repo.Create(ctx, u)
		require.NoError(t, err)
		require.Greater(t, u.ID, 0)

		got, err := repo.GetByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, u.Code, got.Code)
	})

	t.Run("Get by ID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, -1)
		assert.ErrorContains(t, err, "unit of measure not found")
	})

	t.Run("GetIDByCode", func(t *testing.T) {
		id, err := repo.GetIDByCode(ctx, "XUNIT")
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("GetIDByCode inactive returns error", func(t *testing.T) {
		u := &UnitOfMeasure{Code: "INACTV", Name: "Inactive UOM", IsActive: false}
		require.NoError(t, repo.Create(ctx, u))

		_, err := repo.GetIDByCode(ctx, "INACTV")
		assert.Error(t, err)
	})

	t.Run("List all active UOMs", func(t *testing.T) {
		units, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.NotNil(t, units)
	})

	t.Run("Update UOM", func(t *testing.T) {
		u := &UnitOfMeasure{Code: "YUPD", Name: "Before Update", IsActive: true}
		require.NoError(t, repo.Create(ctx, u))

		u.Name = "After Update"
		err := repo.Update(ctx, u)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, "After Update", got.Name)
	})

	t.Run("Delete UOM", func(t *testing.T) {
		u := &UnitOfMeasure{Code: "ZDEL", Name: "Delete Me", IsActive: true}
		require.NoError(t, repo.Create(ctx, u))

		err := repo.Delete(ctx, u.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, u.ID)
		assert.Error(t, err)
	})
}

func TestUOMRepository_Export(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("GetAllForExport returns all UOMs", func(t *testing.T) {
		units, err := repo.GetAllForExport(ctx)
		require.NoError(t, err)
		assert.NotNil(t, units)
	})
}

func TestUOMRepository_Import(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("BulkUpsert inserts new UOMs", func(t *testing.T) {
		records := []ImportRow{
			{Row: 2, Code: "IMP1", Name: "Import UOM 1", IsActive: true},
			{Row: 3, Code: "IMP2", Name: "Import UOM 2", IsActive: false},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 2, result.Inserted)
		assert.Equal(t, 0, result.Updated)
		assert.Empty(t, result.Errors)
	})

	t.Run("BulkUpsert updates existing UOMs", func(t *testing.T) {
		records := []ImportRow{
			{Row: 2, Code: "IMP1", Name: "Updated UOM 1", IsActive: false},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 0, result.Inserted)
		assert.Equal(t, 1, result.Updated)
		assert.Empty(t, result.Errors)
	})

	t.Run("BulkUpsert rejects empty code", func(t *testing.T) {
		records := []ImportRow{
			{Row: 2, Code: "", Name: "No Code", IsActive: true},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 0, result.Inserted)
		assert.Equal(t, 0, result.Updated)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Code is required")
	})

	t.Run("BulkUpsert rejects empty name", func(t *testing.T) {
		records := []ImportRow{
			{Row: 2, Code: "NONAM", Name: "", IsActive: true},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 0, result.Inserted)
		assert.Equal(t, 0, result.Updated)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Name is required")
	})
}
