package validators

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

type TypeValidator struct{}

func (v *TypeValidator) Name() string {
	return "type"
}

func (v *TypeValidator) Validate(_ context.Context, s schema.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexport.ReferenceItem) []importexport.ValidationError {
	var errs []importexport.ValidationError

	for i, row := range rows {
		rowNum := i + 2

		for _, col := range s.Columns {
			val, ok := row[col.Name]
			if !ok {
				val = row[col.Label]
				if val == nil {
					continue
				}
			}

			if val == nil || val == "" {
				continue
			}

			strVal := fmt.Sprintf("%v", val)

			switch col.Type {
			case schema.ColString:
				if col.MaxLength != nil && len(strVal) > *col.MaxLength {
					errs = append(errs, importexport.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason:     fmt.Sprintf("exceeds max length of %d", *col.MaxLength),
						Suggestion: fmt.Sprintf("truncate to %d characters", *col.MaxLength),
						Stage:      importexport.StageType,
					})
				}
				if col.AllowedValues != nil {
					found := false
					for _, av := range col.AllowedValues {
						if strings.EqualFold(strVal, av) {
							found = true
							break
						}
					}
					if !found {
						errs = append(errs, importexport.ValidationError{
							Row: rowNum, Field: col.Name, Value: strVal,
							Reason:     "value is not in allowed list",
							Suggestion: fmt.Sprintf("allowed values: %s", strings.Join(col.AllowedValues, ", ")),
							Stage:      importexport.StageType,
						})
					}
				}

			case schema.ColNumber:
				num, err := strconv.ParseFloat(strVal, 64)
				if err != nil {
					errs = append(errs, importexport.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: "must be a number",
						Stage:  importexport.StageType,
					})
					continue
				}
				if col.MinValue != nil && num < *col.MinValue {
					errs = append(errs, importexport.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: fmt.Sprintf("minimum value is %v", *col.MinValue),
						Stage:  importexport.StageType,
					})
				}
				if col.MaxValue != nil && num > *col.MaxValue {
					errs = append(errs, importexport.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: fmt.Sprintf("maximum value is %v", *col.MaxValue),
						Stage:  importexport.StageType,
					})
				}

			case schema.ColBoolean:
				if !isValidBool(strVal) {
					errs = append(errs, importexport.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: "must be a boolean (true/false/yes/no/1/0)",
						Stage:  importexport.StageType,
					})
				}

			case schema.ColDate:
				if !isValidDateStr(strVal) {
					errs = append(errs, importexport.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: "must be a valid date (YYYY-MM-DD)",
						Stage:  importexport.StageType,
					})
				}
			}
		}
	}

	return errs
}

func isValidBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "false", "yes", "no", "1", "0", "":
		return true
	}
	return false
}

func isValidDateStr(v string) bool {
	if v == "" {
		return true
	}
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return false
	}
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}
