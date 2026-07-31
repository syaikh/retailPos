package websocket

import (
	"context"
	"fmt"
	"log/slog"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/sale"
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
