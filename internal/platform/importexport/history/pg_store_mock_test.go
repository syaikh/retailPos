package history

import (
	"context"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/platform/importexport/schema"
	importexportshared "retail-pos-system/internal/shared/importexport"
)

func TestHistoryStore_SaveSnapshot_ExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO import_snapshots").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("db exec failed"))

	store := NewStore(mock)
	ctx := context.Background()

	rows := []map[string]interface{}{{"name": "test"}}
	s := schema.ModuleSchema{ModuleName: "test", SchemaVersion: "1.0"}

	err = store.SaveSnapshot(ctx, 1, s, rows, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save snapshot")
}

func TestHistoryStore_SaveRow_ExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO import_rows").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("db exec failed"))

	store := NewStore(mock)
	ctx := context.Background()

	err = store.SaveRow(ctx, 1, 1, "success", nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save row")
}

func TestHistoryStore_SaveRow_NilValues(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO import_rows").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("I", 1))

	store := NewStore(mock)
	ctx := context.Background()

	err = store.SaveRow(ctx, 1, 2, "error", nil, nil, nil)
	assert.NoError(t, err)
}

func TestHistoryStore_SaveError_ExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO import_errors").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("db exec failed"))

	store := NewStore(mock)
	ctx := context.Background()

	err = store.SaveError(ctx, 1, 1, "field", "val", "reason", "suggestion", "stage")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")
}

func TestHistoryStore_SaveError_EmptyFields(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO import_errors").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("I", 1))

	store := NewStore(mock)
	ctx := context.Background()

	err = store.SaveError(ctx, 1, 1, "", "", "reason", "", "stage")
	assert.NoError(t, err)
}

func TestHistoryStore_GetSnapshot_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT rows_data, schema_snapshot").WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("query failed"))

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetSnapshot(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get snapshot")
}

func TestHistoryStore_GetSnapshot_NotFound_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"rows_data", "schema_snapshot"})
	mock.ExpectQuery("SELECT rows_data, schema_snapshot").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetSnapshot(ctx, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot not found")
}

func TestHistoryStore_GetSnapshot_BadJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"rows_data", "schema_snapshot"}).
		AddRow([]byte("not-json"), []byte(`{}`))
	mock.ExpectQuery("SELECT rows_data, schema_snapshot").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetSnapshot(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal rows_data")
}

func TestHistoryStore_GetRows_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT ir.row_number").WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("rows query failed"))

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetRows(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query rows")
}

func TestHistoryStore_GetRows_ErrorsQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"row_number", "status", "entity_id", "old_values", "new_values"}).
		AddRow(1, "success", nil, []byte("{}"), []byte("{}"))
	mock.ExpectQuery("SELECT ir.row_number").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	mock.ExpectQuery("SELECT row_number, field, value, reason, suggestion, stage").WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("errors query failed"))

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetRows(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query errors")
}

func TestHistoryStore_GetRows_ErrorRowsIterationError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"row_number", "status", "entity_id", "old_values", "new_values"})
	mock.ExpectQuery("SELECT ir.row_number").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	mock.ExpectQuery("SELECT row_number, field, value, reason, suggestion, stage").WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("error rows iteration failed"))

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetRows(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error rows iteration")
}

func TestHistoryStore_GetRows_WithErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"row_number", "status", "entity_id", "old_values", "new_values"}).
		AddRow(1, "error", nil, []byte(`{"name":"old"}`), []byte(`{"name":"new"}`))
	mock.ExpectQuery("SELECT ir.row_number").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	errRows := pgxmock.NewRows([]string{"row_number", "field", "value", "reason", "suggestion", "stage"}).
		AddRow(1, "email", "bad", "invalid", "fix it", "validation")
	mock.ExpectQuery("SELECT row_number, field, value, reason, suggestion, stage").WithArgs(pgxmock.AnyArg()).WillReturnRows(errRows)

	store := NewStore(mock)
	ctx := context.Background()

	result, err := store.GetRows(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHistoryStore_GetRows_Empty_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"row_number", "status", "entity_id", "old_values", "new_values"})
	mock.ExpectQuery("SELECT ir.row_number").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	errRows := pgxmock.NewRows([]string{"row_number", "field", "value", "reason", "suggestion", "stage"})
	mock.ExpectQuery("SELECT row_number, field, value, reason, suggestion, stage").WithArgs(pgxmock.AnyArg()).WillReturnRows(errRows)

	store := NewStore(mock)
	ctx := context.Background()

	result, err := store.GetRows(ctx, 1)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestHistoryStore_GetErrors_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT row_number, COALESCE").WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("query failed"))

	store := NewStore(mock)
	ctx := context.Background()

	_, err = store.GetErrors(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query errors")
}

func TestHistoryStore_GetErrors_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	errRows := pgxmock.NewRows([]string{"row_number", "field", "value", "reason", "suggestion", "stage"}).
		AddRow(1, "name", "empty", "required", nil, "validation").
		AddRow(2, "email", "bad", "format error", nil, "validation")
	mock.ExpectQuery("SELECT row_number, COALESCE").WithArgs(pgxmock.AnyArg()).WillReturnRows(errRows)

	store := NewStore(mock)
	ctx := context.Background()

	result, err := store.GetErrors(ctx, 1)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Row)
	assert.Equal(t, importexportshared.ErrorStage("validation"), result[0].Stage)
}

func TestHistoryStore_GetErrors_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	errRows := pgxmock.NewRows([]string{"row_number", "field", "value", "reason", "suggestion", "stage"})
	mock.ExpectQuery("SELECT row_number, COALESCE").WithArgs(pgxmock.AnyArg()).WillReturnRows(errRows)

	store := NewStore(mock)
	ctx := context.Background()

	result, err := store.GetErrors(ctx, 1)
	require.NoError(t, err)
	assert.Empty(t, result)
}
