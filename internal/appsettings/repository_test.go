package appsettings

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

func newTestRepo() *Repository {
	return NewRepository(dbPool)
}

func TestRepository_GetAll(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo()
	ctx := context.Background()

	// Seed at least one setting.
	_, err := dbPool.Exec(ctx,
		`INSERT INTO app_settings (key, value, updated_at)
		 VALUES ('test_getall_key', 'test_getall_value', now())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
	)
	require.NoError(t, err)

	result, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test_getall_value", result["test_getall_key"])
}

func TestRepository_GetMultiple(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo()
	ctx := context.Background()

	_, err := dbPool.Exec(ctx,
		`INSERT INTO app_settings (key, value, updated_at)
		 VALUES ('store_name', 'Test Store', now())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
	)
	require.NoError(t, err)

	result, err := repo.GetMultiple(ctx, []string{"store_name", "nonexistent"})
	require.NoError(t, err)
	assert.Equal(t, "Test Store", result["store_name"])
	assert.Empty(t, result["nonexistent"])
}

func TestRepository_GetMultiple_EmptyKeys(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	result, err := repo.GetMultiple(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestRepository_UpsertMultiple(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := newTestRepo()
	ctx := context.Background()

	settings := map[string]string{
		"store_name": "Upsert Store",
		"logo_path":  "logo.png",
	}
	err := repo.UpsertMultiple(ctx, settings)
	require.NoError(t, err)

	got, err := repo.GetMultiple(ctx, []string{"store_name", "logo_path"})
	require.NoError(t, err)
	assert.Equal(t, "Upsert Store", got["store_name"])
	assert.Equal(t, "logo.png", got["logo_path"])

	// Overwrite existing key.
	settings["store_name"] = "Updated Store"
	err = repo.UpsertMultiple(ctx, settings)
	require.NoError(t, err)

	got2, err := repo.GetMultiple(ctx, []string{"store_name"})
	require.NoError(t, err)
	assert.Equal(t, "Updated Store", got2["store_name"])
}

func TestRepository_UpsertMultiple_EmptyMap(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	err := repo.UpsertMultiple(ctx, map[string]string{})
	assert.NoError(t, err)
}
