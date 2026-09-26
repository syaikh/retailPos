package config

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
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
	CartHoldTTLHours       int
	// ReportRefreshDebounce is the base retry delay between consecutive
	// refresh failures in seconds (REPORT_REFRESH_DEBOUNCE, default 30). It no
	// longer debounces sale-triggered refreshes; the coordinator refreshes at
	// each Jakarta hour boundary and uses this only as retry backoff base.
	ReportRefreshDebounce int
	LogLevel              string
	Timezone              *time.Location
	// DBSSLMode is the libpq sslmode used when the DSN is composed from the
	// discrete DB_* variables. It is ignored when DATABASE_URL is set, since
	// that DSN carries its own sslmode parameter.
	DBSSLMode string

	// startupDiagnostics holds the warnings and notices raised while loading
	// configuration, in order.
	//
	// Load runs before the logger is configured (main.go calls
	// shared.InitLogger with the values this function resolves), so anything
	// logged directly here would go through the default slog text handler even
	// in production. That put the "database TLS disabled in production" notice
	// into a different format from every log line around it, in the one message
	// an operator is most likely to be reading. Collecting them and replaying
	// via LogStartupDiagnostics keeps a single log format in production.
	startupDiagnostics diagnosticLog
}

// startupDiagnostic is a log record raised during Load and replayed later.
type startupDiagnostic struct {
	level slog.Level
	msg   string
	attrs []any
}

type diagnosticLog []startupDiagnostic

// diag records a startup diagnostic instead of logging it immediately. See
// Config.startupDiagnostics for why.
func (d *diagnosticLog) diag(level slog.Level, msg string, attrs ...any) {
	*d = append(*d, startupDiagnostic{level: level, msg: msg, attrs: attrs})
}

// LogStartupDiagnostics replays the diagnostics collected by Load through the
// currently configured logger. main.go calls this immediately after
// shared.InitLogger, so these land in the same format as the rest of the
// process. Safe to call more than once and with a nil receiver.
func (c *Config) LogStartupDiagnostics() {
	if c == nil {
		return
	}
	for _, d := range c.startupDiagnostics {
		slog.Log(context.Background(), d.level, d.msg, d.attrs...)
	}
}

// validSSLModes are the values libpq accepts for the sslmode connection
// parameter. Anything else is rejected at startup rather than passed through to
// the driver, where the failure would surface as a connection error with no
// indication that a typo in configuration was the cause.
var validSSLModes = map[string]bool{
	"disable":     true,
	"allow":       true,
	"prefer":      true,
	"require":     true,
	"verify-ca":   true,
	"verify-full": true,
}

// resolveDBSSLMode determines the sslmode for a DSN composed from DB_*.
//
// It defaults to require in production because database traffic should be
// encrypted, and to disable in development where the local server does not
// present certificates. DB_SSLMODE overrides both, which matters when the
// database is unreachable over TLS: a deployment confined to a private network
// may legitimately set disable, but that has to be a recorded decision rather
// than an accident of which variables happened to be set.
//
// The boolean result reports whether the resolved value came from a usable
// DB_SSLMODE; when false the caller should log that the default was used.
//
// The third result is the rejected input, or "" when the value was usable or
// unset. An unusable DB_SSLMODE is returned rather than logged so that the
// caller can emit the notice through the configured logger: resolveDBSSLMode
// runs inside Load, before main.go calls shared.InitLogger, so a direct
// slog.Warn here would always use the default text handler.
func resolveDBSSLMode(raw, env string) (string, bool, string) {
	if mode := strings.TrimSpace(raw); mode != "" {
		if validSSLModes[mode] {
			return mode, true, ""
		}
		return dbSSLModeDefault(env), false, mode
	}
	return dbSSLModeDefault(env), false, ""
}

// dbSSLModeDefault is the mode used when DB_SSLMODE is unset or unusable.
func dbSSLModeDefault(env string) string {
	if env == "production" {
		return "require"
	}
	return "disable"
}

func sslModeList() string {
	modes := make([]string, 0, len(validSSLModes))
	for m := range validSSLModes {
		modes = append(modes, m)
	}
	sort.Strings(modes)
	return strings.Join(modes, ", ")
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

// getEnvInt reads an integer environment variable, falling back to defaultVal
// when unset or unparseable. The warning is recorded on pending rather than
// printed so it is replayed through the configured logger; see
// Config.startupDiagnostics.
func getEnvInt(pending *diagnosticLog, key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		pending.diag(slog.LevelWarn, "invalid env, using default",
			"key", key, "value", val, "default", defaultVal)
		return defaultVal
	}
	return n
}

func Load() *Config {
	configOnce.Do(func() {
		// Collected rather than logged here; see Config.startupDiagnostics.
		var pending diagnosticLog

		env := os.Getenv("ENV")
		if env == "" {
			env = "development"
			pending.diag(slog.LevelWarn, "ENV environment variable not set, defaulting to 'development'. Set ENV=production for production deployments.")
		}

		corsOrigin := os.Getenv("CORS_ORIGIN")
		if corsOrigin == "" {
			corsOrigin = "http://localhost:5173"
		}
		if env == "production" && corsOrigin == "*" {
			// Fatal, so it cannot be deferred to LogStartupDiagnostics: the process
			// exits here, before main.go configures the logger. It is emitted
			// directly and says so in the message.
			slog.Error("CORS_ORIGIN must not be '*' in production. Set it to your actual domain. " +
				"(logged in the default format: the process exits before the production logger is installed)")
			os.Exit(1)
		}
		if corsOrigin != "*" {
			if _, err := url.ParseRequestURI(corsOrigin); err != nil {
				pending.diag(slog.LevelWarn, "invalid CORS_ORIGIN", "origin", corsOrigin, "error", err)
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

		warningThreshold := getEnvInt(&pending, "STOCK_WARNING_THRESHOLD", 10)
		criticalThreshold := getEnvInt(&pending, "STOCK_CRITICAL_THRESHOLD", 5)
		cartHoldTTLHours := getEnvInt(&pending, "CART_HOLD_TTL_HOURS", 24)
		reportRefreshDebounce := getEnvInt(&pending, "REPORT_REFRESH_DEBOUNCE", 30)

		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			if env == "production" {
				logLevel = "info"
			} else {
				logLevel = "debug"
			}
		}

		dbSSLMode, explicit, rejected := resolveDBSSLMode(os.Getenv("DB_SSLMODE"), env)
		if rejected != "" {
			pending.diag(slog.LevelWarn, "invalid DB_SSLMODE, using environment default", "value", rejected, "valid", sslModeList())
		}
		if env == "production" && dbSSLMode == "disable" {
			pending.diag(slog.LevelWarn, "database TLS disabled in production. Ensure the database is unreachable from outside the deployment network, and record this decision.")
		}
		if !explicit {
			pending.diag(slog.LevelInfo, "database sslmode defaulted from environment", "sslmode", dbSSLMode, "env", env)
		}

		cachedConfig = &Config{
			Env:                    env,
			CORSOrigin:             corsOrigin,
			JWTSecret:              jwtSecret,
			JWTSecretRefresh:       jwtSecretRefresh,
			StockWarningThreshold:  warningThreshold,
			StockCriticalThreshold: criticalThreshold,
			CartHoldTTLHours:       cartHoldTTLHours,
			ReportRefreshDebounce:  reportRefreshDebounce,
			LogLevel:               logLevel,
			Timezone:               defaultLocation,
			DBSSLMode:              dbSSLMode,
			startupDiagnostics:     pending,
		}
	})

	return cachedConfig
}
