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
	repo := newTestRepo(t)
	svc := NewService(repo, eventbus.New())
	listener := NewPurchaseReceiptListener(repo, svc)
	ctx := context.Background()

	prodID := insertTestProduct(ctx, t, "LISTENER-PROD")
	insertTestUser(ctx, t, 1)
	storeID := createTestStore(ctx, t, "LISTENER-STORE")
	_, err := dbPool.Exec(ctx, `UPDATE products SET store_id = $1 WHERE id = $2`, storeID, prodID)
	require.NoError(t, err)
	userID := 9999
	_, err = dbPool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET email = excluded.email
	`, userID, "listener_user_"+string(rune(userID)), "listener@test.com", "hash", 1)
	require.NoError(t, err)

	payload := events.PurchaseReceiptCompleted{
		POID:    1,
		GRID:    1,
		StoreID: storeID,
		UserID:  userID,
		Items: []events.PurchaseReceiptItem{
			{ProductID: prodID, QtyGood: 5},
		},
	}

	event := eventbus.Event{
		Type:      events.TopicPurchaseReceiptCompleted,
		Payload:   &payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	err = listener.HandleEvent(ctx, event)
	require.NoError(t, err)

	stockAfter, err := repo.GetStockByProductID(ctx, prodID)
	require.NoError(t, err)
	require.NotNil(t, stockAfter)
	assert.Equal(t, 5, stockAfter.Quantity)
}

func TestPurchaseReceiptListener_HandleEvent_AdjustsMultipleItemsInOneBatch(t *testing.T) {
	repo := newTestRepo(t)
	svc := NewService(repo, eventbus.New())
	listener := NewPurchaseReceiptListener(repo, svc)
	ctx := context.Background()

	prodA := insertTestProduct(ctx, t, "LISTENER-MULTI-A")
	prodB := insertTestProduct(ctx, t, "LISTENER-MULTI-B")
	insertTestUser(ctx, t, 1)
	storeID := createTestStore(ctx, t, "LISTENER-MULTI-STORE")
	_, err := dbPool.Exec(ctx, `UPDATE products SET store_id = $1 WHERE id = ANY($2)`, storeID, []int{prodA, prodB})
	require.NoError(t, err)
	userID := 9998
	_, err = dbPool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET email = excluded.email
	`, userID, "listener_user_multi", "listener_multi@test.com", "hash", 1)
	require.NoError(t, err)

	payload := events.PurchaseReceiptCompleted{
		POID:    2,
		GRID:    2,
		StoreID: storeID,
		UserID:  userID,
		Items: []events.PurchaseReceiptItem{
			{ProductID: prodA, QtyGood: 5},
			{ProductID: prodB, QtyGood: 10},
			{ProductID: prodB, QtyGood: 0},
		},
	}

	event := eventbus.Event{
		Type:      events.TopicPurchaseReceiptCompleted,
		Payload:   &payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	err = listener.HandleEvent(ctx, event)
	require.NoError(t, err)

	stockA, err := repo.GetStockByProductID(ctx, prodA)
	require.NoError(t, err)
	assert.Equal(t, 5, stockA.Quantity)

	stockB, err := repo.GetStockByProductID(ctx, prodB)
	require.NoError(t, err)
	assert.Equal(t, 10, stockB.Quantity)
}

func TestPurchaseReceiptListener_HandleEvent_IsIdempotent(t *testing.T) {
	repo := newTestRepo(t)
	svc := NewService(repo, eventbus.New())
	listener := NewPurchaseReceiptListener(repo, svc)
	ctx := context.Background()

	prodID := insertTestProduct(ctx, t, "LISTENER-IDEMP")
	insertTestUser(ctx, t, 1)
	storeID := createTestStore(ctx, t, "LISTENER-IDEMP-STORE")
	_, err := dbPool.Exec(ctx, `UPDATE products SET store_id = $1 WHERE id = $2`, storeID, prodID)
	require.NoError(t, err)
	userID := 9997
	_, err = dbPool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET email = excluded.email
	`, userID, "listener_user_idemp", "listener_idemp@test.com", "hash", 1)
	require.NoError(t, err)

	payload := events.PurchaseReceiptCompleted{
		POID:    3,
		GRID:    3,
		StoreID: storeID,
		UserID:  userID,
		Items: []events.PurchaseReceiptItem{
			{ProductID: prodID, QtyGood: 5},
		},
	}
	event := eventbus.Event{
		Type:      events.TopicPurchaseReceiptCompleted,
		Payload:   &payload,
		Ctx:       ctx,
		Timestamp: time.Now(),
	}

	require.NoError(t, listener.HandleEvent(ctx, event))
	require.NoError(t, listener.HandleEvent(ctx, event))

	stock, err := repo.GetStockByProductID(ctx, prodID)
	require.NoError(t, err)
	assert.Equal(t, 5, stock.Quantity, "duplicate event must not double-adjust stock")
}
