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

func TestHeaderStyleFor(t *testing.T) {
	e := NewEngine()
	cs := columnStyles{required: 1, optional: 2, ref: 3, readonly: 4}

	tests := []struct {
		col  schema.ColumnSchema
		want int
	}{
		{schema.ColumnSchema{Editable: false}, 4},
		{schema.ColumnSchema{Editable: true, Required: true}, 1},
		{schema.ColumnSchema{Editable: true, Required: false, Reference: "brands"}, 3},
		{schema.ColumnSchema{Editable: true, Required: false, Reference: ""}, 2},
	}
	for _, tt := range tests {
		got := e.headerStyleFor(cs, tt.col)
		if got != tt.want {
			t.Errorf("headerStyleFor(%+v) = %d, want %d", tt.col, got, tt.want)
		}
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a, b"},
		{[]string{"a", "b", "c"}, "a, b, c"},
	}
	for _, tt := range tests {
		got := joinStrings(tt.input)
		if got != tt.expected {
			t.Errorf("joinStrings(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestColumnIndex(t *testing.T) {
	cols := []schema.ColumnSchema{
		{Name: "A"}, {Name: "B"}, {Name: "C"},
	}
	if got := columnIndex(cols, "B"); got != 1 {
		t.Errorf("columnIndex(B) = %d, want 1", got)
	}
	if got := columnIndex(cols, "X"); got != -1 {
		t.Errorf("columnIndex(X) = %d, want -1", got)
	}
}

func TestColWidth(t *testing.T) {
	tests := []struct {
		typ  schema.ColumnType
		want float64
		ok   bool
	}{
		{schema.ColString, 35, true},
		{schema.ColNumber, 15, true},
		{schema.ColBoolean, 12, true},
		{schema.ColDate, 15, true},
		{schema.ColReference, 25, true},
		{"unknown", 0, false},
	}
	for _, tt := range tests {
		w, ok := colWidth(tt.typ)
		if w != tt.want || ok != tt.ok {
			t.Errorf("colWidth(%q) = (%v, %v), want (%v, %v)", tt.typ, w, ok, tt.want, tt.ok)
		}
	}
}

func TestBuildValidationHint(t *testing.T) {
	intP := func(v int) *int { return &v }
	floatP := func(v float64) *float64 { return &v }

	tests := []struct {
		name string
		col  schema.ColumnSchema
		need string
	}{
		{"string no max", schema.ColumnSchema{Type: schema.ColString}, ""},
		{"string with max", schema.ColumnSchema{Type: schema.ColString, MaxLength: intP(50)}, "Max 50 characters"},
		{"number range", schema.ColumnSchema{Type: schema.ColNumber, MinValue: floatP(0), MaxValue: floatP(100)}, "Number 0 – 100"},
		{"number min only", schema.ColumnSchema{Type: schema.ColNumber, MinValue: floatP(5)}, "Number >= 5"},
		{"number max only", schema.ColumnSchema{Type: schema.ColNumber, MaxValue: floatP(50)}, "Number <= 50"},
		{"number no bounds", schema.ColumnSchema{Type: schema.ColNumber}, "Numeric value"},
		{"boolean", schema.ColumnSchema{Type: schema.ColBoolean}, "Yes/No or True/False"},
		{"date", schema.ColumnSchema{Type: schema.ColDate}, "Date format YYYY-MM-DD"},
		{"reference", schema.ColumnSchema{Type: schema.ColReference}, "Must exist in system"},
		{"empty type", schema.ColumnSchema{}, ""},
		{"with allowed", schema.ColumnSchema{Type: schema.ColString, AllowedValues: []string{"a", "b"}}, "Allowed: a, b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildValidationHint(tt.col)
			if got != tt.need {
				t.Errorf("buildValidationHint = %q, want %q", got, tt.need)
			}
		})
	}
}

func TestEngine_GenerateMetaNoDisplayName(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName: "nodisplay",
		Columns:    []schema.ColumnSchema{{Name: "X", Type: schema.ColString, Label: "X", Required: true, Template: true}},
	}
	var buf bytes.Buffer
	if err := e.Generate(s, nil, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

func TestEngine_GenerateRefDataNotFound(t *testing.T) {
	e := NewEngine()
	s := testSchema
	refData := map[string][]string{"other_module": {"val1"}}

	var buf bytes.Buffer
	if err := e.Generate(s, refData, &buf); err != nil {
		t.Fatalf("Generate with mismatched ref data failed: %v", err)
	}
}

func TestEngine_GenerateWithDescription(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName:    "desc",
		DisplayName:   "Desc",
		SchemaVersion: "1.0.0",
		Description:   "A module description",
		Columns:       []schema.ColumnSchema{{Name: "X", Type: schema.ColString, Label: "X", Required: true, Template: true}},
	}
	var buf bytes.Buffer
	if err := e.Generate(s, nil, &buf); err != nil {
		t.Fatal(err)
	}
}

func TestEngine_GenerateRefColumnNotInTemplate(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName:    "ref",
		DisplayName:   "Ref",
		SchemaVersion: "1.0.0",
		Columns:       []schema.ColumnSchema{{Name: "X", Type: schema.ColString, Label: "X", Required: true, Template: true}},
		References: []schema.ReferenceDef{
			{Column: "MissingCol", ReferenceModule: "brands", ReferenceLabel: "Brand"},
		},
	}
	refData := map[string][]string{"brands": {"Nike"}}
	var buf bytes.Buffer
	if err := e.Generate(s, refData, &buf); err != nil {
		t.Fatal(err)
	}
}

func TestEngine_GenerateAllowedValuesIsRef(t *testing.T) {
	e := NewEngine()
	s := schema.ModuleSchema{
		ModuleName:    "refav",
		DisplayName:   "RefAV",
		SchemaVersion: "1.0.0",
		Columns: []schema.ColumnSchema{
			{Name: "Brand", Type: schema.ColReference, Label: "Brand", AllowedValues: []string{"Nike"}, Template: true},
		},
		References: []schema.ReferenceDef{
			{Column: "Brand", ReferenceModule: "brands", ReferenceLabel: "Brand"},
		},
	}
	refData := map[string][]string{"brands": {"Nike"}}
	var buf bytes.Buffer
	if err := e.Generate(s, refData, &buf); err != nil {
		t.Fatal(err)
	}
}
