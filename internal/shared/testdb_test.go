package shared

import (
	"os"
	"strings"
	"testing"
)

func TestGetEnv_ReturnsValue(t *testing.T) {
	os.Setenv("TEST_GETENV_KEY", "hello")
	defer os.Unsetenv("TEST_GETENV_KEY")

	got := getEnv("TEST_GETENV_KEY", "fallback")
	if got != "hello" {
		t.Fatalf("getEnv = %q, want %q", got, "hello")
	}
}

func TestGetEnv_ReturnsFallbackWhenEmpty(t *testing.T) {
	os.Setenv("TEST_GETENV_EMPTY", "")
	defer os.Unsetenv("TEST_GETENV_EMPTY")

	got := getEnv("TEST_GETENV_EMPTY", "fallback")
	if got != "fallback" {
		t.Fatalf("getEnv = %q, want %q", got, "fallback")
	}
}

func TestGetEnv_ReturnsFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("TEST_GETENV_UNSET_KEY")

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
	os.Setenv("TEST_DB_HOST", "myhost")
	os.Setenv("TEST_DB_PORT", "9999")
	os.Setenv("TEST_DB_USER", "myuser")
	os.Setenv("TEST_DB_PASSWORD", "mypass")
	os.Setenv("TEST_DB_NAME", "mydb")
	defer func() {
		os.Unsetenv("TEST_DB_HOST")
		os.Unsetenv("TEST_DB_PORT")
		os.Unsetenv("TEST_DB_USER")
		os.Unsetenv("TEST_DB_PASSWORD")
		os.Unsetenv("TEST_DB_NAME")
	}()

	dsn := GetTestDSN()

	if !strings.Contains(dsn, "myuser:mypass@myhost:9999/mydb") {
		t.Fatalf("DSN should use env overrides, got: %s", dsn)
	}
}

func TestGetTestDSN_Defaults(t *testing.T) {
	os.Unsetenv("TEST_DB_HOST")
	os.Unsetenv("TEST_DB_PORT")
	os.Unsetenv("TEST_DB_USER")
	os.Unsetenv("TEST_DB_PASSWORD")
	os.Unsetenv("TEST_DB_NAME")

	dsn := GetTestDSN()

	if !strings.Contains(dsn, "pos:admin123@localhost:5433/retail_pos_test") {
		t.Fatalf("DSN should use defaults, got: %s", dsn)
	}
}
