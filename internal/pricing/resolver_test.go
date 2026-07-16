package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepo is a test double for the pricing repository.
// It will be replaced by the real repository in TASK-009.
type mockRepo struct {
	basePrices map[int]int
	rules      map[int][]PricingRule
}

func (m *mockRepo) GetBasePrice(_ context.Context, productID int) (int, error) {
	price, ok := m.basePrices[productID]
	if !ok {
		return 0, ErrProductNotFound
	}
	return price, nil
}

func (m *mockRepo) GetActiveRules(_ context.Context, productID int, _ time.Time) ([]PricingRule, error) {
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

func (m *mockRepo) GetActiveRulesBatch(_ context.Context, productIDs []int, _ time.Time) (map[int][]PricingRule, error) {
	result := make(map[int][]PricingRule, len(productIDs))
	for _, pid := range productIDs {
		if rules, ok := m.rules[pid]; ok {
			result[pid] = rules
		}
	}
	return result, nil
}

// --- Helper to build PricingRules for tests ---

func rule(id int, pType PricingType, price, minQty, priority int, active bool) PricingRule {
	return PricingRule{
		ID:              id,
		ProductID:       1,
		PricingType:     pType,
		Name:            string(pType),
		Price:           price,
		MinimumQuantity: minQty,
		Priority:        priority,
		IsActive:        active,
	}
}

func ruleWithDates(id int, pType PricingType, price, minQty, priority int, active bool, from, until *time.Time) PricingRule {
	r := rule(id, pType, price, minQty, priority, active)
	r.EffectiveFrom = from
	r.EffectiveUntil = until
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

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 0, result.Discount)
	assert.Equal(t, PricingTypeNormal, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_ProductNotFound(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{},
		rules:      map[int][]PricingRule{},
	}
	resolver := NewResolver(repo)

	_, err := resolver.Resolve(context.Background(), 999, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestResolver_Resolve_SingleDiscountRule(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypeDiscount, 12000, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 12000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 3000, result.Discount)
	assert.Equal(t, PricingTypeDiscount, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_WholesaleBelowMinimum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypeWholesale, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_WholesaleMeetsMinimum(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypeWholesale, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 3)
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 5000, result.Discount)
	assert.Equal(t, PricingTypeWholesale, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 10, result.Rule.ID)
}

func TestResolver_Resolve_PriorityWins(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypeDiscount, 12000, 1, 0, true),
				rule(11, PricingTypeWholesale, 10000, 1, 1, true),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 5)
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, PricingTypeWholesale, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_PriceTiebreak(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypeDiscount, 12000, 1, 0, true),
				rule(11, PricingTypeWholesale, 11000, 1, 0, true),
			},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 11000, result.UnitPrice)
	assert.Equal(t, PricingTypeWholesale, result.PricingType)
	require.NotNil(t, result.Rule)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_ExpiredRuleFallsBack(t *testing.T) {
	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {ruleWithDates(10, PricingTypeDiscount, 12000, 1, 0, true, nil, &past)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_InactiveRuleFallsBack(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypeDiscount, 12000, 1, 0, false)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_FutureRuleFallsBack(t *testing.T) {
	future := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {ruleWithDates(10, PricingTypeDiscount, 12000, 1, 0, true, &future, nil)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_NoRulesForProduct(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules:      map[int][]PricingRule{1: nil},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 15000, result.UnitPrice)
	assert.Equal(t, PricingTypeNormal, result.PricingType)
	assert.Nil(t, result.Rule)
}

func TestResolver_Resolve_MultipleWholesaleTiers(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {
				rule(10, PricingTypeWholesale, 12000, 3, 0, true),
				rule(11, PricingTypeWholesale, 10000, 5, 0, true),
			},
		},
	}
	resolver := NewResolver(repo)

	// qty=4: only first tier matches
	result, err := resolver.Resolve(context.Background(), 1, 4)
	require.NoError(t, err)
	assert.Equal(t, 12000, result.UnitPrice)

	// qty=6: both match, lower price wins (same priority)
	result, err = resolver.Resolve(context.Background(), 1, 6)
	require.NoError(t, err)
	assert.Equal(t, 10000, result.UnitPrice)
	assert.Equal(t, 11, result.Rule.ID)
}

func TestResolver_Resolve_RuleWithZeroPrice(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypePromotion, 0, 1, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	result, err := resolver.Resolve(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, result.UnitPrice)
	assert.Equal(t, 15000, result.OriginalPrice)
	assert.Equal(t, 15000, result.Discount)
	assert.Equal(t, PricingTypePromotion, result.PricingType)
}

// ============================================================
// ResolveBatch — batch resolution for POS cart
// ============================================================

func TestResolver_ResolveBatch_MixedProducts(t *testing.T) {
	repo := &mockRepo{
		basePrices: map[int]int{1: 15000, 2: 20000, 3: 5000},
		rules: map[int][]PricingRule{
			1: {rule(10, PricingTypeWholesale, 10000, 3, 0, true)},
			2: {},
			3: {rule(20, PricingTypeDiscount, 4000, 1, 0, true)},
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

	// Product 1: wholesale applies (qty=5 >= min_qty=3)
	assert.Equal(t, 10000, results[0].UnitPrice)
	assert.Equal(t, PricingTypeWholesale, results[0].PricingType)

	// Product 2: no rules, base price
	assert.Equal(t, 20000, results[1].UnitPrice)
	assert.Equal(t, PricingTypeNormal, results[1].PricingType)

	// Product 3: discount applies
	assert.Equal(t, 4000, results[2].UnitPrice)
	assert.Equal(t, PricingTypeDiscount, results[2].PricingType)
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
			1: {rule(10, PricingTypeWholesale, 10000, 3, 0, true)},
		},
	}
	resolver := NewResolver(repo)

	// Same product with different quantities — batch should handle both
	items := []ResolveItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 1, Quantity: 5},
	}

	results, err := resolver.ResolveBatch(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// First: qty=2, wholesale doesn't apply
	assert.Equal(t, 15000, results[0].UnitPrice)
	assert.Equal(t, PricingTypeNormal, results[0].PricingType)

	// Second: qty=5, wholesale applies
	assert.Equal(t, 10000, results[1].UnitPrice)
	assert.Equal(t, PricingTypeWholesale, results[1].PricingType)
}
