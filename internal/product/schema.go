package product

import "retail-pos-system/internal/platform/importexport/schema"

var Schema = schema.ModuleSchema{
	ModuleName:    "products",
	DisplayName:   "Products",
	SchemaVersion: "1.0.0",
	Description:   "Master product data",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"SKU"},
	Columns: []schema.ColumnSchema{
		{Name: "SKU", Type: schema.ColString, Label: "SKU", Required: true, MaxLength: schema.IntPtr(50), Editable: false, Exportable: true, Template: true},
		{Name: "Name", Type: schema.ColString, Label: "Product Name", Required: true, MaxLength: schema.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Barcode", Type: schema.ColString, Label: "Barcode", Required: false, MaxLength: schema.IntPtr(50), Editable: true, Exportable: true, Template: true},
		{Name: "Category", Type: schema.ColReference, Label: "Category", Required: false, Reference: "categories", Editable: true, Exportable: true, Template: true},
		{Name: "Brand", Type: schema.ColReference, Label: "Brand", Required: false, Reference: "brands", Editable: true, Exportable: true, Template: true},
		{Name: "Price", Type: schema.ColNumber, Label: "Price", Required: true, MinValue: schema.Float64Ptr(1), Editable: true, Exportable: true, Template: true},
		{Name: "Cost", Type: schema.ColNumber, Label: "Cost", Required: false, MinValue: schema.Float64Ptr(0), Editable: true, Exportable: true, Template: true},
		{Name: "Stock", Type: schema.ColNumber, Label: "Stock", Required: false, MinValue: schema.Float64Ptr(0), Editable: true, Exportable: false, Template: false},
		{Name: "Status", Type: schema.ColString, Label: "Status", Required: false, AllowedValues: []string{"active", "inactive", "draft", "archived"}, Default: "active", Editable: true, Exportable: true, Template: true},
		{Name: "UnitOfMeasure", Type: schema.ColReference, Label: "Unit of Measure", Required: false, Reference: "uoms", Editable: true, Exportable: true, Template: true},
		{Name: "WeightGrams", Type: schema.ColNumber, Label: "Weight (g)", Required: false, MinValue: schema.Float64Ptr(0), Editable: true, Exportable: true, Template: true},
		{Name: "Description", Type: schema.ColString, Label: "Description", Required: false, MaxLength: schema.IntPtr(2000), Editable: true, Exportable: true, Template: true},
	},
	References: []schema.ReferenceDef{
		{Column: "Category", ReferenceModule: "categories", ReferenceColumn: "Name", ReferenceLabel: "Category Name", Policy: schema.RefStrict},
		{Column: "Brand", ReferenceModule: "brands", ReferenceColumn: "Name", ReferenceLabel: "Brand Name", Policy: schema.RefStrict},
		{Column: "UnitOfMeasure", ReferenceModule: "uoms", ReferenceColumn: "Code", ReferenceLabel: "UOM Code", Policy: schema.RefStrict},
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
