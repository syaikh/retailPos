package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/xuri/excelize/v2"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

type Format string

const (
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
)

func (e *Engine) Export(w io.Writer, s schema.ModuleSchema, data []map[string]interface{}, format Format) error {
	switch format {
	case FormatCSV:
		return e.exportCSV(w, s, data)
	case FormatXLSX:
		return e.exportXLSX(w, s, data)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

func (e *Engine) exportCSV(w io.Writer, s schema.ModuleSchema, data []map[string]interface{}) error {
	cols := exportableColumns(s)

	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)

	header := make([]string, len(cols))
	for i, col := range cols {
		header[i] = col.Label
	}
	if err := writeCSVRow(writer, header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, row := range data {
		record := make([]string, len(cols))
		for i, col := range cols {
			record[i] = formatValue(col.Type, row[col.Name])
		}
		if err := writeCSVRow(writer, record); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}

func (e *Engine) exportXLSX(w io.Writer, s schema.ModuleSchema, data []map[string]interface{}) error {
	cols := exportableColumns(s)

	wb := excelize.NewFile()
	sheet := s.ModuleName
	_ = wb.SetSheetName("Sheet1", sheet)

	headerSty, _ := wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
	})

	for i, col := range cols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", colLetter)
		_ = wb.SetCellValue(sheet, cell, col.Label)
		_ = wb.SetCellStyle(sheet, cell, cell, headerSty)
	}

	_ = wb.SetRowHeight(sheet, 1, 22)

	for i, col := range cols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		if fmtID, ok := colNumberFormatID(col.Type); ok {
			styl, _ := wb.NewStyle(&excelize.Style{NumFmt: fmtID})
			if styl > 0 {
				_ = wb.SetColStyle(sheet, colLetter, styl)
			}
		}
	}

	for i, col := range cols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		w, ok := columnWidth(col.Type)
		if ok {
			_ = wb.SetColWidth(sheet, colLetter, colLetter, w)
		}
	}

	for rowIdx, row := range data {
		excelRow := rowIdx + 2
		for colIdx, col := range cols {
			colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
			cell := fmt.Sprintf("%s%d", colLetter, excelRow)
			val := row[col.Name]
			if col.Type == schema.ColNumber && val != nil {
				_ = wb.SetCellValue(sheet, cell, val)
			} else {
				_ = wb.SetCellValue(sheet, cell, formatValue(col.Type, val))
			}
		}
	}

	return wb.Write(w)
}

func exportableColumns(s schema.ModuleSchema) []schema.ColumnSchema {
	var cols []schema.ColumnSchema
	for _, c := range s.Columns {
		if c.Exportable {
			cols = append(cols, c)
		}
	}
	return cols
}

func formatValue(t schema.ColumnType, v interface{}) string {
	if v == nil {
		return ""
	}

	str := fmt.Sprintf("%v", v)

	switch t {
	case schema.ColNumber:
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			if f == float64(int(f)) {
				return strconv.FormatInt(int64(f), 10)
			}
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
	case schema.ColBoolean:
		switch strings.ToLower(strings.TrimSpace(str)) {
		case "true", "1", "yes":
			return "true"
		case "false", "0", "no":
			return "false"
		}
	}

	return str
}

func colNumberFormatID(t schema.ColumnType) (int, bool) {
	switch t {
	case schema.ColNumber:
		return 3, true
	default:
		return 0, false
	}
}

func columnWidth(t schema.ColumnType) (float64, bool) {
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

func writeCSVRow(w *csv.Writer, record []string) error {
	sanitized := make([]string, len(record))
	for i, v := range record {
		sanitized[i] = sanitizeCSVField(v)
	}
	return w.Write(sanitized)
}

var dangerousPrefixes = []string{"=", "+", "-", "@", "\t", "\r"}

func sanitizeCSVField(s string) string {
	for _, p := range dangerousPrefixes {
		if strings.HasPrefix(s, p) {
			return "'" + s
		}
	}
	return s
}
