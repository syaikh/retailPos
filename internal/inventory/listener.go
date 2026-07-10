package inventory

import (
	"context"
	"fmt"
	"log"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/sale"
)

func NewStockDeductListener(repo *Repository) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{"sale.created"},
		func(ctx context.Context, event eventbus.Event) error {
			s, ok := event.Payload.(*sale.Sale)
			if !ok {
				log.Printf("[inventory] unexpected payload type for sale.created: %T", event.Payload)
				return nil
			}

			var lastErr error
			for _, item := range s.Items {
				if err := repo.AdjustStock(ctx, item.ProductID, -item.Quantity, nil, fmt.Sprintf("auto-deduct from sale #%d", s.ID)); err != nil {
					log.Printf("[inventory] failed to adjust stock for product %d: %v", item.ProductID, err)
					lastErr = err
				}
			}
			return lastErr
		},
	)
}
