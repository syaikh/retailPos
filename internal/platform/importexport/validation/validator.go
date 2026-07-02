package validation

import (
	"context"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type Validator interface {
	Name() string
	Validate(ctx context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, refs map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError
}
