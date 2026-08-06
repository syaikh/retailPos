package category

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CreateRequest{
		Name: "SvcReadTest-" + t.Name(),
	})
	require.NoError(t, err)

	t.Run("ListCategories", func(t *testing.T) {
		list, err := svc.ListCategories(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})

	t.Run("GetCategoryByID", func(t *testing.T) {
		got, err := svc.GetCategoryByID(ctx, cat.ID)
		require.NoError(t, err)
		assert.Equal(t, cat.Name, got.Name)
	})

	t.Run("GetAllCategories", func(t *testing.T) {
		cats, total, err := svc.GetAllCategories(ctx, 10, 0, cat.Name)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(cats), 1)
	})

	t.Run("SlugExists", func(t *testing.T) {
		exists, err := svc.SlugExists(ctx, cat.Slug, cat.ID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("HasActiveProducts", func(t *testing.T) {
		has, err := svc.HasActiveProducts(ctx, cat.ID)
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestCategoryService_DeleteWithActiveProductsFails(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CreateRequest{
		Name: "SvcCantDel-" + t.Name(),
	})
	require.NoError(t, err)

	sku := fmt.Sprintf("SVC-BLOCK-%d", cat.ID)
	_, err = dbPool.Exec(ctx, `
		INSERT INTO products (sku, name, price, cost, stock, status, category_id)
		VALUES ($1, 'Svc Blocking Product', 10000, 5000, 10, 'active', $2)
	`, sku, cat.ID)
	require.NoError(t, err)

	err = svc.DeleteCategory(ctx, cat.ID)
	assert.Error(t, err)
}
