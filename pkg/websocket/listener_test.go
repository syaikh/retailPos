package websocket

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/stockopname"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newListenerHub() *Hub {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	return hub
}

func TestNewSaleCreatedListener(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	listener := NewSaleCreatedListener(hub)
	assert.Contains(t, listener.EventTypes(), eventbus.SaleCreated)

	t.Run("broadcasts sale event", func(t *testing.T) {
		s := &sale.Sale{
			ID:            42,
			InvoiceNumber: "INV-042",
			TotalAmount:   75000,
			Items: []sale.SaleItem{
				{ID: 1, Quantity: 2},
				{ID: 2, Quantity: 1},
			},
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.SaleCreated,
			Payload: s,
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "sale_created") && strings.Contains(s, "INV-042")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for sale broadcast")
		}
		_ = msg
	})

	t.Run("nil items count is zero", func(t *testing.T) {
		s := &sale.Sale{
			ID:            99,
			InvoiceNumber: "INV-099",
			TotalAmount:   10000,
			Items:         nil,
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.SaleCreated,
			Payload: s,
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "sale_created")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for sale broadcast")
		}
		_ = msg
	})

	t.Run("wrong payload type returns nil", func(t *testing.T) {
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.SaleCreated,
			Payload: "not a sale",
		})
		assert.NoError(t, err)
	})
}

func TestNewProductUpdatedListener(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	listener := NewProductUpdatedListener(hub)
	assert.Contains(t, listener.EventTypes(), eventbus.ProductUpdated)

	t.Run("direct product pointer", func(t *testing.T) {
		p := &product.Product{
			ID:    10,
			SKU:   "SKU-010",
			Stock: 50,
			Price: 25000,
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.ProductUpdated,
			Payload: p,
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "product_updated") && strings.Contains(s, "SKU-010")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for product broadcast")
		}
		_ = msg
	})

	t.Run("update payload wrapper", func(t *testing.T) {
		p := &product.Product{
			ID:    20,
			SKU:   "SKU-020",
			Stock: 30,
			Price: 15000,
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.ProductUpdated,
			Payload: eventbus.UpdatePayload{Old: nil, New: p},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "product_updated") && strings.Contains(s, "SKU-020")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for product broadcast")
		}
		_ = msg
	})

	t.Run("update payload with wrong new type", func(t *testing.T) {
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.ProductUpdated,
			Payload: eventbus.UpdatePayload{Old: nil, New: "wrong type"},
		})
		assert.NoError(t, err)
	})

	t.Run("wrong payload type", func(t *testing.T) {
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.ProductUpdated,
			Payload: 12345,
		})
		assert.NoError(t, err)
	})
}

type mockProductLookup struct {
	sku     string
	name    string
	stock   int
	storeID *int
	err     error
}

func (m *mockProductLookup) GetProductByID(_ context.Context, id int) (string, string, int, *int, error) {
	return m.sku, m.name, m.stock, m.storeID, m.err
}

func TestNewStockAdjustedListener(t *testing.T) {
	t.Run("broadcasts stock update when lookup succeeds", func(t *testing.T) {
		hub := newListenerHub()
		go hub.Run()
		defer hub.Shutdown()

		client := registerClient(t, hub, 1, nil, true)
		drainMessages(client.send)

		mock := &mockProductLookup{
			sku:   "SKU-100",
			name:  "Widget",
			stock: 42,
		}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.StockAdjusted,
			Payload: inventory.StockAdjustedEvent{
				ProductID:      100,
				QuantityChange: 5,
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "stock_update") && strings.Contains(s, "SKU-100")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for stock broadcast")
		}
		_ = msg
	})

	t.Run("broadcasts low stock alert when stock is zero", func(t *testing.T) {
		hub := newListenerHub()
		go hub.Run()
		defer hub.Shutdown()

		client := registerClient(t, hub, 1, nil, true)
		drainMessages(client.send)

		mock := &mockProductLookup{
			sku:   "SKU-200",
			name:  "Gadget",
			stock: 0,
		}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.StockAdjusted,
			Payload: inventory.StockAdjustedEvent{
				ProductID:      200,
				QuantityChange: -5,
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "stock_update")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for stock broadcast")
		}
		_ = msg

		msg, ok = waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "low_stock_alert")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for low stock alert")
		}
		_ = msg
	})

	t.Run("broadcasts low stock alert when stock is negative", func(t *testing.T) {
		hub := newListenerHub()
		go hub.Run()
		defer hub.Shutdown()

		client := registerClient(t, hub, 1, nil, true)
		drainMessages(client.send)

		mock := &mockProductLookup{
			sku:   "SKU-300",
			name:  "Thing",
			stock: -2,
		}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.StockAdjusted,
			Payload: inventory.StockAdjustedEvent{
				ProductID:      300,
				QuantityChange: -10,
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "stock_update")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for stock broadcast")
		}
		_ = msg

		msg, ok = waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "low_stock_alert")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for low stock alert")
		}
		_ = msg
	})

	t.Run("no broadcast on lookup error", func(t *testing.T) {
		hub := newListenerHub()
		go hub.Run()
		defer hub.Shutdown()

		client := registerClient(t, hub, 1, nil, true)
		time.Sleep(50 * time.Millisecond)
		drainMessages(client.send)
		time.Sleep(50 * time.Millisecond)
		drainMessages(client.send)

		mock := &mockProductLookup{
			err: assert.AnError,
		}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.StockAdjusted,
			Payload: inventory.StockAdjustedEvent{
				ProductID: 999,
			},
		})
		assert.NoError(t, err)

		msg, found := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "stock_update") || strings.Contains(s, "low_stock_alert")
		}, 300*time.Millisecond)
		if found {
			t.Fatalf("should not broadcast on lookup error, got: %s", msg)
		}
	})

	t.Run("wrong payload type", func(t *testing.T) {
		hub := newListenerHub()
		go hub.Run()
		defer hub.Shutdown()

		mock := &mockProductLookup{}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.StockAdjusted,
			Payload: "not an event",
		})
		assert.NoError(t, err)
	})

	t.Run("passes storeID from product lookup", func(t *testing.T) {
		hub := newListenerHub()
		go hub.Run()
		defer hub.Shutdown()

		client := registerClient(t, hub, 1, nil, true)
		drainMessages(client.send)

		storeID := 3
		mock := &mockProductLookup{
			sku:     "SKU-400",
			name:    "Item",
			stock:   10,
			storeID: &storeID,
		}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.StockAdjusted,
			Payload: inventory.StockAdjustedEvent{
				ProductID: 400,
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "stock_update") && strings.Contains(s, "SKU-400")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for broadcast")
		}
		_ = msg
	})
}

func TestExtractPOEventFields(t *testing.T) {
	t.Run("returns all fields when present", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     42,
			"po_number": "PO-042",
			"store_id":  5,
		}
		poID, poNumber, storeID := extractPOEventFields(payload)
		assert.Equal(t, 42, poID)
		assert.Equal(t, "PO-042", poNumber)
		require.NotNil(t, storeID)
		assert.Equal(t, 5, *storeID)
	})

	t.Run("handles float64 po_id and store_id (JSON unmarshaling)", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     float64(42),
			"po_number": "PO-042",
			"store_id":  float64(5),
		}
		poID, poNumber, storeID := extractPOEventFields(payload)
		assert.Equal(t, 42, poID)
		assert.Equal(t, "PO-042", poNumber)
		require.NotNil(t, storeID)
		assert.Equal(t, 5, *storeID)
	})

	t.Run("handles int64 po_id and store_id", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     int64(42),
			"po_number": "PO-042",
			"store_id":  int64(5),
		}
		poID, poNumber, storeID := extractPOEventFields(payload)
		assert.Equal(t, 42, poID)
		assert.Equal(t, "PO-042", poNumber)
		require.NotNil(t, storeID)
		assert.Equal(t, 5, *storeID)
	})

	t.Run("nil store_id when missing", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     1,
			"po_number": "PO-001",
		}
		_, _, storeID := extractPOEventFields(payload)
		assert.Nil(t, storeID)
	})

	t.Run("nil store_id when zero", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     1,
			"po_number": "PO-001",
			"store_id":  0,
		}
		_, _, storeID := extractPOEventFields(payload)
		assert.Nil(t, storeID)
	})

	t.Run("nil store_id when zero float64", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     1,
			"po_number": "PO-001",
			"store_id":  float64(0),
		}
		_, _, storeID := extractPOEventFields(payload)
		assert.Nil(t, storeID)
	})

	t.Run("nil store_id when negative", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     1,
			"po_number": "PO-001",
			"store_id":  -1,
		}
		_, _, storeID := extractPOEventFields(payload)
		assert.Nil(t, storeID)
	})

	t.Run("zero values for missing po_id and po_number", func(t *testing.T) {
		payload := map[string]interface{}{}
		poID, poNumber, storeID := extractPOEventFields(payload)
		assert.Equal(t, 0, poID)
		assert.Equal(t, "", poNumber)
		assert.Nil(t, storeID)
	})

	t.Run("wrong types fall back to zero values", func(t *testing.T) {
		payload := map[string]interface{}{
			"po_id":     "not-an-int",
			"po_number": 123,
			"store_id":  "not-an-int",
		}
		poID, poNumber, storeID := extractPOEventFields(payload)
		assert.Equal(t, 0, poID)
		assert.Equal(t, "", poNumber)
		assert.Nil(t, storeID)
	})
}

type poListenerTestCase struct {
	name           string
	eventType      eventbus.EventType
	broadcastType  EventType
	wsEventName    string
	newListener    func(hub *Hub) eventbus.Listener
}

func TestPOCreatedConfirmedCancelledListeners(t *testing.T) {
	cases := []poListenerTestCase{
		{
			name:          "created",
			eventType:     eventbus.EventType("purchase_order.created"),
			broadcastType: EventPOCreated,
			wsEventName:   "po_created",
			newListener:   NewPOCreatedListener,
		},
		{
			name:          "confirmed",
			eventType:     eventbus.EventType("purchase_order.confirmed"),
			broadcastType: EventPOConfirmed,
			wsEventName:   "po_confirmed",
			newListener:   NewPOConfirmedListener,
		},
		{
			name:          "cancelled",
			eventType:     eventbus.EventType("purchase_order.cancelled"),
			broadcastType: EventPOCancelled,
			wsEventName:   "po_cancelled",
			newListener:   NewPOCancelledListener,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hub := newListenerHub()
			go hub.Run()
			defer hub.Shutdown()

			client := registerClient(t, hub, 1, nil, true)
			drainMessages(client.send)

			listener := tc.newListener(hub)

			t.Run("broadcasts on valid event", func(t *testing.T) {
				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"po_id":     42,
						"po_number": "PO-042",
						"store_id":  5,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "PO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast")
				}
				_ = msg
			})

			t.Run("wrong payload type returns nil without error", func(t *testing.T) {
				drainMessages(client.send)
				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type:    tc.eventType,
					Payload: "not a map",
				})
				assert.NoError(t, err)

				msg, found := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 300*time.Millisecond)
				if found {
					t.Fatalf("should not broadcast on wrong payload type, got: %s", msg)
				}
			})

			t.Run("missing fields produce broadcast with correct event type", func(t *testing.T) {
				drainMessages(client.send)
				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type:    tc.eventType,
					Payload: map[string]interface{}{},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast with empty payload")
				}
				var event Event
				err = json.Unmarshal([]byte(msg), &event)
				assert.NoError(t, err)
				assert.Equal(t, tc.broadcastType, event.Type)
			})

			t.Run("forwards store_id from payload", func(t *testing.T) {
				drainMessages(client.send)

				sid := 5
				storeScopedClient := registerClient(t, hub, 2, &sid, false)
				drainMessages(storeScopedClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"po_id":     99,
						"po_number": "PO-099",
						"store_id":  5,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, storeScopedClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for store-scoped broadcast")
				}
				assert.Contains(t, msg, "PO-099")
			})

			t.Run("different store does not receive event", func(t *testing.T) {
				drainMessages(client.send)

				sid := 99
				otherClient := registerClient(t, hub, 3, &sid, false)
				drainMessages(otherClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"po_id":     55,
						"po_number": "PO-055",
						"store_id":  5,
					},
				})
				assert.NoError(t, err)

				msg, found := waitForMessage(t, otherClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 300*time.Millisecond)
				if found {
					t.Fatalf("store-scoped client should not receive event for different store, got: %s", msg)
				}
			})

			t.Run("admin receives event regardless of store", func(t *testing.T) {
				drainMessages(client.send)

				sid := 7
				adminClient := registerClient(t, hub, 4, &sid, true)
				drainMessages(adminClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"po_id":     77,
						"po_number": "PO-077",
						"store_id":  5,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, adminClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for admin to receive cross-store broadcast")
				}
				assert.Contains(t, msg, "PO-077")
			})

			t.Run("missing store_id broadcasts to all", func(t *testing.T) {
				drainMessages(client.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"po_id":     111,
						"po_number": "PO-111",
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast with missing store_id")
				}
				assert.Contains(t, msg, "PO-111")
			})
		})
	}
}

func TestListener_EventTypes(t *testing.T) {
	hub := newListenerHub()

	saleListener := NewSaleCreatedListener(hub)
	assert.Equal(t, []eventbus.EventType{eventbus.SaleCreated}, saleListener.EventTypes())

	productListener := NewProductUpdatedListener(hub)
	assert.Equal(t, []eventbus.EventType{eventbus.ProductUpdated}, productListener.EventTypes())

	mock := &mockProductLookup{}
	stockListener := NewStockAdjustedListener(hub, mock)
	assert.Equal(t, []eventbus.EventType{eventbus.StockAdjusted}, stockListener.EventTypes())

	poListener := NewPOReceivedListener(hub)
	assert.Equal(t, []eventbus.EventType{eventbus.EventType("goods_receipt.created")}, poListener.EventTypes())

	createdListener := NewPOCreatedListener(hub)
	assert.Contains(t, createdListener.EventTypes(), eventbus.EventType("purchase_order.created"))

	confirmedListener := NewPOConfirmedListener(hub)
	assert.Contains(t, confirmedListener.EventTypes(), eventbus.EventType("purchase_order.confirmed"))

	cancelledListener := NewPOCancelledListener(hub)
	assert.Contains(t, cancelledListener.EventTypes(), eventbus.EventType("purchase_order.cancelled"))

	soListener := NewStockOpnameStatusListener(hub)
	assert.ElementsMatch(t, []eventbus.EventType{
		eventbus.EventType(stockopname.EventStockOpnameCreated),
		eventbus.EventType(stockopname.EventStockOpnameOpened),
		eventbus.EventType(stockopname.EventStockOpnameSubmitted),
		eventbus.EventType(stockopname.EventStockOpnameApproved),
		eventbus.EventType(stockopname.EventStockOpnamePosted),
		eventbus.EventType(stockopname.EventStockOpnameClosed),
		eventbus.EventType(stockopname.EventStockOpnameRejected),
		eventbus.EventType(stockopname.EventStockOpnameRecount),
		eventbus.EventType(stockopname.EventStockOpnameCancelled),
	}, soListener.EventTypes())
}

func TestNewPOReceivedListener(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	listener := NewPOReceivedListener(hub)
	assert.Contains(t, listener.EventTypes(), eventbus.EventType("goods_receipt.created"))

	t.Run("broadcasts po_received on valid event", func(t *testing.T) {
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType("goods_receipt.created"),
			Payload: map[string]interface{}{
				"po_id":     42,
				"po_number": "PO-042",
				"gr_number": "GR-007",
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "po_received") && strings.Contains(s, "PO-042") && strings.Contains(s, "GR-007")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for po_received broadcast")
		}
		_ = msg
	})

	t.Run("wrong payload type returns nil without error", func(t *testing.T) {
		drainMessages(client.send)
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.EventType("goods_receipt.created"),
			Payload: "not a map",
		})
		assert.NoError(t, err)

		msg, found := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "po_received")
		}, 300*time.Millisecond)
		if found {
			t.Fatalf("should not broadcast on wrong payload type, got: %s", msg)
		}
	})

	t.Run("missing fields produce empty broadcast", func(t *testing.T) {
		drainMessages(client.send)
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.EventType("goods_receipt.created"),
			Payload: map[string]interface{}{},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "po_received")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for po_received with empty payload")
		}
		var event Event
		err = json.Unmarshal([]byte(msg), &event)
		assert.NoError(t, err)
		assert.Equal(t, EventPOReceived, event.Type)
	})

	t.Run("forwards store_id from payload to broadcast event", func(t *testing.T) {
		drainMessages(client.send)

		sid := 5
		storeScopedClient := registerClient(t, hub, 2, &sid, false)
		drainMessages(storeScopedClient.send)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType("goods_receipt.created"),
			Payload: map[string]interface{}{
				"po_id":     99,
				"po_number": "PO-099",
				"gr_number": "GR-099",
				"store_id":  5,
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, storeScopedClient.send, func(s string) bool {
			return strings.Contains(s, "po_received")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for po_received with store_id")
		}
		assert.Contains(t, msg, "PO-099")
		assert.Contains(t, msg, "GR-099")
	})

	t.Run("different store does not receive event", func(t *testing.T) {
		drainMessages(client.send)

		sid := 99
		otherClient := registerClient(t, hub, 3, &sid, false)
		drainMessages(otherClient.send)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType("goods_receipt.created"),
			Payload: map[string]interface{}{
				"po_id":     55,
				"po_number": "PO-055",
				"gr_number": "GR-055",
				"store_id":  5,
			},
		})
		assert.NoError(t, err)

		msg, found := waitForMessage(t, otherClient.send, func(s string) bool {
			return strings.Contains(s, "po_received")
		}, 300*time.Millisecond)
		if found {
			t.Fatalf("store-scoped client should not receive event for different store, got: %s", msg)
		}
	})

	t.Run("admin receives event regardless of store", func(t *testing.T) {
		drainMessages(client.send)

		sid := 7
		adminClient := registerClient(t, hub, 4, &sid, true)
		drainMessages(adminClient.send)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType("goods_receipt.created"),
			Payload: map[string]interface{}{
				"po_id":     77,
				"po_number": "PO-077",
				"gr_number": "GR-077",
				"store_id":  5,
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, adminClient.send, func(s string) bool {
			return strings.Contains(s, "po_received")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for admin to receive cross-store po_received")
		}
		assert.Contains(t, msg, "PO-077")
	})

	t.Run("missing store_id defaults to nil (broadcasts to all)", func(t *testing.T) {
		drainMessages(client.send)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType("goods_receipt.created"),
			Payload: map[string]interface{}{
				"po_id":     111,
				"po_number": "PO-111",
				"gr_number": "GR-111",
			},
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, client.send, func(s string) bool {
			return strings.Contains(s, "po_received")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for po_received with missing store_id")
		}
		assert.Contains(t, msg, "PO-111")
	})
}

func TestHub_BroadcastUserCount(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)

	// registration triggers broadcastUserCount
	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "user_online_count")
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		var payload struct {
			Count int `json:"count"`
		}
		err = json.Unmarshal(event.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, 1, payload.Count)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for user count broadcast")
	}
}

func TestNewStockOpnameStatusListener(t *testing.T) {
	cases := []struct {
		name          string
		eventType     eventbus.EventType
		broadcastType EventType
		wsEventName   string
	}{
	{
		name:          "created",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameCreated),
		broadcastType: EventSOCreated,
		wsEventName:   "so_created",
	},
	{
		name:          "opened",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameOpened),
		broadcastType: EventSOOpened,
		wsEventName:   "so_opened",
	},
	{
		name:          "submitted",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameSubmitted),
		broadcastType: EventSOSubmitted,
		wsEventName:   "so_submitted",
	},
	{
		name:          "approved",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameApproved),
		broadcastType: EventSOApproved,
		wsEventName:   "so_approved",
	},
	{
		name:          "posted",
		eventType:     eventbus.EventType(stockopname.EventStockOpnamePosted),
		broadcastType: EventSOPosted,
		wsEventName:   "so_posted",
	},
	{
		name:          "closed",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameClosed),
		broadcastType: EventSOClosed,
		wsEventName:   "so_closed",
	},
	{
		name:          "rejected",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameRejected),
		broadcastType: EventSORejected,
		wsEventName:   "so_rejected",
	},
	{
		name:          "needs_recount",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameRecount),
		broadcastType: EventSORecount,
		wsEventName:   "so_needs_recount",
	},
	{
		name:          "cancelled",
		eventType:     eventbus.EventType(stockopname.EventStockOpnameCancelled),
		broadcastType: EventSOCancelled,
		wsEventName:   "so_cancelled",
	},
}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hub := newListenerHub()
			go hub.Run()
			defer hub.Shutdown()

			client := registerClient(t, hub, 1, nil, true)
			drainMessages(client.send)

			listener := NewStockOpnameStatusListener(hub)

			t.Run("broadcasts on valid event", func(t *testing.T) {
				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"session_id":     42,
						"session_number": "SO-042",
						"status":         "pending_approval",
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "SO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast")
				}
				assert.Contains(t, msg, tc.broadcastType)
				assert.Contains(t, msg, `"session_id":42`)
				assert.Contains(t, msg, "SO-042")
			})

			t.Run("wrong payload type returns nil without error", func(t *testing.T) {
				drainMessages(client.send)
				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type:    tc.eventType,
					Payload: "not a map",
				})
				assert.NoError(t, err)

				msg, found := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 300*time.Millisecond)
				if found {
					t.Fatalf("should not broadcast on wrong payload type, got: %s", msg)
				}
			})

			t.Run("zero session_id produces no broadcast", func(t *testing.T) {
				drainMessages(client.send)
				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type:    tc.eventType,
					Payload: map[string]interface{}{},
				})
				assert.NoError(t, err)

				msg, found := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 300*time.Millisecond)
				if found {
					t.Fatalf("should not broadcast when session_id is missing, got: %s", msg)
				}
			})

			t.Run("forwards store_id and scopes to matching store", func(t *testing.T) {
				sid := 5
				storeScopedClient := registerClient(t, hub, 2, &sid, false)
				drainMessages(storeScopedClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"session_id":     42,
						"session_number": "SO-042",
						"status":         "pending_approval",
						"store_id":       5,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, storeScopedClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "SO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for store-scoped broadcast")
				}
				assert.Contains(t, msg, `"session_id":42`)
				assert.Contains(t, msg, `"status":"pending_approval"`)
			})

			t.Run("different store does not receive event", func(t *testing.T) {
				sid := 99
				otherClient := registerClient(t, hub, 3, &sid, false)
				drainMessages(otherClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"session_id":     42,
						"session_number": "SO-042",
						"status":         "pending_approval",
						"store_id":       5,
					},
				})
				assert.NoError(t, err)

				msg, found := waitForMessage(t, otherClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 300*time.Millisecond)
				if found {
					t.Fatalf("store-scoped client should not receive event for different store, got: %s", msg)
				}
			})

			t.Run("admin receives event regardless of store", func(t *testing.T) {
				sid := 7
				adminClient := registerClient(t, hub, 4, &sid, true)
				drainMessages(adminClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"session_id":     42,
						"session_number": "SO-042",
						"status":         "pending_approval",
						"store_id":       5,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, adminClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "SO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for admin to receive cross-store broadcast")
				}
				assert.Contains(t, msg, `"session_id":42`)
				assert.Contains(t, msg, `"status":"pending_approval"`)
			})

			t.Run("missing store_id broadcasts to all stores", func(t *testing.T) {
				sid := 5
				storeScopedClient := registerClient(t, hub, 5, &sid, false)
				drainMessages(storeScopedClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"session_id":     42,
						"session_number": "SO-042",
						"status":         "pending_approval",
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, storeScopedClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "SO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast with missing store_id")
				}
				assert.Contains(t, msg, `"session_id":42`)
				assert.Contains(t, msg, `"status":"pending_approval"`)
			})

			t.Run("zero store_id broadcasts to all stores", func(t *testing.T) {
				sid := 5
				storeScopedClient := registerClient(t, hub, 6, &sid, false)
				drainMessages(storeScopedClient.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: map[string]interface{}{
						"session_id":     42,
						"session_number": "SO-042",
						"status":         "pending_approval",
						"store_id":       0,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, storeScopedClient.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName) && strings.Contains(s, "SO-042")
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast with zero store_id")
				}
				assert.Contains(t, msg, `"session_id":42`)
				assert.Contains(t, msg, `"status":"pending_approval"`)
			})
		})
	}
}
