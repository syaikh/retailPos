package validators

import (
	"context"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type FileValidator struct{}

func (v *FileValidator) Name() string {
	return "file"
}

func (v *FileValidator) Validate(_ context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	if len(rows) == 0 {
		return []importexportshared.ValidationError{
			{Reason: "file contains no data rows", Stage: importexportshared.StageTemplate},
		}
	}
	return nil
}
