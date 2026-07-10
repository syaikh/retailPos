package template

import (
	"bytes"
	"testing"

	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/xuri/excelize/v2"
)

var testSchema = schema.ModuleSchema{
	ModuleName:    "products",
	DisplayName:   "Products",
	SchemaVersion: "1.0.0",
	Description:   "Test schema",
	Columns: []schema.ColumnSchema{
		{Name: "Name", Type: schema.ColString, Label: "Product Name", Required: true, Template: true},
		{Name: "Price", Type: schema.ColNumber, Label: "Price", Required: true, Template: true},
		{Name: "Active", Type: schema.ColBoolean, Label: "Active", Required: false, Template: true},
		{Name: "Brand", Type: schema.ColReference, Label: "Brand", Required: false, Reference: "brands", Template: true},
		{Name: "InternalNote", Type: schema.ColString, Label: "Internal Note", Required: false, Exportable: true, Template: false},
	},
	References: []schema.ReferenceDef{
		{
			Column: "Brand", ReferenceModule: "brands",
			ReferenceColumn: "Name", ReferenceLabel: "Brand Name",
			Policy: schema.RefStrict,
		},
	},
}

func TestEngine_Generate(t *testing.T) {
	e := NewEngine()
	refData := map[string][]string{
		"brands": {"Nike", "Adidas", "Puma"},
	}

	var buf bytes.Buffer
	err := e.Generate(testSchema, refData, &buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty xlsx output")
	}
}

func TestEngine_GenerateNoRefs(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName:    "simple",
		DisplayName:   "Simple",
		SchemaVersion: "1.0.0",
		Columns: []schema.ColumnSchema{
			{Name: "Name", Type: schema.ColString, Label: "Name", Required: true, Template: true},
		},
	}

	var buf bytes.Buffer
	err := e.Generate(s, nil, &buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty xlsx output")
	}
}

func TestEngine_GenerateStatusDropdown(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName:    "test",
		DisplayName:   "Test",
		SchemaVersion: "1.0.0",
		Columns: []schema.ColumnSchema{
			{Name: "Name", Type: schema.ColString, Label: "Name", Required: true, Template: true},
			{Name: "Status",
				Type:          schema.ColString,
				Label:         "Status",
				Required:      false,
				AllowedValues: []string{"active", "inactive"},
				Template:      true,
			},
		},
	}

	var buf bytes.Buffer
	err := e.Generate(s, nil, &buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	wb, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer wb.Close()

	hasRefStatus := false
	for _, sheet := range wb.GetSheetList() {
		if sheet == "Ref_Status" {
			hasRefStatus = true
			break
		}
	}
	if !hasRefStatus {
		t.Fatal("expected Ref_Status sheet for AllowedValues dropdown")
	}

	rows, err := wb.GetRows("Ref_Status")
	if err != nil {
		t.Fatalf("read Ref_Status: %v", err)
	}
	if len(rows) < 2 {
		t.Fatal("expected at least header + 1 value in Ref_Status")
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header + 2 values), got %d", len(rows))
	}
	if rows[1][0] != "active" || rows[2][0] != "inactive" {
		t.Fatalf("unexpected values: %v", rows)
	}
}

func TestEngine_GenerateEmptyRefData(t *testing.T) {
	e := NewEngine()
	s := testSchema
	refData := map[string][]string{"brands": {}}

	var buf bytes.Buffer
	err := e.Generate(s, refData, &buf)
	if err != nil {
		t.Fatalf("Generate with empty ref data failed: %v", err)
	}
}

func TestEngine_GenerateWritesToWriter(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName:    "test",
		DisplayName:   "Test",
		SchemaVersion: "1.0.0",
		Columns: []schema.ColumnSchema{
			{Name: "A", Type: schema.ColString, Label: "A", Required: true, Template: true},
		},
	}

	var buf bytes.Buffer
	if err := e.Generate(s, nil, &buf); err != nil {
		t.Fatal(err)
	}

	if buf.Len() < 100 {
		t.Fatalf("xlsx too small: %d bytes", buf.Len())
	}
}
