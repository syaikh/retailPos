package category

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/product"
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

func TestCategoryRepository_CRUD(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	repo.SetProductQueryProvider(product.CategoryProductCountProvider{})
	ctx := context.Background()

	t.Run("Create and get by ID", func(t *testing.T) {
		c := &Category{
			Name:        "CreateGet-" + t.Name(),
			Description: "Integration test",
			IsActive:    true,
		}
		err := repo.CreateCategory(ctx, c)
		require.NoError(t, err)
		require.Greater(t, c.ID, 0)
		require.NotEmpty(t, c.Slug)

		got, err := repo.GetCategoryByID(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, c.Name, got.Name)
		assert.Equal(t, c.Slug, got.Slug)
		assert.Equal(t, c.Description, got.Description)
		assert.True(t, got.IsActive)
	})

	t.Run("GetCategoryByID not found", func(t *testing.T) {
		_, err := repo.GetCategoryByID(ctx, -1)
		assert.ErrorContains(t, err, "category not found")
	})

	t.Run("GetCategoryIDByName", func(t *testing.T) {
		name := "GetByName-" + t.Name()
		c := &Category{Name: name, Description: "", IsActive: true}
		require.NoError(t, repo.CreateCategory(ctx, c))

		id, err := repo.GetCategoryIDByName(ctx, name)
		require.NoError(t, err)
		assert.Equal(t, c.ID, id)
	})

	t.Run("GetCategoryIDByName inactive returns error", func(t *testing.T) {
		name := "InactiveByName-" + t.Name()
		c := &Category{Name: name, Description: "", IsActive: false}
		require.NoError(t, repo.CreateCategory(ctx, c))

		_, err := repo.GetCategoryIDByName(ctx, name)
		assert.Error(t, err)
	})

	t.Run("ListCategories returns only active", func(t *testing.T) {
		activeName := "ListActiveOk-" + t.Name()
		inactiveName := "ListInactiveSkip-" + t.Name()

		active := &Category{Name: activeName, Description: "", IsActive: true}
		inactive := &Category{Name: inactiveName, Description: "", IsActive: false}
		require.NoError(t, repo.CreateCategory(ctx, active))
		require.NoError(t, repo.CreateCategory(ctx, inactive))

		list, err := repo.ListCategories(ctx)
		require.NoError(t, err)

		var foundActive, foundInactive bool
		for _, cat := range list {
			if cat.ID == active.ID {
				foundActive = true
			}
			if cat.ID == inactive.ID {
				foundInactive = true
			}
		}
		assert.True(t, foundActive, "active category should be listed")
		assert.False(t, foundInactive, "inactive category should not be listed")
	})

	t.Run("GetAllCategories", func(t *testing.T) {
		prefix := "GetAllCat-" + t.Name()
		for i := 0; i < 3; i++ {
			c := &Category{Name: fmt.Sprintf("%s-%d", prefix, i), IsActive: true}
			require.NoError(t, repo.CreateCategory(ctx, c))
		}

		t.Run("paginated", func(t *testing.T) {
			cats, total, err := repo.GetAllCategories(ctx, 2, 0, "")
			require.NoError(t, err)
			assert.LessOrEqual(t, len(cats), 2)
			assert.Greater(t, total, 0)
		})

		t.Run("search matches", func(t *testing.T) {
			cats, total, err := repo.GetAllCategories(ctx, 10, 0, prefix)
			require.NoError(t, err)
			assert.Equal(t, 3, total)
			assert.Equal(t, 3, len(cats))
		})

		t.Run("search no results", func(t *testing.T) {
			cats, total, err := repo.GetAllCategories(ctx, 10, 0, "ZZZZnoMatchXXXX")
			require.NoError(t, err)
			assert.Equal(t, 0, total)
			assert.Empty(t, cats)
		})

		t.Run("product count counts only active products", func(t *testing.T) {
			cat := &Category{Name: prefix + "-counted", IsActive: true}
			require.NoError(t, repo.CreateCategory(ctx, cat))

			_, err := dbPool.Exec(ctx, `
				INSERT INTO products (sku, name, price, cost, stock, status, category_id)
				VALUES ($1, 'Counted Active', 10000, 5000, 10, 'active', $2)
			`, fmt.Sprintf("CNT-ACT-%d", cat.ID), cat.ID)
			require.NoError(t, err)

			_, err = dbPool.Exec(ctx, `
				INSERT INTO products (sku, name, price, cost, stock, status, category_id, deleted_at)
				VALUES ($1, 'Counted Deleted', 10000, 5000, 10, 'inactive', $2, NOW())
			`, fmt.Sprintf("CNT-DEL-%d", cat.ID), cat.ID)
			require.NoError(t, err)

			cats, _, err := repo.GetAllCategories(ctx, 10, 0, prefix+"-counted")
			require.NoError(t, err)
			require.Len(t, cats, 1)
			assert.Equal(t, 1, cats[0].ProductCount)
		})
	})

	t.Run("SlugExists", func(t *testing.T) {
		name := "SlugCheck-" + t.Name()
		c := &Category{Name: name, IsActive: true}
		require.NoError(t, repo.CreateCategory(ctx, c))

		exists, err := repo.SlugExists(ctx, c.Slug, 0)
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = repo.SlugExists(ctx, c.Slug, c.ID)
		require.NoError(t, err)
		assert.False(t, exists)

		exists, err = repo.SlugExists(ctx, "nonexistent-slug-xyz", 0)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("UpdateCategory", func(t *testing.T) {
		c := &Category{Name: "BeforeUpd-" + t.Name(), Description: "Old", IsActive: true}
		require.NoError(t, repo.CreateCategory(ctx, c))

		c.Name = "AfterUpd-" + t.Name()
		c.Description = "Updated"
		c.IsActive = false
		err := repo.UpdateCategory(ctx, c)
		require.NoError(t, err)

		got, err := repo.GetCategoryByID(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, "AfterUpd-"+t.Name(), got.Name)
		assert.Equal(t, "Updated", got.Description)
		assert.False(t, got.IsActive)
	})

	t.Run("HasActiveProducts", func(t *testing.T) {
		cat := &Category{Name: "HasProd-" + t.Name(), IsActive: true}
		require.NoError(t, repo.CreateCategory(ctx, cat))

		has, err := repo.HasActiveProducts(ctx, cat.ID)
		require.NoError(t, err)
		assert.False(t, has)

		sku := fmt.Sprintf("HAS-PROD-%d", cat.ID)
		_, err = dbPool.Exec(ctx, `
			INSERT INTO products (sku, name, price, cost, stock, status, category_id)
			VALUES ($1, 'HasActive Test Product', 10000, 5000, 10, 'active', $2)
		`, sku, cat.ID)
		require.NoError(t, err)

		has, err = repo.HasActiveProducts(ctx, cat.ID)
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("DeleteCategory", func(t *testing.T) {
		c := &Category{Name: "DeleteOk-" + t.Name(), IsActive: true}
		require.NoError(t, repo.CreateCategory(ctx, c))

		err := repo.DeleteCategory(ctx, c.ID)
		require.NoError(t, err)

		_, err = repo.GetCategoryByID(ctx, c.ID)
		assert.ErrorContains(t, err, "category not found")
	})

	t.Run("DeleteCategory with active products fails", func(t *testing.T) {
		cat := &Category{Name: "CantDelete-" + t.Name(), IsActive: true}
		require.NoError(t, repo.CreateCategory(ctx, cat))

		sku := fmt.Sprintf("BLOCK-DEL-%d", cat.ID)
		_, err := dbPool.Exec(ctx, `
			INSERT INTO products (sku, name, price, cost, stock, status, category_id)
			VALUES ($1, 'Blocking Product', 10000, 5000, 10, 'active', $2)
		`, sku, cat.ID)
		require.NoError(t, err)

		err = repo.DeleteCategory(ctx, cat.ID)
		assert.Error(t, err)
	})
}
