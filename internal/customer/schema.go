package customer

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "customers",
	DisplayName:   "Customers",
	SchemaVersion: "1.0.0",
	Description:   "Customer data",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name", "Phone"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Name", Type: importexportshared.ColString, Label: "Customer Name", Required: true, MaxLength: importexportshared.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Phone", Type: importexportshared.ColString, Label: "Phone", Required: false, MaxLength: importexportshared.IntPtr(20), Editable: true, Exportable: true, Template: true},
		{Name: "Email", Type: importexportshared.ColString, Label: "Email", Required: false, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Address", Type: importexportshared.ColString, Label: "Address", Required: false, MaxLength: importexportshared.IntPtr(500), Editable: true, Exportable: true, Template: true},
		{Name: "Note", Type: importexportshared.ColString, Label: "Note", Required: false, MaxLength: importexportshared.IntPtr(500), Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Default: "true", Editable: true, Exportable: true, Template: true},
	},
	Features: importexportshared.ModuleFeatures{
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
