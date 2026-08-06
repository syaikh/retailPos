package uom

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUOMService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	uom, err := svc.Create(ctx, &CreateRequest{
		Code: "SVCREAD",
		Name: "Svc Read UOM",
	})
	require.NoError(t, err)

	t.Run("GetByID", func(t *testing.T) {
		got, err := svc.GetByID(ctx, uom.ID)
		require.NoError(t, err)
		assert.Equal(t, uom.Code, got.Code)
	})

	t.Run("GetAll", func(t *testing.T) {
		list, err := svc.GetAll(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})

	t.Run("GetIDByCode", func(t *testing.T) {
		id, err := svc.GetIDByCode(ctx, uom.Code)
		require.NoError(t, err)
		assert.Equal(t, uom.ID, id)
	})
}

func TestUOMService_CreateWithIsActiveFalse(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	isActive := false
	uom, err := svc.Create(ctx, &CreateRequest{
		Code:     "SVCINACT",
		Name:     "Svc Inactive UOM",
		IsActive: &isActive,
	})
	require.NoError(t, err)
	assert.False(t, uom.IsActive)
}

func TestUOMService_UpdateNotFound(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Update(ctx, -1, &UpdateRequest{
		Code: "NONEXIST",
		Name: "NonExistent",
	})
	assert.Error(t, err)
}
