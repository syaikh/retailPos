package middleware

import (
	"net/http"
	"os"
	"strconv"
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

// getClientIP returns the real client IP as resolved by gin: X-Forwarded-For
// is honored only when the TCP peer (RemoteAddr) is a trusted proxy per
// router.SetTrustedProxies, so a client on the open internet cannot spoof the
// address that rate limiting is keyed on. Callers must not substitute
// c.Request.RemoteAddr here: behind the bundled nginx every request would then
// collapse into a single proxy-IP bucket shared by the entire deployment.
func getClientIP(c *gin.Context) string {
	return c.ClientIP()
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
	// Coarse flood guard sized for the largest plausible store NAT: all
	// registers in one store share one public IP, so this must absorb a
	// shift-start login burst without 429ing legitimate cashiers. Brute-force
	// protection is not this layer's job — the per-username (5/15 min) and
	// per-IP (10/15 min) failure counters in AuthService.Login do that, and
	// they count failures only. Aligned with the nginx edge zone (30 r/m,
	// burst 20) in deploy/nginx/default.conf.template.
	burst := 20
	rpm := 30
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
	// Every register refreshes its 15-minute access token on that cadence, and
	// after a backend restart they all refresh at once; the previous 10 rpm /
	// burst 10 would 429 a store with more than a handful of devices behind
	// one NAT. Refresh is a cheap JWT sign (no bcrypt), so the guard only needs
	// to stop floods.
	burst := 60
	rpm := 60
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
