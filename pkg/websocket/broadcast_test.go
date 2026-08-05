package websocket

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func waitForMessage(t *testing.T, ch <-chan []byte, predicate func(string) bool, timeout time.Duration) (string, bool) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case msg := <-ch:
			s := string(msg)
			if predicate(s) {
				return s, true
			}
		case <-timer.C:
			return "", false
		}
	}
}

func TestEventTypes(t *testing.T) {
	assert.Equal(t, EventType("stock_update"), EventStockUpdate)
	assert.Equal(t, EventType("sale_created"), EventSaleCreated)
	assert.Equal(t, EventType("low_stock_alert"), EventLowStockAlert)
	assert.Equal(t, EventType("product_updated"), EventProductUpdate)
	assert.Equal(t, EventType("user_online_count"), EventUserOnline)
	assert.Equal(t, EventType("po_received"), EventPOReceived)
	assert.Equal(t, EventType("po_created"), EventPOCreated)
	assert.Equal(t, EventType("po_confirmed"), EventPOConfirmed)
	assert.Equal(t, EventType("po_cancelled"), EventPOCancelled)
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
	hub := newListenerHub()
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

	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg := <-client.send:
			s := string(msg)
			if strings.Contains(s, "stock_update") && strings.Contains(s, "SKU-042") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for stock_update broadcast with SKU-042")
		}
	}
}

func TestBroadcastSaleCreated_NilHub(t *testing.T) {
	BroadcastSaleCreated(nil, SaleCreatedEvent{ID: 1})
}

func TestBroadcastSaleCreated_WithHub(t *testing.T) {
	hub := newListenerHub()
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

	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg := <-client.send:
			if strings.Contains(string(msg), "sale_created") && strings.Contains(string(msg), "INV-001") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for sale_created broadcast")
		}
	}
}

func TestBroadcastProductUpdate_NilHub(t *testing.T) {
	BroadcastProductUpdate(nil, ProductUpdateEvent{ID: 1})
}

func TestBroadcastProductUpdate_WithHub(t *testing.T) {
	hub := newListenerHub()
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

	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg := <-client.send:
			if strings.Contains(string(msg), "product_updated") && strings.Contains(string(msg), "PROD-077") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for product_updated broadcast")
		}
	}
}

func TestBroadcastLowStockAlert_NilHub(t *testing.T) {
	BroadcastLowStockAlert(nil, LowStockAlertEvent{ID: 1})
}

func TestBroadcastLowStockAlert_WithHub(t *testing.T) {
	hub := newListenerHub()
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

	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg := <-client.send:
			if strings.Contains(string(msg), "low_stock_alert") && strings.Contains(string(msg), "SKU-055") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for low_stock_alert broadcast")
		}
	}
}

func TestBroadcastStockUpdate_StoreFiltering(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	adminClient := registerClient(t, hub, 2, store1, true)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)

	BroadcastStockUpdate(hub, StockUpdateEvent{
		ID:      1,
		SKU:     "SKU-001",
		Stock:   5,
		StoreID: store1,
	})

	// Admin at store1 receives it
	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg := <-adminClient.send:
			if strings.Contains(string(msg), "stock_update") {
				goto doneStock
			}
		case <-timeout:
			t.Fatal("timeout: admin should receive broadcast")
		}
	}
doneStock:

	// Regular client at store2 does NOT receive it
	_, received := waitForMessage(t, regularClient.send, func(s string) bool {
		return strings.Contains(s, "stock_update")
	}, 300*time.Millisecond)
	if received {
		t.Fatal("regular client at different store should not receive stock_update")
	}
}

func TestBroadcastSaleCreated_StoreFiltering(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)

	BroadcastSaleCreated(hub, SaleCreatedEvent{
		ID:      1,
		Invoice: "INV-001",
		StoreID: store1,
	})

	_, received := waitForMessage(t, regularClient.send, func(s string) bool {
		return strings.Contains(s, "sale_created")
	}, 300*time.Millisecond)
	if received {
		t.Fatal("regular client at different store should not receive sale_created")
	}
}

func TestBroadcastProductUpdate_StoreFiltering(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	adminClient := registerClient(t, hub, 2, store1, true)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)

	BroadcastProductUpdate(hub, ProductUpdateEvent{
		ID:      1,
		SKU:     "SKU-001",
		StoreID: store1,
	})

	_, ok := waitForMessage(t, adminClient.send, func(s string) bool {
		return strings.Contains(s, "product_updated")
	}, 2*time.Second)
	if !ok {
		t.Fatal("timeout: admin should receive broadcast")
	}

	_, received := waitForMessage(t, regularClient.send, func(s string) bool {
		return strings.Contains(s, "product_updated")
	}, 300*time.Millisecond)
	if received {
		t.Fatal("regular client at different store should not receive product_updated")
	}
}

func TestBroadcastPOReceived_NilHub(t *testing.T) {
	BroadcastPOReceived(nil, POReceivedEvent{POID: 1, PONumber: "PO-001", GRNumber: "GR-001"})
}

func TestBroadcastPOReceived_WithHub(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	BroadcastPOReceived(hub, POReceivedEvent{
		POID:     42,
		PONumber: "PO-042",
		GRNumber: "GR-007",
	})

	msg, ok := waitForMessage(t, client.send, func(s string) bool {
		return strings.Contains(s, "po_received") && strings.Contains(s, "PO-042") && strings.Contains(s, "GR-007")
	}, 2*time.Second)
	if !ok {
		t.Fatal("timeout waiting for po_received broadcast")
	}

	var event Event
	err := json.Unmarshal([]byte(msg), &event)
	assert.NoError(t, err)
	assert.Equal(t, EventPOReceived, event.Type)
}

func TestBroadcastPOReceived_AdminReceivesWithStoreID(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	BroadcastPOReceived(hub, POReceivedEvent{
		POID:     1,
		PONumber: "PO-001",
		GRNumber: "GR-001",
		StoreID:  store1,
	})

	_, ok := waitForMessage(t, client.send, func(s string) bool {
		return strings.Contains(s, "po_received")
	}, 2*time.Second)
	if !ok {
		t.Fatal("timeout waiting for po_received broadcast with store_id")
	}
}

func TestBroadcastPOEvents(t *testing.T) {
	type eventCase struct {
		name          string
		broadcastType EventType
		wsEventName   string
	}

	newEventCases := func() []eventCase {
		return []eventCase{
			{
				name:          "POCreated",
				broadcastType: EventPOCreated,
				wsEventName:   "po_created",
			},
			{
				name:          "POConfirmed",
				broadcastType: EventPOConfirmed,
				wsEventName:   "po_confirmed",
			},
			{
				name:          "POCancelled",
				broadcastType: EventPOCancelled,
				wsEventName:   "po_cancelled",
			},
		}
	}

	t.Run("NilHub", func(t *testing.T) {
		BroadcastPOCreated(nil, POCreatedEvent{POID: 1, PONumber: "PO-001"})
		BroadcastPOConfirmed(nil, POConfirmedEvent{POID: 1, PONumber: "PO-001"})
		BroadcastPOCancelled(nil, POCancelledEvent{POID: 1, PONumber: "PO-001"})
	})

	for _, tc := range newEventCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("WithHub", func(t *testing.T) {
				hub := newListenerHub()
				go hub.Run()
				defer hub.Shutdown()

				client := registerClient(t, hub, 1, nil, true)
				drainMessages(client.send)

				var evt interface{}
				switch tc.broadcastType {
				case EventPOCreated:
					evt = POCreatedEvent{POID: 42, PONumber: "PO-042"}
				case EventPOConfirmed:
					evt = POConfirmedEvent{POID: 42, PONumber: "PO-042"}
				case EventPOCancelled:
					evt = POCancelledEvent{POID: 42, PONumber: "PO-042"}
				}

				switch evt := evt.(type) {
				case POCreatedEvent:
					BroadcastPOCreated(hub, evt)
				case POConfirmedEvent:
					BroadcastPOConfirmed(hub, evt)
				case POCancelledEvent:
					BroadcastPOCancelled(hub, evt)
				}

				msg, ok := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "PO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast")
				}

				var event Event
				err := json.Unmarshal([]byte(msg), &event)
				assert.NoError(t, err)
				assert.Equal(t, tc.broadcastType, event.Type)
			})

			t.Run("StoreFiltering", func(t *testing.T) {
				hub := newListenerHub()
				go hub.Run()
				defer hub.Shutdown()

				store1 := intPtr(1)
				store2 := intPtr(2)

				regularClient := registerClient(t, hub, 1, store2, false)
				drainMessages(regularClient.send)

				var evt interface{}
				switch tc.broadcastType {
				case EventPOCreated:
					evt = POCreatedEvent{POID: 1, PONumber: "PO-001", StoreID: store1}
				case EventPOConfirmed:
					evt = POConfirmedEvent{POID: 1, PONumber: "PO-001", StoreID: store1}
				case EventPOCancelled:
					evt = POCancelledEvent{POID: 1, PONumber: "PO-001", StoreID: store1}
				}

				switch evt := evt.(type) {
				case POCreatedEvent:
					BroadcastPOCreated(hub, evt)
				case POConfirmedEvent:
					BroadcastPOConfirmed(hub, evt)
				case POCancelledEvent:
					BroadcastPOCancelled(hub, evt)
				}

				_, received := waitForMessage(t, regularClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 300*time.Millisecond)
				if received {
					t.Fatal("regular client at different store should not receive event")
				}
			})
		})
	}
}

func TestBroadcastPOReceived_StoreFiltering(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	drainMessages(regularClient.send)

	BroadcastPOReceived(hub, POReceivedEvent{
		POID:     1,
		PONumber: "PO-001",
		GRNumber: "GR-001",
		StoreID:  store1,
	})

	_, received := waitForMessage(t, regularClient.send, func(s string) bool {
		return strings.Contains(s, "po_received")
	}, 300*time.Millisecond)
	if received {
		t.Fatal("regular client at different store should not receive po_received")
	}
}

func TestBroadcastLowStockAlert_StoreFiltering(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)
	time.Sleep(50 * time.Millisecond)
	drainMessages(regularClient.send)

	BroadcastLowStockAlert(hub, LowStockAlertEvent{
		ID:      1,
		SKU:     "SKU-001",
		Name:    "Widget",
		Stock:   0,
		StoreID: store1,
	})

	_, received := waitForMessage(t, regularClient.send, func(s string) bool {
		return strings.Contains(s, "low_stock_alert")
	}, 300*time.Millisecond)
	if received {
		t.Fatal("regular client at different store should not receive low_stock_alert")
	}
}

func TestBroadcastStockOpnameStatus_NilHub(t *testing.T) {
	BroadcastStockOpnameStatus(nil, EventSOCreated, StockOpnameStatusEvent{SessionID: 1, SessionNumber: "SO-001", Status: "draft"})
}

func TestBroadcastStockOpnameStatus_WithHub(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	BroadcastStockOpnameStatus(hub, EventSOCreated, StockOpnameStatusEvent{
		SessionID:     42,
		SessionNumber: "SO-042",
		Status:        "draft",
	})

	msg, ok := waitForMessage(t, client.send, func(s string) bool {
		return strings.Contains(s, "so_created") && strings.Contains(s, "SO-042")
	}, 2*time.Second)
	if !ok {
		t.Fatal("timeout waiting for stock opname status broadcast")
	}
	assert.Contains(t, msg, `"session_id":42`)
	assert.Contains(t, msg, `"status":"draft"`)
}

func TestBroadcastStockOpnameStatus_StoreFiltering(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	regularClient := registerClient(t, hub, 1, store2, false)
	adminClient := registerClient(t, hub, 2, store1, true)
	drainMessages(regularClient.send)
	drainMessages(adminClient.send)

	BroadcastStockOpnameStatus(hub, EventSOCreated, StockOpnameStatusEvent{
		SessionID:     42,
		SessionNumber: "SO-042",
		Status:        "draft",
		StoreID:       store1,
	})

	_, ok := waitForMessage(t, adminClient.send, func(s string) bool {
		return strings.Contains(s, "so_created")
	}, 2*time.Second)
	if !ok {
		t.Fatal("timeout: admin should receive store-scoped so_created")
	}

	_, received := waitForMessage(t, regularClient.send, func(s string) bool {
		return strings.Contains(s, "so_created")
	}, 300*time.Millisecond)
	if received {
		t.Fatal("regular client at different store should not receive store-scoped so_created")
	}
}
