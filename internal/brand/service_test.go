package brand

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrandService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	brand, err := svc.Create(ctx, &BrandCreateRequest{
		Name: "SvcReadBrand",
	})
	require.NoError(t, err)

	t.Run("GetByID", func(t *testing.T) {
		got, err := svc.GetByID(ctx, brand.ID)
		require.NoError(t, err)
		assert.Equal(t, brand.Name, got.Name)
	})

	t.Run("GetAll", func(t *testing.T) {
		list, err := svc.GetAll(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})

	t.Run("GetIDByName", func(t *testing.T) {
		id, err := svc.GetIDByName(ctx, brand.Name)
		require.NoError(t, err)
		assert.Equal(t, brand.ID, id)
	})
}

func TestBrandService_CreateWithIsActiveFalse(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	isActive := false
	brand, err := svc.Create(ctx, &BrandCreateRequest{
		Name:     "SvcInactiveBrand",
		IsActive: &isActive,
	})
	require.NoError(t, err)
	assert.False(t, brand.IsActive)
}

func TestBrandService_UpdateNotFound(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Update(ctx, -1, &BrandUpdateRequest{
		Name: "NonExistent",
	})
	assert.Error(t, err)
}
