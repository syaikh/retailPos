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
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodMarkupPct, 15, 1, 0, true)},
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
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 0, true)},
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
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 3})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 5000, result.Discount)
	assert.Equal(t, PricingTypeSpecialPrice, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_QuantityExceedsMaximum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {ruleWithMaxQty(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 10, 0, true)},
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
				rule(11, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 1, 1, true),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 5})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, PricingTypeSpecialPrice, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_PriceTiebreak(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true),
				rule(11, PricingTypeSpecialPrice, PricingMethodFixedPrice, 11000, 1, 0, true),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 11000, result.UnitPrice)
	assert.Equal(t, PricingTypeSpecialPrice, result.PricingType)
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
				rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 12000, 3, 0, true),
				rule(11, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 5, 0, true),
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
				rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 14000, 1, 0, true),  // category match (score 2)
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
				rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 1, 0, true), // group-only rule
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
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 0, true)},
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
	assert.Equal(t, PricingTypeSpecialPrice, results[0].PricingType)

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
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 0, true)},
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
	assert.Equal(t, PricingTypeSpecialPrice, results[1].PricingType)
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
				rule(11, PricingTypeSpecialPrice, PricingMethodFixedPrice, 80000, 1, 1, true), // 80000 fixed, higher priority
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

// ============================================================
// Additional edge-case tests: stacking, recurrence, time-of-day,
// store filter, category/brand priority, customer group, max qty
// ============================================================

func ruleAdvanced(id int, pType PricingType, method PricingMethod, value float64, minQty int, maxQty *int, priority int, active, combine bool, catID, brandID, custGroup, storeID *int, days []string, tFrom, tTo *string) PricingRule {
	pid := 1
	r := PricingRule{
		ID:              id,
		ProductID:       &pid,
		PricingType:     pType,
		PricingMethod:   method,
		PricingValue:    value,
		Name:            string(pType),
		MinimumQuantity: minQty,
		MaximumQuantity: maxQty,
		Priority:        priority,
		IsActive:        active,
		AllowCombine:    combine,
		CategoryID:      catID,
		BrandID:         brandID,
		CustomerGroupID: custGroup,
		StoreID:         storeID,
		RecurrenceDays:  days,
		TimeFrom:        tFrom,
		TimeTo:          tTo,
	}
	return r
}

func intPtr(v int) *int { return &v }

func TestResolver_Resolve_StackingTwoPromotions(t *testing.T) {
	base := 100000
	repo := &mockRepo{
		basePrices: map[int]int{1: base},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountAmt, 5000, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Stacking: highest priority first: 100000 → 5000 off (pri 2) = 95000 → 10% off (pri 1) = 85500
	assert.Equal(t, 85500, result.UnitPrice)
	assert.Equal(t, 14500, result.Discount)
	assert.Equal(t, PricingTypePromotion, result.PricingType)
}

func TestResolver_Resolve_RecurrenceDayAllowed(t *testing.T) {
	// Today is Friday (2026-07-17) — include friday in recurrence_days
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 15, 1, nil, 0, true, false, nil, nil, nil, nil, []string{"thursday", "friday", "saturday"}, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 85000, result.UnitPrice)
}

func TestResolver_Resolve_RecurrenceDayBlocked(t *testing.T) {
	// Today is Friday (2026-07-17) — rule only applies on mon/tue/wed
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 15, 1, nil, 0, true, false, nil, nil, nil, nil, []string{"monday", "tuesday", "wednesday"}, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	// Today is Friday (2026-07-17) — weekday-only rules for mon/tue/wed should NOT apply
	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 100000, result.UnitPrice) // fallback to base price
	assert.Equal(t, PricingTypeDefault, result.PricingType)
}

func TestResolver_Resolve_StoreFilterMatch(t *testing.T) {
	storeID := intPtr(1)
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {ruleAdvanced(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 80000, 1, nil, 0, true, false, nil, nil, nil, storeID, nil, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1, StoreID: storeID})
	require.NoError(t, err)
	assert.Equal(t, 80000, result.UnitPrice)
}

func TestResolver_Resolve_StoreFilterMismatch(t *testing.T) {
	storeIDRule := intPtr(1)
	storeIDCtx := intPtr(99)
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {ruleAdvanced(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 80000, 1, nil, 0, true, false, nil, nil, nil, storeIDRule, nil, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1, StoreID: storeIDCtx})
	require.NoError(t, err)
	assert.Equal(t, 100000, result.UnitPrice)
}

func TestResolver_Resolve_CategoryScoped_WinsOverBrandScoped(t *testing.T) {
	catID := intPtr(5)
	brandID := intPtr(3)
	catRuleID, brandRuleID := 10, 20
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		scopes:     map[int]ProductScope{1: {CategoryID: catID, BrandID: brandID}},
		rules: map[int][]PricingRule{
			1: {
				{ID: catRuleID, ProductID: nil, CategoryID: catID, PricingType: PricingTypePromotion, PricingMethod: PricingMethodDiscountPct, PricingValue: 15, Name: "cat-discount", MinimumQuantity: 1, Priority: 0, IsActive: true},
				{ID: brandRuleID, ProductID: nil, BrandID: brandID, PricingType: PricingTypePromotion, PricingMethod: PricingMethodDiscountPct, PricingValue: 5, Name: "brand-discount", MinimumQuantity: 1, Priority: 0, IsActive: true},
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Category rule wins: specificity=2 > brand specificity=1
	assert.Equal(t, 85000, result.UnitPrice)
	require.NotNil(t, result.Rule)
	assert.Equal(t, catRuleID, result.Rule.ID)
}

func TestResolver_Resolve_CustomerGroupFilterMatch(t *testing.T) {
	vipGroup := intPtr(3)
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {ruleAdvanced(10, PricingTypeSpecialPrice, PricingMethodDiscountPct, 20, 1, nil, 0, true, false, nil, nil, vipGroup, nil, nil, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1, CustomerGroupID: vipGroup})
	require.NoError(t, err)
	assert.Equal(t, 80000, result.UnitPrice)
}

func TestResolver_Resolve_MaxQuantityFilter(t *testing.T) {
	maxQty := intPtr(5)
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {ruleAdvanced(10, PricingTypeSpecialPrice, PricingMethodDiscountPct, 15, 1, maxQty, 0, true, false, nil, nil, nil, nil, nil, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	// Quantity within max — rule applies
	result1, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 5})
	require.NoError(t, err)
	assert.Equal(t, 85000, result1.UnitPrice)

	// Quantity above max — rule excluded
	result2, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 6})
	require.NoError(t, err)
	assert.Equal(t, 100000, result2.UnitPrice)
}

// ============================================================
// Stacking (allow_combine) — full coverage
// ============================================================

func TestResolver_Resolve_StackingThreePromotions(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil), // priority 1
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountAmt, 5000, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil), // priority 2
				ruleAdvanced(12, PricingTypePromotion, PricingMethodDiscountPct, 5, 1, nil, 3, true, true, nil, nil, nil, nil, nil, nil, nil),  // priority 3
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Highest priority first: 100000 → 5% off (pri 3) = 95000 → 5000 off (pri 2) = 90000 → 10% off (pri 1) = 81000
	assert.Equal(t, 81000, result.UnitPrice)
	assert.Equal(t, 19000, result.Discount)
}

func TestResolver_Resolve_StackingBestNonCombinablePlusCombinable(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 80000, 1, nil, 5, true, false, nil, nil, nil, nil, nil, nil, nil), // best non-combinable: 80000
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),  // combinable: 10% → 72000
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Non-combinable wins: 80000, then 10% off → 72000
	assert.Equal(t, 72000, result.UnitPrice)
	assert.Equal(t, 28000, result.Discount)
}

func TestResolver_Resolve_StackingFixedPriceThenPercent(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 200000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodFixedPrice, 150000, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil), // fixed → 150000
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil),   // 10% → 135000
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Highest priority first: 200000 → 10% off (pri 2) = 180000 → fixed 150000 (pri 1) = 150000
	assert.Equal(t, 150000, result.UnitPrice)
	assert.Equal(t, 50000, result.Discount)
}

func TestResolver_Resolve_StackingFloorAtZero(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 10000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountAmt, 6000, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil), // 6000 → 4000
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountAmt, 8000, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil), // 8000 → floor 0
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, result.UnitPrice)
	assert.Equal(t, 10000, result.Discount)
}

func TestResolver_Resolve_StackingPriorityOrder(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 5, 1, nil, 10, true, true, nil, nil, nil, nil, nil, nil, nil),   // priority 10
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountPct, 15, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil), // priority 1
				ruleAdvanced(12, PricingTypePromotion, PricingMethodDiscountAmt, 10000, 1, nil, 5, true, true, nil, nil, nil, nil, nil, nil, nil), // priority 5
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Highest priority first: 100000 → 5% off (pri 10) = 95000 → 10000 off (pri 5) = 85000 → 15% off (pri 1) = 72250
	assert.Equal(t, 72250, result.UnitPrice)
}

func TestResolver_Resolve_NoStackingWhenAllowCombineFalse(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, false, nil, nil, nil, nil, nil, nil, nil),
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountAmt, 5000, 1, nil, 2, true, false, nil, nil, nil, nil, nil, nil, nil),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	// Single best rule: priority 2 wins → 5000 off → 95000
	assert.Equal(t, 95000, result.UnitPrice)
}

func TestResolver_Resolve_StackingSingleCombinable(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]PricingRule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 20, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 80000, result.UnitPrice)
	assert.Equal(t, 20000, result.Discount)
}
