package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewIPRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(10), 20)
	defer limiter.Stop()

	assert.NotNil(t, limiter)
}

func TestGetLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(100), 2)
	defer limiter.Stop()

	lim1 := limiter.GetLimiter("192.168.1.1")
	lim2 := limiter.GetLimiter("192.168.1.1")
	lim3 := limiter.GetLimiter("10.0.0.1")

	assert.Equal(t, lim1, lim2, "same IP should return same limiter")
	assert.True(t, lim1 != lim3, "different IPs should return different limiter instances")

	assert.True(t, limiter.GetLimiter("192.168.1.100").Allow())
	assert.True(t, limiter.GetLimiter("192.168.1.100").Allow())
	assert.False(t, limiter.GetLimiter("192.168.1.100").Allow())
}

func TestGetClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{
			name:       "IP with port",
			remoteAddr: "192.168.1.1:8080",
			want:       "192.168.1.1",
		},
		{
			name:       "different IP with port",
			remoteAddr: "10.0.0.1:1234",
			want:       "10.0.0.1",
		},
		{
			name:       "unix socket fallback",
			remoteAddr: "/var/run/socket",
			want:       "/var/run/socket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.RemoteAddr = tt.remoteAddr

			got := getClientIP(c)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request under limit passes", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimitMiddleware())
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request over limit returns 429", func(t *testing.T) {
		t.Setenv("RATE_LIMIT_RPS", "1")
		t.Setenv("RATE_LIMIT_BURST", "2")

		router := gin.New()
		router.Use(RateLimitMiddleware())
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "1.2.3.4:5678"
			router.ServeHTTP(w, req)
			if i == 2 {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		t.Setenv("RATE_LIMIT_RPS", "1")
		t.Setenv("RATE_LIMIT_BURST", "1")

		router := gin.New()
		router.Use(RateLimitMiddleware())
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.RemoteAddr = "10.0.0.1:1111"
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.RemoteAddr = "10.0.0.2:2222"
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}
