package brand

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrandAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "brands", a.ModuleName())
}

func TestBrandAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil)
	assert.NotNil(t, a)
	assert.Equal(t, "brands", a.ModuleName())
}

func TestBrandAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestBrandAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    BrandImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":        1,
				"Name":        "Samsung",
				"Description": "Korean electronics brand",
				"IsActive":    "true",
			},
			want: BrandImportRow{
				Row:         1,
				Name:        "Samsung",
				Description: "Korean electronics brand",
				IsActive:    true,
			},
		},
		{
			name: "name only with isActive false",
			row: map[string]interface{}{
				"_row":     2,
				"Name":     "DefunctBrand",
				"IsActive": "false",
			},
			want: BrandImportRow{
				Row:      2,
				Name:     "DefunctBrand",
				IsActive: false,
			},
		},
		{
			name: "name is required - empty string",
			row: map[string]interface{}{
				"_row":  3,
				"Name":  "",
			},
			wantErr: true,
		},
		{
			name: "name key missing",
			row: map[string]interface{}{
				"_row": 4,
			},
			wantErr: true,
		},
		{
			name: "isActive defaults to true when absent",
			row: map[string]interface{}{
				"_row":        5,
				"Name":        "AutoActive",
				"Description": "Should be active",
			},
			want: BrandImportRow{
				Row:         5,
				Name:        "AutoActive",
				Description: "Should be active",
				IsActive:    true,
			},
		},
		{
			name: "isActive yes",
			row: map[string]interface{}{
				"_row":     6,
				"Name":     "YesBrand",
				"IsActive": "YES",
			},
			want: BrandImportRow{
				Row:      6,
				Name:     "YesBrand",
				IsActive: true,
			},
		},
		{
			name: "isActive numeric 1",
			row: map[string]interface{}{
				"_row":     7,
				"Name":     "NumericBrand",
				"IsActive": "1",
			},
			want: BrandImportRow{
				Row:      7,
				Name:     "NumericBrand",
				IsActive: true,
			},
		},
		{
			name: "isActive unrecognized value defaults false",
			row: map[string]interface{}{
				"_row":     8,
				"Name":     "WeirdBrand",
				"IsActive": "maybe",
			},
			want: BrandImportRow{
				Row:      8,
				Name:     "WeirdBrand",
				IsActive: false,
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"Name": "NoRowBrand",
			},
			want: BrandImportRow{
				Row:      0,
				Name:     "NoRowBrand",
				IsActive: true,
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
			importRow, ok := got.(BrandImportRow)
			require.True(t, ok, "expected BrandImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestBrandAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}
