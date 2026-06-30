package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

type testEventBus struct {
	eventbus.Bus
}

func TestProductService_CreatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"product.created"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	p := &Product{
		SKU:    "SVC-EVT-001",
		Name:   "Service Event Test",
		Price:  10000,
		Cost:   5000,
		Stock:  5,
		Status: "active",
	}
	err := svc.CreateProduct(ctx, p)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for product.created event")
	}
}

func TestProductService_UpdatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"product.updated"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	p := &Product{
		SKU:    "SVC-EVT-UPD",
		Name:   "Before",
		Price:  10000,
		Cost:   5000,
		Stock:  5,
		Status: "active",
	}
	require.NoError(t, svc.CreateProduct(ctx, p))

	p.Name = "After"
	err := svc.UpdateProduct(ctx, p)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for product.updated event")
	}
}

func TestProductService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, bus)
	ctx := context.Background()

	sku := "SVC-READ-001"
	p := &Product{
		SKU:    sku,
		Name:   "Service Read Test",
		Price:  10000,
		Cost:   5000,
		Stock:  3,
		Status: "active",
	}
	require.NoError(t, svc.CreateProduct(ctx, p))

	t.Run("GetProductByID", func(t *testing.T) {
		got, err := svc.GetProductByID(ctx, p.ID, 0)
		require.NoError(t, err)
		assert.Equal(t, p.Name, got.Name)
	})

	t.Run("GetProductBySKU", func(t *testing.T) {
		got, err := svc.GetProductBySKU(ctx, sku, 0)
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("GetAllProducts", func(t *testing.T) {
		products, total, err := svc.GetAllProducts(ctx, 10, 0, sku, "", "", "", "", nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(products), 1)
	})
}

func TestProductService_DeletePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, bus)
	ctx := context.Background()

	published := make(chan struct{}, 1)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{"product.deleted"},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	p := &Product{
		SKU:    "SVC-EVT-DEL",
		Name:   "Delete Event Test",
		Price:  5000,
		Cost:   2500,
		Stock:  1,
		Status: "active",
	}
	require.NoError(t, svc.CreateProduct(ctx, p))

	err := svc.DeleteProduct(ctx, p.ID)
	require.NoError(t, err)

	select {
	case <-published:
	case <-ctx.Done():
		t.Fatal("timeout waiting for product.deleted event")
	}
}

func TestProductService_BulkUpdate(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, bus)
	ctx := context.Background()

	p := &Product{
		SKU:    "SVC-BULK-01",
		Name:   "Bulk Target 1",
		Price:  1000,
		Cost:   500,
		Stock:  1,
		Status: "active",
	}
	require.NoError(t, svc.CreateProduct(ctx, p))

	err := svc.BulkUpdateProductStatus(ctx, []int{p.ID}, false)
	require.NoError(t, err)

	got, err := svc.GetProductByID(ctx, p.ID, 0)
	require.NoError(t, err)
	assert.Equal(t, "inactive", got.Status)
}

func TestProductService_SubResourceMethods(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, bus)
	ctx := context.Background()

	t.Run("GetNextSKU", func(t *testing.T) {
		sku, err := svc.GetNextSKU(ctx)
		require.NoError(t, err)
		assert.Contains(t, sku, "SKU-")
	})

	t.Run("Brand operations", func(t *testing.T) {
		b := &Brand{Name: "Svc Brand", IsActive: true}
		err := svc.CreateBrand(ctx, b)
		require.NoError(t, err)
		require.Greater(t, b.ID, 0)

		got, err := svc.GetBrandByID(ctx, b.ID)
		require.NoError(t, err)
		assert.Equal(t, b.Name, got.Name)

		list, err := svc.GetAllBrands(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)

		b.Name = "Svc Brand Updated"
		require.NoError(t, svc.UpdateBrand(ctx, b))

		require.NoError(t, svc.DeleteBrand(ctx, b.ID))
	})

	t.Run("Tax class operations", func(t *testing.T) {
		_, err := svc.GetTaxClassByID(ctx, -1)
		assert.Error(t, err)

		_, err = dbPool.Exec(ctx, `INSERT INTO tax_classes (name, rate_percent, is_active) VALUES ('SvcTaxList', 11, true)`)
		require.NoError(t, err)
		list, err := svc.GetAllTaxClasses(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})

	t.Run("UOM operations", func(t *testing.T) {
		u := &UnitOfMeasure{Code: "SVCM", Name: "Svc UOM", IsActive: true}
		err := svc.CreateUnitOfMeasure(ctx, u)
		require.NoError(t, err)

		got, err := svc.GetUnitOfMeasureByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, u.Code, got.Code)

		list, err := svc.GetAllUnitsOfMeasure(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)

		u.Name = "Updated UOM"
		require.NoError(t, svc.UpdateUnitOfMeasure(ctx, u))

		require.NoError(t, svc.DeleteUnitOfMeasure(ctx, u.ID))
	})

	t.Run("Warehouse operations", func(t *testing.T) {
		_, err := dbPool.Exec(ctx, `INSERT INTO warehouses (name, code, is_active) VALUES ('Svc WH', 'SVWH01', true)`)
		require.NoError(t, err)
		list, err := svc.GetAllWarehouses(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}
