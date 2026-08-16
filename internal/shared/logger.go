package shared

import (
	"log/slog"
	"os"
	"path/filepath"
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
	case "":
		return slog.LevelInfo
	default:
		slog.Warn("unknown LOG_LEVEL, defaulting to info", "level", levelStr)
		return slog.LevelInfo
	}
}

func InitLogger(env, levelStr string) {
	once.Do(func() {
		var handler slog.Handler
		level := parseLogLevel(levelStr)
		opts := &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					if src, ok := a.Value.Any().(*slog.Source); ok {
						src.File = filepath.Base(src.File)
					}
				}
				return a
			},
		}
		if env == "production" {
			opts.AddSource = false
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			opts.AddSource = true
			handler = slog.NewTextHandler(os.Stdout, opts)
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
