package pricing

import (
	"context"
	"sort"
	"time"

	"retail-pos-system/internal/shared"
)

// ResolverRepo is the data access interface the resolver depends on.
// The mockRepo in tests and the real Repository both satisfy this.
type ResolverRepo interface {
	GetBasePrice(ctx context.Context, productID int) (int, error)
	GetActiveRules(ctx context.Context, productID int, now time.Time) ([]PricingRule, error)
	GetBasePricesBatch(ctx context.Context, productIDs []int) (map[int]int, error)
	GetActiveRulesBatch(ctx context.Context, productIDs []int, now time.Time) (map[int][]PricingRule, error)
}

// Resolver implements PriceResolver using a deterministic algorithm.
// See ADR-005.
type Resolver struct {
	repo ResolverRepo
}

// NewResolver creates a Resolver backed by the given repository.
func NewResolver(repo ResolverRepo) *Resolver {
	return &Resolver{repo: repo}
}

// Resolve returns the effective selling price for a product at a given quantity.
func (r *Resolver) Resolve(ctx context.Context, productID int, quantity int) (*ResolvedPrice, error) {
	basePrice, err := r.repo.GetBasePrice(ctx, productID)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(shared.JakartaLocation())
	rules, err := r.repo.GetActiveRules(ctx, productID, now)
	if err != nil {
		return nil, err
	}

	best := selectBestRule(rules, quantity, now)
	if best == nil {
		return &ResolvedPrice{
			UnitPrice:     basePrice,
			OriginalPrice: basePrice,
			Discount:      0,
			PricingType:   PricingTypeNormal,
			Rule:          nil,
		}, nil
	}

	discount := basePrice - best.Price
	if discount < 0 {
		discount = 0
	}

	return &ResolvedPrice{
		UnitPrice:     best.Price,
		OriginalPrice: basePrice,
		Discount:      discount,
		PricingType:   best.PricingType,
		Rule:          best,
	}, nil
}

// ResolveBatch returns effective selling prices for multiple products.
func (r *Resolver) ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error) {
	if len(items) == 0 {
		return []ResolvedPrice{}, nil
	}

	// Collect unique product IDs
	seen := make(map[int]bool)
	var productIDs []int
	for _, item := range items {
		if !seen[item.ProductID] {
			seen[item.ProductID] = true
			productIDs = append(productIDs, item.ProductID)
		}
	}

	basePrices, err := r.repo.GetBasePricesBatch(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(shared.JakartaLocation())
	rulesByProduct, err := r.repo.GetActiveRulesBatch(ctx, productIDs, now)
	if err != nil {
		return nil, err
	}

	results := make([]ResolvedPrice, len(items))
	for i, item := range items {
		basePrice := basePrices[item.ProductID]
		rules := rulesByProduct[item.ProductID]

		best := selectBestRule(rules, item.Quantity, now)
		if best == nil {
			results[i] = ResolvedPrice{
				UnitPrice:     basePrice,
				OriginalPrice: basePrice,
				Discount:      0,
				PricingType:   PricingTypeNormal,
				Rule:          nil,
			}
		} else {
			discount := basePrice - best.Price
			if discount < 0 {
				discount = 0
			}
			results[i] = ResolvedPrice{
				UnitPrice:     best.Price,
				OriginalPrice: basePrice,
				Discount:      discount,
				PricingType:   best.PricingType,
				Rule:          best,
			}
		}
	}

	return results, nil
}

// selectBestRule applies the deterministic resolution algorithm from ADR-005:
// 1. Filter: is_active, not expired, not future, quantity >= minimum_quantity
// 2. Sort: priority DESC, price ASC, id ASC
// 3. Return first (best) rule, or nil if none qualify
func selectBestRule(rules []PricingRule, quantity int, now time.Time) *PricingRule {
	// Filter eligible rules
	var eligible []PricingRule
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		if rule.EffectiveFrom != nil && now.Before(*rule.EffectiveFrom) {
			continue
		}
		if rule.EffectiveUntil != nil && now.After(*rule.EffectiveUntil) {
			continue
		}
		if quantity < rule.MinimumQuantity {
			continue
		}
		eligible = append(eligible, rule)
	}

	if len(eligible) == 0 {
		return nil
	}

	// Sort: priority DESC, price ASC, id ASC
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].Priority != eligible[j].Priority {
			return eligible[i].Priority > eligible[j].Priority
		}
		if eligible[i].Price != eligible[j].Price {
			return eligible[i].Price < eligible[j].Price
		}
		return eligible[i].ID < eligible[j].ID
	})

	return &eligible[0]
}
