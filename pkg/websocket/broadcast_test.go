package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

func TestBroadcastStockUpdate_NilHub(t *testing.T) {
	BroadcastStockUpdate(nil, StockUpdateEvent{ID: 1, SKU: "TEST", Stock: 10})
}

func TestBroadcastStockUpdate_WithHub(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	storeID := 1
	BroadcastStockUpdate(hub, StockUpdateEvent{
		ID:       42,
		SKU:      "SKU-042",
		Stock:    100,
		LowStock: false,
		StoreID:  &storeID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "stock_update")
		assert.Contains(t, string(msg), "SKU-042")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcastSaleCreated_NilHub(t *testing.T) {
	BroadcastSaleCreated(nil, SaleCreatedEvent{ID: 1})
}

func TestBroadcastSaleCreated_WithHub(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	BroadcastSaleCreated(hub, SaleCreatedEvent{
		ID:      10,
		Invoice: "INV-001",
		Total:   50000,
		Items:   3,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "sale_created")
		assert.Contains(t, string(msg), "INV-001")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcastProductUpdate_NilHub(t *testing.T) {
	BroadcastProductUpdate(nil, ProductUpdateEvent{ID: 1})
}

func TestBroadcastProductUpdate_WithHub(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	time.Sleep(50 * time.Millisecond)
	drainMessages(client.send)

	BroadcastProductUpdate(hub, ProductUpdateEvent{
		ID:    77,
		SKU:   "PROD-077",
		Stock: 50,
		Price: 25000,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "product_updated")
		assert.Contains(t, string(msg), "PROD-077")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcastLowStockAlert_NilHub(t *testing.T) {
	BroadcastLowStockAlert(nil, LowStockAlertEvent{ID: 1})
}

func TestBroadcastLowStockAlert_WithHub(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	BroadcastLowStockAlert(hub, LowStockAlertEvent{
		ID:    55,
		SKU:   "SKU-055",
		Name:  "Widget",
		Stock: 0,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "low_stock_alert")
		assert.Contains(t, string(msg), "SKU-055")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcastStockUpdate_StoreFiltering(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	adminClient := registerClient(t, hub, 2, store1, true)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)

	BroadcastStockUpdate(hub, StockUpdateEvent{
		ID:      1,
		SKU:     "SKU-001",
		Stock:   5,
		StoreID: store1,
	})

	// Admin at store1 receives it
	select {
	case msg := <-adminClient.send:
		assert.Contains(t, string(msg), "stock_update")
	case <-time.After(time.Second):
		t.Fatal("timeout: admin should receive broadcast")
	}

	// Regular client at store2 does NOT receive it
	select {
	case msg := <-regularClient.send:
		t.Fatalf("regular client at different store should not receive, got: %s", msg)
	case <-time.After(200 * time.Millisecond):
		// expected
	}
}

func TestBroadcastSaleCreated_StoreFiltering(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	drainMessages(regularClient.send)

	BroadcastSaleCreated(hub, SaleCreatedEvent{
		ID:      1,
		Invoice: "INV-001",
		StoreID: store1,
	})

	select {
	case msg := <-regularClient.send:
		t.Fatalf("regular client at different store should not receive, got: %s", msg)
	case <-time.After(200 * time.Millisecond):
		// expected
	}
}

func TestBroadcastProductUpdate_StoreFiltering(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	adminClient := registerClient(t, hub, 2, store1, true)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)

	BroadcastProductUpdate(hub, ProductUpdateEvent{
		ID:      1,
		SKU:     "SKU-001",
		StoreID: store1,
	})

	select {
	case msg := <-adminClient.send:
		assert.Contains(t, string(msg), "product_updated")
	case <-time.After(time.Second):
		t.Fatal("timeout: admin should receive broadcast")
	}

	select {
	case msg := <-regularClient.send:
		t.Fatalf("regular client at different store should not receive, got: %s", msg)
	case <-time.After(200 * time.Millisecond):
		// expected
	}
}

func TestBroadcastLowStockAlert_StoreFiltering(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	drainMessages(regularClient.send)

	BroadcastLowStockAlert(hub, LowStockAlertEvent{
		ID:      1,
		SKU:     "SKU-001",
		Name:    "Widget",
		Stock:   0,
		StoreID: store1,
	})

	select {
	case msg := <-regularClient.send:
		t.Fatalf("regular client at different store should not receive, got: %s", msg)
	case <-time.After(200 * time.Millisecond):
		// expected
	}
}
