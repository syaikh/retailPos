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
	os.Unsetenv("ENV")
	os.Unsetenv("CORS_ORIGIN")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Unsetenv("STOCK_WARNING_THRESHOLD")
	os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	defer os.Unsetenv("JWT_SECRET")

	cfg := Load()
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "http://localhost:5173", cfg.CORSOrigin)
	assert.Equal(t, "test-secret", cfg.JWTSecret)
	assert.Equal(t, 10, cfg.StockWarningThreshold)
	assert.Equal(t, 5, cfg.StockCriticalThreshold)
	assert.Equal(t, "Asia/Jakarta", cfg.Timezone.String())
}

func TestLoadFromEnv(t *testing.T) {
	resetConfigForTest()
	os.Setenv("ENV", "production")
	os.Setenv("CORS_ORIGIN", "https://example.com")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("STOCK_WARNING_THRESHOLD", "20")
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "8")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("CORS_ORIGIN")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("STOCK_WARNING_THRESHOLD")
		os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
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
	os.Unsetenv("STOCK_WARNING_THRESHOLD")
	assert.Equal(t, 10, getEnvInt("STOCK_WARNING_THRESHOLD", 10))
	os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	assert.Equal(t, 5, getEnvInt("STOCK_CRITICAL_THRESHOLD", 5))
}

func TestGetEnvInt_InvalidFallsBack(t *testing.T) {
	os.Setenv("STOCK_WARNING_THRESHOLD", "abc")
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "-1")
	defer func() {
		os.Unsetenv("STOCK_WARNING_THRESHOLD")
		os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	}()
	assert.Equal(t, 10, getEnvInt("STOCK_WARNING_THRESHOLD", 10))
	assert.Equal(t, 5, getEnvInt("STOCK_CRITICAL_THRESHOLD", 5))
}

func TestGetEnvInt_ZeroAllowed(t *testing.T) {
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "0")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	assert.Equal(t, 0, getEnvInt("STOCK_CRITICAL_THRESHOLD", 5))
}

func TestLoad_InvalidCORSOriginURL(t *testing.T) {
	resetConfigForTest()
	os.Setenv("ENV", "development")
	os.Setenv("CORS_ORIGIN", "://invalid")
	os.Setenv("JWT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("CORS_ORIGIN")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	assert.Equal(t, "://invalid", cfg.CORSOrigin)
	assert.Equal(t, "development", cfg.Env)
}

func TestLoad_ExplicitLogLevel(t *testing.T) {
	resetConfigForTest()
	os.Setenv("ENV", "development")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("LOG_LEVEL", "warn")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg := Load()
	assert.Equal(t, "warn", cfg.LogLevel)
}

func TestLoad_ExplicitStockMinimum(t *testing.T) {
	resetConfigForTest()
	os.Setenv("ENV", "development")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("STOCK_MINIMUM", "3")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("STOCK_MINIMUM")
	}()

	cfg := Load()
	assert.Equal(t, 3, cfg.StockMinimum)
	assert.Equal(t, 10, cfg.StockWarningThreshold)
	assert.Equal(t, 5, cfg.StockCriticalThreshold)
}

func TestLoad_ExplicitJWTSecretRefresh(t *testing.T) {
	resetConfigForTest()
	os.Setenv("ENV", "development")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_SECRET_REFRESH", "refresh-secret")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_SECRET_REFRESH")
	}()

	cfg := Load()
	assert.Equal(t, "refresh-secret", cfg.JWTSecretRefresh)
}

func TestLoad_ProductionCORSExit(t *testing.T) {
	if os.Getenv("TEST_CORS_PRODUCTION") == "1" {
		resetConfigForTest()
		os.Setenv("ENV", "production")
		os.Setenv("CORS_ORIGIN", "*")
		os.Setenv("JWT_SECRET", "test-secret")
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
	os.Setenv("ENV", "production")
	os.Setenv("CORS_ORIGIN", "https://myapp.com")
	os.Setenv("JWT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("CORS_ORIGIN")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "https://myapp.com", cfg.CORSOrigin)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_ProductionLogLevelInfo(t *testing.T) {
	resetConfigForTest()
	os.Setenv("ENV", "production")
	os.Setenv("CORS_ORIGIN", "https://example.com")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Unsetenv("LOG_LEVEL")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("CORS_ORIGIN")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	assert.Equal(t, "info", cfg.LogLevel)
}
