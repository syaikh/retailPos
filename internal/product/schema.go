package product

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "products",
	DisplayName:   "Products",
	SchemaVersion: "1.0.0",
	Description:   "Master product data",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"SKU"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "SKU", Type: importexportshared.ColString, Label: "SKU", Required: true, MaxLength: importexportshared.IntPtr(50), Editable: false, Exportable: true, Template: true},
		{Name: "Name", Type: importexportshared.ColString, Label: "Product Name", Required: true, MaxLength: importexportshared.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Barcode", Type: importexportshared.ColString, Label: "Barcode", Required: false, MaxLength: importexportshared.IntPtr(50), Editable: true, Exportable: true, Template: true},
		{Name: "Category", Type: importexportshared.ColReference, Label: "Category", Required: false, Reference: "categories", Editable: true, Exportable: true, Template: true},
		{Name: "Brand", Type: importexportshared.ColReference, Label: "Brand", Required: false, Reference: "brands", Editable: true, Exportable: true, Template: true},
		{Name: "Price", Type: importexportshared.ColNumber, Label: "Price", Required: true, MinValue: importexportshared.Float64Ptr(1), Editable: true, Exportable: true, Template: true},
		{Name: "Cost", Type: importexportshared.ColNumber, Label: "Cost", Required: false, MinValue: importexportshared.Float64Ptr(0), Editable: true, Exportable: true, Template: true},
		{Name: "Stock", Type: importexportshared.ColNumber, Label: "Stock", Required: false, MinValue: importexportshared.Float64Ptr(0), Editable: true, Exportable: false, Template: false},
		{Name: "Status", Type: importexportshared.ColString, Label: "Status", Required: false, AllowedValues: []string{"active", "inactive", "draft", "archived"}, Default: "active", Editable: true, Exportable: true, Template: true},
		{Name: "UnitOfMeasure", Type: importexportshared.ColReference, Label: "Unit of Measure", Required: false, Reference: "uoms", Editable: true, Exportable: true, Template: true},
		{Name: "WeightGrams", Type: importexportshared.ColNumber, Label: "Weight (g)", Required: false, MinValue: importexportshared.Float64Ptr(0), Editable: true, Exportable: false, Template: false},
		{Name: "Description", Type: importexportshared.ColString, Label: "Description", Required: false, MaxLength: importexportshared.IntPtr(2000), Editable: true, Exportable: false, Template: false},
	},
	References: []importexportshared.ReferenceDef{
		{Column: "Category", ReferenceModule: "categories", ReferenceColumn: "Name", ReferenceLabel: "Category Name", Policy: importexportshared.RefStrict},
		{Column: "Brand", ReferenceModule: "brands", ReferenceColumn: "Name", ReferenceLabel: "Brand Name", Policy: importexportshared.RefStrict},
		{Column: "UnitOfMeasure", ReferenceModule: "uoms", ReferenceColumn: "Code", ReferenceLabel: "UOM Code", Policy: importexportshared.RefStrict},
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
