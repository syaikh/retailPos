package shared

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	globalLogger *slog.Logger
	once         = new(sync.Once)
)

func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func InitLogger(env, levelStr string) {
	once.Do(func() {
		var handler slog.Handler
		level := parseLogLevel(levelStr)
		if env == "production" {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			})
		} else {
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			})
		}
		globalLogger = slog.New(handler)
		slog.SetDefault(globalLogger)
	})
}

func Logger() *slog.Logger {
	if globalLogger == nil {
		InitLogger("development", "debug")
	}
	return globalLogger
}
