package customergroup

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "customer_groups",
	DisplayName:   "Customer Groups",
	SchemaVersion: "1.0.0",
	Description:   "Customer group classifications",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Name", Type: importexportshared.ColString, Label: "Group Name", Required: true, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Description", Type: importexportshared.ColString, Label: "Description", Required: false, Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Default: "true", AllowedValues: []string{"Yes", "No"}, Editable: true, Exportable: true, Template: true},
		{Name: "Color", Type: importexportshared.ColString, Label: "Color", Required: false, Default: "#6C5CE7", Editable: true, Exportable: true, Template: true},
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
