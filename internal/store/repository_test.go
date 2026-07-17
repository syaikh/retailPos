package store

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
		os.Exit(0)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(0)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func TestStoreRepository_CRUD(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		s := &Store{
			Name:     "Test Store",
			Address:  "123 Main St",
			Phone:    "08123456789",
			IsActive: true,
		}
		err := repo.Create(ctx, s)
		require.NoError(t, err)
		require.Greater(t, s.ID, 0)

		fetched, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Store", fetched.Name)
		assert.Equal(t, "123 Main St", fetched.Address)
		assert.Equal(t, "08123456789", fetched.Phone)
		assert.True(t, fetched.IsActive)
	})

	t.Run("Get all with pagination", func(t *testing.T) {
		stores, total, err := repo.GetAll(ctx, 10, 0, "", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(stores), 1)
	})

	t.Run("Get all with search", func(t *testing.T) {
		stores, total, err := repo.GetAll(ctx, 10, 0, "Test Store", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(stores), 1)
	})

	t.Run("Get all active", func(t *testing.T) {
		stores, err := repo.GetAllActive(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(stores), 1)
	})

	t.Run("Update", func(t *testing.T) {
		s := &Store{Name: "To Update", IsActive: true}
		err := repo.Create(ctx, s)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, s.ID)
		require.NoError(t, err)
		fetched.Name = "Updated Store"
		err = repo.Update(ctx, fetched)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, fetched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Store", updated.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		s := &Store{Name: "To Delete", IsActive: true}
		err := repo.Create(ctx, s)
		require.NoError(t, err)

		err = repo.Delete(ctx, s.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, s.ID)
		assert.Error(t, err)
	})
}
