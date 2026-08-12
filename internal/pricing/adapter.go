package pricing

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"retail-pos-system/internal/shared"
	importexportshared "retail-pos-system/internal/shared/importexport"
)

type adapter struct {
	repo *Repository
}

func NewAdapter(repo *Repository) importexportshared.Adapter {
	return &adapter{repo: repo}
}

func (a *adapter) ModuleName() string {
	return "pricing_rules"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ importexportshared.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	var productID *int
	if v, ok := row["ProductID"]; ok && v != nil {
		n := floatToInt(v)
		if n > 0 {
			productID = &n
		}
	}

	var categoryID *int
	if v, ok := row["CategoryID"]; ok && v != nil {
		n := floatToInt(v)
		if n > 0 {
			categoryID = &n
		}
	}

	var brandID *int
	if v, ok := row["BrandID"]; ok && v != nil {
		n := floatToInt(v)
		if n > 0 {
			brandID = &n
		}
	}

	if productID == nil && categoryID == nil && brandID == nil {
		return nil, fmt.Errorf("at least one of ProductID, CategoryID, or BrandID is required")
	}

	pricingType, _ := row["Type"].(string)
	if pricingType == "" {
		return nil, fmt.Errorf("Type is required")
	}

	pricingMethod, _ := row["Method"].(string)
	if pricingMethod == "" {
		pricingMethod = "fixed_price"
	}

	pricingValue := floatToFloat64(row["PricingValue"])
	if pricingValue == 0 {
		// Fallback to legacy "Price" column for backward compatibility
		pricingValue = floatToFloat64(row["Price"])
	}

	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	minQty := floatToInt(row["MinimumQuantity"])
	if minQty == 0 {
		minQty = 1
	}
	priority := floatToInt(row["Priority"])
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = fmt.Sprintf("%v", v) == "true"
	}

	rowNum, _ := row["_row"].(int)

	var effectiveFrom, effectiveUntil *time.Time
	if v, ok := row["EffectiveFrom"]; ok && v != nil {
		t, err := time.ParseInLocation("2006-01-02", fmt.Sprintf("%v", v), shared.JakartaLocation())
		if err == nil {
			effectiveFrom = &t
		}
	}
	if v, ok := row["EffectiveUntil"]; ok && v != nil {
		t, err := time.ParseInLocation("2006-01-02", fmt.Sprintf("%v", v), shared.JakartaLocation())
		if err == nil {
			effectiveUntil = &t
		}
	}

	return RuleImportRow{
		Row:             rowNum,
		ProductID:       productID,
		CategoryID:      categoryID,
		BrandID:         brandID,
		Type:            pricingType,
		Method:          pricingMethod,
		PricingValue:    pricingValue,
		Name:            name,
		MinimumQuantity: minQty,
		Priority:        priority,
		IsActive:        isActive,
		EffectiveFrom:   effectiveFrom,
		EffectiveUntil:  effectiveUntil,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &pricingRepoAdapter{repo: a.repo}
}

type pricingRepoAdapter struct {
	repo *Repository
}

func (r *pricingRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	payloads := make([]RuleImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(RuleImportRow)
		if row.MinimumQuantity < 1 {
			return len(payloads), fmt.Errorf("minimum_quantity must be >= 1 at row %d", row.Row)
		}
		payloads = append(payloads, RuleImportPayload{
			ProductID:       row.ProductID,
			CategoryID:      row.CategoryID,
			BrandID:         row.BrandID,
			Type:            row.Type,
			Method:          row.Method,
			PricingValue:    row.PricingValue,
			Name:            row.Name,
			MinimumQuantity: row.MinimumQuantity,
			Priority:        row.Priority,
			IsActive:        row.IsActive,
			EffectiveFrom:   row.EffectiveFrom,
			EffectiveUntil:  row.EffectiveUntil,
		})
	}
	return r.repo.BulkInsertPricingRules(ctx, payloads)
}

func (r *pricingRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	payloads := make([]RuleImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(RuleImportRow)
		payloads = append(payloads, RuleImportPayload{
			ProductID:       row.ProductID,
			CategoryID:      row.CategoryID,
			BrandID:         row.BrandID,
			Type:            row.Type,
			Method:          row.Method,
			PricingValue:    row.PricingValue,
			Name:            row.Name,
			MinimumQuantity: row.MinimumQuantity,
			Priority:        row.Priority,
			IsActive:        row.IsActive,
			EffectiveFrom:   row.EffectiveFrom,
			EffectiveUntil:  row.EffectiveUntil,
		})
	}
	return r.repo.BulkUpdatePricingRules(ctx, payloads)
}

func (r *pricingRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	rules, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		item := map[string]interface{}{
			"Type":            string(rule.Type),
			"Method":          string(rule.Method),
			"PricingValue":    rule.PricingValue,
			"Name":            rule.Name,
			"MinimumQuantity": rule.MinimumQuantity,
			"Priority":        rule.Priority,
			"IsActive":        rule.IsActive,
		}
		if rule.ProductID != nil {
			item["ProductID"] = *rule.ProductID
		}
		if rule.CategoryID != nil {
			item["CategoryID"] = *rule.CategoryID
		}
		if rule.BrandID != nil {
			item["BrandID"] = *rule.BrandID
		}
		if rule.EffectiveFrom != nil {
			item["EffectiveFrom"] = rule.EffectiveFrom.In(shared.JakartaLocation()).Format("2006-01-02")
		}
		if rule.EffectiveUntil != nil {
			item["EffectiveUntil"] = rule.EffectiveUntil.In(shared.JakartaLocation()).Format("2006-01-02")
		}
		result[i] = item
	}
	return result, nil
}

func (r *pricingRepoAdapter) LoadReferences(_ context.Context, _ importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return map[string][]importexportshared.ReferenceItem{}, nil
}

func floatToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(math.Round(val))
	case int:
		return val
	case string:
		n, _ := strconv.Atoi(val)
		return n
	case bool:
		if val {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func floatToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
