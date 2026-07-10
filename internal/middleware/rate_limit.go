package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter naive in-memory rate limiter per IP (development only).
// Not suitable for multi-instance production (use Redis then).
type IPRateLimiter struct {
	ips      map[string]*rate.Limiter
	mu       *sync.RWMutex
	r        rate.Limit
	b        int
	stopCh   chan struct{}
	stopped  bool
}

func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	l := &IPRateLimiter{
		ips:    make(map[string]*rate.Limiter),
		mu:     &sync.RWMutex{},
		r:      r,
		b:      burst,
		stopCh: make(chan struct{}),
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
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			l.ips = make(map[string]*rate.Limiter)
			l.mu.Unlock()
		case <-l.stopCh:
			return
		}
	}
}

func (l *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter := rate.NewLimiter(l.r, l.b)
	l.ips[ip] = limiter
	return limiter
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.ips[ip]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		limiter, exists = l.ips[ip]
		if !exists {
			limiter = rate.NewLimiter(l.r, l.b)
			l.ips[ip] = limiter
		}
		l.mu.Unlock()
	}
	return limiter
}

// RateLimitMiddleware returns a gin handler that limits requests per IP.
// Exclude sensitive endpoints if needed.
func RateLimitMiddleware() gin.HandlerFunc {
	// 5 requests per second, burst 10 (300 req/min)
	limiter := NewIPRateLimiter(rate.Limit(5), 10)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

// LoginRateLimitMiddleware returns a stricter rate limiter for the login endpoint.
// 5 requests per minute, burst 5 — limits brute-force attempts.
func LoginRateLimitMiddleware() gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Every(time.Minute/5), 5)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts. try again later."})
			return
		}
		c.Next()
	}
}

// RefreshRateLimitMiddleware returns a moderate rate limiter for the refresh endpoint.
// 10 requests per minute, burst 10 — limits rotation abuse while allowing legitimate use.
func RefreshRateLimitMiddleware() gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Every(time.Minute/10), 10)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many refresh attempts. try again later."})
			return
		}
		c.Next()
	}
}
