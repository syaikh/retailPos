package uom

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
	return "uoms"
}

func (a *adapter) ValidateBusiness(_ context.Context, _ schema.ModuleSchema, _ []map[string]interface{}) []importexport.ValidationError {
	return nil
}

func (a *adapter) MapToEntity(_ context.Context, _ schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	code, _ := row["Code"].(string)
	if code == "" {
		return nil, fmt.Errorf("Code is required")
	}
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
	return UOMImportRow{
		Row:         rowNum,
		Code:        code,
		Name:        name,
		Description: desc,
		IsActive:    isActive,
	}, nil
}

func (a *adapter) Repository() importexport.RepositoryActions {
	return &uomRepoAdapter{repo: a.repo}
}

type uomRepoAdapter struct {
	repo *Repository
}

func (r *uomRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]UOMImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(UOMImportRow)
	}
	result := r.repo.BulkUpsert(ctx, rows)
	return result.Inserted, nil
}

func (r *uomRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]UOMImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(UOMImportRow)
	}
	result := r.repo.BulkUpsert(ctx, rows)
	return result.Updated, nil
}

func (r *uomRepoAdapter) ExportData(ctx context.Context, _ schema.ModuleSchema) ([]map[string]interface{}, error) {
	units, err := r.repo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(units))
	for i, u := range units {
		result[i] = map[string]interface{}{
			"Code":        u.Code,
			"Name":        u.Name,
			"Description": u.Description,
			"IsActive":    strconv.FormatBool(u.IsActive),
		}
	}
	return result, nil
}

func (r *uomRepoAdapter) LoadReferences(_ context.Context, _ schema.ModuleSchema) (map[string][]importexport.ReferenceItem, error) {
	return nil, nil
}

func parseBool(v string) bool {
	switch v {
	case "true", "1", "yes", "TRUE", "YES":
		return true
	}
	return false
}
