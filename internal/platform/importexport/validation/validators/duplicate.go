package validators

import (
	"context"
	"fmt"
	"strings"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type DuplicateValidator struct{}

func (v *DuplicateValidator) Name() string {
	return "duplicate"
}

func (v *DuplicateValidator) Validate(_ context.Context, s schema.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexport.ReferenceItem) []importexport.ValidationError {
	if len(s.BusinessKeys) == 0 {
		return nil
	}

	var errs []importexport.ValidationError
	seen := make(map[string]int)

	for i, row := range rows {
		rowNum := i + 2
		key := buildBusinessKey(s.BusinessKeys, row)

		if key == "" {
			continue
		}

		if firstRow, exists := seen[key]; exists {
			errs = append(errs, importexport.ValidationError{
				Row: rowNum,
				Reason: fmt.Sprintf("duplicate %s (also appears at row %d)",
					strings.Join(s.BusinessKeys, "+"), firstRow),
				Stage: importexport.StageTemplate,
			})
		} else {
			seen[key] = rowNum
		}
	}

	return errs
}

func buildBusinessKey(keys []string, row map[string]interface{}) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v, ok := row[k]
		if !ok {
			v = row[findLabelByKey(k)]
		}
		if v == nil {
			return ""
		}
		parts = append(parts, strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v))))
	}
	return strings.Join(parts, "|")
}

func findLabelByKey(key string) string {
	return key
}
