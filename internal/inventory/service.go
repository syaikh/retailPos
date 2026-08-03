package inventory

import (
	"context"
	"fmt"
	"log/slog"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/shared"
)

type Service struct {
	repo     *Repository
	eventBus shared.EventBus
}

func NewService(repo *Repository, eventBus shared.EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error) {
	return s.repo.GetStockByProductID(ctx, productID)
}

// ListLocationStock returns rack-level stock rows for a product and/or location.
func (s *Service) ListLocationStock(ctx context.Context, productID, locationID int) ([]LocationStockItem, error) {
	return s.repo.ListLocationStock(ctx, productID, locationID)
}

// SetLocationStock records how much of a product sits in a rack. Global stock
// is unchanged; rack rows are a sub-account reconciled by rack stock opname.
func (s *Service) SetLocationStock(ctx context.Context, productID, locationID, quantity, userID int) error {
	return s.repo.SetLocationStock(ctx, productID, locationID, quantity, userID)
}

// TransferLocationStock moves stock between two racks. Global stock is unchanged.
func (s *Service) TransferLocationStock(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int) error {
	return s.repo.TransferLocationStock(ctx, productID, fromLocationID, toLocationID, quantity, userID)
}

func (s *Service) AdjustStock(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
	err := s.repo.AdjustStock(ctx, productID, quantityChange, &userID, notes)
	if err != nil {
		return fmt.Errorf("adjust stock: %w", err)
	}

	if err := s.eventBus.Publish(ctx, string(eventbus.StockAdjusted), StockAdjustedEvent{
		ProductID:      productID,
		QuantityChange: quantityChange,
		UserID:         userID,
		Notes:          notes,
	}); err != nil {
		slog.Warn("failed to publish stock adjusted event", "error", err)
	}

	return nil
}
