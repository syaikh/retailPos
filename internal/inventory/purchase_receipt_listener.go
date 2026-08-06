package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"
)

type PurchaseReceiptListener struct {
	repo           *Repository
	svc            *Service
	processedItems map[string]bool
	mu             sync.Mutex
}

func NewPurchaseReceiptListener(repo *Repository, svc *Service) *PurchaseReceiptListener {
	return &PurchaseReceiptListener{
		repo:           repo,
		svc:            svc,
		processedItems: make(map[string]bool),
	}
}

func (l *PurchaseReceiptListener) HandleEvent(ctx context.Context, event eventbus.Event) error {
	payload, ok := event.Payload.(events.PurchaseReceiptCompleted)
	if !ok {
		slog.Warn("invalid payload type for PurchaseReceiptCompleted", "type", fmt.Sprintf("%T", event.Payload))
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var adjustments []StockAdjustment
	for _, item := range payload.Items {
		if item.QtyGood <= 0 {
			continue
		}

		key := fmt.Sprintf("%d-%d-%d", payload.GRID, payload.POID, item.ProductID)
		if l.processedItems[key] {
			slog.Debug("skipping already processed purchase receipt item", "key", key)
			continue
		}

		adjustments = append(adjustments, StockAdjustment{
			ProductID:      item.ProductID,
			QuantityChange: item.QtyGood,
		})
	}

	if len(adjustments) == 0 {
		return nil
	}

	if err := l.svc.AdjustStockBatch(ctx, adjustments, payload.UserID, "purchase_receipt"); err != nil {
		slog.Error("failed to adjust stock for purchase receipt",
			"product_ids", adjustmentProductIDs(adjustments),
			"gr_id", payload.GRID,
			"po_id", payload.POID,
			"error", err,
		)
		return err
	}

	for _, item := range payload.Items {
		if item.QtyGood <= 0 {
			continue
		}
		key := fmt.Sprintf("%d-%d-%d", payload.GRID, payload.POID, item.ProductID)
		l.processedItems[key] = true
	}

	return nil
}

func adjustmentProductIDs(adjustments []StockAdjustment) []int {
	ids := make([]int, len(adjustments))
	for i, adj := range adjustments {
		ids[i] = adj.ProductID
	}
	return ids
}

func (l *PurchaseReceiptListener) EventTypes() []eventbus.EventType {
	return []eventbus.EventType{events.TopicPurchaseReceiptCompleted}
}
