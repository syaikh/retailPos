package inventory

import (
	"context"
	"fmt"
	"log/slog"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
)

type Repo interface {
	AdjustStock(ctx context.Context, productID int, quantityChange int, storeID *int, userID *int, notes string) error
	AdjustStockBatch(ctx context.Context, adjustments []StockAdjustment, userID *int, notes string) error
	GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error)
	ListLocationStock(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error)
	SetLocationStock(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error
	TransferLocationStock(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error
}

type service struct {
	repo     Repo
	eventBus shared.EventBus
}

func NewService(repo Repo, eventBus shared.EventBus) Service {
	return &service{repo: repo, eventBus: eventBus}
}

func (s *service) GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error) {
	return s.repo.GetStockByProductID(ctx, productID)
}

// ListLocationStock returns rack-level stock rows for a product and/or location.
// A non-nil storeID (store-scoped caller) restricts the rows to that store's racks.
func (s *service) ListLocationStock(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error) {
	return s.repo.ListLocationStock(ctx, productID, locationID, storeID)
}

// SetLocationStock records how much of a product sits in a rack. Global stock
// is unchanged; rack rows are a sub-account reconciled by rack stock opname.
// A store-scoped caller may only write racks that belong to their store.
func (s *service) SetLocationStock(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
	return s.repo.SetLocationStock(ctx, productID, locationID, quantity, userID, storeID)
}

// TransferLocationStock moves stock between two racks. Global stock is unchanged.
// A store-scoped caller may only move stock between racks of their own store.
func (s *service) TransferLocationStock(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
	return s.repo.TransferLocationStock(ctx, productID, fromLocationID, toLocationID, quantity, userID, storeID)
}

func (s *service) AdjustStock(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
	err := s.repo.AdjustStock(ctx, productID, quantityChange, storeID, &userID, notes)
	if err != nil {
		return fmt.Errorf("adjust stock: %w", err)
	}

	if err := s.eventBus.Publish(ctx, events.TopicStockAdjusted, &events.StockAdjusted{
		ProductID:      productID,
		QuantityChange: quantityChange,
		UserID:         userID,
		Notes:          notes,
	}); err != nil {
		slog.Warn("failed to publish stock adjusted event", "error", err)
	}

	return nil
}

// AdjustStockBatch applies all deltas in a single transaction, then emits one
// StockAdjusted event per product so realtime listeners still get per-product
// notifications.
func (s *service) AdjustStockBatch(ctx context.Context, adjustments []StockAdjustment, userID int, notes string) error {
	if len(adjustments) == 0 {
		return nil
	}

	if err := s.repo.AdjustStockBatch(ctx, adjustments, &userID, notes); err != nil {
		return fmt.Errorf("adjust stock batch: %w", err)
	}

	for _, adj := range adjustments {
		if err := s.eventBus.Publish(ctx, events.TopicStockAdjusted, &events.StockAdjusted{
			ProductID:      adj.ProductID,
			QuantityChange: adj.QuantityChange,
			UserID:         userID,
			Notes:          notes,
		}); err != nil {
			slog.Warn("failed to publish stock adjusted event", "error", err)
		}
	}

	return nil
}
