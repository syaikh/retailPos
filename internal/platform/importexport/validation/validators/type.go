package validators

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type TypeValidator struct{}

func (v *TypeValidator) Name() string {
	return "type"
}

func (v *TypeValidator) Validate(_ context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	var errs []importexportshared.ValidationError

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
			case importexportshared.ColString:
				if col.MaxLength != nil && len(strVal) > *col.MaxLength {
					errs = append(errs, importexportshared.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason:     fmt.Sprintf("exceeds max length of %d", *col.MaxLength),
						Suggestion: fmt.Sprintf("truncate to %d characters", *col.MaxLength),
						Stage:      importexportshared.StageType,
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
						errs = append(errs, importexportshared.ValidationError{
							Row: rowNum, Field: col.Name, Value: strVal,
							Reason:     "value is not in allowed list",
							Suggestion: fmt.Sprintf("allowed values: %s", strings.Join(col.AllowedValues, ", ")),
							Stage:      importexportshared.StageType,
						})
					}
				}

			case importexportshared.ColNumber:
				num, err := strconv.ParseFloat(strVal, 64)
				if err != nil {
					errs = append(errs, importexportshared.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: "must be a number",
						Stage:  importexportshared.StageType,
					})
					continue
				}
				if col.MinValue != nil && num < *col.MinValue {
					errs = append(errs, importexportshared.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: fmt.Sprintf("minimum value is %v", *col.MinValue),
						Stage:  importexportshared.StageType,
					})
				}
				if col.MaxValue != nil && num > *col.MaxValue {
					errs = append(errs, importexportshared.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: fmt.Sprintf("maximum value is %v", *col.MaxValue),
						Stage:  importexportshared.StageType,
					})
				}

			case importexportshared.ColBoolean:
				if !isValidBool(strVal) {
					errs = append(errs, importexportshared.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: "must be a boolean (true/false/yes/no/1/0)",
						Stage:  importexportshared.StageType,
					})
				}

			case importexportshared.ColDate:
				if !isValidDateStr(strVal) {
					errs = append(errs, importexportshared.ValidationError{
						Row: rowNum, Field: col.Name, Value: strVal,
						Reason: "must be a valid date (YYYY-MM-DD)",
						Stage:  importexportshared.StageType,
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
