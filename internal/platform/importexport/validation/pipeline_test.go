package validation

import (
	"context"
	"testing"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

func TestDefaultPipeline_Run(t *testing.T) {
	p := NewDefaultPipeline()
	s := schema.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
		Columns: []schema.ColumnSchema{
			{Name: "Code", Type: schema.ColString, Label: "Code", Required: true, Template: true},
			{Name: "Name", Type: schema.ColString, Label: "Name", Required: true, Template: true},
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
	s := schema.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
		Columns: []schema.ColumnSchema{
			{Name: "Code", Type: schema.ColString, Label: "Code", Required: true, Template: true},
			{Name: "Price", Type: schema.ColNumber, Label: "Price", Required: false, Template: true},
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

	s := schema.ModuleSchema{ModuleName: "test"}
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

func (m *mockValidator) Validate(_ context.Context, _ schema.ModuleSchema, _ []map[string]interface{}, _ map[string][]importexport.ReferenceItem) []importexport.ValidationError {
	return []importexport.ValidationError{{Reason: "custom error"}}
}
