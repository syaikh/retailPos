package store

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
	return "stores"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ importexportshared.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	rowNum, _ := row["_row"].(int)
	address, _ := row["Address"].(string)
	phone, _ := row["Phone"].(string)
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = parseBool(fmt.Sprintf("%v", v))
	}
	return ImportRow{
		Row:      rowNum,
		Name:     name,
		Address:  address,
		Phone:    phone,
		IsActive: isActive,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &storeRepoAdapter{repo: a.repo}
}

type storeRepoAdapter struct {
	repo *Repository
}

func (r *storeRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	count := 0
	for _, e := range entities {
		row := e.(ImportRow)
		s := &Store{
			Name:     row.Name,
			Address:  row.Address,
			Phone:    row.Phone,
			IsActive: row.IsActive,
		}
		if err := r.repo.Create(ctx, s); err != nil {
			return count, fmt.Errorf("insert store at row %d: %w", row.Row, err)
		}
		count++
	}
	return count, nil
}

func (r *storeRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	count := 0
	for _, e := range entities {
		row := e.(ImportRow)
		existing, err := r.repo.GetByName(ctx, row.Name)
		if err != nil {
			// Not found — insert instead
			s := &Store{Name: row.Name, Address: row.Address, Phone: row.Phone, IsActive: row.IsActive}
			if err := r.repo.Create(ctx, s); err != nil {
				return count, fmt.Errorf("insert store at row %d: %w", row.Row, err)
			}
			count++
			continue
		}
		existing.Address = row.Address
		existing.Phone = row.Phone
		existing.IsActive = row.IsActive
		if err := r.repo.Update(ctx, existing); err != nil {
			return count, fmt.Errorf("update store at row %d: %w", row.Row, err)
		}
		count++
	}
	return count, nil
}

func (r *storeRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	stores, _, err := r.repo.GetAll(ctx, 10000, 0, "", nil)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(stores))
	for i, s := range stores {
		result[i] = map[string]interface{}{
			"Name":     s.Name,
			"Address":  s.Address,
			"Phone":    s.Phone,
			"IsActive": fmt.Sprintf("%v", s.IsActive),
		}
	}
	return result, nil
}

func (r *storeRepoAdapter) LoadReferences(_ context.Context, _ importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, nil
}

func parseBool(v string) bool {
	switch strings.ToLower(v) {
	case "true", "1", "yes":
		return true
	}
	return false
}
