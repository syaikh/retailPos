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

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool) ([]PricingRule, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, productID, pricingType, pricingMethod, categoryID, brandID, customerGroupID, storeID, isActive)
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
	if rule.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidRule)
	}
	// At least one target must be set
	if rule.ProductID == nil && rule.CategoryID == nil && rule.BrandID == nil {
		return fmt.Errorf("%w: at least one of product_id, category_id, or brand_id is required", ErrInvalidRule)
	}
	if rule.PricingType == "" {
		return fmt.Errorf("%w: pricing_type is required", ErrInvalidRule)
	}
	validType := false
	for _, t := range ValidPricingTypes() {
		if rule.PricingType == t {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("%w: invalid pricing_type '%s'", ErrInvalidRule, rule.PricingType)
	}
	if rule.PricingMethod == "" {
		return fmt.Errorf("%w: pricing_method is required", ErrInvalidRule)
	}
	validMethod := false
	for _, m := range ValidPricingMethods() {
		if rule.PricingMethod == m {
			validMethod = true
			break
		}
	}
	if !validMethod {
		return fmt.Errorf("%w: invalid pricing_method '%s'", ErrInvalidRule, rule.PricingMethod)
	}
	if rule.PricingValue < 0 {
		return fmt.Errorf("%w: pricing_value must be non-negative", ErrInvalidRule)
	}
	if rule.PricingMethod == PricingMethodDiscountPct && (rule.PricingValue < 0 || rule.PricingValue > 100) {
		return fmt.Errorf("%w: discount_percent pricing_value must be between 0 and 100", ErrInvalidRule)
	}
	if rule.PricingMethod == PricingMethodMarkupPct && (rule.PricingValue < 0 || rule.PricingValue > 500) {
		return fmt.Errorf("%w: markup_percent pricing_value must be between 0 and 500", ErrInvalidRule)
	}
	if rule.MinimumQuantity < 1 {
		return fmt.Errorf("%w: minimum_quantity must be at least 1", ErrInvalidRule)
	}
	if rule.MaximumQuantity != nil && *rule.MaximumQuantity < rule.MinimumQuantity {
		return fmt.Errorf("%w: maximum_quantity must be >= minimum_quantity", ErrInvalidRule)
	}
	return nil
}
