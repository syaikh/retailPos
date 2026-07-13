package importer

import (
	"context"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/validation"

	"github.com/xuri/excelize/v2"
)

func TestEngine_Preview(t *testing.T) {
	reg := schema.NewRegistry()
	_ = reg.Register(testSchema)
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	csv := "Code,Product Name,Price\nA1,Widget,100\n"
	result, err := e.Preview(context.Background(), "test", "data.csv", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}
	if result.TotalRows != 1 {
		t.Fatalf("TotalRows = %d, want 1", result.TotalRows)
	}
	if result.Module != "test" {
		t.Fatalf("Module = %q, want %q", result.Module, "test")
	}
}

func TestEngine_PreviewUnknownModule(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	_, err := e.Preview(context.Background(), "bogus", "data.csv", strings.NewReader("x\n1\n"))
	if err == nil {
		t.Fatal("expected error for unknown module")
	}
}

func TestEngine_ExecuteNoPreview(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	_, err := e.Execute(context.Background(), "bogus-token")
	if err == nil {
		t.Fatal("expected error: preview state not found")
	}
}

func TestRandomHex_Length(t *testing.T) {
	result := randomHex(16)
	if len(result) != 32 {
		t.Fatalf("expected 32 hex chars, got %d", len(result))
	}
}

func TestRandomHex_Unique(t *testing.T) {
	a := randomHex(16)
	b := randomHex(16)
	if a == b {
		t.Fatal("expected unique random values")
	}
}

func TestStoreAndGetPreview(t *testing.T) {
	reg := schema.NewRegistry()
	_ = reg.Register(testSchema)
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	state := &PreviewState{
		Module:  "test",
		Created: time.Now(),
	}
	e.StorePreview("test-token", state)

	got := e.GetPreview("test-token")
	if got == nil {
		t.Fatal("expected to find preview state")
	}
	if got.Module != "test" {
		t.Fatalf("expected module 'test', got %q", got.Module)
	}
}

func TestGetPreview_NotFound(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	got := e.GetPreview("nonexistent")
	if got != nil {
		t.Fatal("expected nil for nonexistent token")
	}
}

func TestGetPreview_Expired(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	state := &PreviewState{
		Module:  "test",
		Created: time.Now().Add(-31 * time.Minute), // expired (TTL is 30 min)
	}
	e.StorePreview("expired-token", state)

	got := e.GetPreview("expired-token")
	if got != nil {
		t.Fatal("expected nil for expired preview")
	}
}

func TestDeletePreview(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	state := &PreviewState{
		Module:  "test",
		Created: time.Now(),
	}
	e.StorePreview("delete-me", state)
	e.DeletePreview("delete-me")

	got := e.GetPreview("delete-me")
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestEngine_Close(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	e.Close()
	// Should not panic
}

func TestEngine_PreviewEmptyRows(t *testing.T) {
	reg := schema.NewRegistry()
	_ = reg.Register(testSchema)
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	csv := "Code,Product Name,Price\n"
	_, err := e.Preview(context.Background(), "test", "data.csv", strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for header-only CSV")
	}
}

func TestValidateMetaSheet_NoMetaSheet(t *testing.T) {
	wb := excelize.NewFile()
	s := schema.ModuleSchema{SchemaVersion: "1.0.0"}
	err := validateMetaSheet(wb, s)
	if err != nil {
		t.Fatalf("expected nil for no _Meta sheet, got %v", err)
	}
}

func TestValidateMetaSheet_MatchingVersion(t *testing.T) {
	wb := excelize.NewFile()
	wb.SetSheetName("Sheet1", "_Meta")
	wb.SetCellValue("_Meta", "A1", "SchemaVersion")
	wb.SetCellValue("_Meta", "B1", "1.0.0")

	s := schema.ModuleSchema{SchemaVersion: "1.0.0"}
	err := validateMetaSheet(wb, s)
	if err != nil {
		t.Fatalf("expected nil for matching version, got %v", err)
	}
}

func TestValidateMetaSheet_MismatchedVersion(t *testing.T) {
	wb := excelize.NewFile()
	wb.SetSheetName("Sheet1", "_Meta")
	wb.SetCellValue("_Meta", "A1", "SchemaVersion")
	wb.SetCellValue("_Meta", "B1", "2.0.0")

	s := schema.ModuleSchema{SchemaVersion: "1.0.0"}
	err := validateMetaSheet(wb, s)
	if err == nil {
		t.Fatal("expected error for mismatched version")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected 'mismatch' in error, got %q", err.Error())
	}
}

func TestValidateMetaSheet_NoVersionRow(t *testing.T) {
	wb := excelize.NewFile()
	wb.SetSheetName("Sheet1", "_Meta")
	wb.SetCellValue("_Meta", "A1", "SomethingElse")
	wb.SetCellValue("_Meta", "B1", "1.0.0")

	s := schema.ModuleSchema{SchemaVersion: "1.0.0"}
	err := validateMetaSheet(wb, s)
	if err != nil {
		t.Fatalf("expected nil when no SchemaVersion row, got %v", err)
	}
}

func TestEngine_StartImport_NoPreview(t *testing.T) {
	reg := schema.NewRegistry()
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	_, err := e.StartImport(context.Background(), "bogus-token", 1, 1)
	if err == nil {
		t.Fatal("expected error for nonexistent token")
	}
}

func TestEngine_StartImport_NoAdapter(t *testing.T) {
	reg := schema.NewRegistry()
	_ = reg.Register(testSchema)
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng, nil)

	e.StorePreview("token-no-adapter", &PreviewState{
		Module:   "test",
		Schema:   testSchema,
		Rows:     []map[string]interface{}{},
		Result:   &importexport.PreviewResult{TotalRows: 0},
		Created:  time.Now(),
	})

	_, err := e.StartImport(context.Background(), "token-no-adapter", 1, 1)
	if err == nil {
		t.Fatal("expected error for missing adapter")
	}
	if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("expected 'adapter' in error, got %q", err.Error())
	}
}
