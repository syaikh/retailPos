package importexportshared

import "context"

type Adapter interface {
	ModuleName() string
	ValidateBusiness(ctx context.Context, s ModuleSchema, rows []map[string]interface{}) []ValidationError
	MapToEntity(ctx context.Context, s ModuleSchema, row map[string]interface{}) (interface{}, error)
	Repository() RepositoryActions
}

type RepositoryActions interface {
	Insert(ctx context.Context, entities []interface{}) (int, error)
	Update(ctx context.Context, entities []interface{}) (int, error)
	ExportData(ctx context.Context, s ModuleSchema) ([]map[string]interface{}, error)
	LoadReferences(ctx context.Context, s ModuleSchema) (map[string][]ReferenceItem, error)
}

type ReferenceItem struct {
	Key   string
	Value interface{}
}
