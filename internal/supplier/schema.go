package supplier

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "suppliers",
	DisplayName:   "Suppliers",
	SchemaVersion: "1.0.0",
	Description:   "Supplier master data",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Code"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Code", Type: importexportshared.ColString, Label: "Supplier Code", Required: true, MaxLength: importexportshared.IntPtr(50), Editable: false, Exportable: true, Template: true},
		{Name: "Name", Type: importexportshared.ColString, Label: "Supplier Name", Required: true, MaxLength: importexportshared.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "ContactName", Type: importexportshared.ColString, Label: "Contact Name", Required: false, MaxLength: importexportshared.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Phone", Type: importexportshared.ColString, Label: "Phone", Required: false, MaxLength: importexportshared.IntPtr(50), Editable: true, Exportable: true, Template: true},
		{Name: "Email", Type: importexportshared.ColString, Label: "Email", Required: false, MaxLength: importexportshared.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Address", Type: importexportshared.ColString, Label: "Address", Required: false, MaxLength: importexportshared.IntPtr(1000), Editable: true, Exportable: true, Template: false},
		{Name: "Notes", Type: importexportshared.ColString, Label: "Notes", Required: false, MaxLength: importexportshared.IntPtr(2000), Editable: true, Exportable: false, Template: false},
		{Name: "IsActive", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Default: true, Editable: true, Exportable: true, Template: true},
	},
	References: []importexportshared.ReferenceDef{},
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
