package websocket

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"

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
		started:         make(chan struct{}),
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
	assert.Contains(t, listener.EventTypes(), eventbus.EventType(events.TopicSaleCreated))

	t.Run("broadcasts sale event", func(t *testing.T) {
		s := &events.SaleCreated{
			ID:            42,
			InvoiceNumber: "INV-042",
			TotalAmount:   75000,
			ItemCount:     2,
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    events.TopicSaleCreated,
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
		s := &events.SaleCreated{
			ID:            99,
			InvoiceNumber: "INV-099",
			TotalAmount:   10000,
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    events.TopicSaleCreated,
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
			Type:    events.TopicSaleCreated,
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
	assert.Contains(t, listener.EventTypes(), eventbus.EventType(events.TopicProductUpdated))

	t.Run("direct DTO pointer", func(t *testing.T) {
		p := &events.ProductUpdated{
			ID:    10,
			SKU:   "SKU-010",
			Stock: 50,
			Price: 25000,
		}

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.EventType(events.TopicProductUpdated),
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

	t.Run("forwards store_id", func(t *testing.T) {
		sid := 5
		p := &events.ProductUpdated{
			ID:      20,
			SKU:     "SKU-020",
			Stock:   30,
			Price:   15000,
			StoreID: &sid,
		}

		storeScopedClient := registerClient(t, hub, 2, &sid, false)
		drainMessages(storeScopedClient.send)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.EventType(events.TopicProductUpdated),
			Payload: p,
		})
		assert.NoError(t, err)

		msg, ok := waitForMessage(t, storeScopedClient.send, func(s string) bool {
			return strings.Contains(s, "product_updated") && strings.Contains(s, "SKU-020")
		}, 2*time.Second)
		if !ok {
			t.Fatal("timeout waiting for product broadcast")
		}
		_ = msg
	})

	t.Run("wrong payload type", func(t *testing.T) {
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type:    eventbus.EventType(events.TopicProductUpdated),
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
			Type: eventbus.EventType(events.TopicStockAdjusted),
			Payload: &events.StockAdjusted{
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
			Type: eventbus.EventType(events.TopicStockAdjusted),
			Payload: &events.StockAdjusted{
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
			Type: eventbus.EventType(events.TopicStockAdjusted),
			Payload: &events.StockAdjusted{
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
		drainMessages(client.send)

		mock := &mockProductLookup{
			err: assert.AnError,
		}
		listener := NewStockAdjustedListener(hub, mock)

		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType(events.TopicStockAdjusted),
			Payload: &events.StockAdjusted{
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
			Type:    eventbus.EventType(events.TopicStockAdjusted),
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
			Type: eventbus.EventType(events.TopicStockAdjusted),
			Payload: &events.StockAdjusted{
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

type poListenerTestCase struct {
	name          string
	eventType     eventbus.EventType
	broadcastType EventType
	wsEventName   string
	newListener   func(hub *Hub) eventbus.Listener
}

func TestPOCreatedConfirmedCancelledListeners(t *testing.T) {
	cases := []poListenerTestCase{
		{
			name:          "created",
			eventType:     eventbus.EventType(events.TopicPOCreated),
			broadcastType: EventPOCreated,
			wsEventName:   "po_created",
			newListener:   NewPOCreatedListener,
		},
		{
			name:          "confirmed",
			eventType:     eventbus.EventType(events.TopicPOConfirmed),
			broadcastType: EventPOConfirmed,
			wsEventName:   "po_confirmed",
			newListener:   NewPOConfirmedListener,
		},
		{
			name:          "cancelled",
			eventType:     eventbus.EventType(events.TopicPOCancelled),
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
					Payload: &events.PurchaseOrderEvent{
						POID:     42,
						PONumber: "PO-042",
						StoreID:  5,
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
					Payload: &events.PurchaseOrderEvent{},
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
					Payload: &events.PurchaseOrderEvent{
						POID:     99,
						PONumber: "PO-099",
						StoreID:  5,
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
					Payload: &events.PurchaseOrderEvent{
						POID:     55,
						PONumber: "PO-055",
						StoreID:  5,
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
					Payload: &events.PurchaseOrderEvent{
						POID:     77,
						PONumber: "PO-077",
						StoreID:  5,
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
					Payload: &events.PurchaseOrderEvent{
						POID:     111,
						PONumber: "PO-111",
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

			t.Run("zero store_id broadcasts to all", func(t *testing.T) {
				drainMessages(client.send)

				err := listener.HandleEvent(context.Background(), eventbus.Event{
					Type: tc.eventType,
					Payload: &events.PurchaseOrderEvent{
						POID:     112,
						PONumber: "PO-112",
						StoreID:  0,
					},
				})
				assert.NoError(t, err)

				msg, ok := waitForMessage(t, client.send, func(s string) bool {
					return strings.Contains(s, tc.wsEventName)
				}, 2*time.Second)
				if !ok {
					t.Fatal("timeout waiting for broadcast with zero store_id")
				}
				assert.Contains(t, msg, "PO-112")
			})
		})
	}
}

func TestListener_EventTypes(t *testing.T) {
	hub := newListenerHub()

	saleListener := NewSaleCreatedListener(hub)
	assert.Equal(t, []eventbus.EventType{events.TopicSaleCreated}, saleListener.EventTypes())

	productListener := NewProductUpdatedListener(hub)
	assert.Equal(t, []eventbus.EventType{eventbus.EventType(events.TopicProductUpdated)}, productListener.EventTypes())

	mock := &mockProductLookup{}
	stockListener := NewStockAdjustedListener(hub, mock)
	assert.Equal(t, []eventbus.EventType{eventbus.EventType(events.TopicStockAdjusted)}, stockListener.EventTypes())

	poListener := NewPOReceivedListener(hub)
	assert.Equal(t, []eventbus.EventType{eventbus.EventType(events.TopicGoodsReceiptCreated)}, poListener.EventTypes())

	createdListener := NewPOCreatedListener(hub)
	assert.Contains(t, createdListener.EventTypes(), eventbus.EventType(events.TopicPOCreated))

	confirmedListener := NewPOConfirmedListener(hub)
	assert.Contains(t, confirmedListener.EventTypes(), eventbus.EventType(events.TopicPOConfirmed))

	cancelledListener := NewPOCancelledListener(hub)
	assert.Contains(t, cancelledListener.EventTypes(), eventbus.EventType(events.TopicPOCancelled))

	soListener := NewStockOpnameStatusListener(hub)
	assert.ElementsMatch(t, []eventbus.EventType{
		eventbus.EventType(events.TopicStockOpnameCreated),
		eventbus.EventType(events.TopicStockOpnameOpened),
		eventbus.EventType(events.TopicStockOpnameSubmitted),
		eventbus.EventType(events.TopicStockOpnameApproved),
		eventbus.EventType(events.TopicStockOpnamePosted),
		eventbus.EventType(events.TopicStockOpnameClosed),
		eventbus.EventType(events.TopicStockOpnameRejected),
		eventbus.EventType(events.TopicStockOpnameRecount),
		eventbus.EventType(events.TopicStockOpnameCancelled),
	}, soListener.EventTypes())
}

func TestNewPOReceivedListener(t *testing.T) {
	hub := newListenerHub()
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	listener := NewPOReceivedListener(hub)
	assert.Contains(t, listener.EventTypes(), eventbus.EventType(events.TopicGoodsReceiptCreated))

	t.Run("broadcasts po_received on valid event", func(t *testing.T) {
		err := listener.HandleEvent(context.Background(), eventbus.Event{
			Type: eventbus.EventType(events.TopicGoodsReceiptCreated),
			Payload: &events.GoodsReceiptCreated{
				POID:     42,
				PONumber: "PO-042",
				GRNumber: "GR-007",
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
			Type:    eventbus.EventType(events.TopicGoodsReceiptCreated),
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
			Type:    eventbus.EventType(events.TopicGoodsReceiptCreated),
			Payload: &events.GoodsReceiptCreated{},
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
			Type: eventbus.EventType(events.TopicGoodsReceiptCreated),
			Payload: &events.GoodsReceiptCreated{
				POID:     99,
				PONumber: "PO-099",
				GRNumber: "GR-099",
				StoreID:  5,
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
			Type: eventbus.EventType(events.TopicGoodsReceiptCreated),
			Payload: &events.GoodsReceiptCreated{
				POID:     55,
				PONumber: "PO-055",
				GRNumber: "GR-055",
				StoreID:  5,
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
			Type: eventbus.EventType(events.TopicGoodsReceiptCreated),
			Payload: &events.GoodsReceiptCreated{
				POID:     77,
				PONumber: "PO-077",
				GRNumber: "GR-077",
				StoreID:  5,
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
			Type: eventbus.EventType(events.TopicGoodsReceiptCreated),
			Payload: &events.GoodsReceiptCreated{
				POID:     111,
				PONumber: "PO-111",
				GRNumber: "GR-111",
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
			eventType:     eventbus.EventType(events.TopicStockOpnameCreated),
			broadcastType: EventSOCreated,
			wsEventName:   "so_created",
		},
		{
			name:          "opened",
			eventType:     eventbus.EventType(events.TopicStockOpnameOpened),
			broadcastType: EventSOOpened,
			wsEventName:   "so_opened",
		},
		{
			name:          "submitted",
			eventType:     eventbus.EventType(events.TopicStockOpnameSubmitted),
			broadcastType: EventSOSubmitted,
			wsEventName:   "so_submitted",
		},
		{
			name:          "approved",
			eventType:     eventbus.EventType(events.TopicStockOpnameApproved),
			broadcastType: EventSOApproved,
			wsEventName:   "so_approved",
		},
		{
			name:          "posted",
			eventType:     eventbus.EventType(events.TopicStockOpnamePosted),
			broadcastType: EventSOPosted,
			wsEventName:   "so_posted",
		},
		{
			name:          "closed",
			eventType:     eventbus.EventType(events.TopicStockOpnameClosed),
			broadcastType: EventSOClosed,
			wsEventName:   "so_closed",
		},
		{
			name:          "rejected",
			eventType:     eventbus.EventType(events.TopicStockOpnameRejected),
			broadcastType: EventSORejected,
			wsEventName:   "so_rejected",
		},
		{
			name:          "needs_recount",
			eventType:     eventbus.EventType(events.TopicStockOpnameRecount),
			broadcastType: EventSORecount,
			wsEventName:   "so_needs_recount",
		},
		{
			name:          "cancelled",
			eventType:     eventbus.EventType(events.TopicStockOpnameCancelled),
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
					Payload: &events.StockOpnameStatusChanged{
						SessionID:     42,
						SessionNumber: "SO-042",
						Status:        "pending_approval",
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
					Payload: &events.StockOpnameStatusChanged{},
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
					Payload: &events.StockOpnameStatusChanged{
						SessionID:     42,
						SessionNumber: "SO-042",
						Status:        "pending_approval",
						StoreID:       5,
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
					Payload: &events.StockOpnameStatusChanged{
						SessionID:     42,
						SessionNumber: "SO-042",
						Status:        "pending_approval",
						StoreID:       5,
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
					Payload: &events.StockOpnameStatusChanged{
						SessionID:     42,
						SessionNumber: "SO-042",
						Status:        "pending_approval",
						StoreID:       5,
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
					Payload: &events.StockOpnameStatusChanged{
						SessionID:     42,
						SessionNumber: "SO-042",
						Status:        "pending_approval",
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
					Payload: &events.StockOpnameStatusChanged{
						SessionID:     42,
						SessionNumber: "SO-042",
						Status:        "pending_approval",
						StoreID:       0,
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
