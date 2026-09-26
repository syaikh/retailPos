package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// capturingHandler records every slog record so tests can assert on the level
// and attributes the access log actually produced.
type capturingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *capturingHandler) only(t *testing.T) slog.Record {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	require.Len(t, h.records, 1, "expected exactly one access-log record")
	return h.records[0]
}

func recordAttrs(t *testing.T, r slog.Record) map[string]any {
	t.Helper()
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	return attrs
}

// captureAccessLogs installs a capturing slog default for the duration of the
// test and returns the handler plus a function yielding the single record the
// request produced.
func captureAccessLogs(t *testing.T) *capturingHandler {
	t.Helper()
	h := &capturingHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return h
}

func setupRequestLoggingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLoggingMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "pong"})
	})
	r.GET("/boom", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "boom")
	})
	r.GET("/teapot", func(c *gin.Context) {
		c.String(http.StatusForbidden, "nope")
	})
	r.POST("/api/validate", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
	})
	r.POST("/api/refresh", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
	})
	r.POST("/api/sales", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
	})
	r.POST("/api/validate-ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "session valid"})
	})
	r.GET("/api/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	})
	return r
}

func TestRequestLogging_GeneratesRequestID(t *testing.T) {
	r := setupRequestLoggingRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	reqID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, reqID, "X-Request-ID header should be set")
}

func TestRequestLogging_HonorsIncomingRequestID(t *testing.T) {
	r := setupRequestLoggingRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "incoming-id-123")

	r.ServeHTTP(w, req)

	assert.Equal(t, "incoming-id-123", w.Header().Get("X-Request-ID"))
}

func TestRequestLogging_SetsRequestIDInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var captured string
	r.Use(RequestLoggingMiddleware())
	r.GET("/check", func(c *gin.Context) {
		captured = shared.GetRequestID(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, captured, "request ID should be available on request context")
	assert.Equal(t, captured, w.Header().Get("X-Request-ID"))
}

func TestRequestLogging_DoesNotMaskErrorStatus(t *testing.T) {
	r := setupRequestLoggingRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestIDFromContext_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	assert.Equal(t, "", RequestIDFromContext(c))
}

func TestRequestIDFromContext_Present(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("requestID", "abc-123")

	assert.Equal(t, "abc-123", RequestIDFromContext(c))
}

// --- Access-log level selection ---

func serveLogged(t *testing.T, r *gin.Engine, method, path string) slog.Record {
	t.Helper()
	h := captureAccessLogs(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("User-Agent", "POS-Terminal/1.0")
	req.RemoteAddr = "10.0.0.7:5555"
	r.ServeHTTP(w, req)
	return h.only(t)
}

// withSlowThreshold pins the slow-request threshold for the duration of a test
// so level assertions cannot be flipped by a slow or -race-enabled CI runner.
func withSlowThreshold(t *testing.T, d time.Duration) {
	t.Helper()
	orig := slowRequestThreshold
	slowRequestThreshold = d
	t.Cleanup(func() { slowRequestThreshold = orig })
}

func TestRequestLogging_SuccessIsDebug(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	rec := serveLogged(t, setupRequestLoggingRouter(), http.MethodGet, "/ping")

	assert.Equal(t, slog.LevelDebug, rec.Level)
	assert.Equal(t, "http_request", rec.Message)
}

func TestRequestLogging_ServerErrorIsError(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	rec := serveLogged(t, setupRequestLoggingRouter(), http.MethodGet, "/boom")

	assert.Equal(t, slog.LevelError, rec.Level)
}

func TestRequestLogging_ClientErrorIsWarn(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	rec := serveLogged(t, setupRequestLoggingRouter(), http.MethodGet, "/teapot")

	assert.Equal(t, slog.LevelWarn, rec.Level)
}

func TestRequestLogging_ExpectedAuthRejectionIsDebug(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	r := setupRequestLoggingRouter()

	for _, path := range []string{"/api/validate", "/api/refresh"} {
		rec := serveLogged(t, r, http.MethodPost, path)
		assert.Equal(t, slog.LevelDebug, rec.Level, "401 on %s is a routine session-lifecycle event", path)
		assert.Equal(t, path, recordAttrs(t, rec)["path"])
	}
}

func TestRequestLogging_UnexpectedUnauthorizedStaysWarn(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	rec := serveLogged(t, setupRequestLoggingRouter(), http.MethodPost, "/api/sales")

	assert.Equal(t, slog.LevelWarn, rec.Level)
}

func TestRequestLogging_SlowSuccessIsWarn(t *testing.T) {
	withSlowThreshold(t, time.Millisecond)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLoggingMiddleware())
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(5 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	rec := serveLogged(t, r, http.MethodGet, "/slow")
	attrs := recordAttrs(t, rec)

	assert.Equal(t, slog.LevelWarn, rec.Level)
	assert.Equal(t, int64(http.StatusOK), attrs["status"])
	assert.Equal(t, true, attrs["slow_request"])
}

// A slow failure stays ERROR but must still be identifiable as slow.
func TestRequestLogging_SlowFailureKeepsMarker(t *testing.T) {
	withSlowThreshold(t, time.Millisecond)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLoggingMiddleware())
	r.GET("/slow-boom", func(c *gin.Context) {
		time.Sleep(5 * time.Millisecond)
		c.String(http.StatusInternalServerError, "boom")
	})

	rec := serveLogged(t, r, http.MethodGet, "/slow-boom")
	attrs := recordAttrs(t, rec)

	assert.Equal(t, slog.LevelError, rec.Level)
	assert.Equal(t, true, attrs["slow_request"])
}

func TestRequestLogging_RoutineSuccessOmitsClientIdentity(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	rec := serveLogged(t, setupRequestLoggingRouter(), http.MethodGet, "/ping")
	attrs := recordAttrs(t, rec)

	assert.NotContains(t, attrs, "ip")
	assert.NotContains(t, attrs, "user_agent")
}

func TestRequestLogging_ErrorsKeepClientIdentity(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	rec := serveLogged(t, setupRequestLoggingRouter(), http.MethodPost, "/api/sales")
	attrs := recordAttrs(t, rec)

	assert.Equal(t, "10.0.0.7", attrs["ip"])
	assert.Equal(t, "POS-Terminal/1.0", attrs["user_agent"])
}

// A successful session call still records where the token was used, even
// though it logs at DEBUG and nobody is expected to act on the entry. This is
// the only record of the originating IP for /api/validate, which writes no
// audit row of its own, so dropping it would make a replayed access token
// indistinguishable from routine use.
func TestRequestLogging_SessionSuccessKeepsClientIdentity(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLoggingMiddleware())
	r.POST("/api/validate", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "session valid"})
	})

	rec := serveLogged(t, r, http.MethodPost, "/api/validate")
	attrs := recordAttrs(t, rec)

	assert.Equal(t, slog.LevelDebug, rec.Level)
	assert.Equal(t, "10.0.0.7", attrs["ip"])
	assert.Equal(t, "POS-Terminal/1.0", attrs["user_agent"])
}

func TestRetainsClientIdentity(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		status   int
		slow     bool
		expected bool
	}{
		{"routine success", "/api/products", http.StatusOK, false, false},
		{"session success", "/api/validate", http.StatusOK, false, true},
		{"refresh success", "/api/refresh", http.StatusOK, false, true},
		{"login success", "/api/login", http.StatusOK, false, true},
		{"client error", "/api/products", http.StatusBadRequest, false, true},
		{"slow routine success", "/api/products", http.StatusOK, true, true},
		{"unlisted 401", "/api/sales", http.StatusUnauthorized, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, retainsClientIdentity(tt.path, tt.status, tt.slow))
		})
	}
}

func TestRequestLogging_KeepsCorrelationFields(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	r := setupRequestLoggingRouter()
	h := captureAccessLogs(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set("X-Request-ID", "trace-me")
	r.ServeHTTP(w, req)

	attrs := recordAttrs(t, h.only(t))
	assert.Equal(t, "trace-me", attrs["request_id"])
	assert.Equal(t, http.MethodGet, attrs["method"])
	assert.Equal(t, "/boom", attrs["path"])
	assert.Contains(t, attrs, "latency_ms")
}

func TestRequestLogging_LogsAuthenticatedUser(t *testing.T) {
	withSlowThreshold(t, time.Hour)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLoggingMiddleware())
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), CtxKeyUserID, 42))
		c.Next()
	})
	r.GET("/secure", func(c *gin.Context) { c.Status(http.StatusForbidden) })

	rec := serveLogged(t, r, http.MethodGet, "/secure")
	attrs := recordAttrs(t, rec)

	assert.Equal(t, slog.LevelWarn, rec.Level)
	assert.Equal(t, int64(42), attrs["user_id"])
}
