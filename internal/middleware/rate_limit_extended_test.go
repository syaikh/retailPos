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

// The shipped defaults are part of the contract .env.example documents for
// operators (a store's registers share one NAT address, so the flood guard
// must absorb a shift-start burst). Every other test in this file overrides
// the knobs via env, so a silent revert of these values would go unnoticed.
// An empty value fails strconv.Atoi, which is exactly the unset path.
func TestLoginRateLimitMiddleware_ShippedDefaultBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("LOGIN_RATE_LIMIT_RPM", "")
	t.Setenv("LOGIN_RATE_LIMIT_BURST", "")

	router := gin.New()
	router.Use(LoginRateLimitMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "203.0.113.50:9000"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "20", w.Header().Get("X-RateLimit-Limit"),
		"shipped login burst is 20 (30 rpm); see the coarse-guard note in rate_limit.go")
}

func TestRefreshRateLimitMiddleware_ShippedDefaultBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("REFRESH_RATE_LIMIT_RPM", "")
	t.Setenv("REFRESH_RATE_LIMIT_BURST", "")

	router := gin.New()
	router.Use(RefreshRateLimitMiddleware())
	router.POST("/refresh", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.RemoteAddr = "203.0.113.51:9001"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"),
		"shipped refresh burst is 60 (60 rpm): every register refreshes on the same cadence")
}

func TestSetCtxValue(t *testing.T) {
	type ctxKey string
	const testKey ctxKey = "test"

	t.Run("nil context returns background", func(t *testing.T) {
		result := setCtxValue(nil, testKey, "value") //nolint:staticcheck // deliberately tests nil-context handling
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

func TestGetClientIP_TrustedProxyHonorsXForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	// Peer is the nginx container on loopback in the single-pod deployment.
	assert.NoError(t, r.SetTrustedProxies([]string{"127.0.0.1/32"}))
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "127.0.0.1:8080"
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.7")

	assert.Equal(t, "203.0.113.7", getClientIP(c))
}

func TestGetClientIP_UntrustedPeerIgnoresXForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	// Direct exposure: only loopback is a trusted proxy, so an arbitrary
	// remote peer cannot spoof its way past per-IP rate limits.
	assert.NoError(t, r.SetTrustedProxies([]string{"127.0.0.1/32"}))
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "198.51.100.9:44321"
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.7")

	assert.Equal(t, "198.51.100.9", getClientIP(c))
}

// TestRateLimitMiddleware_TrustedProxyKeying wires the limiter the way
// cmd/server/main.go does (router.SetTrustedProxies + RateLimitMiddleware) and
// asserts the bucket key end to end, which the getClientIP unit tests above
// cannot: a peer outside the trusted list must not be able to mint a fresh
// bucket with X-Forwarded-For, while a request that genuinely arrives from the
// trusted proxy must be keyed on the forwarded client.
func TestRateLimitMiddleware_TrustedProxyKeying(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// burst 1 makes the bucket boundary observable: a second request from the
	// same key must 429 instead of silently getting its own allowance.
	t.Setenv("RATE_LIMIT_RPS", "1")
	t.Setenv("RATE_LIMIT_BURST", "1")

	router := gin.New()
	assert.NoError(t, router.SetTrustedProxies([]string{"127.0.0.1/32"}))
	router.Use(RateLimitMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	serve := func(remoteAddr, xff string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		router.ServeHTTP(w, req)
		return w.Code
	}

	t.Run("untrusted peer cannot spoof a new bucket", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve("198.51.100.9:44321", "203.0.113.7"))
		// If X-Forwarded-For were honoured from this peer, this second
		// spoofed client IP would open a fresh bucket and return 200.
		assert.Equal(t, http.StatusTooManyRequests, serve("198.51.100.9:44321", "203.0.113.8"))
	})

	t.Run("trusted proxy keys on the forwarded client", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve("127.0.0.1:61000", "203.0.113.7"))
		// A different real client behind the same proxy gets its own bucket…
		assert.Equal(t, http.StatusOK, serve("127.0.0.1:61000", "203.0.113.9"))
		// …while a repeat from that client exhausts it.
		assert.Equal(t, http.StatusTooManyRequests, serve("127.0.0.1:61000", "203.0.113.9"))
	})
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

func TestWebSocketRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request under limit passes", func(t *testing.T) {
		t.Setenv("WS_RATE_LIMIT_RPM", "2")
		t.Setenv("WS_RATE_LIMIT_BURST", "2")

		router := gin.New()
		router.Use(WebSocketRateLimitMiddleware())
		router.GET("/ws", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request over limit returns 429", func(t *testing.T) {
		t.Setenv("WS_RATE_LIMIT_RPM", "1")
		t.Setenv("WS_RATE_LIMIT_BURST", "1")

		router := gin.New()
		router.Use(WebSocketRateLimitMiddleware())
		router.GET("/ws", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			req.RemoteAddr = "5.5.5.5:1111"
			router.ServeHTTP(w, req)
			if i == 2 {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})

	t.Run("custom env vars", func(t *testing.T) {
		t.Setenv("WS_RATE_LIMIT_RPM", "100")
		t.Setenv("WS_RATE_LIMIT_BURST", "100")

		router := gin.New()
		router.Use(WebSocketRateLimitMiddleware())
		router.GET("/ws", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.RemoteAddr = "9.9.9.9:2222"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
