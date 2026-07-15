package inventory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

type failingEventBus struct{}

func (f *failingEventBus) Publish(_ context.Context, _ string, _ interface{}) error {
	return fmt.Errorf("publish failed")
}

func TestInventoryService_GetStockByProductID(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	productID := insertTestProduct(t, ctx, "SVC-GET-001")
	insertTestStock(t, ctx, productID, 42)

	stock, err := svc.GetStockByProductID(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, productID, stock.ProductID)
	assert.Equal(t, 42, stock.Quantity)
}

func TestInventoryService_AdjustStock(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	t.Run("adjust and publish event", func(t *testing.T) {
		productID := insertTestProduct(t, ctx, "SVC-ADJ-001")
		insertTestStock(t, ctx, productID, 50)
		insertTestUser(t, ctx, 1)

		published := make(chan struct{}, 1)
		bus.Subscribe(eventbus.NewListenerFunc(
			[]eventbus.EventType{eventbus.StockAdjusted},
			func(ctx context.Context, event eventbus.Event) error {
				published <- struct{}{}
				return nil
			},
		))

		err := svc.AdjustStock(ctx, productID, 10, 1, "service test")
		require.NoError(t, err)

		select {
		case <-published:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for stock.adjusted event")
		}

		stock, err := svc.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 60, stock.Quantity)
	})

	t.Run("insufficient stock returns error", func(t *testing.T) {
		productID := insertTestProduct(t, ctx, "SVC-ADJ-INSF-001")
		insertTestStock(t, ctx, productID, 3)

		err := svc.AdjustStock(ctx, productID, -10, 1, "overdraft")
		assert.Error(t, err)

		stock, err := svc.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 3, stock.Quantity, "stock should remain unchanged")
	})

	t.Run("event publish failure does not block adjust", func(t *testing.T) {
		svcFailing := NewService(repo, &failingEventBus{})
		productID := insertTestProduct(t, ctx, "SVC-ADJ-FAILPUB-001")
		insertTestStock(t, ctx, productID, 20)
		insertTestUser(t, ctx, 1)

		err := svcFailing.AdjustStock(ctx, productID, 5, 1, "event fail test")
		require.NoError(t, err)

		stock, err := svcFailing.GetStockByProductID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, 25, stock.Quantity)
	})
}
