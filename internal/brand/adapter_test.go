package brand

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
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

func TestBrandAdapter_ReposAdapter_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	inserted, err := ra.Insert(context.Background(), []interface{}{
		BrandImportRow{Row: 1, Name: "New Brand", Description: "desc", IsActive: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBrandAdapter_ReposAdapter_Insert_Errors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	inserted, err := ra.Insert(context.Background(), []interface{}{
		BrandImportRow{Row: 1, Name: "Valid", IsActive: true},
		BrandImportRow{Row: 2, Name: "", IsActive: true},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "brand import errors")
	_ = inserted
}

func TestBrandAdapter_ReposAdapter_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(false)
	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	updated, err := ra.Update(context.Background(), []interface{}{
		BrandImportRow{Row: 1, Name: "Existing", Description: "upd", IsActive: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBrandAdapter_ReposAdapter_ExportData(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Brand", "desc", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM brands ORDER BY name").WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.Equal(t, "Brand", data[0]["Name"])
	assert.Equal(t, "desc", data[0]["Description"])
	assert.Equal(t, "true", data[0]["IsActive"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBrandAdapter_ReposAdapter_ExportData_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM brands ORDER BY name").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.ExportData(context.Background(), Schema)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBrandAdapter_ReposAdapter_LoadReferences(t *testing.T) {
	a := NewAdapter(nil)
	ra := a.Repository()
	refs, err := ra.LoadReferences(context.Background(), Schema)
	assert.NoError(t, err)
	assert.Nil(t, refs)
}
