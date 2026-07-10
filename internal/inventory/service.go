package inventory

import (
	"context"
	"fmt"
	"log"

	"retail-pos-system/internal/eventbus"
)

type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}

type Service struct {
	repo     *Repository
	eventBus EventBus
}

func NewService(repo *Repository, eventBus EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error) {
	return s.repo.GetStockByProductID(ctx, productID)
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
		log.Printf("[inventory] failed to publish stock adjusted event: %v", err)
	}

	return nil
}
