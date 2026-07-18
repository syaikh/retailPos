package customergroup

import (
	"context"
	"fmt"
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
	return "customer_groups"
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
	desc, _ := row["Description"].(string)
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = parseBool(fmt.Sprintf("%v", v))
	}
	color, _ := row["Color"].(string)
	return CustomerGroupImportRow{
		Row:         rowNum,
		Name:        name,
		Description: desc,
		IsActive:    isActive,
		Color:       color,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &cgRepoAdapter{repo: a.repo}
}

type cgRepoAdapter struct {
	repo *Repository
}

func (r *cgRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]CustomerGroupImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(CustomerGroupImportRow)
	}
	result := r.repo.BulkUpsertCustomerGroups(ctx, rows)
	if len(result.Errors) > 0 {
		return result.Inserted, fmt.Errorf("customer group import errors: %s", strings.Join(result.Errors, "; "))
	}
	return result.Inserted, nil
}

func (r *cgRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]CustomerGroupImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(CustomerGroupImportRow)
	}
	result := r.repo.BulkUpsertCustomerGroups(ctx, rows)
	if len(result.Errors) > 0 {
		return result.Updated, fmt.Errorf("customer group import errors: %s", strings.Join(result.Errors, "; "))
	}
	return result.Updated, nil
}

func (r *cgRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	groups, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(groups))
	for i, g := range groups {
		item := map[string]interface{}{
			"Name":        g.Name,
			"Description": g.Description,
			"IsActive":    fmt.Sprintf("%v", g.IsActive),
		}
		if g.Color != "" {
			item["Color"] = g.Color
		}
		result[i] = item
	}
	return result, nil
}

func (r *cgRepoAdapter) LoadReferences(_ context.Context, _ importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, nil
}

func parseBool(v string) bool {
	switch strings.ToLower(v) {
	case "true", "1", "yes":
		return true
	}
	return false
}
