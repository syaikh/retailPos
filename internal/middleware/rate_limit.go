package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"retail-pos-system/internal/shared"
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips     map[string]*ipEntry
	mu      *sync.Mutex
	r       rate.Limit
	b       int
	stopCh  chan struct{}
	stopped bool
	ttl     time.Duration
	maxSize int
}

func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	l := &IPRateLimiter{
		ips:     make(map[string]*ipEntry),
		mu:      &sync.Mutex{},
		r:       r,
		b:       burst,
		stopCh:  make(chan struct{}),
		ttl:     30 * time.Minute,
		maxSize: 10000,
	}
	go l.cleanupLoop()
	return l
}

func (l *IPRateLimiter) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.stopped {
		l.stopped = true
		close(l.stopCh)
	}
}

func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.cleanupOnce()
		case <-l.stopCh:
			return
		}
	}
}

func (l *IPRateLimiter) cleanupOnce() {
	l.mu.Lock()
	now := time.Now()
	for ip, entry := range l.ips {
		if now.Sub(entry.lastSeen) > l.ttl {
			delete(l.ips, ip)
		}
	}
	l.mu.Unlock()
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.ips[ip]
	if exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	if len(l.ips) >= l.maxSize {
		l.evictOldestLocked()
	}

	entry = &ipEntry{
		limiter:  rate.NewLimiter(l.r, l.b),
		lastSeen: time.Now(),
	}
	l.ips[ip] = entry
	return entry.limiter
}

func (l *IPRateLimiter) evictOldestLocked() {
	var oldestIP string
	var oldestTime time.Time
	for ip, entry := range l.ips {
		if oldestIP == "" || entry.lastSeen.Before(oldestTime) {
			oldestIP = ip
			oldestTime = entry.lastSeen
		}
	}
	if oldestIP != "" {
		delete(l.ips, oldestIP)
	}
}

// getClientIP extracts the real client IP from the TCP connection's RemoteAddr
// instead of trusting X-Forwarded-For, which can be spoofed by clients to bypass
// rate limiting. If X-Forwarded-For is present, a warning is logged because it
// may indicate a misconfigured reverse proxy or a spoofing attempt.
//
// WARNING: This server does not validate trusted proxy IPs. If deployed behind
// a reverse proxy, configure TrustedProxies on the Gin engine and replace this
// with proxy-aware IP extraction. Do not use c.ClientIP() for rate limiting
// with untrusted proxies.
func getClientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		slog.Warn("X-Forwarded-For header detected for rate limiting; using RemoteAddr instead. "+
			"If behind a trusted proxy, configure Gin TrustedProxies.", "xff", xff)
	}

	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		// RemoteAddr may not have a port (e.g. Unix sockets); fall back as-is.
		return strings.TrimSpace(c.Request.RemoteAddr)
	}
	return host
}

func RateLimitMiddleware() gin.HandlerFunc {
	rps := 50
	burst := 100
	if v, err := strconv.Atoi(os.Getenv("RATE_LIMIT_RPS")); err == nil && v > 0 {
		rps = v
	}
	if v, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BURST")); err == nil && v > 0 {
		burst = v
	}
	limiter := NewIPRateLimiter(rate.Limit(rps), burst)

	return func(c *gin.Context) {
		ip := getClientIP(c)
		l := limiter.GetLimiter(ip)
		c.Header("X-RateLimit-Limit", strconv.Itoa(burst))
		if !l.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, shared.NewError(shared.ErrRateLimited, "too many requests"))
			return
		}
		c.Next()
	}
}

func LoginRateLimitMiddleware() gin.HandlerFunc {
	burst := 5
	rpm := 5
	if v, err := strconv.Atoi(os.Getenv("LOGIN_RATE_LIMIT_RPM")); err == nil && v > 0 {
		rpm = v
	}
	if v, err := strconv.Atoi(os.Getenv("LOGIN_RATE_LIMIT_BURST")); err == nil && v > 0 {
		burst = v
	}
	limiter := NewIPRateLimiter(rate.Every(time.Minute/time.Duration(rpm)), burst)

	return func(c *gin.Context) {
		ip := getClientIP(c)
		l := limiter.GetLimiter(ip)
		c.Header("X-RateLimit-Limit", strconv.Itoa(burst))
		if !l.Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, shared.NewError(shared.ErrRateLimited, "too many login attempts. try again later."))
			return
		}
		c.Next()
	}
}

func RefreshRateLimitMiddleware() gin.HandlerFunc {
	burst := 10
	rpm := 10
	if v, err := strconv.Atoi(os.Getenv("REFRESH_RATE_LIMIT_RPM")); err == nil && v > 0 {
		rpm = v
	}
	if v, err := strconv.Atoi(os.Getenv("REFRESH_RATE_LIMIT_BURST")); err == nil && v > 0 {
		burst = v
	}
	limiter := NewIPRateLimiter(rate.Every(time.Minute/time.Duration(rpm)), burst)

	return func(c *gin.Context) {
		ip := getClientIP(c)
		l := limiter.GetLimiter(ip)
		c.Header("X-RateLimit-Limit", strconv.Itoa(burst))
		if !l.Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, shared.NewError(shared.ErrRateLimited, "too many refresh attempts. try again later."))
			return
		}
		c.Next()
	}
}

func WebSocketRateLimitMiddleware() gin.HandlerFunc {
	burst := 5
	rpm := 20
	if v, err := strconv.Atoi(os.Getenv("WS_RATE_LIMIT_RPM")); err == nil && v > 0 {
		rpm = v
	}
	if v, err := strconv.Atoi(os.Getenv("WS_RATE_LIMIT_BURST")); err == nil && v > 0 {
		burst = v
	}
	limiter := NewIPRateLimiter(rate.Every(time.Minute/time.Duration(rpm)), burst)

	return func(c *gin.Context) {
		ip := getClientIP(c)
		l := limiter.GetLimiter(ip)
		c.Header("X-RateLimit-Limit", strconv.Itoa(burst))
		if !l.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, shared.NewError(shared.ErrRateLimited, "too many WebSocket connections. try again later."))
			return
		}
		c.Next()
	}
}
