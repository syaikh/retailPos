package importexport

import (
	"context"

	"retail-pos-system/internal/platform/importexport/schema"
)

type Adapter interface {
	ModuleName() string
	ValidateBusiness(ctx context.Context, s schema.ModuleSchema, rows []map[string]interface{}) []ValidationError
	MapToEntity(ctx context.Context, s schema.ModuleSchema, row map[string]interface{}) (interface{}, error)
	Repository() RepositoryActions
}

type RepositoryActions interface {
	Insert(ctx context.Context, entities []interface{}) (int, error)
	Update(ctx context.Context, entities []interface{}) (int, error)
	ExportData(ctx context.Context, s schema.ModuleSchema) ([]map[string]interface{}, error)
	LoadReferences(ctx context.Context, s schema.ModuleSchema) (map[string][]ReferenceItem, error)
}

type SchemaRegistryProvider interface {
	Register(s schema.ModuleSchema) error
	Get(module string) (schema.ModuleSchema, error)
	All() []schema.ModuleSchema
}

type AdapterRegistryProvider interface {
	Register(adapter Adapter) error
	Get(module string) (Adapter, error)
	Modules() []string
}

type Validator interface {
	Name() string
	Validate(ctx context.Context, s schema.ModuleSchema, rows []map[string]interface{}, refs map[string][]ReferenceItem) []ValidationError
}
