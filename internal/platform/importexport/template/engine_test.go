package template

import (
	"bytes"
	"testing"

	"retail-pos-system/internal/platform/importexport/schema"
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
