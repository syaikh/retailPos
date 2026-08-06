package history

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func insertImportJob(ctx context.Context, t *testing.T) int64 {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("history_user"), bcrypt.MinCost)
	var userID int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (username) DO UPDATE SET username = users.username
		RETURNING id
	`, "history_test_user", "history@test.com", string(hash), 1, true).Scan(&userID)
	require.NoError(t, err)

	var jobID int64
	err = dbPool.QueryRow(ctx, `
		INSERT INTO import_jobs (module, schema_version, filename, status, total_rows, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "test_module", "1.0", "test.csv", "completed", 10, userID).Scan(&jobID)
	require.NoError(t, err)
	return jobID
}

func TestHistoryStore_SaveSnapshot(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()
	jobID := insertImportJob(ctx, t)

	rows := []map[string]interface{}{
		{"name": "John", "age": float64(30)},
		{"name": "Jane", "age": float64(25)},
	}
	moduleSchema := schema.ModuleSchema{
		ModuleName:    "test_module",
		SchemaVersion: "1.0",
	}
	preview := &importexport.PreviewResult{
		Module:      "test_module",
		TotalRows:   2,
		InsertCount: 2,
	}

	err := store.SaveSnapshot(ctx, jobID, moduleSchema, rows, preview)
	require.NoError(t, err)

	snapshot, err := store.GetSnapshot(ctx, jobID)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	assert.Equal(t, 2, len(snapshot.RowsData))
	assert.Equal(t, "John", snapshot.RowsData[0]["name"])
	assert.Equal(t, float64(30), snapshot.RowsData[0]["age"])
	assert.Equal(t, "test_module", snapshot.SchemaSnapshot["module_name"])
}

func TestHistoryStore_SaveRow(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()
	jobID := insertImportJob(ctx, t)

	oldVals := map[string]interface{}{"name": "John"}
	newVals := map[string]interface{}{"name": "Jane"}
	entityID := 42

	err := store.SaveRow(ctx, jobID, 1, "success", &entityID, oldVals, newVals)
	require.NoError(t, err)

	err = store.SaveRow(ctx, jobID, 2, "error", nil, nil, nil)
	require.NoError(t, err)

	rows, err := store.GetRows(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, 1, rows[0].RowNumber)
	assert.Equal(t, "success", rows[0].Status)
	assert.NotNil(t, rows[0].EntityID)
	assert.Equal(t, 42, *rows[0].EntityID)
	assert.Equal(t, "John", rows[0].OldValues["name"])
	assert.Equal(t, "Jane", rows[0].NewValues["name"])
	assert.Equal(t, 2, rows[1].RowNumber)
	assert.Equal(t, "error", rows[1].Status)
	assert.Nil(t, rows[1].EntityID)
}

func TestHistoryStore_SaveError(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()
	jobID := insertImportJob(ctx, t)

	err := store.SaveError(ctx, jobID, 1, "name", "John", "required", "provide a name", "validation")
	require.NoError(t, err)

	err = store.SaveError(ctx, jobID, 2, "", "bad_value", "invalid format", "", "validation")
	require.NoError(t, err)

	errors, err := store.GetErrors(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, errors, 2)
	assert.Equal(t, 1, errors[0].Row)
	assert.Equal(t, "name", errors[0].Field)
	assert.Equal(t, "John", errors[0].Value)
	assert.Equal(t, "required", errors[0].Reason)
	assert.Equal(t, "provide a name", errors[0].Suggestion)

	assert.Equal(t, 2, errors[1].Row)
	assert.Equal(t, "", errors[1].Field)
	assert.Equal(t, "bad_value", errors[1].Value)
	assert.Equal(t, "invalid format", errors[1].Reason)
}

func TestHistoryStore_GetSnapshot_NotFound(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()

	_, err := store.GetSnapshot(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot not found")
}

func TestHistoryStore_GetRows_Empty(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()
	jobID := insertImportJob(ctx, t)

	rows, err := store.GetRows(ctx, jobID)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestStrPtr(t *testing.T) {
	assert.Nil(t, strPtr(""))
	assert.NotNil(t, strPtr("hello"))
	assert.Equal(t, "hello", *strPtr("hello"))
}

func TestSnapshotData_JSONRoundTrip(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()
	jobID := insertImportJob(ctx, t)

	original := SnapshotData{
		RowsData: []map[string]interface{}{
			{"id": float64(1), "name": "Test"},
		},
		SchemaSnapshot: map[string]interface{}{
			"module_name": "test",
			"version":     "1.0",
		},
	}

	err := store.SaveSnapshot(ctx, jobID, schema.ModuleSchema{
		ModuleName:    "test",
		SchemaVersion: "1.0",
	}, original.RowsData, nil)
	require.NoError(t, err)

	got, err := store.GetSnapshot(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, original.RowsData[0]["id"], got.RowsData[0]["id"])
	assert.Equal(t, original.RowsData[0]["name"], got.RowsData[0]["name"])
	assert.Equal(t, "test", got.SchemaSnapshot["module_name"])
}

func TestHistoryStore_SaveAndGetErrorsWithRowErrors(t *testing.T) {
	store := NewStore(dbPool)
	ctx := context.Background()
	jobID := insertImportJob(ctx, t)

	err := store.SaveRow(ctx, jobID, 1, "error", nil, nil, nil)
	require.NoError(t, err)

	err = store.SaveError(ctx, jobID, 1, "email", "bad", "invalid email", "fix format", "validation")
	require.NoError(t, err)

	err = store.SaveError(ctx, jobID, 1, "phone", "123", "too short", "add area code", "validation")
	require.NoError(t, err)

	rows, err := store.GetRows(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].RowNumber)
	assert.Len(t, rows[0].Errors, 2)
	assert.Equal(t, "email", rows[0].Errors[0].Field)
	assert.Equal(t, "phone", rows[0].Errors[1].Field)
}
