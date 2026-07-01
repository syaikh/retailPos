package uom

import "retail-pos-system/internal/platform/importexport/schema"

var Schema = schema.ModuleSchema{
	ModuleName:    "uoms",
	DisplayName:   "Units of Measure",
	SchemaVersion: "1.0.0",
	Description:   "Units of measure for products",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Code"},
	Columns: []schema.ColumnSchema{
		{Name: "Code", Type: schema.ColString, Label: "Code", Required: true, MaxLength: schema.IntPtr(20), Editable: false, Exportable: true, Template: true},
		{Name: "Name", Type: schema.ColString, Label: "Unit Name", Required: true, MaxLength: schema.IntPtr(100), Editable: true, Exportable: true, Template: true},
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
