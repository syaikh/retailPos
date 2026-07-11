package inventory

import (
	"context"
	"fmt"
	"log/slog"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/sale"
)

func NewStockDeductListener(repo *Repository) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{"sale.created"},
		func(ctx context.Context, event eventbus.Event) error {
			s, ok := event.Payload.(*sale.Sale)
			if !ok {
				slog.Warn("unexpected payload type for sale.created", "type", fmt.Sprintf("%T", event.Payload))
				return nil
			}

			var lastErr error
			for _, item := range s.Items {
				if err := repo.AdjustStock(ctx, item.ProductID, -item.Quantity, nil, fmt.Sprintf("auto-deduct from sale #%d", s.ID)); err != nil {
					slog.Warn("failed to adjust stock", "product_id", item.ProductID, "sale_id", s.ID, "error", err)
					lastErr = err
				}
			}
			return lastErr
		},
	)
}
