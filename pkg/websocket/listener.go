package websocket

import (
	"context"
	"fmt"
	"log/slog"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"
)

func NewSaleCreatedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicSaleCreated},
		func(ctx context.Context, event eventbus.Event) error {
			s, ok := event.Payload.(*events.SaleCreated)
			if !ok {
				slog.Warn("[ws] unexpected payload type for sale.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			BroadcastSaleCreated(hub, SaleCreatedEvent{
				ID:      s.ID,
				Invoice: s.InvoiceNumber,
				Total:   s.TotalAmount,
				Items:   s.ItemCount,
				StoreID: s.StoreID,
			})
			return nil
		},
	)
}

func NewProductUpdatedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicProductUpdated},
		func(ctx context.Context, event eventbus.Event) error {
			p, ok := event.Payload.(*events.ProductUpdated)
			if !ok {
				slog.Warn("[ws] unexpected payload type for product.updated", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			BroadcastProductUpdate(hub, ProductUpdateEvent{
				ID:      p.ID,
				SKU:     p.SKU,
				Stock:   p.Stock,
				Price:   p.Price,
				StoreID: p.StoreID,
			})
			return nil
		},
	)
}

type ProductLookup interface {
	GetProductByID(ctx context.Context, id int) (sku string, name string, stock int, storeID *int, err error)
}

// storeIDPtr converts a DTO store id to the broadcast pointer form: zero or
// negative values mean "global" and map to nil (broadcast to all stores).
func storeIDPtr(storeID int) *int {
	if storeID <= 0 {
		return nil
	}
	return &storeID
}

func NewPOReceivedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicGoodsReceiptCreated},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(*events.GoodsReceiptCreated)
			if !ok {
				slog.Warn("[ws] unexpected payload type for goods_receipt.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			BroadcastPOReceived(hub, POReceivedEvent{
				POID:     payload.POID,
				PONumber: payload.PONumber,
				GRNumber: payload.GRNumber,
				StoreID:  storeIDPtr(payload.StoreID),
			})
			return nil
		},
	)
}

func NewPOCreatedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicPOCreated},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(*events.PurchaseOrderEvent)
			if !ok {
				slog.Warn("[ws] unexpected payload type for purchase_order.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			BroadcastPOCreated(hub, POCreatedEvent{
				POID:     payload.POID,
				PONumber: payload.PONumber,
				StoreID:  storeIDPtr(payload.StoreID),
			})
			return nil
		},
	)
}

func NewPOConfirmedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicPOConfirmed},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(*events.PurchaseOrderEvent)
			if !ok {
				slog.Warn("[ws] unexpected payload type for purchase_order.confirmed", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			BroadcastPOConfirmed(hub, POConfirmedEvent{
				POID:     payload.POID,
				PONumber: payload.PONumber,
				StoreID:  storeIDPtr(payload.StoreID),
			})
			return nil
		},
	)
}

func NewPOCancelledListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicPOCancelled},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(*events.PurchaseOrderEvent)
			if !ok {
				slog.Warn("[ws] unexpected payload type for purchase_order.cancelled", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			BroadcastPOCancelled(hub, POCancelledEvent{
				POID:     payload.POID,
				PONumber: payload.PONumber,
				StoreID:  storeIDPtr(payload.StoreID),
			})
			return nil
		},
	)
}

func NewStockAdjustedListener(hub *Hub, products ProductLookup) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{events.TopicStockAdjusted},
		func(ctx context.Context, event eventbus.Event) error {
			sa, ok := event.Payload.(*events.StockAdjusted)
			if !ok {
				slog.Warn("[ws] unexpected payload type for stock.adjusted", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			sku, name, stock, storeID, err := products.GetProductByID(ctx, sa.ProductID)
			if err != nil {
				slog.Warn("[ws] failed to look up product for stock.adjusted", "product_id", sa.ProductID, "error", err)
				return nil
			}
			lowStock := stock <= 0
			BroadcastStockUpdate(hub, StockUpdateEvent{
				ID:       sa.ProductID,
				SKU:      sku,
				Stock:    stock,
				LowStock: lowStock,
				StoreID:  storeID,
			})
			if lowStock {
				BroadcastLowStockAlert(hub, LowStockAlertEvent{
					ID:      sa.ProductID,
					SKU:     sku,
					Name:    name,
					Stock:   stock,
					StoreID: storeID,
				})
			}
			return nil
		},
	)
}

// stockOpnameEventTypes maps a stock opname eventbus topic to its websocket
// event type. Only status transitions are broadcast; item-level counts are
// deliberately excluded to preserve blind counting (BR-008).
var stockOpnameEventTypes = map[eventbus.EventType]EventType{
	eventbus.EventType(events.TopicStockOpnameCreated):   EventSOCreated,
	eventbus.EventType(events.TopicStockOpnameOpened):    EventSOOpened,
	eventbus.EventType(events.TopicStockOpnameSubmitted): EventSOSubmitted,
	eventbus.EventType(events.TopicStockOpnameApproved):  EventSOApproved,
	eventbus.EventType(events.TopicStockOpnamePosted):    EventSOPosted,
	eventbus.EventType(events.TopicStockOpnameClosed):    EventSOClosed,
	eventbus.EventType(events.TopicStockOpnameRejected):  EventSORejected,
	eventbus.EventType(events.TopicStockOpnameRecount):   EventSORecount,
	eventbus.EventType(events.TopicStockOpnameCancelled): EventSOCancelled,
}

func NewStockOpnameStatusListener(hub *Hub) eventbus.Listener {
	eventTypes := make([]eventbus.EventType, 0, len(stockOpnameEventTypes))
	for et := range stockOpnameEventTypes {
		eventTypes = append(eventTypes, et)
	}
	return eventbus.NewListenerFunc(
		eventTypes,
		func(ctx context.Context, event eventbus.Event) error {
			wsType, ok := stockOpnameEventTypes[event.Type]
			if !ok {
				slog.Warn("[ws] unknown stock opname event", "type", event.Type)
				return nil
			}
			payload, ok := event.Payload.(*events.StockOpnameStatusChanged)
			if !ok {
				slog.Warn("[ws] unexpected payload type for stock opname event", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			if payload.SessionID == 0 {
				return nil
			}
			BroadcastStockOpnameStatus(hub, wsType, StockOpnameStatusEvent{
				SessionID:     payload.SessionID,
				SessionNumber: payload.SessionNumber,
				Status:        payload.Status,
				StoreID:       storeIDPtr(payload.StoreID),
			})
			return nil
		},
	)
}
