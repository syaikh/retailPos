package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

func testSchema() importexportshared.ModuleSchema {
	return importexportshared.ModuleSchema{}
}

func TestFloatToInt(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		assert.Equal(t, 42, floatToInt(42.0))
	})

	t.Run("float64 rounds", func(t *testing.T) {
		assert.Equal(t, 43, floatToInt(42.6))
	})

	t.Run("float64 rounds down", func(t *testing.T) {
		assert.Equal(t, 42, floatToInt(42.4))
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 10, floatToInt(10))
	})

	t.Run("string valid", func(t *testing.T) {
		assert.Equal(t, 55, floatToInt("55"))
	})

	t.Run("string invalid", func(t *testing.T) {
		assert.Equal(t, 0, floatToInt("abc"))
	})

	t.Run("bool true", func(t *testing.T) {
		assert.Equal(t, 1, floatToInt(true))
	})

	t.Run("bool false", func(t *testing.T) {
		assert.Equal(t, 0, floatToInt(false))
	})

	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, 0, floatToInt(nil))
	})
}

func TestFloatToFloat64(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		assert.Equal(t, 3.14, floatToFloat64(3.14))
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 7.0, floatToFloat64(7))
	})

	t.Run("string valid", func(t *testing.T) {
		assert.Equal(t, 9.5, floatToFloat64("9.5"))
	})

	t.Run("string invalid", func(t *testing.T) {
		assert.Equal(t, 0.0, floatToFloat64("xyz"))
	})

	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, 0.0, floatToFloat64(nil))
	})
}

func TestAdapter_MapToEntity(t *testing.T) {
	a := NewAdapter(nil)
	schema := testSchema()
	ctx := context.Background()

	t.Run("product id only", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":       float64(10),
			"PricingType":     "special_price",
			"PricingMethod":   "fixed_price",
			"PricingValue":    float64(5000),
			"Name":            "Test Rule",
			"MinimumQuantity": float64(2),
			"Priority":        float64(1),
			"_row":            1,
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult, ok := result.(PricingRuleImportRow)
		require.True(t, ok)
		assert.NotNil(t, rowResult.ProductID)
		assert.Equal(t, 10, *rowResult.ProductID)
		assert.Nil(t, rowResult.CategoryID)
		assert.Nil(t, rowResult.BrandID)
	})

	t.Run("category id", func(t *testing.T) {
		row := map[string]interface{}{
			"CategoryID":      float64(5),
			"PricingType":     "promotion",
			"PricingMethod":   "discount_percent",
			"PricingValue":    float64(10),
			"Name":            "Cat Rule",
			"MinimumQuantity": float64(1),
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.NotNil(t, rowResult.CategoryID)
		assert.Equal(t, 5, *rowResult.CategoryID)
	})

	t.Run("brand id", func(t *testing.T) {
		row := map[string]interface{}{
			"BrandID":         float64(3),
			"PricingType":     "promotion",
			"PricingMethod":   "fixed_price",
			"PricingValue":    float64(8000),
			"Name":            "Brand Rule",
			"MinimumQuantity": float64(1),
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.NotNil(t, rowResult.BrandID)
		assert.Equal(t, 3, *rowResult.BrandID)
	})

	t.Run("no target returns error", func(t *testing.T) {
		row := map[string]interface{}{
			"PricingType":   "promotion",
			"PricingMethod": "fixed_price",
			"PricingValue":  float64(8000),
			"Name":          "No Target Rule",
		}
		_, err := a.MapToEntity(ctx, schema, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of ProductID, CategoryID, or BrandID is required")
	})

	t.Run("missing pricing type returns error", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID": float64(10),
			"Name":      "No Type",
		}
		_, err := a.MapToEntity(ctx, schema, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PricingType is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":   float64(10),
			"PricingType": "promotion",
		}
		_, err := a.MapToEntity(ctx, schema, row)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Name is required")
	})

	t.Run("missing method defaults to fixed_price", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":   float64(10),
			"PricingType": "promotion",
			"Name":        "Default Method",
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.Equal(t, "fixed_price", rowResult.PricingMethod)
	})

	t.Run("missing min qty defaults to 1", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":   float64(10),
			"PricingType": "promotion",
			"PricingMethod": "fixed_price",
			"Name":        "Default MinQty",
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.Equal(t, 1, rowResult.MinimumQuantity)
	})

	t.Run("effective dates parsed", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":       float64(10),
			"PricingType":     "promotion",
			"PricingMethod":   "fixed_price",
			"PricingValue":    float64(5000),
			"Name":            "With Dates",
			"MinimumQuantity": float64(1),
			"EffectiveFrom":   "2025-01-01",
			"EffectiveUntil":  "2025-12-31",
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.NotNil(t, rowResult.EffectiveFrom)
		assert.NotNil(t, rowResult.EffectiveUntil)
	})

	t.Run("invalid effective date ignored", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":       float64(10),
			"PricingType":     "promotion",
			"PricingMethod":   "fixed_price",
			"PricingValue":    float64(5000),
			"Name":            "Bad Dates",
			"MinimumQuantity": float64(1),
			"EffectiveFrom":   "not-a-date",
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.Nil(t, rowResult.EffectiveFrom)
	})

	t.Run("is_active from string", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":       float64(10),
			"PricingType":     "promotion",
			"PricingMethod":   "fixed_price",
			"PricingValue":    float64(5000),
			"Name":            "Active Rule",
			"MinimumQuantity": float64(1),
			"IsActive":        "false",
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.False(t, rowResult.IsActive)
	})

	t.Run("is_active from bool", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":       float64(10),
			"PricingType":     "promotion",
			"PricingMethod":   "fixed_price",
			"PricingValue":    float64(5000),
			"Name":            "Active Rule Bool",
			"MinimumQuantity": float64(1),
			"IsActive":        true,
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.True(t, rowResult.IsActive)
	})

	t.Run("fallback to Price when PricingValue zero", func(t *testing.T) {
		row := map[string]interface{}{
			"ProductID":       float64(10),
			"PricingType":     "promotion",
			"PricingMethod":   "fixed_price",
			"Price":           float64(7500),
			"Name":            "Legacy Price Rule",
			"MinimumQuantity": float64(1),
		}
		result, err := a.MapToEntity(ctx, schema, row)
		require.NoError(t, err)
		rowResult := result.(PricingRuleImportRow)
		assert.Equal(t, 7500.0, rowResult.PricingValue)
	})
}

func TestAdapter_ModuleName(t *testing.T) {
	a := NewAdapter(nil)
	assert.Equal(t, "pricing_rules", a.ModuleName())
}

func TestAdapter_ValidateBusiness(t *testing.T) {
	a := NewAdapter(nil)
	schema := testSchema()
	assert.Nil(t, a.ValidateBusiness(context.Background(), schema, nil))
}

func TestAdapter_LoadReferences(t *testing.T) {
	a := NewAdapter(nil)
	schema := testSchema()
	repoActions := a.Repository()
	refs, err := repoActions.LoadReferences(context.Background(), schema)
	assert.NoError(t, err)
	assert.Empty(t, refs)
}

func TestAdapter_Repository_Insert(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	ctx := t.Context()

	productID := insertTestProduct(t, ctx, "ADP-INS-"+time.Now().Format("0102150405"), "Adapter Insert Product", 15000)

	t.Run("insert single", func(t *testing.T) {
		entities := []interface{}{
			PricingRuleImportRow{
				Row:             1,
				ProductID:       &productID,
				PricingType:     string(PricingTypePromotion),
				PricingMethod:   string(PricingMethodFixedPrice),
				PricingValue:    10000,
				Name:            "Adapter Insert 1 " + time.Now().Format("0102150405.000"),
				MinimumQuantity: 1,
				IsActive:        true,
			},
		}
		count, err := a.Repository().Insert(ctx, entities)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("minimum quantity zero fails", func(t *testing.T) {
		entities := []interface{}{
			PricingRuleImportRow{
				Row:             2,
				ProductID:       &productID,
				PricingType:     string(PricingTypePromotion),
				PricingMethod:   string(PricingMethodFixedPrice),
				PricingValue:    5000,
				Name:            "Bad MinQty",
				MinimumQuantity: 0,
				IsActive:        true,
			},
		}
		_, err := a.Repository().Insert(ctx, entities)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "minimum_quantity must be >= 1")
	})

	t.Run("empty", func(t *testing.T) {
		count, err := a.Repository().Insert(ctx, []interface{}{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestAdapter_Repository_Update(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	ctx := t.Context()

	productID := insertTestProduct(t, ctx, "ADP-UPD-"+time.Now().Format("0102150405"), "Adapter Update Product", 15000)

	t.Run("insert via update (not found, returns 0)", func(t *testing.T) {
		entities := []interface{}{
			PricingRuleImportRow{
				Row:             1,
				ProductID:       &productID,
				PricingType:     string(PricingTypePromotion),
				PricingMethod:   string(PricingMethodFixedPrice),
				PricingValue:    8000,
				Name:            "Adapter Update New " + time.Now().Format("0102150405.000"),
				MinimumQuantity: 1,
				IsActive:        true,
			},
		}
		count, err := a.Repository().Update(ctx, entities)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("update existing", func(t *testing.T) {
		rule := &PricingRule{
			ProductID:       &productID,
			PricingType:     PricingTypePromotion,
			PricingMethod:   PricingMethodFixedPrice,
			PricingValue:    15000,
			Name:            "Will Be Updated " + time.Now().Format("0102150405.000"),
			MinimumQuantity: 1,
			IsActive:        true,
		}
		require.NoError(t, repo.Create(ctx, rule))

		entities := []interface{}{
			PricingRuleImportRow{
				Row:             2,
				ProductID:       &productID,
				PricingType:     string(PricingTypePromotion),
				PricingMethod:   string(PricingMethodFixedPrice),
				PricingValue:    12000,
				Name:            rule.Name,
				MinimumQuantity: 1,
				IsActive:        true,
			},
		}
		count, err := a.Repository().Update(ctx, entities)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		got, err := repo.GetByID(ctx, rule.ID)
		require.NoError(t, err)
		assert.Equal(t, 12000.0, got.PricingValue)
	})

	t.Run("empty", func(t *testing.T) {
		count, err := a.Repository().Update(ctx, []interface{}{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestAdapter_Repository_ExportData(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	a := NewAdapter(repo)
	schema := testSchema()
	ctx := t.Context()

	productID := insertTestProduct(t, ctx, "ADP-EXP-"+time.Now().Format("0102150405"), "Adapter Export Product", 15000)
	now := time.Now()
	rule := &PricingRule{
		ProductID:       &productID,
		PricingType:     PricingTypePromotion,
		PricingMethod:   PricingMethodFixedPrice,
		PricingValue:    10000,
		Name:            "Export Adapter Rule " + time.Now().Format("0102150405.000"),
		MinimumQuantity: 1,
		IsActive:        true,
		EffectiveFrom:   &now,
		EffectiveUntil:  &now,
	}
	require.NoError(t, repo.Create(ctx, rule))

	result, err := a.Repository().ExportData(ctx, schema)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 1)

	found := false
	for _, item := range result {
		if item["Name"] == rule.Name {
			found = true
			assert.Equal(t, "promotion", item["PricingType"])
			assert.NotNil(t, item["ProductID"])
			assert.Nil(t, item["CategoryID"])
			assert.Nil(t, item["BrandID"])
			assert.NotNil(t, item["EffectiveFrom"])
			assert.NotNil(t, item["EffectiveUntil"])
			break
		}
	}
	assert.True(t, found, "expected to find exported rule")
}
