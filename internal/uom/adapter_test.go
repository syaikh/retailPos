package uom

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
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

func TestUOMAdapter_ReposAdapter_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	inserted, err := ra.Insert(context.Background(), []interface{}{
		UOMImportRow{Row: 1, Code: "L", Name: "Liter", IsActive: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUOMAdapter_ReposAdapter_Insert_Errors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.Insert(context.Background(), []interface{}{
		UOMImportRow{Row: 1, Code: "L", Name: "Valid", IsActive: true},
		UOMImportRow{Row: 2, Code: "", Name: "NoCode", IsActive: true},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uom import errors")
}

func TestUOMAdapter_ReposAdapter_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(false)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	updated, err := ra.Update(context.Background(), []interface{}{
		UOMImportRow{Row: 1, Code: "KG", Name: "Kilogram", IsActive: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUOMAdapter_ReposAdapter_Update_Errors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.Update(context.Background(), []interface{}{
		UOMImportRow{Row: 1, Code: "KG", Name: "Valid", IsActive: true},
		UOMImportRow{Row: 2, Code: "", Name: "NoCode", IsActive: true},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uom import errors")
}

func TestUOMAdapter_ReposAdapter_ExportData(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "description", "is_active", "created_at"}).
		AddRow(1, "KG", "Kilogram", "mass", true, now)
	mock.ExpectQuery("SELECT (.+) FROM units_of_measure ORDER BY code").WillReturnRows(rows)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.Equal(t, "KG", data[0]["Code"])
	assert.Equal(t, "Kilogram", data[0]["Name"])
	assert.Equal(t, "true", data[0]["IsActive"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUOMAdapter_ReposAdapter_ExportData_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM units_of_measure ORDER BY code").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.ExportData(context.Background(), Schema)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUOMAdapter_ReposAdapter_LoadReferences(t *testing.T) {
	a := NewAdapter(nil)
	ra := a.Repository()
	refs, err := ra.LoadReferences(context.Background(), Schema)
	assert.NoError(t, err)
	assert.Nil(t, refs)
}
