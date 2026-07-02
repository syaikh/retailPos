package validators

import (
	"context"
	"fmt"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type RequiredValidator struct{}

func (v *RequiredValidator) Name() string {
	return "required"
}

func (v *RequiredValidator) Validate(_ context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	var errs []importexportshared.ValidationError

	for i, row := range rows {
		rowNum := i + 2

		for _, col := range s.Columns {
			if !col.Required {
				continue
			}

			val, ok := row[col.Name]
			if !ok {
				val = row[col.Label]
			}

			if val == nil || fmt.Sprintf("%v", val) == "" {
				errs = append(errs, importexportshared.ValidationError{
					Row: rowNum, Field: col.Name,
					Reason: "field is required",
					Stage:  importexportshared.StageTemplate,
				})
			}
		}
	}

	return errs
}
