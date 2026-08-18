package export

import (
	"bytes"
	"strings"
	"testing"

	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/shared"
)

var testSchema = schema.ModuleSchema{
	ModuleName:    "products",
	DisplayName:   "Products",
	SchemaVersion: "1.0.0",
	Columns: []schema.ColumnSchema{
		{Name: "Name", Type: schema.ColString, Label: "Product Name", Exportable: true},
		{Name: "Price", Type: schema.ColNumber, Label: "Price", Exportable: true},
		{Name: "Active", Type: schema.ColBoolean, Label: "Active", Exportable: true},
		{Name: "InternalNote", Type: schema.ColString, Label: "Internal Note", Exportable: false},
	},
}

var testData = []map[string]interface{}{
	{"Name": "Widget", "Price": "100", "Active": "true"},
	{"Name": "Gadget", "Price": "200.50", "Active": "false"},
}

func TestExportCSV(t *testing.T) {
	e := NewEngine()
	var buf bytes.Buffer
	err := e.Export(&buf, testSchema, testData, FormatCSV)
	if err != nil {
		t.Fatalf("Export CSV failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Product Name") {
		t.Error("missing header 'Product Name'")
	}
	if !strings.Contains(output, "Widget") {
		t.Error("missing data 'Widget'")
	}
	if !strings.Contains(output, "200.5") {
		t.Error("missing data '200.50'")
	}

	if strings.Contains(output, "Internal Note") {
		t.Error("non-exportable column should not appear")
	}
}

func TestExportCSVFormulaInjection(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		Columns: []schema.ColumnSchema{
			{Name: "Name", Type: schema.ColString, Label: "Name", Exportable: true},
		},
	}
	data := []map[string]interface{}{
		{"Name": "=SUM(1,1)"},
	}
	var buf bytes.Buffer
	err := e.Export(&buf, s, data, FormatCSV)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "'=SUM") {
		t.Errorf("expected sanitized output with leading quote, got: %q", output)
	}
}

func TestExportCSVEmptyData(t *testing.T) {
	e := NewEngine()
	var buf bytes.Buffer
	err := e.Export(&buf, testSchema, []map[string]interface{}{}, FormatCSV)
	if err != nil {
		t.Fatalf("Export CSV failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Product Name") {
		t.Error("missing header row")
	}
}

func TestExportXLSX(t *testing.T) {
	e := NewEngine()
	var buf bytes.Buffer
	err := e.Export(&buf, testSchema, testData, FormatXLSX)
	if err != nil {
		t.Fatalf("Export XLSX failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty xlsx output")
	}
}

func TestExportXLSXEmptyData(t *testing.T) {
	e := NewEngine()
	var buf bytes.Buffer
	err := e.Export(&buf, testSchema, []map[string]interface{}{}, FormatXLSX)
	if err != nil {
		t.Fatalf("Export XLSX empty failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty xlsx output with header")
	}
}

func TestExportInvalidFormat(t *testing.T) {
	e := NewEngine()
	var buf bytes.Buffer
	err := e.Export(&buf, testSchema, testData, Format("xml"))
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		t    schema.ColumnType
		v    interface{}
		want string
	}{
		{schema.ColNumber, "100", "100"},
		{schema.ColNumber, "100.5", "100.5"},
		{schema.ColBoolean, "true", "true"},
		{schema.ColBoolean, "yes", "true"},
		{schema.ColBoolean, "0", "false"},
		{schema.ColString, "hello", "hello"},
		{schema.ColString, nil, ""},
	}
	for _, tt := range tests {
		got := formatValue(tt.t, tt.v)
		if got != tt.want {
			t.Errorf("formatValue(%v, %v) = %q, want %q", tt.t, tt.v, got, tt.want)
		}
	}
}

func TestSanitizeCSVField(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"=SUM(1,1)", "'=SUM(1,1)"},
		{"+cmd", "'+cmd"},
		{"@echo", "'@echo"},
		{"-cmd", "'-cmd"},
		{"normal", "normal"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shared.SanitizeCSVField(tt.in)
		if got != tt.want {
			t.Errorf("SanitizeCSVField(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
