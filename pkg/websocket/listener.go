package websocket

import (
	"context"
	"fmt"
	"log/slog"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/stockopname"
)

func NewSaleCreatedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.SaleCreated},
		func(ctx context.Context, event eventbus.Event) error {
			s, ok := event.Payload.(*sale.Sale)
			if !ok {
				slog.Warn("[ws] unexpected payload type for sale.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			items := 0
			if s.Items != nil {
				items = len(s.Items)
			}
			BroadcastSaleCreated(hub, SaleCreatedEvent{
				ID:      s.ID,
				Invoice: s.InvoiceNumber,
				Total:   s.TotalAmount,
				Items:   items,
				StoreID: s.StoreID,
			})
			return nil
		},
	)
}

func NewProductUpdatedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.ProductUpdated},
		func(ctx context.Context, event eventbus.Event) error {
			var p *product.Product
			switch payload := event.Payload.(type) {
			case *product.Product:
				p = payload
			case eventbus.UpdatePayload:
				var ok bool
				p, ok = payload.New.(*product.Product)
				if !ok {
					slog.Warn("[ws] unexpected New type in UpdatePayload for product.updated", "type", fmt.Sprintf("%T", payload.New))
					return nil
				}
			default:
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

func NewPOReceivedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.EventType("goods_receipt.created")},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(map[string]interface{})
			if !ok {
				slog.Warn("[ws] unexpected payload type for goods_receipt.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			poID, _ := payload["po_id"].(int)
			poNumber, _ := payload["po_number"].(string)
			grNumber, _ := payload["gr_number"].(string)
			var storeID *int
			if sid, ok := payload["store_id"].(int); ok && sid > 0 {
				storeID = &sid
			}
			BroadcastPOReceived(hub, POReceivedEvent{
				POID:     poID,
				PONumber: poNumber,
				GRNumber: grNumber,
				StoreID:  storeID,
			})
			return nil
		},
	)
}

func extractPOID(payload map[string]interface{}) int {
	switch v := payload["po_id"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

func extractStoreID(payload map[string]interface{}) *int {
	switch v := payload["store_id"].(type) {
	case int:
		if v > 0 {
			return &v
		}
	case float64:
		if v > 0 {
			sid := int(v)
			return &sid
		}
	case int64:
		if v > 0 {
			sid := int(v)
			return &sid
		}
	}
	return nil
}

func extractPOEventFields(payload map[string]interface{}) (poID int, poNumber string, storeID *int) {
	poID = extractPOID(payload)
	poNumber, _ = payload["po_number"].(string)
	storeID = extractStoreID(payload)
	return
}

func NewPOCreatedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.EventType("purchase_order.created")},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(map[string]interface{})
			if !ok {
				slog.Warn("[ws] unexpected payload type for purchase_order.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			poID, poNumber, storeID := extractPOEventFields(payload)
			BroadcastPOCreated(hub, POCreatedEvent{
				POID:     poID,
				PONumber: poNumber,
				StoreID:  storeID,
			})
			return nil
		},
	)
}

func NewPOConfirmedListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.EventType("purchase_order.confirmed")},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(map[string]interface{})
			if !ok {
				slog.Warn("[ws] unexpected payload type for purchase_order.confirmed", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			poID, poNumber, storeID := extractPOEventFields(payload)
			BroadcastPOConfirmed(hub, POConfirmedEvent{
				POID:     poID,
				PONumber: poNumber,
				StoreID:  storeID,
			})
			return nil
		},
	)
}

func NewPOCancelledListener(hub *Hub) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.EventType("purchase_order.cancelled")},
		func(ctx context.Context, event eventbus.Event) error {
			payload, ok := event.Payload.(map[string]interface{})
			if !ok {
				slog.Warn("[ws] unexpected payload type for purchase_order.cancelled", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			poID, poNumber, storeID := extractPOEventFields(payload)
			BroadcastPOCancelled(hub, POCancelledEvent{
				POID:     poID,
				PONumber: poNumber,
				StoreID:  storeID,
			})
			return nil
		},
	)
}

func NewStockAdjustedListener(hub *Hub, products ProductLookup) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{eventbus.StockAdjusted},
		func(ctx context.Context, event eventbus.Event) error {
			sa, ok := event.Payload.(inventory.StockAdjustedEvent)
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
	eventbus.EventType(stockopname.EventStockOpnameCreated):   EventSOCreated,
	eventbus.EventType(stockopname.EventStockOpnameSubmitted): EventSOSubmitted,
	eventbus.EventType(stockopname.EventStockOpnameApproved):  EventSOApproved,
	eventbus.EventType(stockopname.EventStockOpnameRejected):  EventSORejected,
	eventbus.EventType(stockopname.EventStockOpnameRecount):   EventSORecount,
	eventbus.EventType(stockopname.EventStockOpnameCancelled): EventSOCancelled,
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
			payload, ok := event.Payload.(map[string]interface{})
			if !ok {
				slog.Warn("[ws] unexpected payload type for stock opname event", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}
			sessionID := extractSessionID(payload)
			if sessionID == 0 {
				return nil
			}
			sessionNumber, _ := payload["session_number"].(string)
			status, _ := payload["status"].(string)
			storeID := extractStoreID(payload)
			BroadcastStockOpnameStatus(hub, wsType, StockOpnameStatusEvent{
				SessionID:     sessionID,
				SessionNumber: sessionNumber,
				Status:        status,
				StoreID:       storeID,
			})
			return nil
		},
	)
}

func extractSessionID(payload map[string]interface{}) int {
	switch v := payload["session_id"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}
