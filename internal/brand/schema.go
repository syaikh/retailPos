package brand

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "brands",
	DisplayName:   "Brands",
	SchemaVersion: "1.0.0",
	Description:   "Product brands",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Name", Type: importexportshared.ColString, Label: "Brand Name", Required: true, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Description", Type: importexportshared.ColString, Label: "Description", Required: false, MaxLength: importexportshared.IntPtr(500), Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Default: "true", Editable: true, Exportable: true, Template: true},
	},
	Features: importexportshared.ModuleFeatures{
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
