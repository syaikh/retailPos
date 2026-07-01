package validators

import (
	"context"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type TemplateValidator struct{}

func (v *TemplateValidator) Name() string {
	return "template"
}

func (v *TemplateValidator) Validate(_ context.Context, s importexportshared.ModuleSchema, rows []map[string]interface{}, _ map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	if len(rows) == 0 {
		return nil
	}

	colNames := columnNames(rows[0])
	colSet := make(map[string]bool, len(colNames))
	for _, n := range colNames {
		colSet[n] = true
	}

	var errs []importexportshared.ValidationError

	for _, c := range s.Columns {
		if !c.Template {
			continue
		}
		if !colSet[c.Name] && !colSet[c.Label] {
			if c.Required {
				errs = append(errs, importexportshared.ValidationError{
					Row: -1, Reason: "required column not found",
					Field: c.Name, Stage: importexportshared.StageTemplate,
				})
			}
		}
	}

	return errs
}

func columnNames(row map[string]interface{}) []string {
	names := make([]string, 0, len(row))
	for k := range row {
		names = append(names, k)
	}
	return names
}
