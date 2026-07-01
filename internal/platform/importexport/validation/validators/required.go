package validators

import (
	"context"
	"fmt"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type RequiredValidator struct{}

func (v *RequiredValidator) Name() string {
	return "required"
}

func (v *RequiredValidator) Validate(_ context.Context, s schema.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexport.ReferenceItem) []importexport.ValidationError {
	var errs []importexport.ValidationError

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
				errs = append(errs, importexport.ValidationError{
					Row: rowNum, Field: col.Name,
					Reason: "field is required",
					Stage:  importexport.StageTemplate,
				})
			}
		}
	}

	return errs
}
