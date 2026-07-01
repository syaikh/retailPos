package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCategoryRepo struct {
	CategoryRefRepo
}

type mockBrandRepo struct {
	BrandRefRepo
}

type mockUOMRepo struct {
	UOMRefRepo
}

func TestProductAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "products", a.ModuleName())
}

func TestProductAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil, nil, nil, nil)
	assert.NotNil(t, a)
	assert.Equal(t, "products", a.ModuleName())
}

func TestProductAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestProductAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    ProductImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":           1,
				"SKU":            "SKU-001",
				"Name":           "Widget Pro",
				"Barcode":        "8999999999999",
				"Category":       "Electronics",
				"Brand":          "Samsung",
				"Price":          "50000",
				"Cost":           "35000",
				"Stock":          "100",
				"Status":         "active",
				"UnitOfMeasure":  "PCS",
				"WeightGrams":    "250",
				"Description":    "A premium widget",
			},
			want: ProductImportRow{
				Row:           1,
				SKU:           "SKU-001",
				Name:          "Widget Pro",
				Barcode:       "8999999999999",
				Category:      "Electronics",
				Brand:         "Samsung",
				Price:         50000,
				Cost:          35000,
				Stock:         100,
				Status:        "active",
				UnitOfMeasure: "PCS",
				WeightGrams:   250,
				Description:   "A premium widget",
			},
		},
		{
			name: "SKU is required - empty string",
			row: map[string]interface{}{
				"_row":  2,
				"SKU":   "",
				"Name":  "NoSKU",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "SKU key missing",
			row: map[string]interface{}{
				"_row":  3,
				"Name":  "NoSKU",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "Name is required - empty string",
			row: map[string]interface{}{
				"_row":  4,
				"SKU":   "SKU-004",
				"Name":  "",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "Name key missing",
			row: map[string]interface{}{
				"_row":  5,
				"SKU":   "SKU-005",
				"Price": "1000",
			},
			wantErr: true,
		},
		{
			name: "Status defaults to active",
			row: map[string]interface{}{
				"_row":  6,
				"SKU":   "SKU-006",
				"Name":  "DefaultStatus",
				"Price": "2000",
			},
			want: ProductImportRow{
				Row:    6,
				SKU:    "SKU-006",
				Name:   "DefaultStatus",
				Price:  2000,
				Status: "active",
			},
		},
		{
			name: "Status explicit inactive",
			row: map[string]interface{}{
				"_row":   7,
				"SKU":    "SKU-007",
				"Name":   "InactiveItem",
				"Price":  "3000",
				"Status": "inactive",
			},
			want: ProductImportRow{
				Row:    7,
				SKU:    "SKU-007",
				Name:   "InactiveItem",
				Price:  3000,
				Status: "inactive",
			},
		},
		{
			name: "zero values for missing numeric fields",
			row: map[string]interface{}{
				"_row":  8,
				"SKU":   "SKU-008",
				"Name":  "Minimal",
				"Price": "0",
			},
			want: ProductImportRow{
				Row:     8,
				SKU:     "SKU-008",
				Name:    "Minimal",
				Status:  "active",
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"SKU":   "SKU-009",
				"Name":  "NoRowProduct",
				"Price": "5000",
			},
			want: ProductImportRow{
				Row:    0,
				SKU:    "SKU-009",
				Name:   "NoRowProduct",
				Price:  5000,
				Status: "active",
			},
		},
		{
			name: "float string price gets truncated to int",
			row: map[string]interface{}{
				"_row":  10,
				"SKU":   "SKU-010",
				"Name":  "FloatPrice",
				"Price": "99.99",
			},
			want: ProductImportRow{
				Row:    10,
				SKU:    "SKU-010",
				Name:   "FloatPrice",
				Status: "active",
			},
		},
		{
			name: "non-numeric price becomes 0",
			row: map[string]interface{}{
				"_row":  11,
				"SKU":   "SKU-011",
				"Name":  "BadPrice",
				"Price": "not-a-number",
			},
			want: ProductImportRow{
				Row:    11,
				SKU:    "SKU-011",
				Name:   "BadPrice",
				Status: "active",
			},
		},
	}

	ctx := context.Background()
	a := &adapter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.MapToEntity(ctx, Schema, tt.row)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			importRow, ok := got.(ProductImportRow)
			require.True(t, ok, "expected ProductImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestProductAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}

func TestNilStr(t *testing.T) {
	tests := []struct {
		name string
		s    *string
		want string
	}{
		{"nil pointer", nil, ""},
		{"non-nil", ptrString("hello"), "hello"},
		{"empty string", ptrString(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nilStr(tt.s))
		})
	}
}

func TestNilInt(t *testing.T) {
	tests := []struct {
		name string
		i    *int
		want interface{}
	}{
		{"nil pointer", nil, nil},
		{"non-nil", ptrInt(42), 42},
		{"zero", ptrInt(0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nilInt(tt.i))
		})
	}
}

func ptrString(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}
