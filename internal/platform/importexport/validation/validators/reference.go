package validators

import (
	"context"
	"fmt"
	"strings"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type ReferenceValidator struct{}

func (v *ReferenceValidator) Name() string {
	return "reference"
}

func (v *ReferenceValidator) Validate(_ context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, refs map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	var errs []importexportshared.ValidationError

	for i, row := range rows {
		rowNum := i + 2

		for _, col := range s.Columns {
			if col.Type != importexportshared.ColReference {
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
				errs = append(errs, importexportshared.ValidationError{
					Row: rowNum, Field: col.Name, Value: strVal,
					Reason: fmt.Sprintf("reference data %q not loaded", refKey),
					Stage:  importexportshared.StageReference,
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
				errs = append(errs, importexportshared.ValidationError{
					Row: rowNum, Field: col.Name, Value: strVal,
					Reason: fmt.Sprintf("value not found in %s", refKey),
					Suggestion: fmt.Sprintf("create the %s first or use an existing value", refKey),
					Stage:  importexportshared.StageReference,
				})
			}
		}
	}

	return errs
}
