package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter naive in-memory rate limiter per IP (development only).
// Not suitable for multi-instance production (use Redis then).
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(rps int, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   rate.Limit(rps),
		b:   burst,
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
		return l.AddIP(ip)
	}
	return limiter
}

// RateLimitMiddleware returns a gin handler that limits requests per IP.
// Exclude sensitive endpoints if needed.
func RateLimitMiddleware() gin.HandlerFunc {
	// 5 requests per second, burst 10 (300 req/min)
	limiter := NewIPRateLimiter(5, 10)

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
	limiter := NewIPRateLimiter(5, 5)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts. try again later."})
			return
		}
		c.Next()
	}
}
