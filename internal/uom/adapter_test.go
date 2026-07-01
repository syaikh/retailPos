package uom

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUOMAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "uoms", a.ModuleName())
}

func TestUOMAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil)
	assert.NotNil(t, a)
	assert.Equal(t, "uoms", a.ModuleName())
}

func TestUOMAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestUOMAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    UOMImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":        1,
				"Code":        "PCS",
				"Name":        "Pieces",
				"Description": "Individual pieces count",
				"IsActive":    "true",
			},
			want: UOMImportRow{
				Row:         1,
				Code:        "PCS",
				Name:        "Pieces",
				Description: "Individual pieces count",
				IsActive:    true,
			},
		},
		{
			name: "code and name only, isActive false",
			row: map[string]interface{}{
				"_row":     2,
				"Code":     "OZ",
				"Name":     "Ounces",
				"IsActive": "false",
			},
			want: UOMImportRow{
				Row:      2,
				Code:     "OZ",
				Name:     "Ounces",
				IsActive: false,
			},
		},
		{
			name: "code is required - empty string",
			row: map[string]interface{}{
				"_row":  3,
				"Code":  "",
				"Name":  "EmptyCode",
			},
			wantErr: true,
		},
		{
			name: "code key missing",
			row: map[string]interface{}{
				"_row":  4,
				"Name":  "NoCode",
			},
			wantErr: true,
		},
		{
			name: "name is required - empty string",
			row: map[string]interface{}{
				"_row": 5,
				"Code": "XYZ",
				"Name": "",
			},
			wantErr: true,
		},
		{
			name: "name key missing",
			row: map[string]interface{}{
				"_row": 6,
				"Code": "ABC",
			},
			wantErr: true,
		},
		{
			name: "isActive defaults to true",
			row: map[string]interface{}{
				"_row":  7,
				"Code":  "KG",
				"Name":  "Kilograms",
			},
			want: UOMImportRow{
				Row:      7,
				Code:     "KG",
				Name:     "Kilograms",
				IsActive: true,
			},
		},
		{
			name: "isActive yes case insensitive",
			row: map[string]interface{}{
				"_row":     8,
				"Code":     "LT",
				"Name":     "Litre",
				"IsActive": "YES",
			},
			want: UOMImportRow{
				Row:      8,
				Code:     "LT",
				Name:     "Litre",
				IsActive: true,
			},
		},
		{
			name: "isActive numeric 1",
			row: map[string]interface{}{
				"_row":     9,
				"Code":     "BOX",
				"Name":     "Box",
				"IsActive": "1",
			},
			want: UOMImportRow{
				Row:      9,
				Code:     "BOX",
				Name:     "Box",
				IsActive: true,
			},
		},
		{
			name: "isActive unrecognized defaults false",
			row: map[string]interface{}{
				"_row":     10,
				"Code":     "BAG",
				"Name":     "Bag",
				"IsActive": "nope",
			},
			want: UOMImportRow{
				Row:      10,
				Code:     "BAG",
				Name:     "Bag",
				IsActive: false,
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"Code": "SET",
				"Name": "Set",
			},
			want: UOMImportRow{
				Row:      0,
				Code:     "SET",
				Name:     "Set",
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
			importRow, ok := got.(UOMImportRow)
			require.True(t, ok, "expected UOMImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestUOMAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}
