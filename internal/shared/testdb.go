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
	_, _ = pool.Exec(context.Background(), "CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1")

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

	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	var appliedCount int
	_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&appliedCount)

	if appliedCount == len(files) {
		return nil
	}

	if appliedCount == 0 {
		var tableCount int
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name NOT IN ('schema_migrations')").Scan(&tableCount)
		if tableCount > 0 {
			for _, f := range files {
				_, _ = pool.Exec(context.Background(), "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", f)
			}
			return nil
		}
	}

	// One connection for every remaining statement: migrations carry explicit
	// BEGIN/COMMIT blocks that must stay on a single session, and psql (the
	// production runner) also applies each file on one connection.
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	for _, f := range files {
		var applied bool
		_ = conn.QueryRow(context.Background(), "SELECT TRUE FROM schema_migrations WHERE filename = $1", f).Scan(&applied)
		if applied {
			continue
		}

		path := filepath.Join(migrationsDir, f)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		// Statements execute one per query, the way psql applies a script:
		// Postgres wraps a multi-statement message in a single implicit
		// transaction, which CREATE INDEX CONCURRENTLY rejects.
		for _, stmt := range splitSQLStatements(string(content)) {
			if _, err := conn.Exec(context.Background(), stmt); err != nil {
				return fmt.Errorf("exec %s: %w", f, err)
			}
		}
		// Some migration files self-register into schema_migrations; ON CONFLICT
		// keeps the recording step idempotent for those.
		if _, err := conn.Exec(context.Background(), "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT (filename) DO NOTHING", f); err != nil {
			return fmt.Errorf("record %s: %w", f, err)
		}
	}
	return nil
}

// splitSQLStatements splits a SQL script into its individual statements at
// top-level semicolons. Semicolons inside string literals, dollar-quoted
// bodies, and comments never split a statement, so function and trigger
// definitions survive intact. Comments that trail the last statement are
// dropped rather than sent as their own statement.
func splitSQLStatements(script string) []string {
	var stmts []string
	var buf strings.Builder
	hasCode := false

	flush := func() {
		if hasCode {
			stmts = append(stmts, strings.TrimSpace(buf.String()))
		}
		buf.Reset()
		hasCode = false
	}

	i := 0
	for i < len(script) {
		c := script[i]
		switch {
		case c == '-' && i+1 < len(script) && script[i+1] == '-':
			j := i + 2
			for j < len(script) && script[j] != '\n' {
				j++
			}
			buf.WriteString(script[i:j])
			i = j
		case c == '/' && i+1 < len(script) && script[i+1] == '*':
			j := i + 2
			depth := 1
			for j < len(script) && depth > 0 {
				if j+1 < len(script) && script[j] == '/' && script[j+1] == '*' {
					depth++
					j += 2
					continue
				}
				if j+1 < len(script) && script[j] == '*' && script[j+1] == '/' {
					depth--
					j += 2
					continue
				}
				j++
			}
			buf.WriteString(script[i:j])
			i = j
		case c == '\'':
			escaped := i > 0 && (script[i-1] == 'E' || script[i-1] == 'e') &&
				(i < 2 || !isSQLIdentChar(script[i-2]))
			j := i + 1
			for j < len(script) {
				if escaped && script[j] == '\\' && j+1 < len(script) {
					j += 2
					continue
				}
				if script[j] == '\'' {
					if j+1 < len(script) && script[j+1] == '\'' {
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			buf.WriteString(script[i:j])
			hasCode = true
			i = j
		case c == '"':
			j := i + 1
			for j < len(script) {
				if script[j] == '"' {
					if j+1 < len(script) && script[j+1] == '"' {
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			buf.WriteString(script[i:j])
			hasCode = true
			i = j
		case c == '$':
			closer, ok := dollarQuoteOpener(script, i)
			if !ok {
				buf.WriteByte(c)
				i++
				continue
			}
			j := strings.Index(script[i+len(closer):], closer)
			if j < 0 {
				j = len(script)
			} else {
				j = i + len(closer) + j + len(closer)
			}
			buf.WriteString(script[i:j])
			hasCode = true
			i = j
		case c == ';':
			flush()
			i++
		default:
			if !isSQLSpace(c) {
				hasCode = true
			}
			buf.WriteByte(c)
			i++
		}
	}
	flush()
	return stmts
}

// dollarQuoteOpener returns the full opening delimiter ("$" + tag + "$") of a
// dollar-quoted string starting at script[i], which must hold a '$'.
func dollarQuoteOpener(script string, i int) (string, bool) {
	j := i + 1
	if j >= len(script) {
		return "", false
	}
	if script[j] == '$' {
		return "$$", true
	}
	if !isSQLIdentStart(script[j]) {
		return "", false
	}
	for j < len(script) && isSQLIdentChar(script[j]) {
		j++
	}
	if j >= len(script) || script[j] != '$' {
		return "", false
	}
	return script[i : j+1], true
}

func isSQLIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isSQLIdentChar(c byte) bool {
	return isSQLIdentStart(c) || (c >= '0' && c <= '9')
}

func isSQLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
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
		"customer_groups", "shifts", "stores",
		"pricing_rules",
		"users", "refresh_tokens",
		"sales", "sale_items", "sale_payments",
		"inventory_movements",
		"stock_opnames",
		"storage_locations",
		"audit_logs",
		"consignment_payouts",
		"consignment_settlement_items",
		"consignment_settlements",
		"consignment_sale_items",
		"consignment_return_items",
		"consignment_returns",
		"consignment_pending_returns",
		"consignment_stock",
		"consignment_receipt_items",
		"consignment_receipts",
		"consignment_terms",
		"consignment_arrangements",
	}
	return TruncateAll(pool, tables...)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
