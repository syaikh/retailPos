package inventory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/shared"
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

func TestInventoryService_AdjustStock_RepoError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))

	repo := NewRepository(mock)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()
	svc := NewService(repo, bus)

	err = svc.AdjustStock(context.Background(), 1, 5, 1, "test")
	assert.ErrorContains(t, err, "adjust stock")
}

func TestInventoryService_AdjustStockBatch(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	productA := insertTestProduct(t, ctx, "SVC-BATCH-A")
	insertTestStock(t, ctx, productA, 100)
	productB := insertTestProduct(t, ctx, "SVC-BATCH-B")
	insertTestStock(t, ctx, productB, 200)
	insertTestUser(t, ctx, 1)

	published := make(chan struct{}, 2)
	bus.Subscribe(eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.StockAdjusted},
		func(ctx context.Context, event eventbus.Event) error {
			published <- struct{}{}
			return nil
		},
	))

	err := svc.AdjustStockBatch(ctx, []StockAdjustment{
		{ProductID: productA, QuantityChange: 5},
		{ProductID: productB, QuantityChange: 10},
	}, 1, "service batch test")
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		select {
		case <-published:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for stock.adjusted events")
		}
	}

	stockA, err := svc.GetStockByProductID(ctx, productA)
	require.NoError(t, err)
	assert.Equal(t, 105, stockA.Quantity)

	stockB, err := svc.GetStockByProductID(ctx, productB)
	require.NoError(t, err)
	assert.Equal(t, 210, stockB.Quantity)
}

func TestInventoryService_AdjustStockBatch_Empty(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, &failingEventBus{})
	err := svc.AdjustStockBatch(context.Background(), nil, 1, "empty batch")
	assert.NoError(t, err)
}

func TestInventoryService_AdjustStockBatch_RepoError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))

	repo := NewRepository(mock)
	svc := NewService(repo, &failingEventBus{})

	err = svc.AdjustStockBatch(context.Background(), []StockAdjustment{{ProductID: 1, QuantityChange: 5}}, 1, "test")
	assert.ErrorContains(t, err, "adjust stock batch")
}

func TestInventoryService_LocationDelegation(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	productID := insertTestProduct(t, ctx, "SVC-LOC-001")
	var userID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO users (id, username, email, password_hash, role_id) VALUES (1, 'svc_loc', 'svc_loc@test.com', 'hash', 1) ON CONFLICT (id) DO NOTHING RETURNING id`).Scan(&userID))
	if userID == 0 {
		userID = 1
	}
	whID := createTestWarehouse(t, ctx, "SVC-WH")
	locA := createTestLocation(t, ctx, "SVC-A", "Svc Rack A", &whID, nil, true)
	locB := createTestLocation(t, ctx, "SVC-B", "Svc Rack B", &whID, nil, true)

	require.NoError(t, svc.SetLocationStock(ctx, productID, locA, 20, userID))
	items, err := svc.ListLocationStock(ctx, productID, 0)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	require.NoError(t, svc.TransferLocationStock(ctx, productID, locA, locB, 5, userID))
	items, err = svc.ListLocationStock(ctx, productID, 0)
	require.NoError(t, err)
	require.Len(t, items, 2)
}
