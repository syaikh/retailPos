package supplier

import (
	"context"
	"fmt"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type adapter struct {
	repo *Repository
}

func NewAdapter(repo *Repository) importexportshared.Adapter {
	return &adapter{repo: repo}
}

func (a *adapter) ModuleName() string {
	return "suppliers"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ importexportshared.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	code, _ := row["Code"].(string)
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}
	name, _ := row["Name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	rowNum, _ := row["_row"].(int)
	contactName, _ := row["ContactName"].(string)
	phone, _ := row["Phone"].(string)
	email, _ := row["Email"].(string)
	address, _ := row["Address"].(string)
	notes, _ := row["Notes"].(string)
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = fmt.Sprintf("%v", v) == "true"
	}

	return ImportRow{
		Row:         rowNum,
		Code:        code,
		Name:        name,
		ContactName: contactName,
		Phone:       phone,
		Email:       email,
		Address:     address,
		Notes:       notes,
		IsActive:    isActive,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &supplierRepoAdapter{repo: a.repo}
}

type supplierRepoAdapter struct {
	repo *Repository
}

func (r *supplierRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	payloads := make([]ImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(ImportRow)
		payloads = append(payloads, ImportPayload{
			Code:        row.Code,
			Name:        row.Name,
			ContactName: strPtr(row.ContactName),
			Phone:       strPtr(row.Phone),
			Email:       strPtr(row.Email),
			Address:     strPtr(row.Address),
			Notes:       strPtr(row.Notes),
			IsActive:    row.IsActive,
		})
	}
	return r.repo.BulkInsertSuppliers(ctx, payloads)
}

func (r *supplierRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	payloads := make([]ImportPayload, 0, len(entities))
	for _, e := range entities {
		row := e.(ImportRow)
		payloads = append(payloads, ImportPayload{
			Code:        row.Code,
			Name:        row.Name,
			ContactName: strPtr(row.ContactName),
			Phone:       strPtr(row.Phone),
			Email:       strPtr(row.Email),
			Address:     strPtr(row.Address),
			Notes:       strPtr(row.Notes),
			IsActive:    row.IsActive,
		})
	}
	return r.repo.BulkUpdateSuppliers(ctx, payloads)
}

func (r *supplierRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	suppliers, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(suppliers))
	for i, s := range suppliers {
		item := map[string]interface{}{
			"Code":        s.Code,
			"Name":        s.Name,
			"ContactName": nilStr(s.ContactName),
			"Phone":       nilStr(s.Phone),
			"Email":       nilStr(s.Email),
			"Address":     nilStr(s.Address),
			"Notes":       nilStr(s.Notes),
			"IsActive":    s.IsActive,
		}
		result[i] = item
	}
	return result, nil
}

func (r *supplierRepoAdapter) LoadReferences(_ context.Context, _ importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return map[string][]importexportshared.ReferenceItem{}, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
