package validation

import (
	"context"
	"testing"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

func TestDefaultPipeline_Run(t *testing.T) {
	p := NewDefaultPipeline()
	s := importexportshared.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
		Columns: []importexportshared.ColumnSchema{
			{Name: "Code", Type: importexportshared.ColString, Label: "Code", Required: true, Template: true},
			{Name: "Name", Type: importexportshared.ColString, Label: "Name", Required: true, Template: true},
		},
	}
	rows := []map[string]interface{}{
		{"Code": "A1", "Name": "One"},
		{"Code": "", "Name": "Two"},
	}
	errs := p.Run(context.Background(), s, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestDefaultPipeline_AllValid(t *testing.T) {
	p := NewDefaultPipeline()
	s := importexportshared.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
		Columns: []importexportshared.ColumnSchema{
			{Name: "Code", Type: importexportshared.ColString, Label: "Code", Required: true, Template: true},
			{Name: "Price", Type: importexportshared.ColNumber, Label: "Price", Required: false, Template: true},
		},
	}
	rows := []map[string]interface{}{
		{"Code": "A1", "Price": "100"},
		{"Code": "A2", "Price": "200"},
	}
	errs := p.Run(context.Background(), s, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestPipeline_AddCustomValidator(t *testing.T) {
	p := NewDefaultPipeline()
	p.Add(&mockValidator{name: "custom"})

	s := importexportshared.ModuleSchema{ModuleName: "test"}
	rows := []map[string]interface{}{{"x": "y"}}
	errs := p.Run(context.Background(), s, rows, nil)
	if len(errs) != 1 || errs[0].Reason != "custom error" {
		t.Fatalf("expected custom validator error, got %v", errs)
	}
}

type mockValidator struct {
	name string
}

func (m *mockValidator) Name() string { return m.name }

func (m *mockValidator) Validate(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}, _ map[string][]importexportshared.ReferenceItem) []importexportshared.ValidationError {
	return []importexportshared.ValidationError{{Reason: "custom error"}}
}
