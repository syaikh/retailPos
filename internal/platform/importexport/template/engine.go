package template

import (
	"fmt"
	"io"

	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/xuri/excelize/v2"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Generate(s schema.ModuleSchema, refData map[string][]string, w io.Writer) error {
	wb := excelize.NewFile()

	headerSty, _ := wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
	})

	if err := e.createInstructionSheet(wb, s); err != nil {
		return fmt.Errorf("instruction sheet: %w", err)
	}

	dataSheet := s.ModuleName
	_ = wb.SetSheetName("Sheet1", dataSheet)

	if err := e.createDataSheet(wb, s, dataSheet, refData, headerSty); err != nil {
		return fmt.Errorf("data sheet: %w", err)
	}

	if err := e.createReferenceSheets(wb, s, refData, headerSty); err != nil {
		return fmt.Errorf("reference sheets: %w", err)
	}

	return wb.Write(w)
}

func (e *Engine) createInstructionSheet(wb *excelize.File, s schema.ModuleSchema) error {
	sheet := "Instructions"
	_, _ = wb.NewSheet(sheet)

	titleSty, _ := wb.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	bodySty, _ := wb.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 11}})

	writeCell := func(row int, text string, styl int) {
		cell := fmt.Sprintf("A%d", row)
		_ = wb.SetCellValue(sheet, cell, text)
		if styl != 0 {
			_ = wb.SetCellStyle(sheet, cell, cell, styl)
		}
	}

	writeCell(1, fmt.Sprintf("Module: %s", s.DisplayName), titleSty)
	writeCell(2, fmt.Sprintf("Schema Version: %s", s.SchemaVersion), bodySty)
	writeCell(3, fmt.Sprintf("Description: %s", s.Description), bodySty)
	writeCell(4, "", bodySty)
	writeCell(5, "Instructions:", bodySty)
	writeCell(6, fmt.Sprintf("1. Fill in the data in the '%s' sheet.", s.ModuleName), bodySty)
	writeCell(7, "2. Required columns are marked with a red asterisk (*).", bodySty)
	writeCell(8, "3. For reference columns, use the dropdown list or refer to the reference sheets.", bodySty)
	writeCell(9, "4. Do not modify the header row.", bodySty)
	writeCell(10, "5. Import the completed file using the Import feature.", bodySty)

	_ = wb.SetColWidth(sheet, "A", "A", 80)
	return nil
}

func (e *Engine) createDataSheet(wb *excelize.File, s schema.ModuleSchema, sheet string, refData map[string][]string, headerSty int) error {
	templateCols := make([]schema.ColumnSchema, 0, len(s.Columns))
	for _, col := range s.Columns {
		if col.Template {
			templateCols = append(templateCols, col)
		}
	}

	for i, col := range templateCols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		label := col.Label
		if col.Required {
			label = col.Label + " *"
		}
		_ = wb.SetCellValue(sheet, fmt.Sprintf("%s1", colLetter), label)
		_ = wb.SetCellStyle(sheet, fmt.Sprintf("%s1", colLetter), fmt.Sprintf("%s1", colLetter), headerSty)
	}

	_ = wb.SetRowHeight(sheet, 1, 22)

	for i, col := range templateCols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		w, ok := colWidth(col.Type)
		if ok {
			_ = wb.SetColWidth(sheet, colLetter, colLetter, w)
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

		if err := wb.SetSheetVisible(sheet, false, true); err != nil {
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

		lastRow := len(values) + 1
		refSheet := "Ref_" + ref.ReferenceModule
		listRange := fmt.Sprintf("$%s!$A$2:$A$%d", refSheet, lastRow)

		dv := excelize.NewDataValidation(true)
		dv.SetSqref(dvRange)
		dv.SetSqrefDropList(listRange)
		dv.SetInput(fmt.Sprintf("Select %s", ref.ReferenceLabel),
			fmt.Sprintf("Choose a valid %s from the dropdown", ref.ReferenceLabel))
		dv.SetError(excelize.DataValidationErrorStyleStop, "Invalid Value",
			fmt.Sprintf("Please select a valid %s", ref.ReferenceLabel))

		if err := wb.AddDataValidation(sheet, dv); err != nil {
			return fmt.Errorf("data validation for %s: %w", ref.Column, err)
		}
	}
	return nil
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
