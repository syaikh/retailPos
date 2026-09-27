package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetEnv_ReturnsValue(t *testing.T) {
	_ = os.Setenv("TEST_GETENV_KEY", "hello")
	defer func() { _ = os.Unsetenv("TEST_GETENV_KEY") }()

	got := getEnv("TEST_GETENV_KEY", "fallback")
	if got != "hello" {
		t.Fatalf("getEnv = %q, want %q", got, "hello")
	}
}

func TestGetEnv_ReturnsFallbackWhenEmpty(t *testing.T) {
	_ = os.Setenv("TEST_GETENV_EMPTY", "")
	defer func() { _ = os.Unsetenv("TEST_GETENV_EMPTY") }()

	got := getEnv("TEST_GETENV_EMPTY", "fallback")
	if got != "fallback" {
		t.Fatalf("getEnv = %q, want %q", got, "fallback")
	}
}

func TestGetEnv_ReturnsFallbackWhenUnset(t *testing.T) {
	_ = os.Unsetenv("TEST_GETENV_UNSET_KEY")

	got := getEnv("TEST_GETENV_UNSET_KEY", "fallback")
	if got != "fallback" {
		t.Fatalf("getEnv = %q, want %q", got, "fallback")
	}
}

func TestGetTestDSN_ContainsAllParts(t *testing.T) {
	dsn := GetTestDSN()

	if !strings.HasPrefix(dsn, "postgres://") {
		t.Fatalf("DSN should start with postgres://, got: %s", dsn)
	}
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Fatalf("DSN should contain sslmode=disable, got: %s", dsn)
	}
	if !strings.Contains(dsn, "timezone=Asia/Jakarta") {
		t.Fatalf("DSN should contain timezone=Asia/Jakarta, got: %s", dsn)
	}
}

func TestGetTestDSN_UsesEnvOverrides(t *testing.T) {
	_ = os.Setenv("TEST_DB_HOST", "myhost")
	_ = os.Setenv("TEST_DB_PORT", "9999")
	_ = os.Setenv("TEST_DB_USER", "myuser")
	_ = os.Setenv("TEST_DB_PASSWORD", "mypass")
	_ = os.Setenv("TEST_DB_NAME", "mydb")
	defer func() {
		_ = os.Unsetenv("TEST_DB_HOST")
		_ = os.Unsetenv("TEST_DB_PORT")
		_ = os.Unsetenv("TEST_DB_USER")
		_ = os.Unsetenv("TEST_DB_PASSWORD")
		_ = os.Unsetenv("TEST_DB_NAME")
	}()

	dsn := GetTestDSN()

	if !strings.Contains(dsn, "myuser:mypass@myhost:9999/mydb") {
		t.Fatalf("DSN should use env overrides, got: %s", dsn)
	}
}

func TestGetTestDSN_Defaults(t *testing.T) {
	_ = os.Unsetenv("TEST_DB_HOST")
	_ = os.Unsetenv("TEST_DB_PORT")
	_ = os.Unsetenv("TEST_DB_USER")
	_ = os.Unsetenv("TEST_DB_PASSWORD")
	_ = os.Unsetenv("TEST_DB_NAME")

	dsn := GetTestDSN()

	if !strings.Contains(dsn, "pos:admin123@localhost:5433/retail_pos_test") {
		t.Fatalf("DSN should use defaults, got: %s", dsn)
	}
}

func TestSplitSQLStatements(t *testing.T) {
	fnBody := "CREATE FUNCTION f() RETURNS trigger AS $$\nBEGIN\n\tNEW.x := 1;\n\tRETURN NEW;\nEND;\n$$ LANGUAGE plpgsql"

	tests := []struct {
		name   string
		script string
		want   []string
	}{
		{"splits at top-level semicolons", "SELECT 1; SELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"keeps semicolon in string literal", "SELECT 'a;b'; SELECT 2;", []string{"SELECT 'a;b'", "SELECT 2"}},
		{"keeps semicolon behind doubled quote", "SELECT 'it''s; ok';", []string{"SELECT 'it''s; ok'"}},
		{"keeps dollar-quoted body intact", fnBody + "; SELECT 1;", []string{fnBody, "SELECT 1"}},
		{"keeps semicolon in line comment", "-- don't; split\nSELECT 1;", []string{"-- don't; split\nSELECT 1"}},
		{"keeps semicolon in block comment", "SELECT /* ; */ 1;", []string{"SELECT /* ; */ 1"}},
		{"keeps semicolon in quoted identifier", `SELECT "a;b" FROM t;`, []string{`SELECT "a;b" FROM t`}},
		{"keeps escaped semicolon in E-string", `SELECT E'a\';b'; SELECT 2;`, []string{`SELECT E'a\';b'`, "SELECT 2"}},
		{"drops trailing comment-only chunk", "SELECT 1; -- done", []string{"SELECT 1"}},
		{"statement without trailing semicolon", "SELECT 1", []string{"SELECT 1"}},
		{"comment-only script yields nothing", "  \n-- nothing\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSQLStatements(tt.script)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d statements %q, want %d %q", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("statement %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitSQLStatements_MigrationFiles(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "database", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no migration files found")
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		name := filepath.Base(f)
		stmts := splitSQLStatements(string(data))
		// Comment-only files are legal (038 is an intentional no-op and
		// registers itself through the recording step).
		for i, stmt := range stmts {
			if stmt == "" {
				t.Errorf("%s: statement %d is empty", name, i)
				continue
			}
			if again := splitSQLStatements(stmt); len(again) != 1 {
				t.Errorf("%s: statement %d is not atomic, resplits into %d: %.60s", name, i, len(again), stmt)
			}
		}
	}
}

func TestSplitSQLStatements_PaginationMigration(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "database", "migrations", "051_pagination_indexes.sql"))
	if err != nil {
		t.Fatalf("read 051: %v", err)
	}

	stmts := splitSQLStatements(string(data))
	if len(stmts) != 4 {
		t.Fatalf("051 splits into %d statements, want 4 (3 CREATE INDEX + 1 registration INSERT): %q", len(stmts), stmts)
	}
	for i, stmt := range stmts[:3] {
		if !strings.Contains(stmt, "CREATE INDEX CONCURRENTLY") {
			t.Errorf("statement %d does not create an index concurrently: %.80s", i, stmt)
		}
	}
	if !strings.Contains(stmts[3], "INSERT INTO schema_migrations") {
		t.Errorf("statement 3 should be the registration insert: %.80s", stmts[3])
	}
}
