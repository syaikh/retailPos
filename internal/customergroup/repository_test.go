package customergroup

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

func TestCustomerGroupRepository_CRUD(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		cg := &CustomerGroup{
			Name:        "Test Group CRUD",
			Description: "Integration test",
			IsActive:    true,
		}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)
		require.Greater(t, cg.ID, 0)

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Group CRUD", fetched.Name)
		assert.Equal(t, "Integration test", fetched.Description)
		assert.True(t, fetched.IsActive)
	})

	t.Run("Get all with pagination", func(t *testing.T) {
		groups, total, err := repo.GetAll(ctx, 10, 0, "", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("Get all with search", func(t *testing.T) {
		groups, total, err := repo.GetAll(ctx, 10, 0, "Test Group CRUD", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("Get all with is_active filter", func(t *testing.T) {
		active := true
		groups, _, err := repo.GetAll(ctx, 10, 0, "", &active)
		require.NoError(t, err)
		for _, g := range groups {
			assert.True(t, g.IsActive)
		}
	})

	t.Run("Get all active", func(t *testing.T) {
		groups, err := repo.GetAllActive(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(groups), 1)
	})

	t.Run("NameExists", func(t *testing.T) {
		exists, err := repo.NameExists(ctx, "Test Group CRUD", 0)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Update", func(t *testing.T) {
		// Create a fresh group for update testing
		cg := &CustomerGroup{Name: "To Update", Description: "", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, cg.ID)
		require.NoError(t, err)
		fetched.Name = "Updated Group"
		err = repo.Update(ctx, fetched)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, fetched.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Group", updated.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		cg := &CustomerGroup{Name: "To Delete", Description: "", IsActive: true}
		err := repo.Create(ctx, cg)
		require.NoError(t, err)

		err = repo.Delete(ctx, cg.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, cg.ID)
		assert.Error(t, err)
	})
}
