package pricing

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id int) (*PricingRule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByProductID(ctx context.Context, productID int) ([]PricingRule, error) {
	return s.repo.GetByProductID(ctx, productID)
}

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType string, isActive *bool) ([]PricingRule, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, productID, pricingType, isActive)
}

func (s *Service) Create(ctx context.Context, rule *PricingRule) error {
	if err := validateRule(rule); err != nil {
		return err
	}
	return s.repo.Create(ctx, rule)
}

func (s *Service) Update(ctx context.Context, rule *PricingRule) error {
	if err := validateRule(rule); err != nil {
		return err
	}
	return s.repo.Update(ctx, rule)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func validateRule(rule *PricingRule) error {
	if rule.ProductID <= 0 {
		return fmt.Errorf("%w: product_id is required", ErrInvalidRule)
	}
	if rule.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidRule)
	}
	if rule.Price < 0 {
		return fmt.Errorf("%w: price must be non-negative", ErrInvalidRule)
	}
	if rule.MinimumQuantity < 1 {
		return fmt.Errorf("%w: minimum_quantity must be at least 1", ErrInvalidRule)
	}
	if rule.PricingType == "" {
		return fmt.Errorf("%w: pricing_type is required", ErrInvalidRule)
	}
	return nil
}
