package importer

import (
	"testing"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
)

func TestGeneratePreview_AllInsert(t *testing.T) {
	s := schema.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
		Columns: []schema.ColumnSchema{
			{Name: "Code", Type: schema.ColString},
		},
	}
	rows := []map[string]interface{}{
		{"Code": "A1"},
		{"Code": "A2"},
	}
	result := GeneratePreview(s, rows, nil)
	if result.TotalRows != 2 {
		t.Fatalf("TotalRows = %d, want 2", result.TotalRows)
	}
	if result.InsertCount != 2 {
		t.Fatalf("InsertCount = %d, want 2", result.InsertCount)
	}
	if result.UpdateCount != 0 {
		t.Fatalf("UpdateCount = %d, want 0", result.UpdateCount)
	}
	if result.ErrorCount != 0 {
		t.Fatalf("ErrorCount = %d, want 0", result.ErrorCount)
	}
}

func TestGeneratePreview_WithDuplicates(t *testing.T) {
	s := schema.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
	}
	rows := []map[string]interface{}{
		{"Code": "A1"},
		{"Code": "A1"},
	}
	result := GeneratePreview(s, rows, nil)
	if result.InsertCount != 1 {
		t.Fatalf("InsertCount = %d, want 1 (first occurrence)", result.InsertCount)
	}
	if result.UpdateCount != 1 {
		t.Fatalf("UpdateCount = %d, want 1 (duplicate becomes update)", result.UpdateCount)
	}
}

func TestGeneratePreview_WithErrors(t *testing.T) {
	s := schema.ModuleSchema{
		ModuleName:   "test",
		BusinessKeys: []string{"Code"},
	}
	rows := []map[string]interface{}{
		{"Code": "A1"},
		{"Code": ""},
	}
	errs := []importexportshared.ValidationError{
		{Row: 3, Field: "Code", Reason: "required"},
	}
	result := GeneratePreview(s, rows, errs)
	if result.ErrorCount != 1 {
		t.Fatalf("ErrorCount = %d, want 1", result.ErrorCount)
	}
	if result.Rows[1].Status != "error" {
		t.Fatalf("row 2 status = %q, want %q", result.Rows[1].Status, "error")
	}
}

func TestGeneratePreview_NoBusinessKeys(t *testing.T) {
	s := schema.ModuleSchema{
		ModuleName: "test",
	}
	rows := []map[string]interface{}{
		{"Name": "A"},
		{"Name": "B"},
	}
	result := GeneratePreview(s, rows, nil)
	if result.InsertCount != 2 {
		t.Fatalf("InsertCount = %d, want 2", result.InsertCount)
	}
}
