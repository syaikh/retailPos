package validation

import (
	"context"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type Validator interface {
	Name() string
	Validate(ctx context.Context, s schema.ModuleSchema, rows []map[string]interface{}, refs map[string][]importexport.ReferenceItem) []importexport.ValidationError
}
