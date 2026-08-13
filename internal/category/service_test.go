package category

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/product"
)

func TestCategoryService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	repo.SetProductQueryProvider(product.CategoryProductCountProvider{})

	svc := NewService(repo)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CreateRequest{
		Name: "SvcReadTest-" + t.Name(),
	})
	require.NoError(t, err)

	t.Run("ListCategories", func(t *testing.T) {
		list, err := svc.ListCategories(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
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
	var prodID int
	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status, category_id)
		VALUES ($1, 'Svc Blocking Product', 10000, 5000, 'active', $2)
		RETURNING id
	`, sku, cat.ID).Scan(&prodID))
	_, err = dbPool.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity) VALUES ($1, 10)
	`, prodID)
	require.NoError(t, err)

	err = svc.DeleteCategory(ctx, cat.ID)
	assert.Error(t, err)
}
