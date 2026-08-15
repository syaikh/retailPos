package pricing

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRule() *Rule {
	pid := 1
	return &Rule{
		Name:            "Test Rule",
		ProductID:       &pid,
		Type:            PricingTypeSpecialPrice,
		Method:          PricingMethodFixedPrice,
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
	r.Type = "bogus"
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "invalid pricing_type")
}

func TestValidateRule_DefaultTypeRejected(t *testing.T) {
	r := validRule()
	r.Type = PricingTypeNormal
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "invalid pricing_type")
}

func TestValidateRule_InvalidPricingMethod(t *testing.T) {
	r := validRule()
	r.Method = "bogus"
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "invalid pricing_method")
}

func TestValidateRule_DiscountPercentOutOfRange(t *testing.T) {
	r := validRule()
	r.Method = PricingMethodDiscountPct
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

func TestValidateRule_MarkupPercentOutOfRange(t *testing.T) {
	r := validRule()
	r.Method = PricingMethodMarkupPct
	r.PricingValue = 600
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "markup_percent")
}

func TestValidateRule_MarkupPercentValid(t *testing.T) {
	r := validRule()
	r.Method = PricingMethodMarkupPct
	r.PricingValue = 500
	err := validateRule(r)
	assert.NoError(t, err)
}

func TestValidateRule_MinimumQuantityZero(t *testing.T) {
	r := validRule()
	r.MinimumQuantity = 0
	err := validateRule(r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRule))
	assert.Contains(t, err.Error(), "minimum_quantity must be at least 1")
}

func TestService_GetByID(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "SVC-GID-"+time.Now().Format("0102150405"), "Service GetByID Product", 15000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypePromotion,
		Method:          PricingMethodFixedPrice,
		PricingValue:    12000,
		Name:            "SVC GetByID Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(ctx, rule))

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, rule.Name, got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(ctx, 999999)
		assert.Error(t, err)
	})
}

func TestService_GetByProductID(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "SVC-GBPID-"+time.Now().Format("0102150405"), "Service GetByProductID Product", 15000)
	rule := &Rule{
		ProductID:       &productID,
		Type:            PricingTypeSpecialPrice,
		Method:          PricingMethodDiscountAmt,
		PricingValue:    3000,
		Name:            "SVC GetByProductID Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
	}
	require.NoError(t, repo.Create(ctx, rule))

	t.Run("found", func(t *testing.T) {
		rules, err := svc.GetByProductID(ctx, productID)
		require.NoError(t, err)
		assert.NotEmpty(t, rules)
	})

	t.Run("empty", func(t *testing.T) {
		rules, err := svc.GetByProductID(ctx, -99999)
		require.NoError(t, err)
		assert.Empty(t, rules)
	})
}

func TestService_Delete(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "SVC-DEL-"+time.Now().Format("0102150405"), "Service Delete Product", 15000)

	t.Run("success", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            "SVC Delete Rule " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))
		err := svc.Delete(ctx, rule.ID)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, 999999)
		assert.NoError(t, err)
	})
}

func TestService_Update(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "SVC-UPD-"+time.Now().Format("0102150405"), "Service Update Product", 15000)

	t.Run("success", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    15000,
			Name:            "SVC Update Rule " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))
		rule.Name = "SVC Updated Rule"
		rule.PricingValue = 12000
		err := svc.Update(ctx, rule)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		rule := &Rule{
			ID:              999999,
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            "Non-existent Update",
			MinimumQuantity: 1,
			IsActive:        true,
		}
		err := svc.Update(ctx, rule)
		assert.NoError(t, err)
	})

	t.Run("validation fails", func(t *testing.T) {
		rule := &Rule{
			ID:              999999,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            "",
			MinimumQuantity: 1,
			IsActive:        true,
		}
		err := svc.Update(ctx, rule)
		assert.Error(t, err)
	})
}

func TestService_Create(t *testing.T) {
	skipIfNoDB(t)
	repo := newWiredRepo()
	svc := NewService(repo)
	ctx := t.Context()

	productID := insertTestProduct(ctx, t, "SVC-CRT-"+time.Now().Format("0102150405"), "Service Create Product", 15000)

	t.Run("success", func(t *testing.T) {
		rule := &Rule{
			ProductID:       &productID,
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            "SVC Create Rule " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
		}
		err := svc.Create(ctx, rule)
		assert.NoError(t, err)
		assert.Greater(t, rule.ID, 0)
	})

	t.Run("validation fails", func(t *testing.T) {
		rule := &Rule{
			Type:            PricingTypePromotion,
			Method:          PricingMethodFixedPrice,
			PricingValue:    10000,
			Name:            "",
			MinimumQuantity: 1,
			IsActive:        true,
		}
		err := svc.Create(ctx, rule)
		assert.Error(t, err)
	})
}
