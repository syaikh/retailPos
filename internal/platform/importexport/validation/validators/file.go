package validators

import (
	"context"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type FileValidator struct{}

func (v *FileValidator) Name() string {
	return "file"
}

func (v *FileValidator) Validate(_ context.Context, s schema.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexport.ReferenceItem) []importexport.ValidationError {
	if len(rows) == 0 {
		return []importexport.ValidationError{
			{Reason: "file contains no data rows", Stage: importexport.StageTemplate},
		}
	}
	return nil
}
