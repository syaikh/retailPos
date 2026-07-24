package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestIPRateLimiter_Stop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(10), 20)

	limiter.Stop()
	assert.True(t, limiter.stopped)

	// Second stop should be a no-op, no panic.
	limiter.Stop()
	assert.True(t, limiter.stopped)
}

func TestIPRateLimiter_StopConcurrentSafety(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(10), 20)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter.Stop()
		}()
	}
	wg.Wait()
	assert.True(t, limiter.stopped)
}

func TestIPRateLimiter_EvictOldestLocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(100), 10)
	defer limiter.Stop()

	limiter.mu.Lock()
	limiter.maxSize = 2

	// Add two entries.
	limiter.ips["10.0.0.1"] = &ipEntry{
		limiter:  rate.NewLimiter(100, 10),
		lastSeen: time.Now().Add(-10 * time.Minute),
	}
	limiter.ips["10.0.0.2"] = &ipEntry{
		limiter:  rate.NewLimiter(100, 10),
		lastSeen: time.Now().Add(-5 * time.Minute),
	}
	limiter.mu.Unlock()

	// Getting a third IP should trigger eviction.
	limiter.GetLimiter("10.0.0.3")

	limiter.mu.Lock()
	count := len(limiter.ips)
	_, oldestExists := limiter.ips["10.0.0.1"]
	limiter.mu.Unlock()

	assert.Equal(t, 2, count, "should have evicted one entry to make room")
	assert.False(t, oldestExists, "oldest entry (10.0.0.1) should have been evicted")
}

func TestIPRateLimiter_EvictOldestLockedEmptyMap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(100), 10)
	defer limiter.Stop()

	// Manually set maxSize to 0 so next GetLimiter triggers eviction on empty map.
	limiter.mu.Lock()
	limiter.maxSize = 0
	limiter.mu.Unlock()

	// Should not panic on empty map eviction.
	limiter.GetLimiter("10.0.0.1")

	limiter.mu.Lock()
	count := len(limiter.ips)
	limiter.mu.Unlock()
	assert.Equal(t, 1, count)
}

func TestLoginRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request under limit passes", func(t *testing.T) {
		t.Setenv("LOGIN_RATE_LIMIT_RPM", "2")
		t.Setenv("LOGIN_RATE_LIMIT_BURST", "2")

		router := gin.New()
		router.Use(LoginRateLimitMiddleware())
		router.POST("/login", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request over limit returns 429", func(t *testing.T) {
		t.Setenv("LOGIN_RATE_LIMIT_RPM", "1")
		t.Setenv("LOGIN_RATE_LIMIT_BURST", "1")

		router := gin.New()
		router.Use(LoginRateLimitMiddleware())
		router.POST("/login", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = "5.5.5.5:1111"
			router.ServeHTTP(w, req)
			if i == 2 {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})

	t.Run("custom env vars", func(t *testing.T) {
		t.Setenv("LOGIN_RATE_LIMIT_RPM", "100")
		t.Setenv("LOGIN_RATE_LIMIT_BURST", "100")

		router := gin.New()
		router.Use(LoginRateLimitMiddleware())
		router.POST("/login", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "9.9.9.9:2222"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRefreshRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request under limit passes", func(t *testing.T) {
		t.Setenv("REFRESH_RATE_LIMIT_RPM", "2")
		t.Setenv("REFRESH_RATE_LIMIT_BURST", "2")

		router := gin.New()
		router.Use(RefreshRateLimitMiddleware())
		router.POST("/refresh", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request over limit returns 429", func(t *testing.T) {
		t.Setenv("REFRESH_RATE_LIMIT_RPM", "1")
		t.Setenv("REFRESH_RATE_LIMIT_BURST", "1")

		router := gin.New()
		router.Use(RefreshRateLimitMiddleware())
		router.POST("/refresh", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			req.RemoteAddr = "6.6.6.6:3333"
			router.ServeHTTP(w, req)
			if i == 2 {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})

	t.Run("custom env vars", func(t *testing.T) {
		t.Setenv("REFRESH_RATE_LIMIT_RPM", "100")
		t.Setenv("REFRESH_RATE_LIMIT_BURST", "100")

		router := gin.New()
		router.Use(RefreshRateLimitMiddleware())
		router.POST("/refresh", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
		req.RemoteAddr = "7.7.7.7:4444"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSetCtxValue(t *testing.T) {
	type ctxKey string
	const testKey ctxKey = "test"

	t.Run("nil context returns background", func(t *testing.T) {
		result := setCtxValue(nil, testKey, "value") //lint:ignore SA1012 deliberately tests nil-context handling
		assert.NotNil(t, result)
		assert.Equal(t, "value", result.Value(testKey))
	})

	t.Run("non-nil context wraps", func(t *testing.T) {
		type outerKey string
		const oKey outerKey = "outer"

		base := context.WithValue(context.Background(), oKey, "base")
		result := setCtxValue(base, testKey, "value")
		assert.Equal(t, "value", result.Value(testKey))
		assert.Equal(t, "base", result.Value(oKey))
	})
}

func TestGetClientIP_XForwardedForLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "10.0.0.1:8080"
	c.Request.Header.Set("X-Forwarded-For", "1.2.3.4")

	// Should still use RemoteAddr, not X-Forwarded-For.
	got := getClientIP(c)
	assert.Equal(t, "10.0.0.1", got)
}

func TestCleanupOnce_RemovesExpiredEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(100), 10)
	defer limiter.Stop()

	// Set TTL to 5 minutes; entries older than that should be removed.
	limiter.mu.Lock()
	limiter.ttl = 5 * time.Minute

	// Add entries well past TTL.
	limiter.ips["10.0.0.1"] = &ipEntry{
		limiter:  rate.NewLimiter(100, 10),
		lastSeen: time.Now().Add(-10 * time.Minute),
	}
	limiter.ips["10.0.0.2"] = &ipEntry{
		limiter:  rate.NewLimiter(100, 10),
		lastSeen: time.Now().Add(-10 * time.Minute),
	}
	// Add a fresh entry (seen 1 minute ago, within 5-min TTL).
	limiter.ips["10.0.0.3"] = &ipEntry{
		limiter:  rate.NewLimiter(100, 10),
		lastSeen: time.Now().Add(-1 * time.Minute),
	}
	limiter.mu.Unlock()

	limiter.cleanupOnce()

	limiter.mu.Lock()
	count := len(limiter.ips)
	_, exists1 := limiter.ips["10.0.0.1"]
	_, exists2 := limiter.ips["10.0.0.2"]
	_, exists3 := limiter.ips["10.0.0.3"]
	limiter.mu.Unlock()

	assert.Equal(t, 1, count, "should only have fresh entry")
	assert.False(t, exists1, "expired entry 1 should be removed")
	assert.False(t, exists2, "expired entry 2 should be removed")
	assert.True(t, exists3, "fresh entry should remain")
}

func TestCleanupOnce_EmptyMap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(100), 10)
	defer limiter.Stop()

	// Should not panic on empty map.
	limiter.cleanupOnce()

	limiter.mu.Lock()
	count := len(limiter.ips)
	limiter.mu.Unlock()
	assert.Equal(t, 0, count)
}

func TestCleanupOnce_EntriesWithinTTLPreserved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(100), 10)
	defer limiter.Stop()

	// Set TTL to 10 minutes.
	limiter.mu.Lock()
	limiter.ttl = 10 * time.Minute

	// Entry seen 5 minutes ago (within TTL).
	limiter.ips["10.0.0.1"] = &ipEntry{
		limiter:  rate.NewLimiter(100, 10),
		lastSeen: time.Now().Add(-5 * time.Minute),
	}
	limiter.mu.Unlock()

	limiter.cleanupOnce()

	limiter.mu.Lock()
	count := len(limiter.ips)
	_, exists := limiter.ips["10.0.0.1"]
	limiter.mu.Unlock()

	assert.Equal(t, 1, count)
	assert.True(t, exists, "entry within TTL should be preserved")
}
