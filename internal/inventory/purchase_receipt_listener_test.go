package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/purchase"
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

	payload := purchase.PurchaseReceiptPayload{
		POID:    1,
		GRID:    1,
		StoreID: 1,
		UserID:  userID,
		Items: []purchase.PurchaseReceiptItem{
			{ProductID: prodID, QtyGood: 5},
		},
	}

	event := eventbus.Event{
		Type:      "PurchaseReceiptCompleted",
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
