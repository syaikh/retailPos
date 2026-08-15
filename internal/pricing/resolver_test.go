package pricing

import (
	"context"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	basePrices map[int]int
	scopes     map[int]ProductScope
	rules      map[int][]Rule
	costTax    map[int]ProductCostTax
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

func (m *mockRepo) GetActiveRules(_ context.Context, productID int, categoryID, brandID *int, now time.Time, customerGroupID, storeID *int) ([]Rule, error) {
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

func (m *mockRepo) GetActiveRulesBatch(_ context.Context, productIDs []int, now time.Time) (map[int][]Rule, error) {
	result := make(map[int][]Rule, len(productIDs))
	for _, pid := range productIDs {
		if rules, ok := m.rules[pid]; ok {
			result[pid] = rules
		}
	}
	return result, nil
}

func (m *mockRepo) GetProductCostAndTax(_ context.Context, productID int) (ProductCostTax, error) {
	ct, ok := m.costTax[productID]
	if !ok {
		return ProductCostTax{}, ErrProductNotFound
	}
	return ct, nil
}

func (m *mockRepo) GetProductCostAndTaxBatch(_ context.Context, productIDs []int) (map[int]ProductCostTax, error) {
	result := make(map[int]ProductCostTax, len(productIDs))
	for _, pid := range productIDs {
		if ct, ok := m.costTax[pid]; ok {
			result[pid] = ct
		}
	}
	return result, nil
}

// --- Helper to build PricingRules for tests ---

func rule(id int, pType Type, method Method, value float64, minQty, priority int, active bool) Rule {
	pid := 1
	return Rule{
		ID:              id,
		ProductID:       &pid,
		Type:            pType,
		Method:          method,
		PricingValue:    value,
		Name:            string(pType),
		MinimumQuantity: minQty,
		Priority:        priority,
		IsActive:        active,
	}
}

func ruleWithDates(id int, pType Type, method Method, value float64, minQty, priority int, active bool, from, until *time.Time) Rule {
	r := rule(id, pType, method, value, minQty, priority, active)
	r.EffectiveFrom = from
	r.EffectiveUntil = until
	return r
}

func ruleWithMaxQty(id int, pType Type, method Method, value float64, minQty, maxQty, priority int, active bool) Rule {
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
		rules:      map[int][]Rule{1: {}},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 0, result.Discount)
	assert.Equal(t, PricingTypeNormal, result.Type)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_ProductNotFound(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{},
		rules:      map[int][]Rule{},
	}
	resolver := NewResolver(repo)

	_, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 999, Quantity: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestResolver_Resolve_FixedPriceRule(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
			1: {rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 12000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 3000, result.Discount)
	assert.Equal(t, PricingTypePromotion, result.Type)
	assert.Equal(t, PricingMethodFixedPrice, result.Method)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_DiscountPercent(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 2})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.Type)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_QuantityMeetsMinimum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
			1: {rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 3})
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 5000, result.Discount)
	assert.Equal(t, PricingTypeSpecialPrice, result.Type)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_QuantityExceedsMaximum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
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
	assert.Equal(t, PricingTypeSpecialPrice, result.Type)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_PriceTiebreak(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
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
	assert.Equal(t, PricingTypeSpecialPrice, result.Type)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_ExpiredRuleFallsBack(t *testing.T) {
	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
			1: {ruleWithDates(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true, nil, &past)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.Type)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_InactiveRuleFallsBack(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
			1: {rule(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, false)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.Type)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_FutureRuleFallsBack(t *testing.T) {
	future := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
			1: {ruleWithDates(10, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true, &future, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.Type)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_NoRulesForProduct(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules:      map[int][]Rule{1: nil},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.Type)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_MultipleTiers(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
			1: {rule(10, PricingTypePromotion, PricingMethodFixedPrice, 0, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 15000, result.Discount)
	assert.Equal(t, PricingTypePromotion, result.Type)
}

// ============================================================
// Scope-based resolution
// ============================================================

func TestResolver_Resolve_ProductScopeWins(t *testing.T) {
	catID := 5
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		scopes:     map[int]ProductScope{1: {CategoryID: &catID}},
		rules: map[int][]Rule{
			1: {
				rule(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 14000, 1, 0, true), // category match (score 2)
				rule(11, PricingTypePromotion, PricingMethodFixedPrice, 12000, 1, 0, true),    // product match (score 3)
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
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
	assert.Equal(t, PricingTypeSpecialPrice, results[0].Type)

	assert.Equal(t, 20000, results[1].UnitPrice)
	assert.Equal(t, PricingTypeNormal, results[1].Type)

	assert.Equal(t, 4000, results[2].UnitPrice)
	assert.Equal(t, PricingTypePromotion, results[2].Type)
}

func TestResolver_ResolveBatch_ProductNotFound(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules:      map[int][]Rule{},
	}
	resolver := NewResolver(repo)

	_, err := resolver.ResolveBatch(context.Background(), []ResolveItem{
		{ProductID: 1, Quantity: 1},
		{ProductID: 999, Quantity: 1},
	})
	require.ErrorIs(t, err, ErrProductNotFound)
}

func TestResolver_ResolveBatch_EmptyItems(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{},
		rules:      map[int][]Rule{},
	}
	resolver := NewResolver(repo)

	results, err := resolver.ResolveBatch(context.Background(), []ResolveItem{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestResolver_ResolveBatch_DuplicateProducts(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]Rule{
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
	assert.Equal(t, PricingTypeNormal, results[0].Type)

	assert.Equal(t, 10000, results[1].UnitPrice)
	assert.Equal(t, PricingTypeSpecialPrice, results[1].Type)
}

func TestResolver_ResolveBatch_CombinableRules(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000, 2: 50000},
		scopes:     map[int]ProductScope{1: {}, 2: {}},
		rules: map[int][]Rule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),   // combinable 10%
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountAmt, 5000, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil), // combinable 5000 off
			},
			2: {
				ruleAdvanced(20, PricingTypeSpecialPrice, PricingMethodFixedPrice, 30000, 1, nil, 5, true, false, nil, nil, nil, nil, nil, nil, nil), // non-combinable fixed
				ruleAdvanced(21, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),       // combinable 10%
			},
		},
	}
	resolver := NewResolver(repo)

	items := []ResolveItem{
		{ProductID: 1, Quantity: 1}, // all combinable → chain
		{ProductID: 2, Quantity: 1}, // non-combinable + combinable → best non + chain
	}

	results, err := resolver.ResolveBatch(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Product 1: 100000 → 5000 off (pri 2) = 95000 → 10% off (pri 1) = 85500
	assert.Equal(t, 85500, results[0].UnitPrice)
	assert.Equal(t, 14500, results[0].Discount)

	// Product 2: best non-combinable = 30000, then 10% combinable → 27000
	assert.Equal(t, 27000, results[1].UnitPrice)
	assert.Equal(t, 23000, results[1].Discount)
}

// ============================================================
// Stacking (allow_combine)
// ============================================================

func TestResolver_Resolve_NoStacking_BestSingleRule(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]Rule{
			1: {
				rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, 0, true),      // 10% off
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
		rules: map[int][]Rule{
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

func ruleAdvanced(id int, pType Type, method Method, value float64, minQty int, maxQty *int, priority int, active, combine bool, catID, brandID, custGroup, storeID *int, days []string, tFrom, tTo *string) Rule {
	pid := 1
	r := Rule{
		ID:              id,
		ProductID:       &pid,
		Type:            pType,
		Method:          method,
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
		rules: map[int][]Rule{
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
	assert.Equal(t, PricingTypePromotion, result.Type)
}

func TestResolver_Resolve_RecurrenceDayAllowed(t *testing.T) {
	today := strings.ToLower(time.Now().In(shared.JakartaLocation()).Weekday().String())
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]Rule{
			1: {ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 15, 1, nil, 0, true, false, nil, nil, nil, nil, []string{today}, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 85000, result.UnitPrice)
}

func TestResolver_Resolve_RecurrenceDayBlocked(t *testing.T) {
	today := strings.ToLower(time.Now().In(shared.JakartaLocation()).Weekday().String())
	blocked := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	withoutToday := make([]string, 0, len(blocked)-1)
	for _, d := range blocked {
		if d != today {
			withoutToday = append(withoutToday, d)
		}
	}
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]Rule{
			1: {ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 15, 1, nil, 0, true, false, nil, nil, nil, nil, withoutToday, nil, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 100000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.Type)
}

func TestResolver_Resolve_StoreFilterMatch(t *testing.T) {
	storeID := intPtr(1)
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
			1: {
				{ID: catRuleID, ProductID: nil, CategoryID: catID, Type: PricingTypePromotion, Method: PricingMethodDiscountPct, PricingValue: 15, Name: "cat-discount", MinimumQuantity: 1, Priority: 0, IsActive: true},
				{ID: brandRuleID, ProductID: nil, BrandID: brandID, Type: PricingTypePromotion, Method: PricingMethodDiscountPct, PricingValue: 5, Name: "brand-discount", MinimumQuantity: 1, Priority: 0, IsActive: true},
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),   // priority 1
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountAmt, 5000, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil), // priority 2
				ruleAdvanced(12, PricingTypePromotion, PricingMethodDiscountPct, 5, 1, nil, 3, true, true, nil, nil, nil, nil, nil, nil, nil),    // priority 3
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
		rules: map[int][]Rule{
			1: {
				ruleAdvanced(10, PricingTypeSpecialPrice, PricingMethodFixedPrice, 80000, 1, nil, 5, true, false, nil, nil, nil, nil, nil, nil, nil), // best non-combinable: 80000
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),       // combinable: 10% → 72000
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
		rules: map[int][]Rule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodFixedPrice, 150000, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil), // fixed → 150000
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, nil, 2, true, true, nil, nil, nil, nil, nil, nil, nil),    // 10% → 135000
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
		rules: map[int][]Rule{
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
		rules: map[int][]Rule{
			1: {
				ruleAdvanced(10, PricingTypePromotion, PricingMethodDiscountPct, 5, 1, nil, 10, true, true, nil, nil, nil, nil, nil, nil, nil),    // priority 10
				ruleAdvanced(11, PricingTypePromotion, PricingMethodDiscountPct, 15, 1, nil, 1, true, true, nil, nil, nil, nil, nil, nil, nil),    // priority 1
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
		rules: map[int][]Rule{
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

func TestResolver_Resolve_SingleCombinableRule(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 100000},
		rules: map[int][]Rule{
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

// ============================================================
// RT-01..RT-04 — ResolveSnapshot / ResolveSnapshotsBatch
// ============================================================

// snapshotRepo returns a resolver seeded with cost/tax + base price for product 1.
func snapshotRepo() *mockRepo {
	cost := 2500
	taxRate := 11.0
	return &mockRepo{
		basePrices: map[int]int{1: 3500, 2: 5000, 3: 7000},
		costTax: map[int]ProductCostTax{
			1: {Cost: cost, TaxRate: &taxRate, ProductName: "Product One"},
			2: {Cost: 3000, TaxRate: &taxRate, ProductName: "Product Two"},
			3: {Cost: 4000, ProductName: "Product Three"},
		},
	}
}

func TestResolver_ResolveSnapshot_NoRule(t *testing.T) {
	repo := snapshotRepo()
	resolver := NewResolver(repo)

	before := time.Now()
	snap, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	after := time.Now()

	assert.Equal(t, 1, snap.ProductID)
	assert.Equal(t, "Product One", snap.ProductName)
	assert.Equal(t, 3500, snap.UnitPrice)
	assert.Equal(t, 3500, snap.OriginalPrice)
	assert.Equal(t, 0, snap.Discount)
	assert.Equal(t, PricingTypeNormal, snap.Type)
	assert.Nil(t, snap.Rule)
	assert.Equal(t, 2500, snap.Cost)
	require.NotNil(t, snap.TaxRate)
	assert.Equal(t, 11.0, *snap.TaxRate)
	assert.False(t, snap.SnapshotAt.IsZero())
	assert.True(t, !snap.SnapshotAt.Before(before.Add(-time.Second)) && !snap.SnapshotAt.After(after.Add(time.Second)),
		"SnapshotAt should be near now, got %v", snap.SnapshotAt)
}

func TestResolver_ResolveSnapshot_PromoRule(t *testing.T) {
	repo := snapshotRepo()
	repo.rules = map[int][]Rule{
		1: {rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, 1, true)},
	}
	resolver := NewResolver(repo)

	snap, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.NoError(t, err)
	assert.Equal(t, 3150, snap.UnitPrice)
	assert.Equal(t, 3500, snap.OriginalPrice)
	assert.Equal(t, 350, snap.Discount)
	assert.Equal(t, PricingTypePromotion, snap.Type)
	assert.Equal(t, PricingMethodDiscountPct, snap.Method)
	require.NotNil(t, snap.Rule)
	assert.Equal(t, 10, snap.Rule.ID)
	assert.Equal(t, 2500, snap.Cost)
}

func TestResolver_ResolveSnapshot_QtyBelowMinimumUsesBase(t *testing.T) {
	repo := snapshotRepo()
	repo.rules = map[int][]Rule{
		1: {rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 5, 1, true)},
	}
	resolver := NewResolver(repo)

	snap, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 2})
	require.NoError(t, err)
	assert.Equal(t, 3500, snap.UnitPrice)
	assert.Equal(t, 0, snap.Discount)
	assert.Equal(t, PricingTypeNormal, snap.Type)

	snap, err = resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 5})
	require.NoError(t, err)
	assert.Equal(t, 3150, snap.UnitPrice)
	assert.Equal(t, PricingTypePromotion, snap.Type)
}

func TestResolver_ResolveSnapshot_Deterministic(t *testing.T) {
	repo := snapshotRepo()
	repo.rules = map[int][]Rule{
		1: {rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, 1, true)},
	}
	resolver := NewResolver(repo)

	a, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 3})
	require.NoError(t, err)
	b, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 3})
	require.NoError(t, err)

	assert.Equal(t, a.UnitPrice, b.UnitPrice)
	assert.Equal(t, a.OriginalPrice, b.OriginalPrice)
	assert.Equal(t, a.Discount, b.Discount)
	assert.Equal(t, a.Type, b.Type)
	assert.Equal(t, a.Cost, b.Cost)
	assert.Equal(t, a.TaxRate, b.TaxRate)
}

func TestResolver_ResolveSnapshotsBatch_OrderMatchesInput(t *testing.T) {
	repo := snapshotRepo()
	repo.rules = map[int][]Rule{
		1: {rule(10, PricingTypePromotion, PricingMethodDiscountPct, 10, 1, 1, true)},
	}
	resolver := NewResolver(repo)

	snaps, err := resolver.ResolveSnapshotsBatch(context.Background(), []ResolveItem{
		{ProductID: 3, Quantity: 1},
		{ProductID: 1, Quantity: 1},
		{ProductID: 2, Quantity: 1},
	})
	require.NoError(t, err)
	require.Len(t, snaps, 3)

	assert.Equal(t, 3, snaps[0].ProductID)
	assert.Equal(t, 7000, snaps[0].UnitPrice)
	assert.Equal(t, "Product Three", snaps[0].ProductName)
	assert.Equal(t, 4000, snaps[0].Cost)

	assert.Equal(t, 1, snaps[1].ProductID)
	assert.Equal(t, 3150, snaps[1].UnitPrice)
	assert.Equal(t, PricingTypePromotion, snaps[1].Type)

	assert.Equal(t, 2, snaps[2].ProductID)
	assert.Equal(t, 5000, snaps[2].UnitPrice)
}

func TestResolver_ResolveSnapshotsBatch_EmptyInput(t *testing.T) {
	resolver := NewResolver(snapshotRepo())
	snaps, err := resolver.ResolveSnapshotsBatch(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, snaps)
}

func TestResolver_ResolveSnapshot_ProductNotFound(t *testing.T) {
	repo := snapshotRepo()
	resolver := NewResolver(repo)

	_, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 999, Quantity: 1})
	require.ErrorIs(t, err, ErrProductNotFound)
}

func TestResolver_ResolveSnapshot_ErrorPropagates(t *testing.T) {
	// ResolveSnapshot resolves the price first, then fetches cost/tax. If the
	// cost/tax lookup fails (no entry for product 1), the error must propagate
	// and no partial snapshot may be returned.
	repo := &mockRepo{
		basePrices: map[int]int{1: 3500},
		costTax:    map[int]ProductCostTax{2: {Cost: 500}}, // missing product 1
	}
	resolver := NewResolver(repo)
	_, err := resolver.ResolveSnapshot(context.Background(), ResolveContext{ProductID: 1, Quantity: 1})
	require.ErrorIs(t, err, ErrProductNotFound)
}
