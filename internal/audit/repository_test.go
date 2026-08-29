package audit

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/metrics"
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

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestAuditRepository_CreateAndGet(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	var userID int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('audit_repo_user', 'audit_repo@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	t.Run("Create audit log with all fields", func(t *testing.T) {
		al := &Log{
			UserID:     &userID,
			Role:       "admin",
			Action:     "test_action_create_full",
			EntityType: "product",
			EntityID:   intPtr(123),
			IPAddress:  "192.168.1.1",
			UserAgent:  "GoTest/1.0",
			OldValues:  map[string]interface{}{"name": "old_name", "price": 1000},
			NewValues:  map[string]interface{}{"name": "new_name", "price": 2000},
		}
		err := repo.CreateAuditLog(ctx, al)
		require.NoError(t, err)
		require.Greater(t, al.ID, 0)
	})

	t.Run("Create audit log without user", func(t *testing.T) {
		al := &Log{
			Role:       "system",
			Action:     "test_action_no_user",
			EntityType: "settings",
		}
		err := repo.CreateAuditLog(ctx, al)
		require.NoError(t, err)
		require.Greater(t, al.ID, 0)
	})

	t.Run("Create audit log with minimal fields", func(t *testing.T) {
		al := &Log{
			Action: "test_action_minimal",
		}
		err := repo.CreateAuditLog(ctx, al)
		require.NoError(t, err)
		require.Greater(t, al.ID, 0)
	})

	t.Run("GetAuditLogs returns logs with total count", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		assert.GreaterOrEqual(t, len(logs), 3)
	})

	t.Run("GetAuditLogs limit and offset", func(t *testing.T) {
		first, total, err := repo.GetAuditLogs(ctx, 1, 0, nil, "", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.Len(t, first, 1)
		require.Greater(t, total, 1)

		second, _, err := repo.GetAuditLogs(ctx, 1, 1, nil, "", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.Len(t, second, 1)
		assert.NotEqual(t, first[0].ID, second[0].ID)
	})

	t.Run("GetAuditLogs filtered by userID", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, &userID, "", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, l := range logs {
			require.NotNil(t, l.UserID)
			assert.Equal(t, userID, *l.UserID)
		}
	})

	t.Run("GetAuditLogs filtered by action", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "test_action_create_full", "", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, logs, 1)
		assert.Equal(t, "test_action_create_full", logs[0].Action)
	})

	t.Run("GetAuditLogs filtered by entityType", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "", "product", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, l := range logs {
			assert.Equal(t, "product", l.EntityType)
		}
	})

	t.Run("GetAuditLogs filtered by entityID", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "", "product", intPtr(123), nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, logs, 1)
		assert.Equal(t, 123, *logs[0].EntityID)
	})

	t.Run("GetAuditLogs search matches username", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "audit_repo_user", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		for _, l := range logs {
			assert.Equal(t, "audit_repo_user", l.Username)
		}
	})

	t.Run("GetAuditLogs search matches action", func(t *testing.T) {
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "test_action_no_user", "", "", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "test_action_no_user", logs[0].Action)
	})

	t.Run("GetAuditLogs with date range", func(t *testing.T) {
		now := time.Now()
		start := now.Add(-1 * time.Hour)
		end := now.Add(1 * time.Hour)
		gotLogs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "", "", nil, &start, &end)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		assert.GreaterOrEqual(t, len(gotLogs), 3)
	})

	t.Run("GetAuditLogs with future start date returns empty", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "", "", nil, &future, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, logs)
	})
}

func TestAuditRepository_StoreAttribution(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	var storeID int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name, is_active) VALUES ('audit_store', true) RETURNING id`).Scan(&storeID)
	require.NoError(t, err)
	storeIDPtr := storeID

	var userID int
	err = dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('audit_store_user', 'audit_store@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	t.Run("Create and GetAuditLogs returns store_id and joined store_name", func(t *testing.T) {
		al := &Log{
			UserID:     &userID,
			StoreID:    &storeIDPtr,
			Role:       "admin",
			Action:     "test_action_store_attr",
			EntityType: "product",
			EntityID:   intPtr(321),
		}
		err := repo.CreateAuditLog(ctx, al)
		require.NoError(t, err)
		require.Greater(t, al.ID, 0)

		logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "test_action_store_attr", "", nil, nil, nil)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, logs, 1)
		require.NotNil(t, logs[0].StoreID)
		assert.Equal(t, storeID, *logs[0].StoreID)
		assert.Equal(t, "audit_store", logs[0].StoreName)
	})

	t.Run("GetAuditLogByID returns store_id and joined store_name", func(t *testing.T) {
		al := &Log{
			UserID:     &userID,
			StoreID:    &storeIDPtr,
			Role:       "admin",
			Action:     "test_action_store_attr_byid",
			EntityType: "product",
		}
		require.NoError(t, repo.CreateAuditLog(ctx, al))

		got, err := repo.GetAuditLogByID(ctx, al.ID)
		require.NoError(t, err)
		require.NotNil(t, got.StoreID)
		assert.Equal(t, storeID, *got.StoreID)
		assert.Equal(t, "audit_store", got.StoreName)
	})

	t.Run("Log without store_id has nil store and empty store_name", func(t *testing.T) {
		al := &Log{
			UserID:     &userID,
			Role:       "system",
			Action:     "test_action_no_store",
			EntityType: "settings",
		}
		require.NoError(t, repo.CreateAuditLog(ctx, al))

		got, err := repo.GetAuditLogByID(ctx, al.ID)
		require.NoError(t, err)
		assert.Nil(t, got.StoreID)
		assert.Empty(t, got.StoreName)
	})
}

func TestAuditRepository_AppendOnly(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	var userID int
	err := dbPool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash, role_id) VALUES ('audit_immutable_user', 'audit_immutable@test.com', 'hash', 1) ON CONFLICT (username) DO UPDATE SET email = excluded.email RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	al := &Log{
		UserID:     &userID,
		Role:       "admin",
		Action:     "test_action_immutable",
		EntityType: "product",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))
	require.Greater(t, al.ID, 0)

	t.Run("UPDATE is rejected by the append-only trigger", func(t *testing.T) {
		_, err := dbPool.Exec(ctx, `UPDATE audit_logs SET description = 'mutated' WHERE id = $1`, al.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "append-only")
	})

	t.Run("DELETE is rejected by the append-only trigger", func(t *testing.T) {
		_, err := dbPool.Exec(ctx, `DELETE FROM audit_logs WHERE id = $1`, al.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "append-only")
	})

	t.Run("bypass allowed when maintenance GUC is set", func(t *testing.T) {
		conn, err := dbPool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()
		_, err = conn.Exec(ctx, `SET app.allow_audit_mod = 'on'`)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, `UPDATE audit_logs SET description = 'backfilled' WHERE id = $1`, al.ID)
		require.NoError(t, err)
	})
}

func TestAuditRepository_GetAuditLogs_CreatedAtJakartaTimezone(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &Log{
		Role:       "admin",
		Action:     "timezone_format_test_" + time.Now().Format("0102150405"),
		EntityType: "product",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))
	require.Greater(t, al.ID, 0)

	logs, total, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", al.Action, "", nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, logs, 1)

	createdAt := logs[0].CreatedAt
	assert.NotEmpty(t, createdAt)

	// Must match ISO 8601 with Jakarta +07:00 offset: YYYY-MM-DDTHH:MM:SS+07:00
	jakartaFormat := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+07:00$`)
	assert.Regexp(t, jakartaFormat, createdAt, "CreatedAt should be in Jakarta timezone format (YYYY-MM-DDTHH:MM:SS+07:00)")
}

func TestAuditRepository_GetAuditLogByID_CreatedAtJakartaTimezone(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	repo := NewRepository(dbPool)
	ctx := context.Background()

	al := &Log{
		Role:       "admin",
		Action:     "timezone_byid_test_" + time.Now().Format("0102150405"),
		EntityType: "product",
	}
	require.NoError(t, repo.CreateAuditLog(ctx, al))
	require.Greater(t, al.ID, 0)

	got, err := repo.GetAuditLogByID(ctx, al.ID)
	require.NoError(t, err)

	createdAt := got.CreatedAt
	assert.NotEmpty(t, createdAt)

	jakartaFormat := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+07:00$`)
	assert.Regexp(t, jakartaFormat, createdAt, "CreatedAt should be in Jakarta timezone format (YYYY-MM-DDTHH:MM:SS+07:00)")
}

func intPtr(i int) *int {
	return &i
}

func TestAuditRepository_CreateAuditLog_FailureIncrementsMetric(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	before := metrics.AuditWriteFailures.Value()

	// action is varchar(100) NOT NULL; an over-length value forces a
	// deterministic DB write failure unrelated to fixture data.
	al := &Log{
		Role:       "system",
		Action:     strings.Repeat("x", 200),
		EntityType: "product",
	}
	err := repo.CreateAuditLog(ctx, al)
	require.Error(t, err)

	after := metrics.AuditWriteFailures.Value()
	assert.Equal(t, before+1, after, "a failed audit write must increment the failure metric")
}

func TestAuditRepository_CorrelationID(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := shared.SetRequestID(context.Background(), "req-trace-123")

	t.Run("correlation id is taken from request context when not set", func(t *testing.T) {
		al := &Log{Role: "admin", Action: "corr_from_ctx", EntityType: "system"}
		require.NoError(t, repo.CreateAuditLog(ctx, al))
		require.Greater(t, al.ID, 0)

		got, err := repo.GetAuditLogByID(ctx, al.ID)
		require.NoError(t, err)
		assert.Equal(t, "req-trace-123", got.CorrelationID)
	})

	t.Run("explicit correlation id overrides context", func(t *testing.T) {
		al := &Log{Role: "admin", Action: "corr_explicit", EntityType: "system", CorrelationID: "explicit-xyz"}
		require.NoError(t, repo.CreateAuditLog(ctx, al))

		got, err := repo.GetAuditLogByID(ctx, al.ID)
		require.NoError(t, err)
		assert.Equal(t, "explicit-xyz", got.CorrelationID)
	})

	t.Run("list items carry correlation id", func(t *testing.T) {
		al := &Log{Role: "admin", Action: "corr_list", EntityType: "system", CorrelationID: "list-trace"}
		require.NoError(t, repo.CreateAuditLog(ctx, al))

		logs, _, err := repo.GetAuditLogs(ctx, 10, 0, nil, "", "corr_list", "system", nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Equal(t, "list-trace", logs[0].CorrelationID)
	})
}

func TestAuditRepository_PurgeOlderThan(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, err := dbPool.Exec(ctx, `INSERT INTO audit_logs (role, action, entity_type, created_at) VALUES ('system', 'purge_stale', 'system', '2000-01-01 00:00:00+00')`)
	require.NoError(t, err)

	recent := &Log{Role: "admin", Action: "purge_recent", EntityType: "system"}
	require.NoError(t, repo.CreateAuditLog(ctx, recent))

	affected, err := repo.PurgeOlderThan(ctx, time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "exactly the 2000-01-01 row should be purged")

	remaining, _, err := repo.GetAuditLogs(ctx, 100, 0, nil, "", "purge_recent", "system", nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, remaining, 1, "the recent row must survive retention purge")
}
