package customer

import "retail-pos-system/internal/platform/importexport/schema"

var Schema = schema.ModuleSchema{
	ModuleName:    "customers",
	DisplayName:   "Customers",
	SchemaVersion: "1.0.0",
	Description:   "Customer data",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name", "Phone"},
	Columns: []schema.ColumnSchema{
		{Name: "Name", Type: schema.ColString, Label: "Customer Name", Required: true, MaxLength: schema.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Phone", Type: schema.ColString, Label: "Phone", Required: false, MaxLength: schema.IntPtr(20), Editable: true, Exportable: true, Template: true},
		{Name: "Email", Type: schema.ColString, Label: "Email", Required: false, MaxLength: schema.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Address", Type: schema.ColString, Label: "Address", Required: false, MaxLength: schema.IntPtr(500), Editable: true, Exportable: true, Template: true},
		{Name: "Note", Type: schema.ColString, Label: "Note", Required: false, MaxLength: schema.IntPtr(500), Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: schema.ColBoolean, Label: "Active", Required: false, Default: "true", Editable: true, Exportable: true, Template: true},
	},
	Features: schema.ModuleFeatures{
		ImportEnabled:   true,
		ExportEnabled:   true,
		TemplateEnabled: true,
		PartialUpdate:   true,
		MassInsert:      true,
		MassUpdate:      true,
		SupportsPreview: true,
		SupportsHistory: true,
	},
}
