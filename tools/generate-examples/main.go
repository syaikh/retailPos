package main

import (
	"bytes"
	"fmt"
	"os"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/customer"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/uom"

	"github.com/xuri/excelize/v2"
)

type moduleDef struct {
	filename string
	schema   schema.ModuleSchema
	refKey   string // reference key column (e.g. "Name", "Code") for extracting ref values
	refLabel string // display label for the reference column
	refData  map[string][]string
	dataRows []map[string]interface{}
}

// canonical reference data — all entries used across modules
var canonicalRefs = struct {
	categories []map[string]interface{}
	brands     []map[string]interface{}
	uoms       []map[string]interface{}
}{
	categories: []map[string]interface{}{
		{"Name": "Elektronik", "Slug": "elektronik", "Description": "Electronic devices and accessories", "IsActive": true},
		{"Name": "Pakaian", "Slug": "pakaian", "Description": "Clothing and apparel", "IsActive": true},
		{"Name": "Makanan & Minuman", "Slug": "makanan-minuman", "Description": "Food and beverage products", "IsActive": true},
		{"Name": "Minuman", "Slug": "minuman", "Description": "Beverages and drinks", "IsActive": true},
		{"Name": "Kesehatan", "Slug": "kesehatan", "Description": "Health and personal care", "IsActive": true},
		{"Name": "Rumah Tangga", "Slug": "rumah-tangga", "Description": "Household supplies", "IsActive": true},
		{"Name": "Olahraga", "Slug": "olahraga", "Description": "Sports equipment and apparel", "IsActive": true},
		{"Name": "Otomotif", "Slug": "otomotif", "Description": "Automotive parts and accessories", "IsActive": true},
		{"Name": "Mainan", "Slug": "mainan", "Description": "Toys and games", "IsActive": true},
		{"Name": "Buku & Alat Tulis", "Slug": "buku-alat-tulis", "Description": "Books and stationery", "IsActive": true},
	},
	brands: []map[string]interface{}{
		{"Name": "Nike", "Description": "Sports footwear and apparel", "IsActive": true},
		{"Name": "Adidas", "Description": "Performance sportswear", "IsActive": true},
		{"Name": "Samsung", "Description": "Consumer electronics", "IsActive": true},
		{"Name": "Sony", "Description": "Audio and visual equipment", "IsActive": true},
		{"Name": "LG", "Description": "Home appliances and electronics", "IsActive": true},
		{"Name": "Aqua", "Description": "Bottled drinking water", "IsActive": true},
		{"Name": "Indofood", "Description": "Food and beverage products", "IsActive": true},
		{"Name": "Unilever", "Description": "Personal care and household", "IsActive": true},
		{"Name": "Mayora", "Description": "Snacks and confectionery", "IsActive": true},
		{"Name": "Wings", "Description": "Household and personal care", "IsActive": true},
	},
	uoms: []map[string]interface{}{
		{"Code": "PCS", "Name": "Pieces", "Description": "Individual units", "IsActive": true},
		{"Code": "KG", "Name": "Kilogram", "Description": "Weight in kilograms", "IsActive": true},
		{"Code": "LTR", "Name": "Liter", "Description": "Volume in liters", "IsActive": true},
		{"Code": "BOX", "Name": "Box", "Description": "Box or carton quantity", "IsActive": true},
		{"Code": "DUS", "Name": "Dus", "Description": "Carton or case quantity", "IsActive": true},
		{"Code": "PAK", "Name": "Pak", "Description": "Pack or bundle", "IsActive": true},
		{"Code": "SET", "Name": "Set", "Description": "Set of items", "IsActive": true},
		{"Code": "MTR", "Name": "Meter", "Description": "Length in meters", "IsActive": true},
		{"Code": "LSN", "Name": "Lusin", "Description": "Dozen (12 items)", "IsActive": true},
		{"Code": "BLT", "Name": "Bal", "Description": "Bale quantity", "IsActive": true},
	},
}

// refValues extracts reference key values (e.g., "Name", "Code") from a data slice.
func refValues(rows []map[string]interface{}, key string) []string {
	vals := make([]string, 0, len(rows))
	for _, r := range rows {
		if v, ok := r[key]; ok {
			if s, ok := v.(string); ok {
				vals = append(vals, s)
			}
		}
	}
	return vals
}

func main() {
	outDir := "docs/examples"
	_ = os.MkdirAll(outDir, 0755)

	eng := template.NewEngine()

	// Build refData for products from canonical data (all entries, not a subset)
	productsRefData := map[string][]string{
		"categories": refValues(canonicalRefs.categories, "Name"),
		"brands":     refValues(canonicalRefs.brands, "Name"),
		"uoms":       refValues(canonicalRefs.uoms, "Code"),
	}

	modules := []moduleDef{
		{
			filename: "example_brands_filled.xlsx",
			schema:   brand.Schema,
			refKey:   "Name",
			refLabel: "Brand Name",
			dataRows: canonicalRefs.brands,
		},
		{
			filename: "example_categories_filled.xlsx",
			schema:   category.Schema,
			refKey:   "Name",
			refLabel: "Category Name",
			dataRows: canonicalRefs.categories,
		},
		{
			filename: "example_uoms_filled.xlsx",
			schema:   uom.Schema,
			refKey:   "Code",
			refLabel: "UOM Code",
			dataRows: canonicalRefs.uoms,
		},
		{
			filename: "example_customers_filled.xlsx",
			schema:   customer.Schema,
			dataRows: []map[string]interface{}{
				{"Name": "John Doe", "Phone": "081234567890", "Email": "john@example.com", "Address": "Jl. Merdeka No. 10, Jakarta", "Note": "Regular customer", "IsActive": true},
				{"Name": "PT Maju Jaya", "Phone": "0211234567", "Email": "info@majujaya.com", "Address": "Jl. Sudirman Kav. 25, Jakarta", "Note": "Corporate account", "IsActive": true},
				{"Name": "Sari Store", "Phone": "087812345678", "Email": "", "Address": "Jl. Ahmad Yani No. 45, Bandung", "Note": "New customer", "IsActive": true},
				{"Name": "CV Berkah Abadi", "Phone": "03198765432", "Email": "berkah@abadi.co.id", "Address": "Jl. Diponegoro 88, Surabaya", "Note": "", "IsActive": true},
				{"Name": "Toko Sukses", "Phone": "0251123456", "Email": "sukses@tokosukses.com", "Address": "Jl. Merpati Raya No. 7, Bogor", "Note": "", "IsActive": true},
				{"Name": "Andi Pratama", "Phone": "085611223344", "Email": "andi@pratama.com", "Address": "Perumahan Bumi Indah Blok A3, Tangerang", "Note": "Prefer WhatsApp", "IsActive": true},
				{"Name": "UD Makmur", "Phone": "0274123456", "Email": "makmur@udmakmur.com", "Address": "Jl. Kaliurang Km 5, Yogyakarta", "Note": "Regular restock every 2 weeks", "IsActive": true},
				{"Name": "Siti Rahayu", "Phone": "082198765432", "Email": "siti.rahayu@gmail.com", "Address": "Jl. Pahlawan No. 15, Semarang", "Note": "", "IsActive": true},
				{"Name": "PT Global Elektronik", "Phone": "02198765432", "Email": "sales@globalelektronik.com", "Address": "Jl. Gatot Subroto Kav. 12, Jakarta", "Note": "Distributor", "IsActive": true},
				{"Name": "Toko ABC", "Phone": "0611234567", "Email": "abc@tokotoabc.com", "Address": "Jl. Sisingamangaraja No. 5, Medan", "Note": "", "IsActive": true},
			},
		},
		{
			filename: "example_products_filled.xlsx",
			schema:   product.Schema,
			refData:  productsRefData,
			dataRows: []map[string]interface{}{
				{"SKU": "SKU001", "Name": "T-Shirt Cotton", "Barcode": "8991234567890", "Category": "Pakaian", "Brand": "Nike", "Price": 150000, "Cost": 90000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU002", "Name": "Running Shoes Pro", "Barcode": "", "Category": "Pakaian", "Brand": "Adidas", "Price": 350000, "Cost": 250000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU003", "Name": "Mineral Water 600ml", "Barcode": "8998765432109", "Category": "Minuman", "Brand": "Aqua", "Price": 5000, "Cost": 3000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU004", "Name": "Smart TV 43 Inch", "Barcode": "", "Category": "Elektronik", "Brand": "Samsung", "Price": 5500000, "Cost": 4500000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU005", "Name": "Wireless Headphone", "Barcode": "8991122334455", "Category": "Elektronik", "Brand": "Sony", "Price": 750000, "Cost": 500000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU006", "Name": "Instant Noodles Cup", "Barcode": "8995544332211", "Category": "Makanan & Minuman", "Brand": "Indofood", "Price": 3500, "Cost": 2500, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU007", "Name": "Cotton Jersey", "Barcode": "", "Category": "Pakaian", "Brand": "Nike", "Price": 250000, "Cost": 175000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU008", "Name": "Air Mineral Galon 19L", "Barcode": "", "Category": "Minuman", "Brand": "Aqua", "Price": 55000, "Cost": 40000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU009", "Name": "Soundbar 2.1", "Barcode": "", "Category": "Elektronik", "Brand": "Samsung", "Price": 1250000, "Cost": 900000, "Status": "active", "UnitOfMeasure": "PCS"},
				{"SKU": "SKU010", "Name": "Sports Shorts", "Barcode": "8996677889900", "Category": "Pakaian", "Brand": "Adidas", "Price": 175000, "Cost": 120000, "Status": "active", "UnitOfMeasure": "PCS"},
			},
		},
	}

	for _, m := range modules {
		if err := generateExample(eng, outDir, m.filename, m.schema, m.refData, m.dataRows); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", m.filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s\n", m.filename)
	}
}

func generateExample(eng *template.Engine, outDir, filename string, s schema.ModuleSchema, refData map[string][]string, dataRows []map[string]interface{}) error {
	var buf bytes.Buffer
	if err := eng.Generate(s, refData, &buf); err != nil {
		return fmt.Errorf("generate template: %w", err)
	}

	wb, err := excelize.OpenReader(&buf)
	if err != nil {
		return fmt.Errorf("open generated template: %w", err)
	}
	defer func() { _ = wb.Close() }()

	sheetName := s.ModuleName

	templateCols := make([]schema.ColumnSchema, 0, len(s.Columns))
	for _, col := range s.Columns {
		if col.Template {
			templateCols = append(templateCols, col)
		}
	}

	for i, row := range dataRows {
		for j, col := range templateCols {
			val, ok := row[col.Name]
			if !ok || val == nil {
				val = col.Default
				if val == "" {
					continue
				}
			}
			colLetter, _ := excelize.ColumnNumberToName(j + 1)
			cell := fmt.Sprintf("%s%d", colLetter, i+2)
			if err := wb.SetCellValue(sheetName, cell, val); err != nil {
				return fmt.Errorf("set cell %s: %w", cell, err)
			}
		}
	}

	path := fmt.Sprintf("%s/%s", outDir, filename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := wb.Write(f); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}
	return nil
}
