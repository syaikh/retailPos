package category

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "categories", a.ModuleName())
}

func TestCategoryAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil)
	assert.NotNil(t, a)
	assert.Equal(t, "categories", a.ModuleName())
}

func TestCategoryAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestCategoryAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    CategoryImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":        1,
				"Name":        "Electronics",
				"Slug":        "electronics",
				"Description": "All electronic items",
				"IsActive":    "true",
			},
			want: CategoryImportRow{
				Row:         1,
				Name:        "Electronics",
				Slug:        "electronics",
				Description: "All electronic items",
				IsActive:    true,
			},
		},
		{
			name: "name only, defaults for optional",
			row: map[string]interface{}{
				"_row":     2,
				"Name":     "Beverages",
				"IsActive": "false",
			},
			want: CategoryImportRow{
				Row:      2,
				Name:     "Beverages",
				IsActive: false,
			},
		},
		{
			name: "name is required",
			row: map[string]interface{}{
				"_row":  3,
				"Name":  "",
				"Slug":  "empty",
			},
			wantErr: true,
		},
		{
			name: "name missing key",
			row: map[string]interface{}{
				"_row": 4,
				"Slug": "no-name",
			},
			wantErr: true,
		},
		{
			name: "isActive defaults to true",
			row: map[string]interface{}{
				"_row": 5,
				"Name": "Actives",
			},
			want: CategoryImportRow{
				Row:      5,
				Name:     "Actives",
				IsActive: true,
			},
		},
		{
			name: "isActive truthy variations",
			row: map[string]interface{}{
				"_row":     6,
				"Name":     "Truthy",
				"IsActive": "1",
			},
			want: CategoryImportRow{
				Row:      6,
				Name:     "Truthy",
				IsActive: true,
			},
		},
		{
			name: "isActive yes",
			row: map[string]interface{}{
				"_row":     7,
				"Name":     "YesActive",
				"IsActive": "yes",
			},
			want: CategoryImportRow{
				Row:      7,
				Name:     "YesActive",
				IsActive: true,
			},
		},
		{
			name: "isActive TRUE uppercase",
			row: map[string]interface{}{
				"_row":     8,
				"Name":     "UpperActive",
				"IsActive": "TRUE",
			},
			want: CategoryImportRow{
				Row:      8,
				Name:     "UpperActive",
				IsActive: true,
			},
		},
		{
			name: "isActive random string defaults to false",
			row: map[string]interface{}{
				"_row":     9,
				"Name":     "Random",
				"IsActive": "bogus",
			},
			want: CategoryImportRow{
				Row:      9,
				Name:     "Random",
				IsActive: false,
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"Name": "NoRow",
			},
			want: CategoryImportRow{
				Row:      0,
				Name:     "NoRow",
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
			importRow, ok := got.(CategoryImportRow)
			require.True(t, ok, "expected CategoryImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestCategoryAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}

