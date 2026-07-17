package store

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "stores",
	DisplayName:   "Stores",
	SchemaVersion: "1.0.0",
	Description:   "Store/outlet locations",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Name", Type: importexportshared.ColString, Label: "Store Name", Required: true, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Address", Type: importexportshared.ColString, Label: "Address", Required: false, Editable: true, Exportable: true, Template: true},
		{Name: "Phone", Type: importexportshared.ColString, Label: "Phone", Required: false, MaxLength: importexportshared.IntPtr(20), Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Default: "true", AllowedValues: []string{"Yes", "No"}, Editable: true, Exportable: true, Template: true},
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
