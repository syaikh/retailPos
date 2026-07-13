package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBuildCSP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedOrigins []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "single origin",
			allowedOrigins: []string{"https://example.com"},
			wantContains:   []string{"ws://example.com", "wss://example.com"},
		},
		{
			name:           "multiple origins",
			allowedOrigins: []string{"https://a.com", "http://b.com"},
			wantContains:   []string{"ws://a.com", "wss://a.com", "ws://b.com", "wss://b.com"},
		},
		{
			name:           "empty origins",
			allowedOrigins: []string{},
			wantContains:   []string{"connect-src 'self'"},
			wantNotContain: []string{"ws://", "wss://"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csp := buildCSP(tt.allowedOrigins)
			for _, s := range tt.wantContains {
				assert.Contains(t, csp, s)
			}
			for _, s := range tt.wantNotContain {
				assert.NotContains(t, csp, s)
			}
		})
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	middleware := SecurityHeadersMiddleware([]string{"https://example.com"})
	middleware(c)

	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "0", w.Header().Get("X-XSS-Protection"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
}

func TestCSRFMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		authHeader     string
		refreshHeader  string
		wantCode       int
	}{
		{
			name:     "GET request passes",
			method:   http.MethodGet,
			wantCode: http.StatusOK,
		},
		{
			name:     "HEAD request passes",
			method:   http.MethodHead,
			wantCode: http.StatusOK,
		},
		{
			name:     "OPTIONS request passes",
			method:   http.MethodOptions,
			wantCode: http.StatusOK,
		},
		{
			name:     "POST without Authorization returns 403",
			method:   http.MethodPost,
			wantCode: http.StatusForbidden,
		},
		{
			name:       "POST with Authorization passes",
			method:     http.MethodPost,
			authHeader: "Bearer abc123",
			wantCode:   http.StatusOK,
		},
		{
			name:          "POST with X-Refresh-Token passes",
			method:        http.MethodPost,
			refreshHeader: "some-refresh-token",
			wantCode:      http.StatusOK,
		},
		{
			name:     "PUT without Authorization returns 403",
			method:   http.MethodPut,
			wantCode: http.StatusForbidden,
		},
		{
			name:     "PATCH without Authorization returns 403",
			method:   http.MethodPatch,
			wantCode: http.StatusForbidden,
		},
		{
			name:     "DELETE without Authorization returns 403",
			method:   http.MethodDelete,
			wantCode: http.StatusForbidden,
		},
		{
			name:       "PUT with Authorization passes",
			method:     http.MethodPut,
			authHeader: "Bearer token",
			wantCode:   http.StatusOK,
		},
		{
			name:       "DELETE with X-Refresh-Token passes",
			method:     http.MethodDelete,
			refreshHeader: "refresh",
			wantCode:   http.StatusOK,
		},
		{
			name:       "POST with empty Authorization returns 403",
			method:     http.MethodPost,
			authHeader: "",
			wantCode:   http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, "/", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}
			if tt.refreshHeader != "" {
				c.Request.Header.Set("X-Refresh-Token", tt.refreshHeader)
			}

			middleware := CSRFMiddleware()
			middleware(c)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}
