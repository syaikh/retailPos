package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
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
