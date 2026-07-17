package shared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetTestDSN() string {
	host := getEnv("TEST_DB_HOST", "localhost")
	port := getEnv("TEST_DB_PORT", "5433")
	user := getEnv("TEST_DB_USER", "pos")
	password := getEnv("TEST_DB_PASSWORD", "admin123")
	dbname := getEnv("TEST_DB_NAME", "retail_pos_test")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=Asia/Jakarta",
		user, password, host, port, dbname)
}

func NewTestDB() (*pgxpool.Pool, error) {
	dsn := GetTestDSN()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("connect test db: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping test db: %w", err)
	}
	return pool, nil
}

func RunMigrations(pool *pgxpool.Pool, migrationsDir string) error {
	_, _ = pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS pgcrypto")

	var tableCount int
	_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount)

	if tableCount > 0 {
		_, _ = pool.Exec(context.Background(), "CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1")
		return nil
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		path := filepath.Join(migrationsDir, f)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := pool.Exec(context.Background(), string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
	}
	return nil
}

func TruncateAll(pool *pgxpool.Pool, tables ...string) error {
	for _, t := range tables {
		if _, err := pool.Exec(context.Background(), "TRUNCATE TABLE "+t+" CASCADE"); err != nil {
			return fmt.Errorf("truncate %s: %w", t, err)
		}
	}
	return nil
}

func TruncateTestData(pool *pgxpool.Pool) error {
	tables := []string{
		"import_rows", "import_errors", "import_snapshots", "import_jobs",
		"categories", "brands", "units_of_measure", "warehouses", "tax_classes",
		"products", "product_stock",
		"customers",
		"customer_groups", "stores",
		"pricing_rules",
		"users", "refresh_tokens",
		"sales", "sale_items",
		"inventory_movements",
		"audit_logs",
	}
	return TruncateAll(pool, tables...)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
