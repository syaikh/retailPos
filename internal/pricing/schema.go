package pricing

import importexportshared "retail-pos-system/internal/shared/importexport"

var Schema = importexportshared.ModuleSchema{
	ModuleName:    "pricing_rules",
	DisplayName:   "Pricing Rules",
	SchemaVersion: "1.0.0",
	Description:   "Product pricing rules (discount, wholesale, etc.)",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"ProductID", "PricingType", "Name"},
	Columns: []importexportshared.ColumnSchema{
		{Name: "ProductID", Type: importexportshared.ColNumber, Label: "Product ID", Required: true, Editable: false, Exportable: true, Template: true},
		{Name: "PricingType", Type: importexportshared.ColString, Label: "Pricing Type", Required: true, AllowedValues: []string{"discount", "wholesale", "member", "promotion"}, Editable: true, Exportable: true, Template: true},
		{Name: "Name", Type: importexportshared.ColString, Label: "Rule Name", Required: true, MaxLength: importexportshared.IntPtr(200), Editable: true, Exportable: true, Template: true},
		{Name: "Price", Type: importexportshared.ColNumber, Label: "Price", Required: true, MinValue: importexportshared.Float64Ptr(0), Editable: true, Exportable: true, Template: true},
		{Name: "MinimumQuantity", Type: importexportshared.ColNumber, Label: "Minimum Quantity", Required: false, MinValue: importexportshared.Float64Ptr(1), Default: 1, Editable: true, Exportable: true, Template: true},
		{Name: "Priority", Type: importexportshared.ColNumber, Label: "Priority", Required: false, Default: 0, Editable: true, Exportable: true, Template: true},
		{Name: "IsActive", Type: importexportshared.ColBoolean, Label: "Active", Required: false, Default: true, Editable: true, Exportable: true, Template: true},
		{Name: "EffectiveFrom", Type: importexportshared.ColDate, Label: "Effective From", Required: false, Editable: true, Exportable: true, Template: false},
		{Name: "EffectiveUntil", Type: importexportshared.ColDate, Label: "Effective Until", Required: false, Editable: true, Exportable: true, Template: false},
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
