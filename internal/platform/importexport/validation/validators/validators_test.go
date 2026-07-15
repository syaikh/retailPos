package validators

import (
	"context"
	"testing"

	importexportshared "retail-pos-system/internal/shared/importexport"
)

type named interface {
	Name() string
}

var testSchema = importexportshared.ModuleSchema{
	ModuleName:   "test",
	BusinessKeys: []string{"Code"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Code", Type: importexportshared.ColString, Label: "Code", Required: true, MaxLength: importexportshared.IntPtr(10), Template: true},
		{Name: "Name", Type: importexportshared.ColString, Label: "Name", Required: true, MaxLength: importexportshared.IntPtr(50), Template: true},
		{Name: "Price", Type: importexportshared.ColNumber, Label: "Price", Required: false, MinValue: importexportshared.Float64Ptr(0), Template: true},
		{Name: "Active", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Template: true},
		{Name: "Brand", Type: importexportshared.ColReference, Label: "Brand", Required: false, Reference: "brands", Template: true},
		{Name: "Date", Type: importexportshared.ColDate, Label: "Date", Required: false, Template: true},
	},
}

func ctx() context.Context {
	return context.Background()
}

func TestFileValidator_EmptyRows(t *testing.T) {
	v := &FileValidator{}
	errs := v.Validate(ctx(), testSchema, []map[string]interface{}{}, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for empty rows")
	}
}

func TestFileValidator_NonEmptyRows(t *testing.T) {
	v := &FileValidator{}
	errs := v.Validate(ctx(), testSchema, []map[string]interface{}{{"Code": "1"}}, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestTemplateValidator_MissingRequiredColumn(t *testing.T) {
	v := &TemplateValidator{}
	rows := []map[string]interface{}{{"Name": "test"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for missing required column")
	}
}

func TestTemplateValidator_AllColumnsPresent(t *testing.T) {
	v := &TemplateValidator{}
	rows := []map[string]interface{}{{"Code": "1", "Name": "test", "Price": "100", "Active": "true", "Brand": "Nike", "Date": "2024-01-01"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestTypeValidator_NumberOverMax(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Price": "100", "Code": "x"}}
	s := testSchema
	s.Columns[2].MaxValue = importexportshared.Float64Ptr(50)
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for price over max")
	}
}

func TestTypeValidator_InvalidNumber(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Price": "not-a-number", "Code": "x"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid number")
	}
}

func TestTypeValidator_InvalidBoolean(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Active": "not-bool", "Code": "x"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid boolean")
	}
}

func TestTypeValidator_ValidBoolean(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Active": "true", "Code": "x"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestTypeValidator_StringMaxLength(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Code": "12345678901"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for string exceeding max length")
	}
}

func TestTypeValidator_Date(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Date": "2024-01-01", "Code": "x"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestTypeValidator_InvalidDate(t *testing.T) {
	v := &TypeValidator{}
	rows := []map[string]interface{}{{"Date": "not-a-date", "Code": "x"}}
	errs := v.Validate(ctx(), testSchema, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid date")
	}
}

func TestTypeValidator_AllowedValues(t *testing.T) {
	v := &TypeValidator{}
	s := testSchema
	s.Columns = append(s.Columns, importexportshared.ColumnSchema{
		Name: "Status", Type: importexportshared.ColString, AllowedValues: []string{"active", "inactive"},
	})

	t.Run("valid", func(t *testing.T) {
		rows := []map[string]interface{}{{"Status": "active", "Code": "x"}}
		errs := v.Validate(ctx(), s, rows, nil)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		rows := []map[string]interface{}{{"Status": "bogus", "Code": "x"}}
		errs := v.Validate(ctx(), s, rows, nil)
		if len(errs) == 0 {
			t.Fatal("expected error for disallowed value")
		}
	})
}

func TestRequiredValidator(t *testing.T) {
	v := &RequiredValidator{}

	t.Run("missing", func(t *testing.T) {
		rows := []map[string]interface{}{{"Code": ""}}
		errs := v.Validate(ctx(), testSchema, rows, nil)
		if len(errs) == 0 {
			t.Fatal("expected error for missing required field")
		}
	})

	t.Run("present", func(t *testing.T) {
		rows := []map[string]interface{}{{"Code": "X1", "Name": "Test"}}
		errs := v.Validate(ctx(), testSchema, rows, nil)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})
}

func TestReferenceValidator(t *testing.T) {
	v := &ReferenceValidator{}
	brandRefs := map[string][]importexportshared.ReferenceItem{
		"brands": {{Key: "Nike"}, {Key: "Adidas"}},
	}

	t.Run("found", func(t *testing.T) {
		rows := []map[string]interface{}{{"Brand": "Nike", "Code": "x"}}
		errs := v.Validate(ctx(), testSchema, rows, brandRefs)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})

	t.Run("not found", func(t *testing.T) {
		rows := []map[string]interface{}{{"Brand": "Puma", "Code": "x"}}
		errs := v.Validate(ctx(), testSchema, rows, brandRefs)
		if len(errs) == 0 {
			t.Fatal("expected error for unknown reference")
		}
	})

	t.Run("empty value skipped", func(t *testing.T) {
		rows := []map[string]interface{}{{"Brand": "", "Code": "x"}}
		errs := v.Validate(ctx(), testSchema, rows, brandRefs)
		if len(errs) != 0 {
			t.Fatalf("expected no errors for empty reference, got %v", errs)
		}
	})
}

func TestDuplicateValidator(t *testing.T) {
	v := &DuplicateValidator{}

	t.Run("no duplicates", func(t *testing.T) {
		rows := []map[string]interface{}{
			{"Code": "A1", "Name": "One"},
			{"Code": "B2", "Name": "Two"},
		}
		errs := v.Validate(ctx(), testSchema, rows, nil)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})

	t.Run("duplicate business key", func(t *testing.T) {
		rows := []map[string]interface{}{
			{"Code": "A1", "Name": "One"},
			{"Code": "A1", "Name": "Duplicate"},
		}
		errs := v.Validate(ctx(), testSchema, rows, nil)
		if len(errs) == 0 {
			t.Fatal("expected error for duplicate business key")
		}
	})
}

func TestDuplicateValidator_NoBusinessKeys(t *testing.T) {
	v := &DuplicateValidator{}
	s := testSchema
	s.BusinessKeys = nil

	rows := []map[string]interface{}{
		{"Code": "A1"},
		{"Code": "A1"},
	}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors when no business keys, got %v", errs)
	}
}

func TestValidatorNames(t *testing.T) {
	tests := []struct {
		v    named
		name string
	}{
		{&DuplicateValidator{}, "duplicate"},
		{&FileValidator{}, "file"},
		{&ReferenceValidator{}, "reference"},
		{&RequiredValidator{}, "required"},
		{&TemplateValidator{}, "template"},
		{&TypeValidator{}, "type"},
	}
	for _, tt := range tests {
		if got := tt.v.Name(); got != tt.name {
			t.Errorf("%T.Name() = %q, want %q", tt.v, got, tt.name)
		}
	}
}

func TestTemplateValidator_EmptyRows(t *testing.T) {
	v := &TemplateValidator{}
	errs := v.Validate(ctx(), testSchema, nil, nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for nil rows, got %v", errs)
	}
}

func TestTemplateValidator_NonRequiredColumnMissing(t *testing.T) {
	v := &TemplateValidator{}
	s := testSchema
	s.Columns[2].Required = false // Price is non-required

	rows := []map[string]interface{}{{"Code": "x", "Name": "test", "Date": "2024-01-01"}}
	errs := v.Validate(ctx(), s, rows, nil)
	for _, e := range errs {
		if e.Field == "Price" {
			t.Fatalf("non-required missing column should not produce error, got %v", e)
		}
	}
}

func TestTemplateValidator_NonTemplateColumnSkipped(t *testing.T) {
	v := &TemplateValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Code", Type: importexportshared.ColString, Label: "Code", Required: true, Template: true},
			{Name: "Hidden", Type: importexportshared.ColString, Label: "Hidden", Required: true, Template: false},
		},
	}
	rows := []map[string]interface{}{{"Code": "x"}}
	errs := v.Validate(ctx(), s, rows, nil)
	for _, e := range errs {
		if e.Field == "Hidden" {
			t.Fatalf("non-template column should be skipped, got error %v", e)
		}
	}
}

func TestReferenceValidator_EmptyReferenceSkip(t *testing.T) {
	v := &ReferenceValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Brand", Type: importexportshared.ColReference, Label: "Brand", Reference: ""},
		},
	}
	rows := []map[string]interface{}{{"Brand": "Nike"}}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("empty reference should be skipped, got %v", errs)
	}
}

func TestReferenceValidator_LabelFallback(t *testing.T) {
	v := &ReferenceValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Brand", Type: importexportshared.ColReference, Label: "Brand Label", Reference: "brands"},
		},
	}
	refs := map[string][]importexportshared.ReferenceItem{
		"brands": {{Key: "Nike"}},
	}
	rows := []map[string]interface{}{{"Brand Label": "Nike"}}
	errs := v.Validate(ctx(), s, rows, refs)
	if len(errs) != 0 {
		t.Fatalf("label fallback should find value, got %v", errs)
	}
}

func TestReferenceValidator_MissingRefData(t *testing.T) {
	v := &ReferenceValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Brand", Type: importexportshared.ColReference, Label: "Brand", Reference: "brands"},
		},
	}
	refs := map[string][]importexportshared.ReferenceItem{}
	rows := []map[string]interface{}{{"Brand": "Nike"}}
	errs := v.Validate(ctx(), s, rows, refs)
	if len(errs) == 0 {
		t.Fatal("expected error when reference data not loaded")
	}
}

func TestDuplicateValidator_NilValues(t *testing.T) {
	v := &DuplicateValidator{}
	s := testSchema
	rows := []map[string]interface{}{
		{"Code": nil, "Name": "One"},
		{"Code": nil, "Name": "Two"},
	}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("nil business key values should be skipped, got %v", errs)
	}
}

func TestTypeValidator_LabelFallback(t *testing.T) {
	v := &TypeValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Price", Type: importexportshared.ColNumber, Label: "Price Label"},
		},
	}
	rows := []map[string]interface{}{{"Price Label": "not-a-number"}}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) == 0 {
		t.Fatal("label fallback should validate type")
	}
}

func TestTypeValidator_NumericDate(t *testing.T) {
	v := &TypeValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Date", Type: importexportshared.ColDate, Label: "Date"},
		},
	}
	rows := []map[string]interface{}{{"Date": "2024"}}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) == 0 {
		t.Fatal("numeric date should be rejected")
	}
}

func TestTypeValidator_NumberBothMinMax(t *testing.T) {
	v := &TypeValidator{}
	s := testSchema
	s.Columns[2].MinValue = importexportshared.Float64Ptr(0)
	s.Columns[2].MaxValue = importexportshared.Float64Ptr(100)

	rows := []map[string]interface{}{{"Price": "200", "Code": "x"}}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for number exceeding both min and max")
	}
}

func TestTypeValidator_NumberNoMinMax(t *testing.T) {
	v := &TypeValidator{}
	s := testSchema
	s.Columns[2].MinValue = nil
	s.Columns[2].MaxValue = nil

	rows := []map[string]interface{}{{"Price": "999999", "Code": "x"}}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("number with no min/max should pass, got %v", errs)
	}
}

func TestTypeValidator_StringNoMaxLength(t *testing.T) {
	v := &TypeValidator{}
	s := importexportshared.ModuleSchema{
		ModuleName: "test",
		Columns: []importexportshared.ColumnSchema{
			{Name: "Code", Type: importexportshared.ColString, Label: "Code"},
		},
	}
	rows := []map[string]interface{}{{"Code": "very long string that should pass"}}
	errs := v.Validate(ctx(), s, rows, nil)
	if len(errs) != 0 {
		t.Fatalf("string with no max length should pass, got %v", errs)
	}
}
