package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/shared"
)

// slowRequestThreshold is the wall-clock latency above which a request is
// surfaced at WARN even when it succeeded. Routine POS traffic (product
// search, cart edits, checkout) stays orders of magnitude below this; report
// and export queries are what the warning is meant to catch. Declared as a var
// so tests can lower it.
var slowRequestThreshold = time.Second

// expectedAuthPaths are endpoints where a 401 is a routine step of the session
// lifecycle rather than a symptom of a fault: the client probes /api/validate
// with a token that may have expired, and /api/refresh is called with a
// refresh token that may have been rotated away or revoked by a logout. A 401
// on these paths is expected and logged at DEBUG. Actual credential abuse
// (guessing, token replay) stays visible through the login rate limiter, the
// failed-login log and the audit log.
var expectedAuthPaths = map[string]bool{
	"/api/validate": true,
	"/api/refresh":  true,
}

// identityPaths are session endpoints that carry client identity (ip,
// user_agent) on every entry, including successes. They are a deliberate
// exception to the "identity only when somebody will act on it" rule: these
// are the calls that establish, extend and end a terminal session, and the
// originating IP is the only way to tell a routine session from a stolen
// token replayed from a new machine. /api/login and /api/logout are already
// covered by the >= 400 and audit-log paths, but are listed so the set stays
// complete as handlers are added.
var identityPaths = map[string]bool{
	"/api/validate":        true,
	"/api/refresh":         true,
	"/api/login":           true,
	"/api/logout":          true,
	"/api/change-password": true,
}

// isExpectedAuthRejection reports whether a 401 on path is part of the normal
// session lifecycle.
func isExpectedAuthRejection(path string, status int) bool {
	return status == http.StatusUnauthorized && expectedAuthPaths[path]
}

// retainsClientIdentity reports whether an access-log entry should carry ip and
// user_agent. Errors and slow requests always qualify; session endpoints
// qualify at any status.
func retainsClientIdentity(path string, status int, slow bool) bool {
	return status >= 400 || slow || identityPaths[path]
}

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
//
// Level selection keeps the signal-to-noise ratio high: successful traffic is
// DEBUG, lateness above slowRequestThreshold is WARN regardless of status,
// genuine server faults are ERROR, and client errors are WARN except for the
// expected session-lifecycle 401s on expectedAuthPaths. Client identity is
// attached for errors, slow requests, and every identityPaths entry.
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

		status := c.Writer.Status()
		elapsed := time.Since(start)
		slow := elapsed >= slowRequestThreshold

		attrs := []any{
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", float64(elapsed.Microseconds()) / 1000.0,
		}
		if userID := UserIDFromContext(c.Request.Context()); userID != nil {
			attrs = append(attrs, "user_id", *userID)
		}

		// Client identity only earns its place on entries somebody will act
		// on — errors and slow requests — plus session endpoints, where the
		// originating IP is the only evidence of where a token was used.
		// Dropping it from routine 2xx traffic removes the bulk of the
		// per-request noise while keeping the authentication and security
		// diagnostics that need it.
		if retainsClientIdentity(c.Request.URL.Path, status, slow) {
			attrs = append(attrs,
				"ip", shared.GetIPAddress(c),
				"user_agent", c.Request.UserAgent(),
			)
		}
		if slow {
			// Set before the switch so a slow failure stays identifiable as
			// such instead of looking like an ordinary error.
			attrs = append(attrs, "slow_request", true)
		}

		logger := slog.Default()
		switch {
		case status >= 500:
			logger.Error("http_request", attrs...)
		case slow:
			logger.Warn("http_request", attrs...)
		case isExpectedAuthRejection(c.Request.URL.Path, status):
			logger.Debug("http_request", attrs...)
		case status >= 400:
			logger.Warn("http_request", attrs...)
		default:
			logger.Debug("http_request", attrs...)
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
