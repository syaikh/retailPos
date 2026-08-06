package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"
)

func TestPurchaseReceiptListener_HandleEvent_AdjustsStock(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, eventbus.New())
	listener := NewPurchaseReceiptListener(repo, svc)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "LISTENER-PROD")
	insertTestStock(t, ctx, prodID, 100)
	userID := 9999
	_, err := dbPool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET email = excluded.email
	`, userID, "listener_user_"+string(rune(userID)), "listener@test.com", "hash", 1)
	require.NoError(t, err)

	stockBefore, err := repo.GetStockByProductID(ctx, prodID)
	require.NoError(t, err)
	require.NotNil(t, stockBefore)

	payload := events.PurchaseReceiptCompleted{
		POID:    1,
		GRID:    1,
		StoreID: 1,
		UserID:  userID,
		Items: []events.PurchaseReceiptItem{
			{ProductID: prodID, QtyGood: 5},
		},
	}

	event := eventbus.Event{
		Type:      events.TopicPurchaseReceiptCompleted,
		Payload:   payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	err = listener.HandleEvent(ctx, event)
	require.NoError(t, err)

	stockAfter, err := repo.GetStockByProductID(ctx, prodID)
	require.NoError(t, err)
	require.NotNil(t, stockAfter)
	assert.Equal(t, 105, stockAfter.Quantity)
}

func TestPurchaseReceiptListener_HandleEvent_AdjustsMultipleItemsInOneBatch(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, eventbus.New())
	listener := NewPurchaseReceiptListener(repo, svc)
	ctx := context.Background()

	prodA := insertTestProduct(t, ctx, "LISTENER-MULTI-A")
	insertTestStock(t, ctx, prodA, 100)
	prodB := insertTestProduct(t, ctx, "LISTENER-MULTI-B")
	insertTestStock(t, ctx, prodB, 50)
	userID := 9998
	_, err := dbPool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET email = excluded.email
	`, userID, "listener_user_multi", "listener_multi@test.com", "hash", 1)
	require.NoError(t, err)

	payload := events.PurchaseReceiptCompleted{
		POID:    2,
		GRID:    2,
		StoreID: 1,
		UserID:  userID,
		Items: []events.PurchaseReceiptItem{
			{ProductID: prodA, QtyGood: 5},
			{ProductID: prodB, QtyGood: 10},
			{ProductID: prodB, QtyGood: 0},
		},
	}

	event := eventbus.Event{
		Type:      events.TopicPurchaseReceiptCompleted,
		Payload:   payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	err = listener.HandleEvent(ctx, event)
	require.NoError(t, err)

	stockA, err := repo.GetStockByProductID(ctx, prodA)
	require.NoError(t, err)
	assert.Equal(t, 105, stockA.Quantity)

	stockB, err := repo.GetStockByProductID(ctx, prodB)
	require.NoError(t, err)
	assert.Equal(t, 60, stockB.Quantity)
}

func TestPurchaseReceiptListener_HandleEvent_IsIdempotent(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, eventbus.New())
	listener := NewPurchaseReceiptListener(repo, svc)
	ctx := context.Background()

	prodID := insertTestProduct(t, ctx, "LISTENER-IDEMP")
	insertTestStock(t, ctx, prodID, 100)
	userID := 9997
	_, err := dbPool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET email = excluded.email
	`, userID, "listener_user_idemp", "listener_idemp@test.com", "hash", 1)
	require.NoError(t, err)

	payload := events.PurchaseReceiptCompleted{
		POID:    3,
		GRID:    3,
		StoreID: 1,
		UserID:  userID,
		Items: []events.PurchaseReceiptItem{
			{ProductID: prodID, QtyGood: 5},
		},
	}
	event := eventbus.Event{
		Type:      events.TopicPurchaseReceiptCompleted,
		Payload:   payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	require.NoError(t, listener.HandleEvent(ctx, event))
	require.NoError(t, listener.HandleEvent(ctx, event))

	stock, err := repo.GetStockByProductID(ctx, prodID)
	require.NoError(t, err)
	assert.Equal(t, 105, stock.Quantity, "duplicate event must not double-adjust stock")
}
