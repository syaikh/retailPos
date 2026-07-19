package product

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func TestProductService_UpdatePublishesEvent(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, nil, nil, bus)
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
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for product.updated event")
	}
}

func TestProductService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, nil, nil, bus)
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
		products, total, err := svc.GetAllProducts(ctx, 10, 0, sku, "", "", "", nil, nil, nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(products), 1)
	})

	t.Run("GetProductsByIDs", func(t *testing.T) {
		p2 := &Product{
			SKU:    "SVC-READ-002",
			Name:   "Service Read Test 2",
			Price:  20000,
			Cost:   10000,
			Stock:  5,
			Status: "active",
		}
		require.NoError(t, svc.CreateProduct(ctx, p2))

		products, err := svc.GetProductsByIDs(ctx, []int{p.ID, p2.ID})
		require.NoError(t, err)
		assert.Len(t, products, 2)
	})

	t.Run("GetProductsByIDs empty", func(t *testing.T) {
		products, err := svc.GetProductsByIDs(ctx, []int{})
		require.NoError(t, err)
		assert.Empty(t, products)
	})

	t.Run("GetProductsByIDs nonexistent", func(t *testing.T) {
		products, err := svc.GetProductsByIDs(ctx, []int{-999})
		require.NoError(t, err)
		assert.Empty(t, products)
	})
}

func TestProductService_BulkUpdate(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, nil, nil, nil, bus)
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

	err := svc.BulkUpdateProductStatus(ctx, []int{p.ID}, false, nil)
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

	svc := NewService(repo, nil, nil, nil, bus)
	ctx := context.Background()

	t.Run("GetNextSKU", func(t *testing.T) {
		sku, err := svc.GetNextSKU(ctx)
		require.NoError(t, err)
		assert.Contains(t, sku, "SKU-")
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

	t.Run("Warehouse operations", func(t *testing.T) {
		_, err := dbPool.Exec(ctx, `INSERT INTO warehouses (name, code, is_active) VALUES ('Svc WH', 'SVWH01', true)`)
		require.NoError(t, err)
		list, err := svc.GetAllWarehouses(ctx)
		require.NoError(t, err)
		assert.NotNil(t, list)
	})
}
