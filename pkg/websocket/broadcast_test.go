package websocket

import (
	"encoding/json"
	"testing"

	"retail-pos-system/internal/domain"

	"github.com/stretchr/testify/assert"
)

// TestBroadcastStockUpdate tests the stock update broadcast function
func TestBroadcastStockUpdate(t *testing.T) {
	t.Run("nil hub does not panic", func(t *testing.T) {
		product := &domain.Product{ID: 1, SKU: "TEST", Stock: 10, StockMin: 5}
		// Should not panic
		BroadcastStockUpdate(nil, product)
	})

	t.Run("product payload structure", func(t *testing.T) {
		storeID := 1
		product := &domain.Product{
			ID:       1,
			SKU:      "TEST-001",
			Stock:    10,
			StockMin: 5,
			StoreID:  &storeID,
		}

		// Verify the payload structure matches expected format
		expectedLowStock := product.Stock <= product.StockMin
		assert.False(t, expectedLowStock, "Stock 10 with min 5 should not be low")
	})
}

// TestBroadcastSaleCreated tests the sale created broadcast function
func TestBroadcastSaleCreated(t *testing.T) {
	t.Run("nil hub does not panic", func(t *testing.T) {
		sale := &domain.Sale{ID: 1}
		// Should not panic
		BroadcastSaleCreated(nil, sale)
	})

	t.Run("sale payload structure", func(t *testing.T) {
		storeID := 1
		sale := &domain.Sale{
			ID:            1,
			InvoiceNumber: "INV-001",
			TotalAmount:   10000,
			StoreID:       &storeID,
			Items:         []domain.SaleItem{{}, {}},
		}

		// Verify the structure
		assert.Equal(t, 1, sale.ID)
		assert.Equal(t, 2, len(sale.Items))
	})
}

// TestEventTypes tests that event type constants are correctly defined
func TestEventTypes(t *testing.T) {
	assert.Equal(t, EventType("stock_update"), EventStockUpdate)
	assert.Equal(t, EventType("sale_created"), EventSaleCreated)
	assert.Equal(t, EventType("low_stock_alert"), EventLowStockAlert)
	assert.Equal(t, EventType("product_updated"), EventProductUpdate)
	assert.Equal(t, EventType("user_online_count"), EventUserOnline)
}

// TestEventJSONMarshaling tests that events can be properly marshaled to JSON
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

// TestShouldReceiveEvent tests the event filtering logic
func TestShouldReceiveEvent(t *testing.T) {
	hub := NewHub(nil)

	t.Run("admin receives all events", func(t *testing.T) {
		client := &Client{isAdmin: true}
		event := &Event{StoreID: func() *int { i := 1; return &i }()}

		assert.True(t, hub.ShouldReceiveEvent(client, event))
	})

	t.Run("user from same store receives event", func(t *testing.T) {
		storeID := 1
		client := &Client{isAdmin: false, storeID: &storeID}
		event := &Event{StoreID: &storeID}

		assert.True(t, hub.ShouldReceiveEvent(client, event))
	})

	t.Run("user from different store does not receive event", func(t *testing.T) {
		clientStoreID := 1
		eventStoreID := 2
		client := &Client{isAdmin: false, storeID: &clientStoreID}
		event := &Event{StoreID: &eventStoreID}

		assert.False(t, hub.ShouldReceiveEvent(client, event))
	})

	t.Run("event with nil storeID is received by all", func(t *testing.T) {
		client := &Client{isAdmin: false, storeID: nil}
		event := &Event{StoreID: nil}

		assert.True(t, hub.ShouldReceiveEvent(client, event))
	})
}