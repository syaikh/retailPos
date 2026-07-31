package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/shared"
)

// newRequestID generates a cryptographically random hex request ID.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}

// RequestLoggingMiddleware emits a structured access-log entry per request and
// attaches a request ID (X-Request-ID) that downstream logs can correlate on.
// Request ID is honored from the incoming header when present, otherwise generated.
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = newRequestID()
		}
		c.Set("requestID", reqID)
		c.Header("X-Request-ID", reqID)

		ctx := c.Request.Context()
		ctx = shared.SetRequestID(ctx, reqID)
		ctx = shared.SetRequestPath(ctx, c.Request.URL.Path)
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()

		attrs := []any{
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", float64(time.Since(start).Microseconds()) / 1000.0,
			"ip", shared.GetIPAddress(c),
			"user_agent", c.Request.UserAgent(),
		}
		if userID := UserIDFromContext(c.Request.Context()); userID != nil {
			attrs = append(attrs, "user_id", *userID)
		}

		logger := slog.Default()
		switch status := c.Writer.Status(); {
		case status >= 500:
			logger.Error("http_request", attrs...)
		case status >= 400:
			logger.Warn("http_request", attrs...)
		default:
			logger.Info("http_request", attrs...)
		}
	}
}

// RequestIDFromContext returns the request ID stored on the gin context, if any.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get("requestID"); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
