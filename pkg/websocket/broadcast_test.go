package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBroadcastStockUpdate(t *testing.T) {
	t.Run("nil hub does not panic", func(t *testing.T) {
		evt := StockUpdateEvent{ID: 1, SKU: "TEST", Stock: 10}
		BroadcastStockUpdate(nil, evt)
	})

	t.Run("product payload structure", func(t *testing.T) {
		storeID := 1
		evt := StockUpdateEvent{
			ID:       1,
			SKU:      "TEST-001",
			Stock:    10,
			StoreID:  &storeID,
		}
		assert.Equal(t, 10, evt.Stock)
	})
}

func TestBroadcastSaleCreated(t *testing.T) {
	t.Run("nil hub does not panic", func(t *testing.T) {
		evt := SaleCreatedEvent{ID: 1}
		BroadcastSaleCreated(nil, evt)
	})

	t.Run("sale payload structure", func(t *testing.T) {
		storeID := 1
		evt := SaleCreatedEvent{
			ID:      1,
			Invoice: "INV-001",
			Total:   10000,
			StoreID: &storeID,
		}
		assert.Equal(t, 1, evt.ID)
		assert.Equal(t, "INV-001", evt.Invoice)
	})
}

func TestEventTypes(t *testing.T) {
	assert.Equal(t, EventType("stock_update"), EventStockUpdate)
	assert.Equal(t, EventType("sale_created"), EventSaleCreated)
	assert.Equal(t, EventType("low_stock_alert"), EventLowStockAlert)
	assert.Equal(t, EventType("product_updated"), EventProductUpdate)
	assert.Equal(t, EventType("user_online_count"), EventUserOnline)
}

func TestEventJSONMarshaling(t *testing.T) {
	t.Run("event marshals correctly", func(t *testing.T) {
		storeID := 1
		event := Event{
			Type:    EventStockUpdate,
			Payload: json.RawMessage(`{"test":123}`),
			StoreID: &storeID,
		}

		data, err := json.Marshal(event)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "stock_update")
		assert.Contains(t, string(data), "test")
	})
}


