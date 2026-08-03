package brand

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

func TestBrandRepository_CRUD(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		b := &Brand{Name: "Test Brand", Description: "A test brand", IsActive: true}
		err := repo.Create(ctx, b)
		require.NoError(t, err)
		require.Greater(t, b.ID, 0)

		got, err := repo.GetByID(ctx, b.ID)
		require.NoError(t, err)
		assert.Equal(t, b.Name, got.Name)
	})

	t.Run("Get by ID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, -1)
		assert.ErrorContains(t, err, "brand not found")
	})

	t.Run("List all active brands", func(t *testing.T) {
		brands, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.NotNil(t, brands)
	})

	t.Run("Update brand", func(t *testing.T) {
		b := &Brand{Name: "Update Brand", Description: "Before", IsActive: true}
		require.NoError(t, repo.Create(ctx, b))

		b.Name = "Updated Brand"
		b.IsActive = false
		err := repo.Update(ctx, b)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, b.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Brand", got.Name)
		assert.False(t, got.IsActive)
	})

	t.Run("Delete brand", func(t *testing.T) {
		b := &Brand{Name: "Delete Brand", IsActive: true}
		require.NoError(t, repo.Create(ctx, b))

		err := repo.Delete(ctx, b.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, b.ID)
		assert.Error(t, err)
	})

	t.Run("GetIDByName", func(t *testing.T) {
		b := &Brand{Name: "BrandByName", IsActive: true}
		require.NoError(t, repo.Create(ctx, b))

		id, err := repo.GetIDByName(ctx, "BrandByName")
		require.NoError(t, err)
		assert.Equal(t, b.ID, id)
	})

	t.Run("GetIDByName inactive returns error", func(t *testing.T) {
		b := &Brand{Name: "BrandInactiveByName", IsActive: false}
		require.NoError(t, repo.Create(ctx, b))

		_, err := repo.GetIDByName(ctx, "BrandInactiveByName")
		assert.Error(t, err)
	})
}

func TestBrandRepository_Export(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("GetAllForExport returns all brands", func(t *testing.T) {
		brands, err := repo.GetAllForExport(ctx)
		require.NoError(t, err)
		assert.NotNil(t, brands)
	})
}

func TestBrandRepository_Import(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("BulkUpsert inserts new brands", func(t *testing.T) {
		records := []BrandImportRow{
			{Row: 2, Name: "ImportBrand1", Description: "Imported", IsActive: true},
			{Row: 3, Name: "ImportBrand2", Description: "Imported 2", IsActive: false},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 2, result.Inserted)
		assert.Equal(t, 0, result.Updated)
		assert.Empty(t, result.Errors)
	})

	t.Run("BulkUpsert updates existing brands", func(t *testing.T) {
		records := []BrandImportRow{
			{Row: 2, Name: "ImportBrand1", Description: "Updated", IsActive: false},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 0, result.Inserted)
		assert.Equal(t, 1, result.Updated)
		assert.Empty(t, result.Errors)
	})

	t.Run("BulkUpsert rejects empty name", func(t *testing.T) {
		records := []BrandImportRow{
			{Row: 2, Name: "", Description: "", IsActive: true},
		}
		result := repo.BulkUpsert(ctx, records)
		assert.Equal(t, 0, result.Inserted)
		assert.Equal(t, 0, result.Updated)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Name is required")
	})
}
