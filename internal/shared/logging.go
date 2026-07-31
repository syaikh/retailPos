package shared

import (
	"context"
	"log/slog"
)

// contextAttrs enriches log records with request-scoped correlation data so
// they can be tied back to a specific access-log entry via request_id.
func contextAttrs(ctx context.Context) []any {
	var attrs []any
	if ctx == nil {
		return attrs
	}
	if id := GetRequestID(ctx); id != "" {
		attrs = append(attrs, "request_id", id)
	}
	if path := GetRequestPath(ctx); path != "" {
		attrs = append(attrs, "path", path)
	}
	return attrs
}

// LogError logs a structured error record enriched with request context.
// Callers should pass the request-scoped ctx so error logs carry request_id/path.
func LogError(ctx context.Context, msg string, err error, extra ...any) {
	attrs := contextAttrs(ctx)
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	attrs = append(attrs, extra...)
	slog.Error(msg, attrs...)
}

// LogWarn logs a structured warning record enriched with request context.
func LogWarn(ctx context.Context, msg string, extra ...any) {
	attrs := contextAttrs(ctx)
	attrs = append(attrs, extra...)
	slog.Warn(msg, attrs...)
}

// LogInfo logs a structured info record enriched with request context.
func LogInfo(ctx context.Context, msg string, extra ...any) {
	attrs := contextAttrs(ctx)
	attrs = append(attrs, extra...)
	slog.Info(msg, attrs...)
}
