package customer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/middleware"
)

type adapter struct {
	repo *Repository
}

func NewAdapter(repo *Repository) importexportshared.Adapter {
	return &adapter{repo: repo}
}

func (a *adapter) ModuleName() string {
	return "customers"
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
	var storeID *int
	if sid, ok := row["_store_id"]; ok {
		if v, ok2 := sid.(int); ok2 && v > 0 {
			storeID = &v
		}
	}
	phone, _ := row["Phone"].(string)
	email, _ := row["Email"].(string)
	address, _ := row["Address"].(string)
	note, _ := row["Note"].(string)
	isActive := true
	if v, ok := row["IsActive"]; ok {
		isActive = parseBool(fmt.Sprintf("%v", v))
	}
	return CustomerImportRow{
		Row:      rowNum,
		Name:     name,
		Phone:    phone,
		Email:    email,
		Address:  address,
		Note:     note,
		IsActive: isActive,
		StoreID:  storeID,
	}, nil
}

func (a *adapter) Repository() importexportshared.RepositoryActions {
	return &customerRepoAdapter{repo: a.repo}
}

type customerRepoAdapter struct {
	repo *Repository
}

func (r *customerRepoAdapter) Insert(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]CustomerImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(CustomerImportRow)
	}
	storeID := extractStoreID(rows)
	result := r.repo.BulkUpsertCustomers(ctx, rows, storeID)
	if len(result.Errors) > 0 {
		return result.Inserted, fmt.Errorf("customer import errors: %v", strings.Join(result.Errors, "; "))
	}
	return result.Inserted, nil
}

func (r *customerRepoAdapter) Update(ctx context.Context, entities []interface{}) (int, error) {
	rows := make([]CustomerImportRow, len(entities))
	for i, e := range entities {
		rows[i] = e.(CustomerImportRow)
	}
	storeID := extractStoreID(rows)
	result := r.repo.BulkUpsertCustomers(ctx, rows, storeID)
	if len(result.Errors) > 0 {
		return result.Updated, fmt.Errorf("customer import errors: %v", strings.Join(result.Errors, "; "))
	}
	return result.Updated, nil
}

func extractStoreID(rows []CustomerImportRow) *int {
	for _, r := range rows {
		if r.StoreID != nil {
			return r.StoreID
		}
	}
	return nil
}

func (r *customerRepoAdapter) ExportData(ctx context.Context, _ importexportshared.ModuleSchema) ([]map[string]interface{}, error) {
	storeID := middleware.StoreIDFromContext(ctx)
	customers, err := r.repo.GetAllCustomersForExport(ctx, storeID)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(customers))
	for i, c := range customers {
		result[i] = map[string]interface{}{
			"Name":     c.Name,
			"Phone":    strVal(c.Phone),
			"Email":    strVal(c.Email),
			"Address":  strVal(c.Address),
			"Note":     strVal(c.Note),
			"IsActive": strconv.FormatBool(c.IsActive),
		}
	}
	return result, nil
}

func (r *customerRepoAdapter) LoadReferences(_ context.Context, _ importexportshared.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, nil
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseBool(v string) bool {
	switch v {
	case "true", "1", "yes", "Yes", "TRUE", "YES":
		return true
	}
	return false
}
