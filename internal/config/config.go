package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env                    string
	CORSOrigin             string
	JWTSecret              string
	StockWarningThreshold  int
	StockCriticalThreshold int
	StockMinimum           int
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
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
	}

	warningThreshold := getEnvInt("STOCK_WARNING_THRESHOLD", 10)
	criticalThreshold := getEnvInt("STOCK_CRITICAL_THRESHOLD", 5)
	stockMinimum := getEnvInt("STOCK_MINIMUM", 10)

	return &Config{
		Env:                    env,
		CORSOrigin:             corsOrigin,
		JWTSecret:              jwtSecret,
		StockWarningThreshold:  warningThreshold,
		StockCriticalThreshold: criticalThreshold,
		StockMinimum:           stockMinimum,
		Timezone:               defaultLocation,
	}
}