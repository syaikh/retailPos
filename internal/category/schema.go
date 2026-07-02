package category

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "categories",
	DisplayName:   "Categories",
	SchemaVersion: "1.0.0",
	Description:   "Product categories",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Name"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "Name", Type: importexportshared.ColString, Label: "Category Name", Required: true, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
		{Name: "Slug", Type: importexportshared.ColString, Label: "Slug", Required: false, MaxLength: importexportshared.IntPtr(100), Editable: true, Exportable: true, Template: true},
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
