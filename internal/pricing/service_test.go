package pricing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validRule() *PricingRule {
	pid := 1
	return &PricingRule{
		Name:            "Test Rule",
		ProductID:       &pid,
		PricingType:     PricingTypeSpecialPrice,
		PricingMethod:   PricingMethodFixedPrice,
		PricingValue:    50000,
		MinimumQuantity: 1,
		IsActive:        true,
	}
}

func TestValidateRule_Valid(t *testing.T) {
	err := validateRule(validRule())
	assert.NoError(t, err)
}

func TestValidateRule_EmptyName(t *testing.T) {
	r := validRule()
	r.Name = ""
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidateRule_NoTarget(t *testing.T) {
	r := validRule()
	r.ProductID = nil
	r.CategoryID = nil
	r.BrandID = nil
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "at least one of")
}

func TestValidateRule_InvalidPricingType(t *testing.T) {
	r := validRule()
	r.PricingType = "bogus"
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "invalid pricing_type")
}

func TestValidateRule_DefaultTypeRejected(t *testing.T) {
	r := validRule()
	r.PricingType = PricingTypeDefault
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "invalid pricing_type")
}

func TestValidateRule_InvalidPricingMethod(t *testing.T) {
	r := validRule()
	r.PricingMethod = "bogus"
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "invalid pricing_method")
}

func TestValidateRule_DiscountPercentOutOfRange(t *testing.T) {
	r := validRule()
	r.PricingMethod = PricingMethodDiscountPct
	r.PricingValue = 150
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "discount_percent")
}

func TestValidateRule_MaxQtyLessThanMinQty(t *testing.T) {
	r := validRule()
	minQty := 10
	maxQty := 5
	r.MinimumQuantity = minQty
	r.MaximumQuantity = &maxQty
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "maximum_quantity must be >= minimum_quantity")
}

func TestValidateRule_CategoryScopedValid(t *testing.T) {
	r := validRule()
	r.ProductID = nil
	r.CategoryID = intPtr(5)
	err := validateRule(r)
	assert.NoError(t, err)
}

func TestValidateRule_BrandScopedValid(t *testing.T) {
	r := validRule()
	r.ProductID = nil
	r.BrandID = intPtr(3)
	err := validateRule(r)
	assert.NoError(t, err)
}

func TestValidateRule_NegativeValue(t *testing.T) {
	r := validRule()
	r.PricingValue = -10
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "non-negative")
}
