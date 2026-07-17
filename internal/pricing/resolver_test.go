package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	basePrices map[int]int
	scopes     map[int]ProductScope
	rules      map[int][]PricingRule
}

func (m *mockRepo) GetBasePrice(_ context.Context, productID int) (int, error) {
	price, ok := m.basePrices[productID]
	if !ok {
		return 0, ErrProductNotFound
	}
	return price, nil
}

func (m *mockRepo) GetProductScope(_ context.Context, productID int) (*int, *int, error) {
	if _, ok := m.basePrices[productID]; !ok {
		return nil, nil, ErrProductNotFound
	}
	s, ok := m.scopes[productID]
	if !ok {
		return nil, nil, nil
	}
	return s.CategoryID, s.BrandID, nil
}

func (m *mockRepo) GetActiveRules(_ context.Context, productID int, categoryID, brandID *int, now time.Time, customerGroupID, storeID *int) ([]PricingRule, error) {
	return m.rules[productID], nil
}

func (m *mockRepo) GetBasePricesBatch(_ context.Context, productIDs []int) (map[int]int, error) {
	result := make(map[int]int, len(productIDs))
	for _, pid := range productIDs {
		if price, ok := m.basePrices[pid]; ok {
			result[pid] = price
		}
	}
	return result, nil
}

func (m *mockRepo) GetProductScopesBatch(_ context.Context, productIDs []int) (map[int]ProductScope, error) {
	result := make(map[int]ProductScope, len(productIDs))
	for _, pid := range productIDs {
		if s, ok := m.scopes[pid]; ok {
			result[pid] = s
		}
	}
	return result, nil
}

func (m *mockRepo) GetActiveRulesBatch(_ context.Context, productIDs []int, now time.Time) (map[int][]PricingRule, error) {
	result := make(map[int][]PricingRule, len(productIDs))
	for _, pid := range productIDs {
		if rules, ok := m.rules[pid]; ok {
			result[pid] = rules
		}
	}
	return result, nil
}

// --- Helper to build PricingRules for tests ---

func rule(id int, pType PricingType, method PricingMethod, value float64, minQty, priority int, active bool) PricingRule {
	pid := 1
	return PricingRule{
		ID:              id,
		ProductID:       &pid,
		PricingType:     pType,
		PricingMethod:   method,
		PricingValue:    value,
		Name:            string(pType),
		MinimumQuantity: minQty,
		Priority:        priority,
		IsActive:        active,
	}
}

func ruleWithDates(id int, pType PricingType, method PricingMethod, value float64, minQty, priority int, active bool, from, until *time.Time) PricingRule {
	r := rule(id, pType, method, value, minQty, priority, active)
	r.EffectiveFrom = from
	r.EffectiveUntil = until
	return r
}

func ruleWithMaxQty(id int, pType PricingType, method PricingMethod, value float64, minQty, maxQty, priority int, active bool) PricingRule {
	r := rule(id, pType, method, value, minQty, priority, active)
	r.MaximumQuantity = &maxQty
	return r
}

// ============================================================
// Resolve — single product resolution
// ============================================================

func TestResolver_Resolve_BasePriceOnly(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules:      map[int][]PricingRule{1: {}},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 0, result.Discount)
	assert.Equal(t, PricingTypeDefault, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_ProductNotFound(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{},
		rules:      map[int][]PricingRule{},
	}
	resolver := NewResolver(repo)

	_, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 999, Quantity: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestResolver_Resolve_FixedPriceRule(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 12000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 3000, result.Discount)
	assert.Equal(t, PricingTypePromotion, result.PricingType)
	assert.Equal(t, PricingMethodFixedPrice, result.PricingMethod)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_DiscountPercent(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 90000, result.UnitPrice) // 100000 * 0.9
	assert.Equal(t, 100000, result.OriginalPrice)
	assert.Equal(t, 10000, result.Discount)
}

func TestResolver_Resolve_DiscountAmount(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 50000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, PricingMethodDiscountAmt, 5000, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 45000, result.UnitPrice) // 50000 - 5000
	assert.Equal(t, 50000, result.OriginalPrice)
	assert.Equal(t, 5000, result.Discount)
}

func TestResolver_Resolve_MarkupPercent(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypeDefault, PricingMethodMarkupPct, 15, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 115000, result.UnitPrice) // 100000 * 1.15
	assert.Equal(t, 100000, result.OriginalPrice)
	assert.Equal(t, 0, result.Discount) // markup has no discount
}

func TestResolver_Resolve_QuantityBelowMinimum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePriceList, PricingMethodFixedPrice, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 2})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeDefault, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_QuantityMeetsMinimum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePriceList, PricingMethodFixedPrice, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 3})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 5000, result.Discount)
	assert.Equal(t, PricingTypePriceList, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_QuantityExceedsMaximum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {ruleWithMaxQty(10, PricingTypePriceList, PricingMethodFixedPrice, 10000, 3, 10, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	// qty=11 exceeds max_qty=10
	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 11})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Nil(t, result.Rule)

	// qty=10 matches max
	result, err = resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 10})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.NotNil(t, result.Rule)
}

func TestResolver_Resolve_PriorityWins(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true),
				rule(11, PricingTypePriceList, PricingMethodFixedPrice, 10000, 1, 1, true),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 5})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, PricingTypePriceList, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_PriceTiebreak(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true),
				rule(11, PricingTypePriceList, PricingMethodFixedPrice, 11000, 1, 0, true),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 11000, result.UnitPrice)
	assert.Equal(t, PricingTypePriceList, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_ExpiredRuleFallsBack(t *testing.T) {
	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {ruleWithDates(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true, nil, &past)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeDefault, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_InactiveRuleFallsBack(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, false)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeDefault, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_FutureRuleFallsBack(t *testing.T) {
	future := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {ruleWithDates(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true, &future, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeDefault, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_NoRulesForProduct(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules:      map[int][]PricingRule{1: nil},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeDefault, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_MultipleTiers(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePriceList, PricingMethodFixedPrice, 12000, 3, 0, true),
				rule(11, PricingTypePriceList, PricingMethodFixedPrice, 10000, 5, 0, true),
			},
		},
	}
	resolver := NewResolver(repo)

	// qty=4: only first tier matches
	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 4})
	require.NoError(t, err)
	assert.Equal(t, 12000, result.UnitPrice)

	// qty=6: both match, lower price wins (same priority)
	result, err = resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 6})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_RuleWithZeroValue(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, PricingMethodFixedPrice, 0, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 15000, result.Discount)
	assert.Equal(t, PricingTypePromotion, result.PricingType)
}

// ============================================================
// Scope-based resolution
// ============================================================

func TestResolver_Resolve_ProductScopeWins(t *testing.T) {
	catID := 5
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		scopes:     map[int]ProductScope{1: {CategoryID: &catID}},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePriceList, PricingMethodFixedPrice, 14000, 1, 0, true),  // category match (score 2)
				rule(11, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true),  // product match (score 3)
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 12000, result.UnitPrice) // product-scope rule wins
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_CustomerGroupFilter(t *testing.T) {
	groupID := 2
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePriceList, PricingMethodFixedPrice, 10000, 1, 0, true), // group-only rule
			},
		},
	}
	repo.rules[1][0].CustomerGroupID = &groupID
	resolver := NewResolver(repo)

	// No customer group — rule should not apply
	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Nil(t, result.Rule)

	// With matching customer group — rule applies
	result, err = resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1, CustomerGroupID: &groupID})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.NotNil(t, result.Rule)
}

// ============================================================
// ResolveBatch — batch resolution for POS cart
// ============================================================

func TestResolver_ResolveBatch_MixedProducts(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000, 2: 20000, 3: 5000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePriceList, PricingMethodFixedPrice, 10000, 3, 0, true)},
			2: {},
			3: {rule(20, PricingTypePromotion, PricingMethodFixedPrice, 4000, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	items := []ResolveItem{
		{ProductID: 1, Quantity: 5},
		{ProductID: 2, Quantity: 1},
		{ProductID: 3, Quantity: 2},
	}

	results, err := resolver.ResolveBatch(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Equal(t, 10000, results[0].UnitPrice)
	assert.Equal(t, PricingTypePriceList, results[0].PricingType)

	assert.Equal(t, 20000, results[1].UnitPrice)
	assert.Equal(t, PricingTypeDefault, results[1].PricingType)

	assert.Equal(t, 4000, results[2].UnitPrice)
	assert.Equal(t, PricingTypePromotion, results[2].PricingType)
}

func TestResolver_ResolveBatch_EmptyItems(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{},
		rules:      map[int][]PricingRule{},
	}
	resolver := NewResolver(repo)

	results, err := resolver.ResolveBatch(context.Background(), []ResolveItem{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestResolver_ResolveBatch_DuplicateProducts(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePriceList, PricingMethodFixedPrice, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	items := []ResolveItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 1, Quantity: 5},
	}

	results, err := resolver.ResolveBatch(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, 15000, results[0].UnitPrice)
	assert.Equal(t, PricingTypeDefault, results[0].PricingType)

	assert.Equal(t, 10000, results[1].UnitPrice)
	assert.Equal(t, PricingTypePriceList, results[1].PricingType)
}

// ============================================================
// Stacking (allow_combine)
// ============================================================

func TestResolver_Resolve_NoStacking_BestSingleRule(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, 0, true),   // 10% off
				rule(11, PricingTypePriceList, PricingMethodFixedPrice, 80000, 1, 1, true), // 80000 fixed, higher priority
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Best single rule: fixed price 80000 (higher priority wins)
	assert.Equal(t, 80000, result.UnitPrice)
}

func TestResolver_Resolve_DiscountDoesNotGoBelowZero(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 10000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, PricingMethodDiscountAmt, 50000, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, result.UnitPrice) // clamped to 0
	assert.Equal(t, 10000, result.Discount)
}
