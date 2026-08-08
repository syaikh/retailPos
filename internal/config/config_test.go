package config

import (
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, 10, getEnvInt("STOCK_WARNING_THRESHOLD", 10))
	_ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	assert.Equal(t, 5, getEnvInt("STOCK_CRITICAL_THRESHOLD", 5))
}

func TestGetEnvInt_InvalidFallsBack(t *testing.T) {
	_ = os.Setenv("STOCK_WARNING_THRESHOLD", "abc")
	_ = os.Setenv("STOCK_CRITICAL_THRESHOLD", "-1")
	defer func() {
		_ = os.Unsetenv("STOCK_WARNING_THRESHOLD")
		_ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	}()
	assert.Equal(t, 10, getEnvInt("STOCK_WARNING_THRESHOLD", 10))
	assert.Equal(t, 5, getEnvInt("STOCK_CRITICAL_THRESHOLD", 5))
}

func TestGetEnvInt_ZeroAllowed(t *testing.T) {
	_ = os.Setenv("STOCK_CRITICAL_THRESHOLD", "0")
	defer func() { _ = os.Unsetenv("STOCK_CRITICAL_THRESHOLD") }()
	assert.Equal(t, 0, getEnvInt("STOCK_CRITICAL_THRESHOLD", 5))
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

func TestLoad_ExplicitStockMinimum(t *testing.T) {
	resetConfigForTest()
	_ = os.Setenv("ENV", "development")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("STOCK_MINIMUM", "3")
	defer func() {
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("STOCK_MINIMUM")
	}()

	cfg := Load()
	assert.Equal(t, 3, cfg.StockMinimum)
	assert.Equal(t, 10, cfg.StockWarningThreshold)
	assert.Equal(t, 5, cfg.StockCriticalThreshold)
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
