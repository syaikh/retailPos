package customer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "customers", a.ModuleName())
}

func TestCustomerAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil)
	assert.NotNil(t, a)
	assert.Equal(t, "customers", a.ModuleName())
}

func TestCustomerAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestCustomerAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    CustomerImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":     1,
				"Name":     "John Doe",
				"Phone":    "08123456789",
				"Email":    "john@example.com",
				"Address":  "Jl. Merdeka No.1",
				"Note":     "VIP customer",
				"IsActive": "true",
			},
			want: CustomerImportRow{
				Row:      1,
				Name:     "John Doe",
				Phone:    "08123456789",
				Email:    "john@example.com",
				Address:  "Jl. Merdeka No.1",
				Note:     "VIP customer",
				IsActive: true,
			},
		},
		{
			name: "name only, optional fields empty",
			row: map[string]interface{}{
				"_row":  2,
				"Name":  "Jane Smith",
			},
			want: CustomerImportRow{
				Row:      2,
				Name:     "Jane Smith",
				IsActive: true,
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
				"_row":  5,
				"Name":  "DefaultActive",
				"Phone": "081111111",
			},
			want: CustomerImportRow{
				Row:      5,
				Name:     "DefaultActive",
				Phone:    "081111111",
				IsActive: true,
			},
		},
		{
			name: "isActive false",
			row: map[string]interface{}{
				"_row":     6,
				"Name":     "InactiveGuy",
				"IsActive": "false",
			},
			want: CustomerImportRow{
				Row:      6,
				Name:     "InactiveGuy",
				IsActive: false,
			},
		},
		{
			name: "isActive yes",
			row: map[string]interface{}{
				"_row":     7,
				"Name":     "YesMan",
				"IsActive": "yes",
			},
			want: CustomerImportRow{
				Row:      7,
				Name:     "YesMan",
				IsActive: true,
			},
		},
		{
			name: "isActive numeric 1",
			row: map[string]interface{}{
				"_row":     8,
				"Name":     "NumberOne",
				"IsActive": "1",
			},
			want: CustomerImportRow{
				Row:      8,
				Name:     "NumberOne",
				IsActive: true,
			},
		},
		{
			name: "isActive unrecognized defaults false",
			row: map[string]interface{}{
				"_row":     9,
				"Name":     "Maybe",
				"IsActive": "maybe",
			},
			want: CustomerImportRow{
				Row:      9,
				Name:     "Maybe",
				IsActive: false,
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"Name": "ZeroRow",
			},
			want: CustomerImportRow{
				Row:      0,
				Name:     "ZeroRow",
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
			importRow, ok := got.(CustomerImportRow)
			require.True(t, ok, "expected CustomerImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestCustomerAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}

func TestStrVal(t *testing.T) {
	tests := []struct {
		name string
		s    *string
		want string
	}{
		{"nil pointer", nil, ""},
		{"non-nil value", ptrHelper("hello"), "hello"},
		{"empty string", ptrHelper(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, strVal(tt.s))
		})
	}
}

func ptrHelper(s string) *string {
	return &s
}
