package template

import (
	"fmt"
	"io"
	"time"

	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/xuri/excelize/v2"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

type columnStyles struct {
	required int
	optional int
	ref      int
	readonly int
}

func (e *Engine) Generate(s schema.ModuleSchema, refData map[string][]string, w io.Writer) error {
	wb := excelize.NewFile()

	cs, err := e.createStyles(wb)
	if err != nil {
		return fmt.Errorf("create styles: %w", err)
	}

	headerSty, err := wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
	})
	if err != nil {
		return fmt.Errorf("header style: %w", err)
	}

	if err := e.createMetaSheet(wb, s); err != nil {
		return fmt.Errorf("meta sheet: %w", err)
	}

	if err := e.createInstructionSheet(wb, s); err != nil {
		return fmt.Errorf("instruction sheet: %w", err)
	}

	dataSheet := s.ModuleName
	_ = wb.SetSheetName("Sheet1", dataSheet)

	if err := e.createDataSheet(wb, s, dataSheet, refData, cs); err != nil {
		return fmt.Errorf("data sheet: %w", err)
	}

	if err := e.createReferenceSheets(wb, s, refData, headerSty); err != nil {
		return fmt.Errorf("reference sheets: %w", err)
	}

	return wb.Write(w)
}

func (e *Engine) createStyles(wb *excelize.File) (columnStyles, error) {
	var cs columnStyles
	var err error
	cs.required, err = wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF3CD"}},
	})
	if err != nil {
		return cs, fmt.Errorf("required style: %w", err)
	}
	cs.optional, err = wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8F4FD"}},
	})
	if err != nil {
		return cs, fmt.Errorf("optional style: %w", err)
	}
	cs.ref, err = wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D4EDDA"}},
	})
	if err != nil {
		return cs, fmt.Errorf("ref style: %w", err)
	}
	cs.readonly, err = wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F0F0F0"}},
	})
	if err != nil {
		return cs, fmt.Errorf("readonly style: %w", err)
	}
	return cs, nil
}

func (e *Engine) headerStyleFor(cs columnStyles, col schema.ColumnSchema) int {
	if !col.Editable {
		return cs.readonly
	}
	if col.Required {
		return cs.required
	}
	if col.Reference != "" {
		return cs.ref
	}
	return cs.optional
}

func (e *Engine) createMetaSheet(wb *excelize.File, s schema.ModuleSchema) error {
	sheet := "_Meta"
	_, err := wb.NewSheet(sheet)
	if err != nil {
		return err
	}

	type entry struct{ Key, Value string }
	meta := []entry{
		{"Module", s.ModuleName},
		{"SchemaVersion", s.SchemaVersion},
		{"GeneratedAt", time.Now().UTC().Format(time.RFC3339)},
		{"TotalColumns", fmt.Sprintf("%d", len(s.Columns))},
	}
	if s.DisplayName != "" {
		meta = append(meta, entry{"DisplayName", s.DisplayName})
	}

	for i, m := range meta {
		row := i + 1
		_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", row), m.Key)
		_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", row), m.Value)
	}

	if err := wb.SetSheetVisible(sheet, false, true); err != nil {
		return fmt.Errorf("hide meta sheet: %w", err)
	}
	return nil
}

func (e *Engine) createInstructionSheet(wb *excelize.File, s schema.ModuleSchema) error {
	sheet := "Instructions"
	_, _ = wb.NewSheet(sheet)

	titleSty, _ := wb.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	bodySty, _ := wb.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 11}})
	tableHeaderSty, _ := wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
	})

	writeCell := func(row int, text string, styl int) {
		cell := fmt.Sprintf("A%d", row)
		_ = wb.SetCellValue(sheet, cell, text)
		if styl != 0 {
			_ = wb.SetCellStyle(sheet, cell, cell, styl)
		}
	}

	row := 1
	writeCell(row, fmt.Sprintf("Module: %s", s.DisplayName), titleSty); row++
	writeCell(row, fmt.Sprintf("Schema Version: %s", s.SchemaVersion), bodySty); row++
	if s.Description != "" {
		writeCell(row, fmt.Sprintf("Description: %s", s.Description), bodySty); row++
	}
	row++

	writeCell(row, "Instructions:", bodySty); row++
	writeCell(row, fmt.Sprintf("1. Fill in the data in the '%s' sheet. Do not modify the header row.", s.ModuleName), bodySty); row++
	writeCell(row, "2. Required columns have a yellow header. They must contain a value.", bodySty); row++
	writeCell(row, "3. Reference columns have a green header. Use the dropdown or enter an existing value.", bodySty); row++
	writeCell(row, "4. Optional columns have a blue header. Leave blank if not needed.", bodySty); row++
	writeCell(row, "5. Read-only columns have a gray header. These are informational and cannot be changed.", bodySty); row++
	row++

	writeCell(row, "Column Reference", titleSty); row++

	_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Column")
	_ = wb.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), tableHeaderSty)
	_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", row), "Required")
	_ = wb.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), tableHeaderSty)
	_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", row), "Editable")
	_ = wb.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), tableHeaderSty)
	_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", row), "Description")
	_ = wb.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), tableHeaderSty)
	_ = wb.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Validation")
	_ = wb.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), tableHeaderSty)
	row++

	for _, col := range s.Columns {
		if !col.Template {
			continue
		}
		_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", row), col.Label)
		if col.Required {
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", row), "Yes")
		} else {
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", row), "No")
		}
		if col.Editable {
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", row), "Yes")
		} else {
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", row), "No")
		}
		_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", row), col.Description)

		validation := buildValidationHint(col)
		_ = wb.SetCellValue(sheet, fmt.Sprintf("E%d", row), validation)
		row++
	}

	_ = wb.SetColWidth(sheet, "A", "A", 25)
	_ = wb.SetColWidth(sheet, "B", "B", 12)
	_ = wb.SetColWidth(sheet, "C", "C", 12)
	_ = wb.SetColWidth(sheet, "D", "D", 40)
	_ = wb.SetColWidth(sheet, "E", "E", 35)
	return nil
}

func buildValidationHint(col schema.ColumnSchema) string {
	var hints []string
	switch col.Type {
	case schema.ColString:
		if col.MaxLength != nil {
			hints = append(hints, fmt.Sprintf("Max %d characters", *col.MaxLength))
		}
	case schema.ColNumber:
		if col.MinValue != nil && col.MaxValue != nil {
			hints = append(hints, fmt.Sprintf("Number %.0f \u2013 %.0f", *col.MinValue, *col.MaxValue))
		} else if col.MinValue != nil {
			hints = append(hints, fmt.Sprintf("Number >= %.0f", *col.MinValue))
		} else if col.MaxValue != nil {
			hints = append(hints, fmt.Sprintf("Number <= %.0f", *col.MaxValue))
		} else {
			hints = append(hints, "Numeric value")
		}
	case schema.ColBoolean:
		hints = append(hints, "Yes/No or True/False")
	case schema.ColDate:
		hints = append(hints, "Date format YYYY-MM-DD")
	case schema.ColReference:
		hints = append(hints, "Must exist in system")
	}
	if len(col.AllowedValues) > 0 {
		hints = append(hints, "Allowed: "+joinStrings(col.AllowedValues))
	}
	if len(hints) == 0 {
		return ""
	}
	result := hints[0]
	for _, h := range hints[1:] {
		result += "; " + h
	}
	return result
}

func joinStrings(s []string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return s[0]
	}
	result := s[0]
	for _, v := range s[1:] {
		result += ", " + v
	}
	return result
}

func (e *Engine) createDataSheet(wb *excelize.File, s schema.ModuleSchema, sheet string, refData map[string][]string, cs columnStyles) error {
	templateCols := make([]schema.ColumnSchema, 0, len(s.Columns))
	for _, col := range s.Columns {
		if col.Template {
			templateCols = append(templateCols, col)
		}
	}

	for i, col := range templateCols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", colLetter)
		_ = wb.SetCellValue(sheet, cell, col.Label)
		styl := e.headerStyleFor(cs, col)
		_ = wb.SetCellStyle(sheet, cell, cell, styl)
	}

	_ = wb.SetRowHeight(sheet, 1, 22)

	textSty, _ := wb.NewStyle(&excelize.Style{
		NumFmt: 49,
	})

	for i, col := range templateCols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		w, ok := colWidth(col.Type)
		if ok {
			_ = wb.SetColWidth(sheet, colLetter, colLetter, w)
		}
		if col.Type == schema.ColString && textSty > 0 {
			_ = wb.SetColStyle(sheet, colLetter, textSty)
		}
	}

	return e.addDataValidation(wb, sheet, s, templateCols, refData)
}

func (e *Engine) createReferenceSheets(wb *excelize.File, s schema.ModuleSchema, refData map[string][]string, headerSty int) error {
	for _, ref := range s.References {
		sheet := "Ref_" + ref.ReferenceModule
		_, err := wb.NewSheet(sheet)
		if err != nil {
			return err
		}

		_ = wb.SetCellValue(sheet, "A1", ref.ReferenceLabel)
		_ = wb.SetCellValue(sheet, "B1", "Description")
		_ = wb.SetCellStyle(sheet, "A1", "B1", headerSty)

		values, ok := refData[ref.ReferenceModule]
		if ok {
			for i, val := range values {
				row := i + 2
				_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", row), val)
			}
		}

		_ = wb.SetColWidth(sheet, "A", "A", 30)
		_ = wb.SetColWidth(sheet, "B", "B", 50)

		if err := wb.SetSheetVisible(sheet, false); err != nil {
			return fmt.Errorf("hide ref sheet %s: %w", sheet, err)
		}
	}

	for _, col := range s.Columns {
		if len(col.AllowedValues) == 0 {
			continue
		}
		isRef := false
		for _, ref := range s.References {
			if ref.Column == col.Name {
				isRef = true
				break
			}
		}
		if isRef {
			continue
		}

		sheet := "Ref_" + col.Name
		_, err := wb.NewSheet(sheet)
		if err != nil {
			return err
		}

		_ = wb.SetCellValue(sheet, "A1", col.Label)
		_ = wb.SetCellStyle(sheet, "A1", "A1", headerSty)

		for i, val := range col.AllowedValues {
			row := i + 2
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", row), val)
		}

		_ = wb.SetColWidth(sheet, "A", "A", 30)

		if err := wb.SetSheetVisible(sheet, false); err != nil {
			return fmt.Errorf("hide ref sheet %s: %w", sheet, err)
		}
	}

	return nil
}

func (e *Engine) addDataValidation(wb *excelize.File, sheet string, s schema.ModuleSchema, templateCols []schema.ColumnSchema, refData map[string][]string) error {
	for _, ref := range s.References {
		values, ok := refData[ref.ReferenceModule]
		if !ok || len(values) == 0 {
			continue
		}

		colIdx := -1
		for i, tc := range templateCols {
			if tc.Name == ref.Column {
				colIdx = i
				break
			}
		}
		if colIdx == -1 {
			continue
		}

		colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
		dvRange := fmt.Sprintf("%s2:%s1048576", colLetter, colLetter)
		refSheet := "Ref_" + ref.ReferenceModule

		dv := excelize.NewDataValidation(true)
		dv.SetSqref(dvRange)
		dv.SetSqrefDropList(fmt.Sprintf("%s!$A$2:$A$%d", refSheet, len(values)+1))
		dv.SetInput(fmt.Sprintf("Select %s", ref.ReferenceLabel),
			fmt.Sprintf("Choose a valid %s from the dropdown", ref.ReferenceLabel))
		dv.SetError(excelize.DataValidationErrorStyleStop, "Invalid Value",
			fmt.Sprintf("Please select a valid %s", ref.ReferenceLabel))

		if err := wb.AddDataValidation(sheet, dv); err != nil {
			return fmt.Errorf("data validation for %s: %w", ref.Column, err)
		}
	}

	for _, tc := range templateCols {
		if len(tc.AllowedValues) == 0 {
			continue
		}
		isRef := false
		for _, ref := range s.References {
			if ref.Column == tc.Name {
				isRef = true
				break
			}
		}
		if isRef {
			continue
		}

		colLetter, _ := excelize.ColumnNumberToName(columnIndex(templateCols, tc.Name) + 1)
		dvRange := fmt.Sprintf("%s2:%s1048576", colLetter, colLetter)
		refSheet := "Ref_" + tc.Name

		dv := excelize.NewDataValidation(true)
		dv.SetSqref(dvRange)
		dv.SetSqrefDropList(fmt.Sprintf("%s!$A$2:$A$%d", refSheet, len(tc.AllowedValues)+1))
		dv.SetInput(fmt.Sprintf("Select %s", tc.Label),
			fmt.Sprintf("Choose a valid %s from the dropdown", tc.Label))
		dv.SetError(excelize.DataValidationErrorStyleStop, "Invalid Value",
			fmt.Sprintf("Please select a valid %s", tc.Label))

		if err := wb.AddDataValidation(sheet, dv); err != nil {
			return fmt.Errorf("data validation for %s: %w", tc.Name, err)
		}
	}

	return nil
}

func columnIndex(cols []schema.ColumnSchema, name string) int {
	for i, c := range cols {
		if c.Name == name {
			return i
		}
	}
	return -1
}

func colWidth(t schema.ColumnType) (float64, bool) {
	switch t {
	case schema.ColString:
		return 35, true
	case schema.ColNumber:
		return 15, true
	case schema.ColBoolean:
		return 12, true
	case schema.ColDate:
		return 15, true
	case schema.ColReference:
		return 25, true
	}
	return 0, false
}
