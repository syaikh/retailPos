package schema

type ColumnType string

const (
	ColString    ColumnType = "string"
	ColNumber    ColumnType = "number"
	ColBoolean   ColumnType = "boolean"
	ColDate      ColumnType = "date"
	ColReference ColumnType = "reference"
)

type ReferencePolicy string

const (
	RefStrict     ReferencePolicy = "strict"
	RefAutoCreate ReferencePolicy = "auto_create"
	RefIgnore     ReferencePolicy = "ignore"
)

type ColumnSchema struct {
	Name          string       `json:"name"`
	Type          ColumnType   `json:"type"`
	Label         string       `json:"label"`
	Required      bool         `json:"required"`
	MaxLength     *int         `json:"max_length,omitempty"`
	MinValue      *float64     `json:"min_value,omitempty"`
	MaxValue      *float64     `json:"max_value,omitempty"`
	AllowedValues []string     `json:"allowed_values,omitempty"`
	Reference     string       `json:"reference,omitempty"`
	Default       interface{}  `json:"default,omitempty"`
	Description   string       `json:"description,omitempty"`
	Editable      bool         `json:"editable"`
	Exportable    bool         `json:"exportable"`
	Template      bool         `json:"template"`
	ImportGroup   string       `json:"import_group,omitempty"`
}

type ReferenceDef struct {
	Column          string          `json:"column"`
	ReferenceModule string          `json:"reference_module"`
	ReferenceColumn string          `json:"reference_column"`
	ReferenceLabel  string          `json:"reference_label"`
	Policy          ReferencePolicy `json:"policy"`
}

type ModuleFeatures struct {
	ImportEnabled   bool `json:"import_enabled"`
	ExportEnabled   bool `json:"export_enabled"`
	TemplateEnabled bool `json:"template_enabled"`
	PartialUpdate   bool `json:"partial_update"`
	MassInsert      bool `json:"mass_insert"`
	MassUpdate      bool `json:"mass_update"`
	SupportsPreview bool `json:"supports_preview"`
	SupportsHistory bool `json:"supports_history"`
}

type ModuleSchema struct {
	ModuleName    string           `json:"module_name"`
	DisplayName   string           `json:"display_name"`
	SchemaVersion string           `json:"schema_version"`
	Description   string           `json:"description"`
	Columns       []ColumnSchema   `json:"columns"`
	PrimaryKey    string           `json:"primary_key"`
	BusinessKeys  []string         `json:"business_keys"`
	References    []ReferenceDef   `json:"references"`
	Features      ModuleFeatures   `json:"features"`
}

func IntPtr(v int) *int {
	return &v
}

func Float64Ptr(v float64) *float64 {
	return &v
}
