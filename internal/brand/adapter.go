package brand

import (
	"context"
	"fmt"
	"strconv"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type adapter struct {
	repo *Repository
}

func NewAdapter(repo *Repository) importexport.Adapter {
	return &adapter{repo: repo}
}

func (a *adapter) ModuleName() string {
	return "brands"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ schema.ModuleSchema, _ []map[string]interface{}) []importexport.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	rowNum, _ := row["_row"].(int)
	desc, _ := row["Description"].(string)
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = parseBool(fmt.Sprintf("%v", v))
	}
	return BrandImportRow{
		Row:         rowNum,
		Name:        name,
		Description: desc,
		IsActive:    isActive,
	}, nil
}

func (a *adapter) Repository() importexport.RepositoryActions {
	return &brandRepoAdapter{repo: a.repo}
}

type brandRepoAdapter struct {
	repo *Repository
}

func (r *brandRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]BrandImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(BrandImportRow)
	}
	result := r.repo.BulkUpsert(ctx, rows)
	return result.Inserted, nil
}

func (r *brandRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]BrandImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(BrandImportRow)
	}
	result := r.repo.BulkUpsert(ctx, rows)
	return result.Updated, nil
}

func (r *brandRepoAdapter) ExportData(ctx context.Context, _ schema.ModuleSchema) ([]map[string]interface{}, error) {
	brands, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(brands))
	for i, b := range brands {
		result[i] = map[string]interface{}{
			"Name":        b.Name,
			"Description": b.Description,
			"IsActive":    strconv.FormatBool(b.IsActive),
		}
	}
	return result, nil
}

func (r *brandRepoAdapter) LoadReferences(_ context.Context, _ schema.ModuleSchema) (map[string][]importexport.ReferenceItem, error) {
	return nil, nil
}

func parseBool(v string) bool {
	switch v {
	case "true", "1", "yes", "TRUE", "YES":
		return true
	}
	return false
}
