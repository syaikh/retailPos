package importer

import (
	"bytes"
	"strings"
	"testing"

	"retail-pos-system/internal/platform/importexport/schema"
)

// fuzzImportSchema is the smallest schema that exercises header mapping and row
// collection without depending on a real module registry.
func fuzzImportSchema() schema.ModuleSchema {
	return schema.ModuleSchema{
		ModuleName: "fuzz",
		Columns: []schema.ColumnSchema{
			{Name: "sku", Type: schema.ColString, Label: "SKU", Required: true},
			{Name: "price", Type: schema.ColNumber, Label: "Price"},
		},
		PrimaryKey: "sku",
	}
}

// FuzzParseFile exercises the CSV import parser on arbitrary bytes. Uploaded
// spreadsheets are untrusted input, so the parser must never panic, must reject
// malformed data with an error instead of partially populated rows, and must
// trim every cell (a stray space in " sku " would otherwise become part of a
// business key).
//
// The .xlsx branch is deliberately not exercised: excelize decompresses XML
// from the input, which turns random bytes into an unbounded resource test
// rather than a parsing test. Filenames are still fuzzed so the suffix dispatch
// itself is covered.
func FuzzParseFile(f *testing.F) {
	f.Add("products.csv", []byte("sku,price\nABC-1,15000\n"))
	f.Add("products.CSV", []byte("sku,price\n"))
	f.Add("products.csv", []byte("\"unclosed,quote\n"))
	f.Add("products.csv", []byte("sku,price\nABC-1,15000,extra\n"))
	f.Add("products.csv", []byte("sku,sku\n1,2\n"))
	f.Add("products.xlsx", []byte("not really a workbook"))
	f.Add("no-extension", []byte("sku,price\nABC-1,15000\n"))

	f.Fuzz(func(t *testing.T, filename string, data []byte) {
		if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
			return // see doc comment: zip/XML resource cost is not the target here
		}

		rows, err := ParseFile(filename, bytes.NewReader(data), fuzzImportSchema())
		if err != nil {
			if rows != nil {
				t.Fatalf("ParseFile(%q) returned %d rows alongside error %v", filename, len(rows), err)
			}
			return
		}

		if len(rows) == 0 {
			t.Fatalf("ParseFile(%q) succeeded with no rows", filename)
		}
		for i, row := range rows {
			if row == nil {
				t.Fatalf("ParseFile(%q) row %d is nil", filename, i)
			}
			tag, ok := row["_row"]
			if !ok {
				t.Fatalf("ParseFile(%q) row %d has no _row marker: %v", filename, i, row)
			}
			line, ok := tag.(int)
			if !ok || line != i+2 {
				t.Fatalf("ParseFile(%q) row %d _row = %#v, want %d", filename, i, tag, i+2)
			}
			for key, value := range row {
				if key == "_row" {
					continue // int line marker, checked above
				}
				str, ok := value.(string)
				if !ok {
					t.Fatalf("ParseFile(%q) row %d column %q has non-string %#v", filename, i, key, value)
				}
				if str != strings.TrimSpace(str) {
					t.Fatalf("ParseFile(%q) row %d column %q kept untrimmed value %q", filename, i, key, str)
				}
			}
		}
	})
}
