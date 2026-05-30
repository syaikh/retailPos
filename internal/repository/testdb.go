package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestDB represents a test database helper
type TestDB struct {
	pool   *pgxpool.Pool
	dbName string
}

// NewTestDB creates a new test database
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Connect to postgres system database to create test database
	systemPool, err := createTestPool("postgres")
	require.NoError(t, err, "Failed to connect to system database")

	// Generate unique database name
	dbName := fmt.Sprintf("retailpos_test_%d_%d", time.Now().Unix(), time.Now().Nanosecond())

	// Create test database
	_, err = systemPool.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err, "Failed to create test database")

	systemPool.Close()

	// Connect to the new test database
	testPool, err := createTestPool(dbName)
	require.NoError(t, err, "Failed to connect to test database")

	testDB := &TestDB{
		pool:   testPool,
		dbName: dbName,
	}

	// Run migrations and seed data
	testDB.setupSchema(t)

	return testDB
}

// Close drops the test database and closes connections
func (tdb *TestDB) Close(t *testing.T) {
	t.Helper()

	if tdb.pool != nil {
		tdb.pool.Close()
	}

	// Connect back to system database to drop test database
	systemPool, err := createTestPool("postgres")
	if err != nil {
		t.Logf("Warning: Failed to connect to system database for cleanup: %v", err)
		return
	}
	defer systemPool.Close()

	// Force disconnect all connections to our test database
	_, err = systemPool.Exec(context.Background(),
		fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'", tdb.dbName))
	if err != nil {
		t.Logf("Warning: Failed to terminate connections: %v", err)
	}

	// Drop the test database
	_, err = systemPool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.dbName))
	if err != nil {
		t.Logf("Warning: Failed to drop test database %s: %v", tdb.dbName, err)
	}
}

// Pool returns the database connection pool
func (tdb *TestDB) Pool() *pgxpool.Pool {
	return tdb.pool
}

// setupSchema runs migrations and seeds test data
func (tdb *TestDB) setupSchema(t *testing.T) {
	t.Helper()

	// Get project root directory (go up from internal/repository)
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Find project root by looking for go.mod
	projectRoot := wd
	for {
		if _, err := os.Stat(projectRoot + "/go.mod"); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod)")
		}
		projectRoot = parent
	}

// Read migration file
 	migrationPath := filepath.Join(projectRoot, "database/migrations/001_create_tables.sql")
 	migrationSQL, err := os.ReadFile(migrationPath)
 	require.NoError(t, err, "Failed to read migration file")

 	// Create pgcrypto extension for password hashing
 	_, err = tdb.pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS pgcrypto")
 	require.NoError(t, err, "Failed to create pgcrypto extension")

 	// Run migration
 	_, err = tdb.pool.Exec(context.Background(), string(migrationSQL))
 	require.NoError(t, err, "Failed to run migration")

 	// Run additional migrations for extended schema
  	migrationFiles := []string{
  		"database/migrations/002_upsert_tables.sql",
  		"database/migrations/003_seed_data.sql",
  		"database/migrations/004_add_aggregation_indexes.sql",
  		"database/migrations/005_product_extensions.sql",
  		"database/migrations/006_product_schema_update.sql",
  		"database/migrations/007_drop_dead_product_columns.sql",
  	}

 	for _, migrationFile := range migrationFiles {
 		migrationPath := filepath.Join(projectRoot, migrationFile)
 		migrationSQL, err := os.ReadFile(migrationPath)
 		if err != nil {
 			t.Logf("Skipping migration file %s: %v", migrationFile, err)
 			continue
 		}
 		_, err = tdb.pool.Exec(context.Background(), string(migrationSQL))
 		require.NoError(t, err, "Failed to run migration file %s", migrationFile)
 	}

 	// Run seed files
 	seedFiles := []string{
 		"database/seeds/001_roles.sql",
 		"database/seeds/002_permissions.sql",
 		"database/seeds/003_role_permissions.sql",
 		"database/seeds/004_users.sql",
 		"database/seeds/005_stores.sql",
 		"database/seeds/006_categories.sql",
 		"database/seeds/008_brands.sql",
 		"database/seeds/010_units_of_measure.sql",
 		"database/seeds/009_tax_classes.sql",
 		"database/seeds/007_products.sql",
 	}

	for _, seedFile := range seedFiles {
		seedPath := filepath.Join(projectRoot, seedFile)
		seedSQL, err := os.ReadFile(seedPath)
		if err != nil {
			// Skip missing seed files (they might be optional)
			t.Logf("Skipping seed file %s: %v", seedFile, err)
			continue
		}

		_, err = tdb.pool.Exec(context.Background(), string(seedSQL))
		require.NoError(t, err, "Failed to run seed file %s", seedFile)
	}
}

// createTestPool creates a connection pool for testing
func createTestPool(dbName string) (*pgxpool.Pool, error) {
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		host = os.Getenv("DB_HOST")
	}
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = os.Getenv("DB_PORT")
	}
	if port == "" {
		port = "5433"
	}
	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		user = os.Getenv("DB_USER")
	}
	if user == "" {
		user = "devuser"
	}
	password := os.Getenv("TEST_DB_PASSWORD")
	if password == "" {
		password = os.Getenv("DB_PASSWORD")
	}
	if password == "" {
		password = "admin123"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=Asia/Jakarta",
		user, password, host, port, dbName)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Smaller pool for tests
	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
