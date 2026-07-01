package validators

import (
	"context"
	"fmt"
	"strings"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type ReferenceValidator struct{}

func (v *ReferenceValidator) Name() string {
	return "reference"
}

func (v *ReferenceValidator) Validate(_ context.Context, s schema.ModuleSchema, rows []map[string]interface{}, refs map[string][]importexport.ReferenceItem) []importexport.ValidationError {
	var errs []importexport.ValidationError

	for i, row := range rows {
		rowNum := i + 2

		for _, col := range s.Columns {
			if col.Type != schema.ColReference {
				continue
			}
			if col.Reference == "" {
				continue
			}

			val, ok := row[col.Name]
			if !ok {
				val = row[col.Label]
			}
			if val == nil || fmt.Sprintf("%v", val) == "" {
				continue
			}

			strVal := strings.TrimSpace(fmt.Sprintf("%v", val))
			refKey := col.Reference

			refItems, exists := refs[refKey]
			if !exists {
				errs = append(errs, importexport.ValidationError{
					Row: rowNum, Field: col.Name, Value: strVal,
					Reason: fmt.Sprintf("reference data %q not loaded", refKey),
					Stage:  importexport.StageReference,
				})
				continue
			}

			found := false
			for _, item := range refItems {
				if strings.EqualFold(strVal, item.Key) {
					found = true
					break
				}
			}

			if !found {
				errs = append(errs, importexport.ValidationError{
					Row: rowNum, Field: col.Name, Value: strVal,
					Reason: fmt.Sprintf("value not found in %s", refKey),
					Suggestion: fmt.Sprintf("create the %s first or use an existing value", refKey),
					Stage:  importexport.StageReference,
				})
			}
		}
	}

	return errs
}
