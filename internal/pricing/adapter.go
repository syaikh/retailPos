package pricing

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

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
	productID := floatToInt(row["ProductID"])
	if productID == 0 {
		return nil, fmt.Errorf("ProductID is required")
	}
	pricingType, _ := row["PricingType"].(string)
	if pricingType == "" {
		return nil, fmt.Errorf("PricingType is required")
	}
	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	price := floatToInt(row["Price"])
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
		t, err := time.Parse("2006-01-02", fmt.Sprintf("%v", v))
		if err == nil {
			effectiveFrom = &t
		}
	}
	if v, ok := row["EffectiveUntil"]; ok && v != nil {
		t, err := time.Parse("2006-01-02", fmt.Sprintf("%v", v))
		if err == nil {
			effectiveUntil = &t
		}
	}

	return PricingRuleImportRow{
		Row:             rowNum,
		ProductID:       productID,
		PricingType:     pricingType,
		Name:            name,
		Price:           price,
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
	payloads := make([]PricingRuleImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(PricingRuleImportRow)
		if row.Price < 0 {
			return len(payloads), fmt.Errorf("price must not be negative at row %d", row.Row)
		}
		if row.MinimumQuantity < 1 {
			return len(payloads), fmt.Errorf("minimum_quantity must be >= 1 at row %d", row.Row)
		}
		payloads = append(payloads, PricingRuleImportPayload{
			ProductID:       row.ProductID,
			PricingType:     row.PricingType,
			Name:            row.Name,
			Price:           row.Price,
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
	payloads := make([]PricingRuleImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(PricingRuleImportRow)
		if row.Price < 0 {
			return len(payloads), fmt.Errorf("price must not be negative at row %d", row.Row)
		}
		payloads = append(payloads, PricingRuleImportPayload{
			ProductID:       row.ProductID,
			PricingType:     row.PricingType,
			Name:            row.Name,
			Price:           row.Price,
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
			"ProductID":       rule.ProductID,
			"PricingType":     string(rule.PricingType),
			"Name":            rule.Name,
			"Price":           rule.Price,
			"MinimumQuantity": rule.MinimumQuantity,
			"Priority":        rule.Priority,
			"IsActive":        rule.IsActive,
		}
		if rule.EffectiveFrom != nil {
			item["EffectiveFrom"] = rule.EffectiveFrom.Format("2006-01-02")
		}
		if rule.EffectiveUntil != nil {
			item["EffectiveUntil"] = rule.EffectiveUntil.Format("2006-01-02")
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
