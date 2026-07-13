package middleware

import (
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips      map[string]*ipEntry
	mu       *sync.RWMutex
	r        rate.Limit
	b        int
	stopCh   chan struct{}
	stopped  bool
	ttl      time.Duration
}

func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	l := &IPRateLimiter{
		ips:    make(map[string]*ipEntry),
		mu:     &sync.RWMutex{},
		r:      r,
		b:      burst,
		stopCh: make(chan struct{}),
		ttl:    30 * time.Minute,
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
			l.mu.Lock()
			now := time.Now()
			for ip, entry := range l.ips {
				if now.Sub(entry.lastSeen) > l.ttl {
					delete(l.ips, ip)
				}
			}
			l.mu.Unlock()
		case <-l.stopCh:
			return
		}
	}
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	entry, exists := l.ips[ip]
	l.mu.RUnlock()

	if exists {
		l.mu.Lock()
		entry.lastSeen = time.Now()
		l.mu.Unlock()
		return entry.limiter
	}

	l.mu.Lock()
	entry, exists = l.ips[ip]
	if !exists {
		entry = &ipEntry{
			limiter:  rate.NewLimiter(l.r, l.b),
			lastSeen: time.Now(),
		}
		l.ips[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	l.mu.Unlock()
	return entry.limiter
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
		log.Printf("warning: X-Forwarded-For header detected (%s) for rate limiting; using RemoteAddr instead. "+
			"If behind a trusted proxy, configure Gin TrustedProxies.", xff)
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
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
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
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts. try again later."})
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
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many refresh attempts. try again later."})
			return
		}
		c.Next()
	}
}
