package category

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type adapter struct {
	repo *Repository
}

func NewAdapter(repo *Repository) importexportshared.Adapter {
	return &adapter{repo: repo}
}

func (a *adapter) ModuleName() string {
	return "categories"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ importexportshared.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	rowNum, _ := row["_row"].(int)
	slug, _ := row["Slug"].(string)
	desc, _ := row["Description"].(string)
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = parseBool(fmt.Sprintf("%v", v))
	}
	return CategoryImportRow{
		Row:         rowNum,
		Name:        name,
		Slug:        slug,
		Description: desc,
		IsActive:    isActive,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &categoryRepoAdapter{repo: a.repo}
}

type categoryRepoAdapter struct {
	repo *Repository
}

func (r *categoryRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]CategoryImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(CategoryImportRow)
	}
	result := r.repo.BulkUpsertCategories(ctx, rows)
	if len(result.Errors) > 0 {
		return result.Inserted, fmt.Errorf("category import errors: %v", strings.Join(result.Errors, "; "))
	}
	return result.Inserted, nil
}

func (r *categoryRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]CategoryImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(CategoryImportRow)
	}
	result := r.repo.BulkUpsertCategories(ctx, rows)
	if len(result.Errors) > 0 {
		return result.Updated, fmt.Errorf("category import errors: %v", strings.Join(result.Errors, "; "))
	}
	return result.Updated, nil
}

func (r *categoryRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	categories, err := r.repo.GetAllCategoriesForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(categories))
	for i, c := range categories {
		result[i] = map[string]interface{}{
			"Name":        c.Name,
			"Slug":        c.Slug,
			"Description": c.Description,
			"IsActive":    strconv.FormatBool(c.IsActive),
		}
	}
	return result, nil
}

func (r *categoryRepoAdapter) LoadReferences(_ context.Context, _ importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, nil
}

func parseBool(v string) bool {
	switch v {
	case "true", "1", "yes", "TRUE", "YES":
		return true
	}
	return false
}
