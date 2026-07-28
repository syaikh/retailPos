package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/purchase"
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
	payload, ok := event.Payload.(purchase.PurchaseReceiptPayload)
	if !ok {
		slog.Warn("invalid payload type for PurchaseReceiptCompleted", "type", fmt.Sprintf("%T", event.Payload))
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, item := range payload.Items {
		if item.QtyGood <= 0 {
			continue
		}

		key := fmt.Sprintf("%d-%d-%d", payload.GRID, payload.POID, item.ProductID)
		if l.processedItems[key] {
			slog.Debug("skipping already processed purchase receipt item", "key", key)
			continue
		}

		if err := l.svc.AdjustStock(ctx, item.ProductID, item.QtyGood, payload.UserID, "purchase_receipt"); err != nil {
			slog.Error("failed to adjust stock for purchase receipt",
				"product_id", item.ProductID,
				"qty", item.QtyGood,
				"gr_id", payload.GRID,
				"error", err,
			)
			return err
		}

		l.processedItems[key] = true
	}

	return nil
}

func (l *PurchaseReceiptListener) EventTypes() []eventbus.EventType {
	return []eventbus.EventType{"PurchaseReceiptCompleted"}
}
