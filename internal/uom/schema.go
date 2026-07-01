package uom

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "uoms",
	DisplayName:   "Units of Measure",
	SchemaVersion: "1.0.0",
	Description:   "Units of measure for products",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Code"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Code", Type: importexportshared.ColString, Label: "Code", Required: true, MaxLength: importexportshared.IntPtr(20), Editable: false, Exportable: true, Template: true},
		{Name: "Name", Type: importexportshared.ColString, Label: "Unit Name", Required: true, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
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
