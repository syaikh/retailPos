package shared

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingHandler struct {
	records []slog.Record
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

func captureLogs(t *testing.T) *capturingHandler {
	t.Helper()
	h := &capturingHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return h
}

func recordAttrs(r slog.Record) map[string]any {
	out := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.Any()
		return true
	})
	return out
}

func TestSetGetRequestID(t *testing.T) {
	ctx := SetRequestID(context.Background(), "req-1")
	assert.Equal(t, "req-1", GetRequestID(ctx))
	assert.Equal(t, "", GetRequestID(context.Background()))
	assert.Equal(t, "", GetRequestID(nil)) //nolint:staticcheck // deliberately tests nil-context handling
}

func TestSetGetRequestPath(t *testing.T) {
	ctx := SetRequestPath(context.Background(), "/api/products")
	assert.Equal(t, "/api/products", GetRequestPath(ctx))
	assert.Equal(t, "", GetRequestPath(context.Background()))
}

func TestLogError_IncludesRequestContext(t *testing.T) {
	h := captureLogs(t)
	ctx := SetRequestID(SetRequestPath(context.Background(), "/api/x"), "abc-123")

	LogError(ctx, "boom", errors.New("kaboom"), "k", "v")

	require.Len(t, h.records, 1)
	attrs := recordAttrs(h.records[0])
	assert.Equal(t, slog.LevelError, h.records[0].Level)
	assert.Equal(t, "abc-123", attrs["request_id"])
	assert.Equal(t, "/api/x", attrs["path"])
	assert.EqualError(t, attrs["error"].(error), "kaboom")
	assert.Equal(t, "v", attrs["k"])
}

func TestLogWarn_NoContextNoPanic(t *testing.T) {
	h := captureLogs(t)

	LogWarn(nil, "warning without context", "k", "v") //nolint:staticcheck // deliberately tests nil-context handling

	require.Len(t, h.records, 1)
	attrs := recordAttrs(h.records[0])
	assert.Equal(t, slog.LevelWarn, h.records[0].Level)
	assert.Equal(t, "v", attrs["k"])
}

func TestLogInfo(t *testing.T) {
	h := captureLogs(t)

	LogInfo(context.Background(), "hello")

	require.Len(t, h.records, 1)
	assert.Equal(t, slog.LevelInfo, h.records[0].Level)
}

func TestParseLogLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, parseLogLevel("DEBUG"))
	assert.Equal(t, slog.LevelInfo, parseLogLevel("info"))
	assert.Equal(t, slog.LevelWarn, parseLogLevel("warn"))
	assert.Equal(t, slog.LevelError, parseLogLevel("error"))
	assert.Equal(t, slog.LevelInfo, parseLogLevel(""))
	assert.Equal(t, slog.LevelInfo, parseLogLevel("bogus"))
}
