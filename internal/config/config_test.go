package config

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetConfigForTest() {
	cachedConfig = nil
	configOnce = sync.Once{}
}

func TestLoadDefaults(t *testing.T) {
	resetConfigForTest()
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("CORS_ORIGIN")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Unsetenv("STOCK_WARNING_THRESHOLD")
	_ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	defer func() { _ = os.Unsetenv("JWT_SECRET") }()

	cfg := Load()
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "http://localhost:5173", cfg.CORSOrigin)
	assert.Equal(t, "test-secret", cfg.JWTSecret)
	assert.Equal(t, 10, cfg.StockWarningThreshold)
	assert.Equal(t, 5, cfg.StockCriticalThreshold)
	assert.Equal(t, "Asia/Jakarta", cfg.Timezone.String())
	assert.Equal(t, 30, cfg.ReportRefreshDebounce)
}

func TestLoadFromEnv(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://example.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("STOCK_WARNING_THRESHOLD", "20")
	_ = os.Setenv("STOCK_CRITICAL_THRESHOLD", "8")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("STOCK_WARNING_THRESHOLD")
		_ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	}()

	cfg := Load()
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "https://example.com", cfg.CORSOrigin)
	assert.Equal(t, "test-secret", cfg.JWTSecret)
	assert.Equal(t, 20, cfg.StockWarningThreshold)
	assert.Equal(t, 8, cfg.StockCriticalThreshold)
	assert.Equal(t, "Asia/Jakarta", cfg.Timezone.String())
}

func TestGetEnvInt_Defaults(t *testing.T) {
	_ = os.Unsetenv("STOCK_WARNING_THRESHOLD")
	assert.Equal(t, 10, getEnvInt(&diagnosticLog{}, "STOCK_WARNING_THRESHOLD", 10))
	_ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	assert.Equal(t, 5, getEnvInt(&diagnosticLog{}, "STOCK_CRITICAL_THRESHOLD", 5))
}

func TestGetEnvInt_InvalidFallsBack(t *testing.T) {
	_ = os.Setenv("STOCK_WARNING_THRESHOLD", "abc")
	_ = os.Setenv("STOCK_CRITICAL_THRESHOLD", "-1")
	defer func() {
		_ = os.Unsetenv("STOCK_WARNING_THRESHOLD")
		_ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	}()

	var pending diagnosticLog
	assert.Equal(t, 10, getEnvInt(&pending, "STOCK_WARNING_THRESHOLD", 10))
	assert.Equal(t, 5, getEnvInt(&pending, "STOCK_CRITICAL_THRESHOLD", 5))

	// Both must be deferred diagnostics, not printed, so they reach the
	// configured logger in the same format as every other startup line.
	require.Len(t, pending, 2, "each invalid value should record one diagnostic")
	for _, d := range pending {
		assert.Equal(t, slog.LevelWarn, d.level)
		assert.Equal(t, "invalid env, using default", d.msg)
		assert.Contains(t, d.attrs, "key")
	}
}

func TestGetEnvInt_ZeroAllowed(t *testing.T) {
	_ = os.Setenv("STOCK_CRITICAL_THRESHOLD", "0")
	defer func() { _ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD") }()
	assert.Equal(t, 0, getEnvInt(&diagnosticLog{}, "STOCK_CRITICAL_THRESHOLD", 5))
}

func TestLoad_InvalidCORSOriginURL(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("CORS_ORIGIN", "://invalid")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	assert.Equal(t, "://invalid", cfg.CORSOrigin)
	assert.Equal(t, "development", cfg.Env)
}

func TestLoad_ExplicitLogLevel(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("LOG_LEVEL", "warn")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	cfg := Load()
	assert.Equal(t, "warn", cfg.LogLevel)
}

func TestLoad_ExplicitJWTSecretRefresh(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("JWT_SECRET_REFRESH", "refresh-secret")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("JWT_SECRET_REFRESH")
	}()

	cfg := Load()
	assert.Equal(t, "refresh-secret", cfg.JWTSecretRefresh)
}

func TestLoad_ExplicitReportRefreshDebounce(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("REPORT_REFRESH_DEBOUNCE", "45")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("REPORT_REFRESH_DEBOUNCE")
	}()

	cfg := Load()
	assert.Equal(t, 45, cfg.ReportRefreshDebounce)
}

func TestLoad_ProductionCORSExit(t *testing.T) {
	if os.Getenv("TEST_CORS_PRODUCTION") == "1" {
		resetConfigForTest()
		_ = os.Setenv("ENV", "production")
		_ = os.Setenv("CORS_ORIGIN", "*")
		_ = os.Setenv("JWT_SECRET", "test-secret")
		Load()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLoad_ProductionCORSExit")
	cmd.Env = append(os.Environ(), "TEST_CORS_PRODUCTION=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatal("expected process to exit with status 1 for CORS_ORIGIN=* in production")
}

func TestLoad_ProductionValidCORS(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://myapp.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "https://myapp.com", cfg.CORSOrigin)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_ProductionLogLevelInfo(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://example.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Unsetenv("LOG_LEVEL")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestResolveDBSSLMode(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		env      string
		want     string
		explicit bool
	}{
		{name: "production defaults to require", raw: "", env: "production", want: "require", explicit: false},
		{name: "development defaults to disable", raw: "", env: "development", want: "disable", explicit: false},

		// The point of DB_SSLMODE: a deployment that genuinely cannot use TLS
		// says so, instead of inheriting a default chosen for a different
		// environment.
		{name: "explicit disable overrides production", raw: "disable", env: "production", want: "disable", explicit: true},
		{name: "explicit verify-full in development", raw: "verify-full", env: "development", want: "verify-full", explicit: true},
		{name: "prefer honoured", raw: "prefer", env: "production", want: "prefer", explicit: true},
		{name: "verify-ca honoured", raw: "verify-ca", env: "production", want: "verify-ca", explicit: true},
		{name: "allow honoured", raw: "allow", env: "production", want: "allow", explicit: true},
		{name: "require honoured", raw: "require", env: "development", want: "require", explicit: true},

		// Secrets files are hand-edited; trailing whitespace is common.
		{name: "surrounding whitespace trimmed", raw: "  require  ", env: "development", want: "require", explicit: true},

		// An unrecognised value must not reach the driver, where it would
		// surface as an opaque connection error with no hint that a typo in
		// configuration was the cause.
		{name: "invalid falls back to production default", raw: "yes-please", env: "production", want: "require", explicit: false},
		{name: "invalid falls back to development default", raw: "true", env: "development", want: "disable", explicit: false},
		{name: "whitespace-only is not treated as explicit", raw: "   ", env: "production", want: "require", explicit: false},
		{name: "value is case sensitive", raw: "REQUIRE", env: "production", want: "require", explicit: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, explicit, rejected := resolveDBSSLMode(tt.raw, tt.env)
			assert.Equal(t, tt.want, got, "resolveDBSSLMode(%q, %q)", tt.raw, tt.env)
			assert.Equal(t, tt.explicit, explicit, "explicit flag for %q", tt.raw)
			// A rejected value is reported back so Load can log it; a usable or
			// unset value must not look like a rejection.
			wantRejected := ""
			if !tt.explicit && strings.TrimSpace(tt.raw) != "" {
				wantRejected = strings.TrimSpace(tt.raw)
			}
			assert.Equal(t, wantRejected, rejected, "rejected input for %q", tt.raw)
		})
	}
}

func TestResolveDBSSLModeNeverEmpty(t *testing.T) {
	// An empty sslmode would be interpolated into the DSN as `sslmode=`, which
	// pgx reads as the library default rather than a deliberate choice.
	for _, env := range []string{"production", "development", "staging", ""} {
		got, _, _ := resolveDBSSLMode("", env)
		assert.NotEmpty(t, got, "resolveDBSSLMode with env %q", env)
	}
}

func TestLoad_DBSSLModeDefaultsByEnv(t *testing.T) {
	for _, tt := range []struct{ env, want string }{
		{env: "production", want: "require"},
		{env: "development", want: "disable"},
	} {
		resetConfigForTest()
		_ = os.Setenv("ENV", tt.env)
		_ = os.Setenv("CORS_ORIGIN", "https://example.com")
		_ = os.Setenv("JWT_SECRET", "test-secret")
		_ = os.Unsetenv("DB_SSLMODE")

		cfg := Load()
		assert.Equal(t, tt.want, cfg.DBSSLMode, "default sslmode for ENV=%s", tt.env)

		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
	}
}

func TestLoad_DBSSLModeExplicitOverrideInProduction(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://example.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("DB_SSLMODE", "disable")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("DB_SSLMODE")
	}()

	cfg := Load()
	assert.Equal(t, "disable", cfg.DBSSLMode)
}

// startupMsgs returns the messages of the diagnostics Load collected, for
// assertions that do not care about levels or attributes.
func startupMsgs(c *Config) []string {
	var out []string
	for _, d := range c.startupDiagnostics {
		out = append(out, d.msg)
	}
	return out
}

func TestLoad_DefersDiagnosticsUntilLoggerConfigured(t *testing.T) {
	// Load runs before shared.InitLogger, so a diagnostic logged directly from
	// Load would be written by the default slog text handler even in production.
	// That is what put "database TLS disabled in production" into a different
	// format from every other line. These must be collected instead.
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://example.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("DB_SSLMODE", "disable")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("DB_SSLMODE")
	}()

	cfg := Load()
	msgs := startupMsgs(cfg)

	assert.Contains(t, msgs, "database TLS disabled in production. Ensure the database is unreachable from outside the deployment network, and record this decision.",
		"the production TLS notice must be deferred, not logged during Load")
}

func TestLoad_DeferredDiagnosticsIncludeInvalidDBSSLMode(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://example.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("DB_SSLMODE", "requrie")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("DB_SSLMODE")
	}()

	cfg := Load()
	assert.Equal(t, "require", cfg.DBSSLMode, "an unusable value falls back to the environment default")
	assert.Contains(t, startupMsgs(cfg), "invalid DB_SSLMODE, using environment default",
		"a silently ignored DB_SSLMODE must be reported")
}

func TestLoad_DeferredDiagnosticsPreserveLevel(t *testing.T) {
	// The defaulted-sslmode notice is informational; the TLS notice is a
	// warning. Collapsing them would let operators filter warnings and lose the
	// fact that TLS was disabled.
	resetConfigForTest()
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGIN", "https://example.com")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("DB_SSLMODE", "disable")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGIN")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("DB_SSLMODE")
	}()

	cfg := Load()
	levels := map[string]slog.Level{}
	for _, d := range cfg.startupDiagnostics {
		levels[d.msg] = d.level
	}
	assert.Equal(t, slog.LevelWarn, levels["database TLS disabled in production. Ensure the database is unreachable from outside the deployment network, and record this decision."])
	assert.Equal(t, slog.LevelInfo, levels["database sslmode defaulted from environment"])
}

func TestLogStartupDiagnostics_NilReceiverIsSafe(t *testing.T) {
	// main.go calls this on the result of Load, which is non-nil, but a nil
	// receiver must not panic if that ever changes.
	var c *Config
	assert.NotPanics(t, func() { c.LogStartupDiagnostics() })
}
