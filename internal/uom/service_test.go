package uom

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func TestUOMService_CreatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"uom.created"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	uom, err := svc.Create(ctx, &UOMCreateRequest{
		Code: "SVCEVT",
		Name: "Svc Create UOM",
	})
	require.NoError(t, err)
	require.Greater(t, uom.ID, 0)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for uom.created event")
	}
}

func TestUOMService_UpdatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	uom, err := svc.Create(ctx, &UOMCreateRequest{
		Code: "SVCUPD",
		Name: "Svc Update Before",
	})
	require.NoError(t, err)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"uom.updated"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	updated, err := svc.Update(ctx, uom.ID, &UOMUpdateRequest{
		Code: "SVCUPD",
		Name: "Svc Update After",
	})
	require.NoError(t, err)
	assert.Equal(t, "Svc Update After", updated.Name)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for uom.updated event")
	}
}

func TestUOMService_DeletePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	uom, err := svc.Create(ctx, &UOMCreateRequest{
		Code: "SVCDEL",
		Name: "Svc Delete UOM",
	})
	require.NoError(t, err)

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"uom.deleted"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	err = svc.Delete(ctx, uom.ID)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for uom.deleted event")
	}
}

func TestUOMService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	uom, err := svc.Create(ctx, &UOMCreateRequest{
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
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	isActive := false
	uom, err := svc.Create(ctx, &UOMCreateRequest{
		Code:     "SVCINACT",
		Name:     "Svc Inactive UOM",
		IsActive: &isActive,
	})
	require.NoError(t, err)
	assert.False(t, uom.IsActive)
}

func TestUOMService_UpdateNotFound(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	_, err := svc.Update(ctx, -1, &UOMUpdateRequest{
		Code: "NONEXIST",
		Name: "NonExistent",
	})
	assert.Error(t, err)
}
