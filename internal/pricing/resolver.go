package pricing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
)

// ResolverRepo is the data access interface the resolver depends on.
type ResolverRepo interface {
	GetBasePrice(ctx context.Context, productID int) (int, error)
	GetProductScope(ctx context.Context, productID int) (*int, *int, error)
	GetActiveRules(ctx context.Context, productID int, categoryID, brandID *int, now time.Time, customerGroupID, storeID *int) ([]Rule, error)
	GetBasePricesBatch(ctx context.Context, productIDs []int) (map[int]int, error)
	GetProductScopesBatch(ctx context.Context, productIDs []int) (map[int]ProductScope, error)
	GetActiveRulesBatch(ctx context.Context, productIDs []int, now time.Time) (map[int][]Rule, error)
	GetProductCostAndTax(ctx context.Context, productID int) (ProductCostTax, error)
	GetProductCostAndTaxBatch(ctx context.Context, productIDs []int) (map[int]ProductCostTax, error)
}

// Resolver implements PriceResolver using a deterministic 8-step algorithm.
type Resolver struct {
	repo ResolverRepo
}

// NewResolver creates a Resolver backed by the given repository.
func NewResolver(repo ResolverRepo) *Resolver {
	return &Resolver{repo: repo}
}

// Resolve returns the effective selling price for a product at a given quantity and context.
func (r *Resolver) Resolve(ctx context.Context, rc ResolveContext) (*ResolvedPrice, error) {
	basePrice, err := r.repo.GetBasePrice(ctx, rc.ProductID)
	if err != nil {
		return nil, err
	}

	categoryID, brandID, err := r.repo.GetProductScope(ctx, rc.ProductID)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(shared.JakartaLocation())
	rules, err := r.repo.GetActiveRules(ctx, rc.ProductID, categoryID, brandID, now, rc.CustomerGroupID, rc.StoreID)
	if err != nil {
		return nil, err
	}

	eligible := filterEligible(rules, rc.Quantity, now, rc.CustomerGroupID, rc.StoreID)
	result := resolvePricing(basePrice, eligible, rc.ProductID, categoryID, brandID)
	return &result, nil
}

// ResolveBatch returns effective selling prices for multiple products.
func (r *Resolver) ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error) {
	if len(items) == 0 {
		return []ResolvedPrice{}, nil
	}

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
	for _, id := range productIDs {
		if _, ok := basePrices[id]; !ok {
			return nil, fmt.Errorf("%w: product %d", ErrProductNotFound, id)
		}
	}

	scopes, err := r.repo.GetProductScopesBatch(ctx, productIDs)
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
		scope := scopes[item.ProductID]
		rules := rulesByProduct[item.ProductID]

		eligible := filterEligible(rules, item.Quantity, now, item.CustomerGroupID, item.StoreID)
		results[i] = resolvePricing(basePrice, eligible, item.ProductID, scope.CategoryID, scope.BrandID)
	}

	return results, nil
}

// ResolveSnapshot returns an immutable price snapshot (including cost & tax) for a single item.
// The pricing engine runs the same deterministic algorithm as Resolve, then captures the
// additional cost/tax fields and the snapshot timestamp.
func (r *Resolver) ResolveSnapshot(ctx context.Context, rc ResolveContext) (*PriceSnapshot, error) {
	resolved, err := r.Resolve(ctx, rc)
	if err != nil {
		return nil, err
	}

	costTax, err := r.repo.GetProductCostAndTax(ctx, rc.ProductID)
	if err != nil {
		return nil, err
	}

	return &PriceSnapshot{
		ProductID:     rc.ProductID,
		ProductName:   costTax.ProductName,
		UnitPrice:     resolved.UnitPrice,
		OriginalPrice: resolved.OriginalPrice,
		Discount:      resolved.Discount,
		Type:          resolved.Type,
		Method:        resolved.Method,
		Rule:          resolved.Rule,
		Cost:          costTax.Cost,
		TaxClassID:    costTax.TaxClassID,
		TaxRate:       costTax.TaxRate,
		SnapshotAt:    time.Now().In(shared.JakartaLocation()),
	}, nil
}

// ResolveSnapshotsBatch returns immutable price snapshots for multiple items.
func (r *Resolver) ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error) {
	if len(items) == 0 {
		return []PriceSnapshot{}, nil
	}

	resolved, err := r.ResolveBatch(ctx, items)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]bool)
	var productIDs []int
	for _, item := range items {
		if !seen[item.ProductID] {
			seen[item.ProductID] = true
			productIDs = append(productIDs, item.ProductID)
		}
	}

	costTax, err := r.repo.GetProductCostAndTaxBatch(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(shared.JakartaLocation())
	snapshots := make([]PriceSnapshot, len(items))
	for i, item := range items {
		ct, ok := costTax[item.ProductID]
		if !ok {
			ct = ProductCostTax{}
		}
		snapshots[i] = PriceSnapshot{
			ProductID:     item.ProductID,
			ProductName:   ct.ProductName,
			UnitPrice:     resolved[i].UnitPrice,
			OriginalPrice: resolved[i].OriginalPrice,
			Discount:      resolved[i].Discount,
			Type:          resolved[i].Type,
			Method:        resolved[i].Method,
			Rule:          resolved[i].Rule,
			Cost:          ct.Cost,
			TaxClassID:    ct.TaxClassID,
			TaxRate:       ct.TaxRate,
			SnapshotAt:    now,
		}
	}

	return snapshots, nil
}

// resolvePricing applies the deterministic resolution algorithm to a single product's
// eligible rules and base price. This is the shared core used by both Resolve and ResolveBatch.
func resolvePricing(basePrice int, eligible []Rule, productID int, categoryID, brandID *int) ResolvedPrice {
	if len(eligible) == 0 {
		return ResolvedPrice{
			UnitPrice:     basePrice,
			OriginalPrice: basePrice,
			Discount:      0,
			Type:          PricingTypeDefault,
			Method:        PricingMethodFixedPrice,
			Rule:          nil,
		}
	}

	combinable, nonCombinable := splitByCombine(eligible)
	if len(nonCombinable) > 0 && len(combinable) > 0 {
		bestNon := selectBestRule(nonCombinable, productID, categoryID, brandID)
		running := float64(computePrice(basePrice, bestNon))
		chainCombinable(combinable, productID, categoryID, brandID, &running)
		discount := basePrice - int(running+0.5)
		if discount < 0 {
			discount = 0
		}
		return ResolvedPrice{
			UnitPrice:     int(running + 0.5),
			OriginalPrice: basePrice,
			Discount:      discount,
			Type:          bestNon.Type,
			Method:        bestNon.Method,
			Rule:          bestNon,
		}
	}

	if len(combinable) > 0 {
		running := float64(basePrice)
		chainCombinable(combinable, productID, categoryID, brandID, &running)
		discount := basePrice - int(running+0.5)
		if discount < 0 {
			discount = 0
		}
		return ResolvedPrice{
			UnitPrice:     int(running + 0.5),
			OriginalPrice: basePrice,
			Discount:      discount,
			Type:          combinable[0].Type,
			Method:        combinable[0].Method,
			Rule:          &combinable[0],
		}
	}

	best := selectBestRule(nonCombinable, productID, categoryID, brandID)
	if best == nil {
		return ResolvedPrice{
			UnitPrice:     basePrice,
			OriginalPrice: basePrice,
			Discount:      0,
			Type:          PricingTypeDefault,
			Method:        PricingMethodFixedPrice,
			Rule:          nil,
		}
	}

	unitPrice := computePrice(basePrice, best)
	discount := basePrice - unitPrice
	if discount < 0 {
		discount = 0
	}

	return ResolvedPrice{
		UnitPrice:     unitPrice,
		OriginalPrice: basePrice,
		Discount:      discount,
		Type:          best.Type,
		Method:        best.Method,
		Rule:          best,
	}
}

// filterEligible filters rules by schedule, quantity range, scope, and active status.
func filterEligible(rules []Rule, quantity int, now time.Time, customerGroupID, storeID *int) []Rule {
	var eligible []Rule
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
		if rule.MaximumQuantity != nil && quantity > *rule.MaximumQuantity {
			continue
		}
		// Customer group filter
		if rule.CustomerGroupID != nil {
			if customerGroupID == nil || *rule.CustomerGroupID != *customerGroupID {
				continue
			}
		}
		// Store filter
		if rule.StoreID != nil {
			if storeID == nil || *rule.StoreID != *storeID {
				continue
			}
		}
		// Day-of-week filter
		if len(rule.RecurrenceDays) > 0 {
			dayName := strings.ToLower(now.Weekday().String())
			found := false
			for _, d := range rule.RecurrenceDays {
				if strings.ToLower(d) == dayName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Time-of-day filter
		if rule.TimeFrom != nil && rule.TimeTo != nil {
			nowTime := now.Format("15:04:05")
			if nowTime < *rule.TimeFrom || nowTime > *rule.TimeTo {
				continue
			}
		}
		eligible = append(eligible, rule)
	}
	return eligible
}

// splitByCombine separates eligible rules into combinable and non-combinable.
func splitByCombine(rules []Rule) (combinable, nonCombinable []Rule) {
	for _, r := range rules {
		if r.AllowCombine {
			combinable = append(combinable, r)
		} else {
			nonCombinable = append(nonCombinable, r)
		}
	}
	return
}

// chainCombinable sorts combinable rules by specificity, priority, value, then id
// and applies each sequentially, mutating running.
func chainCombinable(rules []Rule, productID int, categoryID, brandID *int, running *float64) {
	sort.Slice(rules, func(i, j int) bool {
		si := specificityScore(rules[i], productID, categoryID, brandID)
		sj := specificityScore(rules[j], productID, categoryID, brandID)
		if si != sj {
			return si > sj
		}
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		if rules[i].PricingValue != rules[j].PricingValue {
			return rules[i].PricingValue < rules[j].PricingValue
		}
		return rules[i].ID < rules[j].ID
	})
	for i := range rules {
		*running = float64(computePrice(int(*running+0.5), &rules[i]))
	}
}

// selectBestRule applies the deterministic resolution algorithm:
// 1. Sort by specificity (product > category > brand)
// 2. Then by priority DESC
// 3. Then by pricing_value ASC (best price wins)
// 4. Then by id ASC (tie-break)
func selectBestRule(rules []Rule, productID int, categoryID, brandID *int) *Rule {
	if len(rules) == 0 {
		return nil
	}

	sort.Slice(rules, func(i, j int) bool {
		si := specificityScore(rules[i], productID, categoryID, brandID)
		sj := specificityScore(rules[j], productID, categoryID, brandID)
		if si != sj {
			return si > sj // higher specificity wins
		}
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		if rules[i].PricingValue != rules[j].PricingValue {
			return rules[i].PricingValue < rules[j].PricingValue
		}
		return rules[i].ID < rules[j].ID
	})

	return &rules[0]
}

// specificityScore returns a score for scope specificity:
// product match = 3, category match = 2, brand match = 1, none = 0
func specificityScore(rule Rule, productID int, categoryID, brandID *int) int {
	if rule.ProductID != nil && *rule.ProductID == productID {
		return 3
	}
	if rule.CategoryID != nil && categoryID != nil && *rule.CategoryID == *categoryID {
		return 2
	}
	if rule.BrandID != nil && brandID != nil && *rule.BrandID == *brandID {
		return 1
	}
	return 0
}

// computePrice calculates the final unit price based on the pricing method.
func computePrice(basePrice int, rule *Rule) int {
	var unitPrice float64
	switch rule.Method {
	case PricingMethodFixedPrice:
		unitPrice = rule.PricingValue
	case PricingMethodDiscountPct:
		unitPrice = float64(basePrice) * (1 - rule.PricingValue/100)
	case PricingMethodDiscountAmt:
		unitPrice = float64(basePrice) - rule.PricingValue
	case PricingMethodMarkupPct:
		unitPrice = float64(basePrice) * (1 + rule.PricingValue/100)
	default:
		unitPrice = float64(basePrice)
	}
	if unitPrice < 0 {
		unitPrice = 0
	}
	return int(unitPrice + 0.5) // round to nearest integer
}
