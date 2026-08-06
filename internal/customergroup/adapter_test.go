package customergroup

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCGAdapter_NewAdapter(t *testing.T) {
	a := NewAdapter(nil)
	assert.NotNil(t, a)
	assert.Equal(t, "customer_groups", a.ModuleName())
}

func TestCGAdapter_ModuleName(t *testing.T) {
	a := &adapter{}
	assert.Equal(t, "customer_groups", a.ModuleName())
}

func TestCGAdapter_ValidateBusiness(t *testing.T) {
	a := &adapter{}
	ctx := context.Background()
	errs := a.ValidateBusiness(ctx, Schema, nil)
	assert.Nil(t, errs)
}

func TestCGAdapter_parseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"maybe", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, parseBool(tt.input))
		})
	}
}

func TestCGAdapter_MapToEntity(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]interface{}
		want    ImportRow
		wantErr bool
	}{
		{
			name: "happy path all fields",
			row: map[string]interface{}{
				"_row":        1,
				"Name":        "VIP",
				"Description": "VIP customers",
				"IsActive":    "true",
				"Color":       "#FF0000",
			},
			want: ImportRow{
				Row:         1,
				Name:        "VIP",
				Description: "VIP customers",
				IsActive:    true,
				Color:       "#FF0000",
			},
		},
		{
			name: "name only with isActive false",
			row: map[string]interface{}{
				"_row":     2,
				"Name":     "Inactive",
				"IsActive": "false",
			},
			want: ImportRow{
				Row:      2,
				Name:     "Inactive",
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
				"_row": 5,
				"Name": "AutoActive",
			},
			want: ImportRow{
				Row:      5,
				Name:     "AutoActive",
				IsActive: true,
			},
		},
		{
			name: "isActive yes",
			row: map[string]interface{}{
				"_row":     6,
				"Name":     "YesGroup",
				"IsActive": "YES",
			},
			want: ImportRow{
				Row:      6,
				Name:     "YesGroup",
				IsActive: true,
			},
		},
		{
			name: "isActive numeric 1",
			row: map[string]interface{}{
				"_row":     7,
				"Name":     "NumericGroup",
				"IsActive": "1",
			},
			want: ImportRow{
				Row:      7,
				Name:     "NumericGroup",
				IsActive: true,
			},
		},
		{
			name: "isActive random string defaults false",
			row: map[string]interface{}{
				"_row":     8,
				"Name":     "RandomGroup",
				"IsActive": "maybe",
			},
			want: ImportRow{
				Row:      8,
				Name:     "RandomGroup",
				IsActive: false,
			},
		},
		{
			name: "no _row defaults to 0",
			row: map[string]interface{}{
				"Name": "NoRow",
			},
			want: ImportRow{
				Row:      0,
				Name:     "NoRow",
				IsActive: true,
			},
		},
		{
			name: "with color",
			row: map[string]interface{}{
				"_row":  9,
				"Name":  "ColoredGroup",
				"Color": "#ABCDEF",
			},
			want: ImportRow{
				Row:      9,
				Name:     "ColoredGroup",
				IsActive: true,
				Color:    "#ABCDEF",
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
			importRow, ok := got.(ImportRow)
			require.True(t, ok, "expected ImportRow")
			assert.Equal(t, tt.want, importRow)
		})
	}
}

func TestCGAdapter_Repository(t *testing.T) {
	a := &adapter{}
	ra := a.Repository()
	assert.NotNil(t, ra)
}

func TestCGAdapter_ReposAdapter_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	cgCols := []string{"id", "name", "description", "is_active", "color", "customer_count", "created_at", "updated_at"}
	mock.ExpectQuery("(?si)SELECT .+ FROM customer_groups .+ WHERE LOWER").WithArgs(pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows(cgCols))
	mock.ExpectQuery("(?si)INSERT INTO customer_groups").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, now, now))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	inserted, err := ra.Insert(context.Background(), []interface{}{
		ImportRow{Row: 1, Name: "New CG", Description: "new", IsActive: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCGAdapter_ReposAdapter_Insert_Errors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	cgCols := []string{"id", "name", "description", "is_active", "color", "customer_count", "created_at", "updated_at"}
	mock.ExpectQuery("(?si)SELECT .+ FROM customer_groups .+ WHERE LOWER").WithArgs(pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows(cgCols))
	mock.ExpectQuery("(?si)INSERT INTO customer_groups").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, now, now))
	mock.ExpectQuery("(?si)SELECT .+ FROM customer_groups .+ WHERE LOWER").WithArgs(pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows(cgCols))
	mock.ExpectQuery("(?si)INSERT INTO customer_groups").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("insert failed"))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.Insert(context.Background(), []interface{}{
		ImportRow{Row: 1, Name: "Valid", IsActive: true},
		ImportRow{Row: 2, Name: "", IsActive: true},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer group import errors")
}

func TestCGAdapter_ReposAdapter_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	cgCols := []string{"id", "name", "description", "is_active", "color", "customer_count", "created_at", "updated_at"}
	mock.ExpectQuery("(?si)SELECT .+ FROM customer_groups .+ WHERE LOWER").WithArgs(pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows(cgCols).AddRow(1, "Existing CG", "desc", true, "#6C5CE7", 0, now, now),
	)
	mock.ExpectExec("(?si)UPDATE customer_groups").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	updated, err := ra.Update(context.Background(), []interface{}{
		ImportRow{Row: 1, Name: "Existing CG", Description: "updated", IsActive: false},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCGAdapter_ReposAdapter_Update_Errors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	cgCols := []string{"id", "name", "description", "is_active", "color", "customer_count", "created_at", "updated_at"}
	mock.ExpectQuery("(?si)SELECT .+ FROM customer_groups .+ WHERE LOWER").WithArgs(pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows(cgCols).AddRow(1, "Valid", "desc", true, "#6C5CE7", 0, now, now),
	)
	mock.ExpectExec("(?si)UPDATE customer_groups").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery("(?si)SELECT .+ FROM customer_groups .+ WHERE LOWER").WithArgs(pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows(cgCols).AddRow(2, "Valid2", "desc", true, "#6C5CE7", 0, now, now),
	)
	mock.ExpectExec("(?si)UPDATE customer_groups").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("update failed"))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.Update(context.Background(), []interface{}{
		ImportRow{Row: 1, Name: "Valid", Description: "upd", IsActive: true},
		ImportRow{Row: 2, Name: "Valid2", Description: "upd2", IsActive: false},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "customer group import errors")
}

func TestCGAdapter_ReposAdapter_ExportData(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	cgCols := []string{"id", "name", "description", "is_active", "color", "customer_count", "created_at", "updated_at"}
	mock.ExpectQuery("(?i)SELECT .+ FROM customer_groups .+ ORDER BY cg.id ASC").WillReturnRows(
		pgxmock.NewRows(cgCols).AddRow(1, "VIP", "VIP group", true, "#FF0000", 5, now, now),
	)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.Equal(t, "VIP", data[0]["Name"])
	assert.Equal(t, "VIP group", data[0]["Description"])
	assert.Equal(t, "true", data[0]["IsActive"])
	assert.Equal(t, "#FF0000", data[0]["Color"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCGAdapter_ReposAdapter_ExportData_NoColor(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	cgCols := []string{"id", "name", "description", "is_active", "color", "customer_count", "created_at", "updated_at"}
	mock.ExpectQuery("(?i)SELECT .+ FROM customer_groups .+ ORDER BY cg.id ASC").WillReturnRows(
		pgxmock.NewRows(cgCols).AddRow(1, "Basic", "basic group", true, "", 0, now, now),
	)

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	data, err := ra.ExportData(context.Background(), Schema)
	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.Equal(t, "Basic", data[0]["Name"])
	_, hasColor := data[0]["Color"]
	assert.False(t, hasColor)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCGAdapter_ReposAdapter_ExportData_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("(?i)SELECT .+ FROM customer_groups .+ ORDER BY cg.id ASC").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	a := NewAdapter(repo)
	ra := a.Repository()

	_, err = ra.ExportData(context.Background(), Schema)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCGAdapter_ReposAdapter_LoadReferences(t *testing.T) {
	a := NewAdapter(nil)
	ra := a.Repository()
	refs, err := ra.LoadReferences(context.Background(), Schema)
	assert.NoError(t, err)
	assert.Nil(t, refs)
}
