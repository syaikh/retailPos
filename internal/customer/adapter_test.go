package customer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	importexportshared "retail-pos-system/internal/shared/importexport"
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
				StoreID:  nil,
			},
		},
		{
			name: "name only, optional fields empty",
			row: map[string]interface{}{
				"_row": 2,
				"Name": "Jane Smith",
			},
			want: CustomerImportRow{
				Row:      2,
				Name:     "Jane Smith",
				IsActive: true,
				StoreID:  nil,
			},
		},
		{
			name: "name is required - empty string",
			row: map[string]interface{}{
				"_row": 3,
				"Name": "",
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
				StoreID:  nil,
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
				StoreID:  nil,
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
				StoreID:  nil,
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
				StoreID:  nil,
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
				StoreID:  nil,
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
				StoreID:  nil,
			},
		},
		{
			name: "with store_id",
			row: map[string]interface{}{
				"_row":      10,
				"_store_id": 42,
				"Name":      "StoreOwner",
			},
			want: CustomerImportRow{
				Row:      10,
				Name:     "StoreOwner",
				IsActive: true,
				StoreID:  intPtr(42),
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

func intPtr(i int) *int { return &i }

func ptrHelper(s string) *string {
	return &s
}

func TestCustomerAdapter_Insert_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(dbPool)
	adapter := NewAdapter(repo)
	ra := adapter.Repository()

	rows := []interface{}{
		CustomerImportRow{Row: 1, Name: "DB Insert 1", Phone: "0811111111", Email: "dbins1@test.com", IsActive: true},
		CustomerImportRow{Row: 2, Name: "DB Insert 2", Phone: "0812222222", Email: "dbins2@test.com", IsActive: true},
	}

	count, err := ra.Insert(ctx, rows)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	_, total, err := repo.GetAllCustomers(ctx, 100, 0, "", nil, nil, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 2)
}

func TestCustomerAdapter_Insert_WithStoreID(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(dbPool)
	adapter := NewAdapter(repo)
	ra := adapter.Repository()

	storeID := 1
	rows := []interface{}{
		CustomerImportRow{Row: 1, Name: "Store Cust", Phone: "0813333333", Email: "store@test.com", IsActive: true, StoreID: &storeID},
	}

	count, err := ra.Insert(ctx, rows)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCustomerAdapter_Update_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(dbPool)
	adapter := NewAdapter(repo)
	ra := adapter.Repository()

	phone := "0814444444"
	customer := &Customer{Name: "Before Update", Phone: &phone, Email: ptr("before@test.com"), IsActive: true}
	err := repo.CreateCustomer(ctx, customer)
	require.NoError(t, err)

	rows := []interface{}{
		CustomerImportRow{Row: 1, Name: "After Update", Phone: "0814444444", Email: "after@test.com", IsActive: true},
	}

	count, err := ra.Update(ctx, rows)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	got, err := repo.GetCustomerByID(ctx, customer.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "After Update", got.Name)
	assert.Equal(t, "after@test.com", *got.Email)
}

func TestCustomerAdapter_ExportData_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(dbPool)
	adapter := NewAdapter(repo)
	ra := adapter.Repository()

	phone := "0815555555"
	customer := &Customer{Name: "Export Test", Phone: &phone, Email: ptr("export@test.com"), Address: ptr("Jl. Export"), IsActive: true}
	err := repo.CreateCustomer(ctx, customer)
	require.NoError(t, err)

	data, err := ra.ExportData(ctx, importexportshared.ModuleSchema{})
	require.NoError(t, err)
	require.NotEmpty(t, data)

	found := false
	for _, row := range data {
		if row["Name"] == "Export Test" {
			found = true
			assert.Equal(t, "0815555555", row["Phone"])
			assert.Equal(t, "export@test.com", row["Email"])
			assert.Equal(t, "Jl. Export", row["Address"])
		}
	}
	assert.True(t, found, "exported customer not found")
}
