package importutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type ImportResult struct {
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Errors   []string `json:"errors"`
}

func (r *ImportResult) AddError(row int, msg string) {
	r.Errors = append(r.Errors, fmt.Sprintf("row %d: %s", row, msg))
}

func ParseCSV(r io.Reader) (headers []string, rows [][]string, err error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = false

	all, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(all) < 2 {
		return nil, nil, fmt.Errorf("CSV must have header row and at least one data row")
	}

	headers = make([]string, len(all[0]))
	for i, h := range all[0] {
		headers[i] = strings.TrimSpace(h)
	}

	rows = make([][]string, len(all)-1)
	for i, row := range all[1:] {
		rows[i] = make([]string, len(headers))
		for j, v := range row {
			rows[i][j] = strings.TrimSpace(v)
		}
	}

	return headers, rows, nil
}

func HeaderIndex(headers []string, name string) int {
	for i, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h), name) {
			return i
		}
	}
	return -1
}
