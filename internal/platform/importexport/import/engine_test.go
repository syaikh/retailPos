package importer

import (
	"context"
	"strings"
	"testing"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/validation"
)

func TestEngine_Preview(t *testing.T) {
	reg := schema.NewRegistry()
	_ = reg.Register(testSchema)
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	e := NewEngine(reg, v, adapterReg, progEng)

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
	e := NewEngine(reg, v, adapterReg, progEng)

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
	e := NewEngine(reg, v, adapterReg, progEng)

	_, err := e.Execute(context.Background(), "bogus-token")
	if err == nil {
		t.Fatal("expected error: preview state not found")
	}
}
