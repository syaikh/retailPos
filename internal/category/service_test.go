package category

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func TestCategoryService_CreatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"category.created"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	cat, err := svc.CreateCategory(ctx, &CategoryCreateRequest{
		Name:        "SvcCreateEvt-" + t.Name(),
		Description: "Event test",
	})
	require.NoError(t, err)
	require.Greater(t, cat.ID, 0)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for category.created event")
	}
}

func TestCategoryService_UpdatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CategoryCreateRequest{
		Name: "SvcUpdBefore-" + t.Name(),
	})
	require.NoError(t, err)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"category.updated"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	updated, err := svc.UpdateCategory(ctx, cat.ID, &CategoryUpdateRequest{
		Name:        "SvcUpdAfter-" + t.Name(),
		Description: "Updated desc",
	})
	require.NoError(t, err)
	assert.Equal(t, "SvcUpdAfter-"+t.Name(), updated.Name)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for category.updated event")
	}
}

func TestCategoryService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CategoryCreateRequest{
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

func TestCategoryService_DeletePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CategoryCreateRequest{
		Name: "SvcDelBefore-" + t.Name(),
	})
	require.NoError(t, err)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"category.deleted"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	err = svc.DeleteCategory(ctx, cat.ID)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for category.deleted event")
	}
}

func TestCategoryService_DeleteWithActiveProductsFails(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, &CategoryCreateRequest{
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
