package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
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

		if c.Request.Header.Get("Authorization") != "" {
			c.Next()
			return
		}

		if c.Request.Header.Get("X-Refresh-Token") != "" {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(403, gin.H{"error": "CSRF token missing or invalid"})
	}
}
