package importer

import (
	"bytes"
	"strings"
	"testing"

	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/xuri/excelize/v2"
)

var testSchema = schema.ModuleSchema{
	ModuleName: "test",
	Columns: []schema.ColumnSchema{
		{Name: "Code", Type: schema.ColString, Label: "Code"},
		{Name: "Name", Type: schema.ColString, Label: "Product Name"},
		{Name: "Price", Type: schema.ColNumber, Label: "Price"},
	},
}

func TestParseCSV(t *testing.T) {
	csv := "Code,Product Name,Price\nA1,Widget,100\nA2,Gadget,200\n"
	rows, err := ParseFile("data.csv", strings.NewReader(csv), testSchema)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0]["Code"] != "A1" {
		t.Errorf("rows[0][Code] = %q, want %q", rows[0]["Code"], "A1")
	}
	if rows[1]["Name"] != "Gadget" {
		t.Errorf("rows[1][Name] = %q, want %q", rows[1]["Name"], "Gadget")
	}
}

func TestParseCSVEmptyFile(t *testing.T) {
	_, err := ParseFile("data.csv", strings.NewReader(""), testSchema)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseCSVHeaderOnly(t *testing.T) {
	_, err := ParseFile("data.csv", strings.NewReader("Code,Name\n"), testSchema)
	if err == nil {
		t.Fatal("expected error for header-only file")
	}
}

func TestParseXLSX(t *testing.T) {
	wb := excelize.NewFile()
	_ = wb.SetCellValue("Sheet1", "A1", "Code")
	_ = wb.SetCellValue("Sheet1", "B1", "Product Name")
	_ = wb.SetCellValue("Sheet1", "C1", "Price")
	_ = wb.SetCellValue("Sheet1", "A2", "A1")
	_ = wb.SetCellValue("Sheet1", "B2", "Widget")
	_ = wb.SetCellValue("Sheet1", "C2", "100")
	_ = wb.SetCellValue("Sheet1", "A3", "A2")
	_ = wb.SetCellValue("Sheet1", "B3", "Gadget")
	_ = wb.SetCellValue("Sheet1", "C3", "200")

	var buf bytes.Buffer
	if err := wb.Write(&buf); err != nil {
		t.Fatalf("create test xlsx: %v", err)
	}

	rows, err := ParseFile("data.xlsx", &buf, testSchema)
	if err != nil {
		t.Fatalf("ParseFile xlsx failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0]["Code"] != "A1" {
		t.Errorf("rows[0][Code] = %q, want %q", rows[0]["Code"], "A1")
	}
}

func TestParseXLSXEmpty(t *testing.T) {
	wb := excelize.NewFile()
	_ = wb.SetCellValue("Sheet1", "A1", "Code")

	var buf bytes.Buffer
	_ = wb.Write(&buf)

	_, err := ParseFile("data.xlsx", &buf, testSchema)
	if err == nil {
		t.Fatal("expected error for header-only xlsx")
	}
}
