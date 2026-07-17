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
	return CustomerGroupImportRow{
		Row:         rowNum,
		Name:        name,
		Description: desc,
		IsActive:    isActive,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &cgRepoAdapter{repo: a.repo}
}

type cgRepoAdapter struct {
	repo *Repository
}

func (r *cgRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	count := 0
	for _, e := range entities {
		row := e.(CustomerGroupImportRow)
		cg := &CustomerGroup{
			Name:        row.Name,
			Description: row.Description,
			IsActive:    row.IsActive,
		}
		if err := r.repo.Create(ctx, cg); err != nil {
			return count, fmt.Errorf("insert customer group at row %d: %w", row.Row, err)
		}
		count++
	}
	return count, nil
}

func (r *cgRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	count := 0
	for _, e := range entities {
		row := e.(CustomerGroupImportRow)
		existing, err := r.repo.GetByName(ctx, row.Name)
		if err != nil {
			cg := &CustomerGroup{Name: row.Name, Description: row.Description, IsActive: row.IsActive}
			if err := r.repo.Create(ctx, cg); err != nil {
				return count, fmt.Errorf("insert customer group at row %d: %w", row.Row, err)
			}
			count++
			continue
		}
		existing.Description = row.Description
		existing.IsActive = row.IsActive
		if err := r.repo.Update(ctx, existing); err != nil {
			return count, fmt.Errorf("update customer group at row %d: %w", row.Row, err)
		}
		count++
	}
	return count, nil
}

func (r *cgRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	groups, _, err := r.repo.GetAll(ctx, 10000, 0, "", nil)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(groups))
	for i, g := range groups {
		result[i] = map[string]interface{}{
			"Name":        g.Name,
			"Description": g.Description,
			"IsActive":    fmt.Sprintf("%v", g.IsActive),
		}
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
