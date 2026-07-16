package shared

import (
	"log/slog"
	"os"
	"sync"
)

var (
	globalLogger *slog.Logger
	once         = new(sync.Once)
)

func InitLogger(env string) {
	once.Do(func() {
		var handler slog.Handler
		if env == "production" {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
		} else {
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})
		}
		globalLogger = slog.New(handler)
		slog.SetDefault(globalLogger)
	})
}

func Logger() *slog.Logger {
	if globalLogger == nil {
		InitLogger("development")
	}
	return globalLogger
}
