package importexport

import (
	"context"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type SchemaRegistryProvider interface {
	Register(s importexportshared.ModuleSchema) error
	Get(module string) (importexportshared.ModuleSchema, error)
	All() []importexportshared.ModuleSchema
}

type AdapterRegistryProvider interface {
	Register(adapter importexportshared.Adapter) error
	Get(module string) (importexportshared.Adapter, error)
	Modules() []string
}

type Validator interface {
	Name() string
	Validate(ctx context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, refs map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError
}
