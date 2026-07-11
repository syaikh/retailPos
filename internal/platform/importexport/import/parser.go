package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/xuri/excelize/v2"
)

func ParseFile(filename string, r io.Reader, s schema.ModuleSchema) ([]map[string]interface{}, error) {
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return parseXLSX(r, s)
	}
	return parseCSV(r, s)
}

func parseCSV(r io.Reader, s schema.ModuleSchema) ([]map[string]interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	all, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(all) < 2 {
		return nil, fmt.Errorf("csv must have header row and at least one data row")
	}

	headers := make([]string, len(all[0]))
	for i, h := range all[0] {
		headers[i] = strings.TrimSpace(h)
	}

	headerMap := buildHeaderMap(headers, s)

	rows := make([]map[string]interface{}, 0, len(all)-1)
	for i, row := range all[1:] {
		rowMap := make(map[string]interface{}, len(headers))
		for j, val := range row {
			colName := headers[j]
			if mapped, ok := headerMap[colName]; ok {
				colName = mapped
			}
			rowMap[colName] = strings.TrimSpace(val)
		}
		rowMap["_row"] = i + 2
		rows = append(rows, rowMap)
	}

	return rows, nil
}

func parseXLSX(r io.Reader, s schema.ModuleSchema) ([]map[string]interface{}, error) {
	wb, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer wb.Close()

	return ParseXLSXWorkbook(wb, s)
}

func ParseXLSXWorkbook(wb *excelize.File, s schema.ModuleSchema) ([]map[string]interface{}, error) {
	sheet := wb.GetSheetName(0)
	all, err := wb.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("read xlsx sheet: %w", err)
	}
	if len(all) < 2 {
		return nil, fmt.Errorf("xlsx must have header row and at least one data row")
	}

	headers := make([]string, len(all[0]))
	for i, h := range all[0] {
		headers[i] = strings.TrimSpace(h)
	}

	headerMap := buildHeaderMap(headers, s)

	rows := make([]map[string]interface{}, 0, len(all)-1)
	for i, row := range all[1:] {
		hasData := false
		for _, val := range row {
			if strings.TrimSpace(val) != "" {
				hasData = true
				break
			}
		}
		if !hasData {
			continue
		}

		rowMap := make(map[string]interface{}, len(headers))
		for j, val := range row {
			if j >= len(headers) {
				continue
			}
			colName := headers[j]
			if mapped, ok := headerMap[colName]; ok {
				colName = mapped
			}
			rowMap[colName] = strings.TrimSpace(val)
		}
		rowMap["_row"] = i + 2
		rows = append(rows, rowMap)
	}

	return rows, nil
}

func buildHeaderMap(headers []string, s schema.ModuleSchema) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		for _, col := range s.Columns {
			colLower := strings.ToLower(col.Label)
			nameLower := strings.ToLower(col.Name)
			if lower == colLower || lower == nameLower {
				m[h] = col.Name
				break
			}
		}
	}
	return m
}
