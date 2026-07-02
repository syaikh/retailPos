package importer

import (
	"fmt"
	"strings"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

func GeneratePreview(s schema.ModuleSchema, rows []map[string]interface{}, errs []importexportshared.ValidationError) *importexport.PreviewResult {
	errMap := buildErrorMap(errs)
	seenKeys := make(map[string]int)

	result := &importexport.PreviewResult{
		Module:    s.ModuleName,
		TotalRows: len(rows),
	}

	for i, row := range rows {
		rowNum := i + 2
		rowErrs := errMap[rowNum]

		cleanValues := stripInternalKeys(row)

		pr := importexport.PreviewRow{
			RowNumber: rowNum,
			NewValues: cleanValues,
			Errors:    rowErrs,
		}

		if len(rowErrs) > 0 {
			pr.Status = "error"
			result.ErrorCount++
		} else if isUpdate(s, row, seenKeys) {
			pr.Status = "update"
			pr.OldValues = nil
			result.UpdateCount++
		} else {
			pr.Status = "insert"
			result.InsertCount++
		}

		if bk := buildRowBusinessKey(s.BusinessKeys, row); bk != "" {
			seenKeys[bk] = rowNum
		}

		result.Rows = append(result.Rows, pr)
	}

	return result
}

func buildErrorMap(errs []importexportshared.ValidationError) map[int][]importexportshared.ValidationError {
	m := make(map[int][]importexportshared.ValidationError)
	for _, e := range errs {
		m[e.Row] = append(m[e.Row], e)
	}
	return m
}

func isUpdate(s schema.ModuleSchema, row map[string]interface{}, seenKeys map[string]int) bool {
	bk := buildRowBusinessKey(s.BusinessKeys, row)
	if bk == "" {
		return false
	}
	_, exists := seenKeys[bk]
	return exists
}

func buildRowBusinessKey(keys []string, row map[string]interface{}) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v, ok := row[k]
		if !ok || v == nil {
			return ""
		}
		parts = append(parts, strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v))))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "|")
}

func stripInternalKeys(row map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(row))
	for k, v := range row {
		if strings.HasPrefix(k, "_") {
			continue
		}
		out[k] = v
	}
	return out
}
