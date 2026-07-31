package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	Env                    string
	CORSOrigin             string
	JWTSecret              string
	JWTSecretRefresh       string
	StockWarningThreshold  int
	StockCriticalThreshold int
	StockMinimum           int
	CartHoldTTLHours       int
	LogLevel               string
	Timezone               *time.Location
}

var defaultLocation *time.Location

func init() {
	var err error
	defaultLocation, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		fmt.Printf("Warning: failed to load Asia/Jakarta timezone: %v. Falling back to UTC.\n", err)
		defaultLocation = time.UTC
	}
}

var cachedConfig *Config
var configOnce sync.Once

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		fmt.Printf("Warning: invalid env %s=%q, using default %d\n", key, val, defaultVal)
		return defaultVal
	}
	return n
}

func Load() *Config {
	configOnce.Do(func() {
		env := os.Getenv("ENV")
		if env == "" {
			env = "development"
			slog.Warn("ENV environment variable not set, defaulting to 'development'. Set ENV=production for production deployments.")
		}

		corsOrigin := os.Getenv("CORS_ORIGIN")
		if corsOrigin == "" {
			corsOrigin = "http://localhost:5173"
		}
		if env == "production" && corsOrigin == "*" {
			slog.Error("CORS_ORIGIN must not be '*' in production. Set it to your actual domain.")
			os.Exit(1)
		}
		if corsOrigin != "*" {
			if _, err := url.ParseRequestURI(corsOrigin); err != nil {
				slog.Warn("invalid CORS_ORIGIN", "origin", corsOrigin, "error", err)
			}
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			panic("FATAL: JWT_SECRET environment variable is required. Set it to a secure random value (256-bit recommended).")
		}

		jwtSecretRefresh := os.Getenv("JWT_SECRET_REFRESH")
		if jwtSecretRefresh == "" {
			jwtSecretRefresh = jwtSecret
		}

		warningThreshold := getEnvInt("STOCK_WARNING_THRESHOLD", 10)
		criticalThreshold := getEnvInt("STOCK_CRITICAL_THRESHOLD", 5)
		stockMinimum := getEnvInt("STOCK_MINIMUM", 10)
		cartHoldTTLHours := getEnvInt("CART_HOLD_TTL_HOURS", 24)

		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			if env == "production" {
				logLevel = "info"
			} else {
				logLevel = "debug"
			}
		}

		cachedConfig = &Config{
			Env:                    env,
			CORSOrigin:             corsOrigin,
			JWTSecret:              jwtSecret,
			JWTSecretRefresh:       jwtSecretRefresh,
			StockWarningThreshold:  warningThreshold,
			StockCriticalThreshold: criticalThreshold,
			StockMinimum:           stockMinimum,
			CartHoldTTLHours:       cartHoldTTLHours,
			LogLevel:               logLevel,
			Timezone:               defaultLocation,
		}
	})

	return cachedConfig
}
