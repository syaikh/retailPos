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

	t.Run("GetByName success", func(t *testing.T) {
		s := &Store{Name: "Repo GetByName Target", Address: "Target Addr", Phone: "333", IsActive: true}
		err := repo.Create(ctx, s)
		require.NoError(t, err)

		fetched, err := repo.GetByName(ctx, "Repo GetByName Target")
		require.NoError(t, err)
		assert.Equal(t, "Repo GetByName Target", fetched.Name)
		assert.Equal(t, "Target Addr", fetched.Address)
		assert.Equal(t, "333", fetched.Phone)
	})

	t.Run("GetByName case insensitive", func(t *testing.T) {
		s := &Store{Name: "Repo CaseStore", IsActive: true}
		err := repo.Create(ctx, s)
		require.NoError(t, err)

		fetched, err := repo.GetByName(ctx, "repo casestore")
		require.NoError(t, err)
		assert.Equal(t, s.ID, fetched.ID)

		fetched2, err := repo.GetByName(ctx, "REPO CASESTORE")
		require.NoError(t, err)
		assert.Equal(t, s.ID, fetched2.ID)
	})

	t.Run("GetByName not found", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "Nonexistent Store XYZ 999")
		assert.Error(t, err)
	})

	t.Run("GetAll with is_active false", func(t *testing.T) {
		s := &Store{Name: "Repo Inactive Filter", IsActive: false}
		err := repo.Create(ctx, s)
		require.NoError(t, err)

		ff := false
		stores, total, err := repo.GetAll(ctx, 10, 0, "", &ff)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, st := range stores {
			assert.False(t, st.IsActive)
		}
	})

	t.Run("GetAll with is_active true", func(t *testing.T) {
		tf := true
		stores, _, err := repo.GetAll(ctx, 10, 0, "", &tf)
		require.NoError(t, err)
		for _, st := range stores {
			assert.True(t, st.IsActive)
		}
	})

	t.Run("GetAll with search and is_active", func(t *testing.T) {
		s := &Store{Name: "Repo DualFilter", IsActive: true}
		err := repo.Create(ctx, s)
		require.NoError(t, err)

		tf := true
		stores, total, err := repo.GetAll(ctx, 10, 0, "DualFilter", &tf)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Repo DualFilter", stores[0].Name)
	})

	t.Run("GetAll empty result", func(t *testing.T) {
		stores, total, err := repo.GetAll(ctx, 10, 0, "ZZZNonexistentXYZ", nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.NotNil(t, stores)
		assert.Equal(t, 0, len(stores))
	})

	t.Run("GetAllActive", func(t *testing.T) {
		stores, err := repo.GetAllActive(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stores)
	})

	t.Run("Delete non-existent", func(t *testing.T) {
		err := repo.Delete(ctx, 999999)
		assert.NoError(t, err)
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 999999)
		assert.Error(t, err)
	})
}
