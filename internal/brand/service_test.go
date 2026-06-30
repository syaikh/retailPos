package brand

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func TestBrandService_CreatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"brand.created"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	brand, err := svc.Create(ctx, &BrandCreateRequest{
		Name:        "SvcCreateBrand",
		Description: "Service create test",
	})
	require.NoError(t, err)
	require.Greater(t, brand.ID, 0)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for brand.created event")
	}
}

func TestBrandService_UpdatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	brand, err := svc.Create(ctx, &BrandCreateRequest{
		Name: "SvcUpdateBefore",
	})
	require.NoError(t, err)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"brand.updated"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	updated, err := svc.Update(ctx, brand.ID, &BrandUpdateRequest{
		Name:        "SvcUpdateAfter",
		Description: "Updated desc",
	})
	require.NoError(t, err)
	assert.Equal(t, "SvcUpdateAfter", updated.Name)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for brand.updated event")
	}
}

func TestBrandService_DeletePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	brand, err := svc.Create(ctx, &BrandCreateRequest{
		Name: "SvcDeleteBrand",
	})
	require.NoError(t, err)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"brand.deleted"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	err = svc.Delete(ctx, brand.ID)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for brand.deleted event")
	}
}

func TestBrandService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
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

func TestBrandService_GetAllForExport(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	brands, err := svc.GetAllForExport(ctx)
	require.NoError(t, err)
	assert.NotNil(t, brands)
}

func TestBrandService_Import(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	records := []BrandImportRow{
		{Row: 2, Name: "SvcImportBrand", Description: "Imported", IsActive: true},
	}
	result := svc.Import(ctx, records)
	assert.Equal(t, 1, result.Inserted)
	assert.Empty(t, result.Errors)
}

func TestBrandService_CreateWithIsActiveFalse(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
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
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	_, err := svc.Update(ctx, -1, &BrandUpdateRequest{
		Name: "NonExistent",
	})
	assert.Error(t, err)
}
