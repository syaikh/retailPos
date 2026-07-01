package category

import "retail-pos-system/internal/platform/importexport/schema"

var Schema = schema.ModuleSchema{
	ModuleName:    "categories",
	DisplayName:   "Categories",
	SchemaVersion: "1.0.0",
	Description:   "Product categories",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name"},
	Columns: []schema.ColumnSchema{
		{Name: "Name", Type: schema.ColString, Label: "Category Name", Required: true, MaxLength: schema.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Slug", Type: schema.ColString, Label: "Slug", Required: false, MaxLength: schema.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Description", Type: schema.ColString, Label: "Description", Required: false, MaxLength: schema.IntPtr(500), Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: schema.ColBoolean, Label: "Active", Required: false, Default: "true", Editable: true, Exportable: true, Template: true},
	},
	Features: schema.ModuleFeatures{
		ImportEnabled:   true,
		ExportEnabled:   true,
		TemplateEnabled: true,
		PartialUpdate:   false,
		MassInsert:      true,
		MassUpdate:      true,
		SupportsPreview: true,
		SupportsHistory: true,
	},
}
