package sale

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== computeLineTotals ====================

func TestComputeLineTotals_TaxZero(t *testing.T) {
	rate := 0.0
	subtotal, dpp, tax := computeLineTotals(2, 5000, &rate)
	assert.Equal(t, 10000, subtotal)
	assert.Equal(t, 10000, dpp)
	assert.Equal(t, 0, tax)
}

func TestComputeLineTotals_TaxNil(t *testing.T) {
	subtotal, dpp, tax := computeLineTotals(1, 5000, nil)
	assert.Equal(t, 5000, subtotal)
	assert.Equal(t, 5000, dpp)
	assert.Equal(t, 0, tax)
}

func TestComputeLineTotals_Tax11(t *testing.T) {
	rate := 11.0
	subtotal, dpp, tax := computeLineTotals(1, 3500, &rate)
	assert.Equal(t, 3500, subtotal)
	assert.Equal(t, 3153, dpp) // round(3500*100/111)
	assert.Equal(t, 347, tax)  // 3500-3153
	assert.Equal(t, subtotal, dpp+tax)
}

func TestComputeLineTotals_Tax115(t *testing.T) {
	rate := 11.5
	subtotal, dpp, tax := computeLineTotals(2, 10000, &rate)
	assert.Equal(t, 20000, subtotal)
	assert.Equal(t, 17937, dpp) // round(20000*100/111.5)
	assert.Equal(t, 2063, tax)
	assert.Equal(t, subtotal, dpp+tax)
}

func TestComputeLineTotals_ZeroQuantity(t *testing.T) {
	rate := 11.0
	subtotal, dpp, tax := computeLineTotals(0, 3500, &rate)
	assert.Equal(t, 0, subtotal)
	assert.Equal(t, 0, dpp)
	assert.Equal(t, 0, tax)
}

// ==================== computeCartTotals ====================

func TestComputeCartTotals_MixedTaxedAndNonTaxed(t *testing.T) {
	items := []CartItem{
		{Subtotal: 3500, DPPAmount: 3153, TaxAmount: 347}, // taxed @ 11%
		{Subtotal: 5000, DPPAmount: 5000, TaxAmount: 0},   // non-taxed
		{Subtotal: 7000, DPPAmount: 7000, TaxAmount: 0},   // non-taxed
	}
	subtotal, discount, tax, total := computeCartTotals(items)
	assert.Equal(t, 15500, subtotal)
	assert.Equal(t, 0, discount)
	assert.Equal(t, 347, tax)
	assert.Equal(t, 15500, total)
}

func TestComputeCartTotals_Empty(t *testing.T) {
	subtotal, discount, tax, total := computeCartTotals(nil)
	assert.Equal(t, 0, subtotal)
	assert.Equal(t, 0, discount)
	assert.Equal(t, 0, tax)
	assert.Equal(t, 0, total)
}

// ==================== CartItem.ToSaleItem ====================

func TestCartItem_ToSaleItem_VerbatimSnapshot(t *testing.T) {
	ruleID := 7
	ruleName := "Promo 10%"
	ruleType := "promotion"
	pricingType := "promotion"
	taxClassID := 3
	taxRate := 11.0
	originalPrice := 3500

	ci := CartItem{
		ID:                1,
		CartSessionID:     9,
		ProductID:         42,
		ProductName:       "Produk Uji",
		Quantity:          3,
		UnitPrice:         3150,
		OriginalPrice:     originalPrice,
		Discount:          350,
		PricingRuleID:     &ruleID,
		PricingRuleName:   &ruleName,
		PricingRuleType:   &ruleType,
		Type:              &pricingType,
		Cost:              2500,
		TaxClassID:        &taxClassID,
		TaxRate:           &taxRate,
		SnapshotCreatedAt: "2026-07-31T10:00:00+07:00",
		Subtotal:          9450,
		DPPAmount:         8514,
		TaxAmount:         936,
	}

	si := ci.ToSaleItem()
	assert.Equal(t, 42, si.ProductID)
	assert.Equal(t, "Produk Uji", si.Name)
	assert.Equal(t, "Produk Uji", si.ProductName)
	assert.Equal(t, 3, si.Quantity)
	assert.Equal(t, 3150, si.UnitPrice)
	assert.Equal(t, 9450, si.Subtotal)
	assert.Equal(t, 8514, si.DPPAmount)
	assert.Equal(t, 936, si.TaxAmount)
	require.NotNil(t, si.PricingRuleID)
	assert.Equal(t, 7, *si.PricingRuleID)
	require.NotNil(t, si.PricingRuleName)
	assert.Equal(t, "Promo 10%", *si.PricingRuleName)
	require.NotNil(t, si.PricingRuleType)
	assert.Equal(t, "promotion", *si.PricingRuleType)
	require.NotNil(t, si.Type)
	assert.Equal(t, "promotion", *si.Type)
	require.NotNil(t, si.OriginalPrice)
	assert.Equal(t, 3500, *si.OriginalPrice)
	assert.Equal(t, 2500, si.Cost)
	require.NotNil(t, si.TaxClassID)
	assert.Equal(t, 3, *si.TaxClassID)
	require.NotNil(t, si.TaxRate)
	assert.Equal(t, 11.0, *si.TaxRate)
	assert.Equal(t, "2026-07-31T10:00:00+07:00", si.SnapshotCreatedAt)
}
