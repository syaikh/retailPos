package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/shared"
)

func SecurityHeadersMiddleware(allowedOrigins []string) gin.HandlerFunc {
	csp := buildCSP(allowedOrigins)

	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", csp)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// X-XSS-Protection: 0 — deliberately disabled per modern security guidance.
		// This header is deprecated and can introduce XSS vulnerabilities in some
		// older browsers. CSP handles XSS prevention via script-src and style-src.
		c.Header("X-XSS-Protection", "0")
		c.Next()
	}
}

func buildCSP(allowedOrigins []string) string {
	wsOrigins := make([]string, 0)
	for _, o := range allowedOrigins {
		host := strings.TrimPrefix(strings.TrimPrefix(o, "https://"), "http://")
		wsOrigins = append(wsOrigins, "ws://"+host, "wss://"+host)
	}

	connectSrc := "'self'"
	if len(wsOrigins) > 0 {
		connectSrc += " " + strings.Join(wsOrigins, " ")
	}

	directives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"font-src 'self' data:",
		"connect-src " + connectSrc,
		"frame-ancestors 'none'",
		"form-action 'self'",
		"base-uri 'self'",
	}

	return strings.Join(directives, "; ")
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// SPA pattern: browser forms cannot set custom headers.
		// If Authorization header is present with a valid JWT format (header.payload.signature),
		// this is a legitimate API call, not a CSRF attack.
		// CSRF attacks exploit cookie-based auth; JWT in Authorization header is immune.
		if auth := c.Request.Header.Get("Authorization"); auth != "" {
			parts := strings.Split(auth, " ")
			if len(parts) == 2 {
				token := parts[1]
				if strings.Count(token, ".") == 2 {
					c.Next()
					return
				}
			}
		}

		if c.Request.Header.Get("X-Refresh-Token") != "" {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(403, shared.NewError(shared.ErrForbidden, "CSRF token missing or invalid"))
	}
}
