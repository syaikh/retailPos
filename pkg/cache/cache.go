package cache

import (
	"math/rand"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache is a thread-safe in-memory cache with TTL and jittered eviction.
type Cache struct {
	store      *gocache.Cache
	mu         sync.RWMutex
	defaultTTL time.Duration
}

// New creates a cache with the given default TTL and cleanup interval.
// Cleanup runs in the background to reclaim expired entries.
func New(defaultTTL, cleanupInterval time.Duration) *Cache {
	return &Cache{
		store:      gocache.New(cleanupInterval, cleanupInterval*2),
		defaultTTL: defaultTTL,
	}
}

// Set stores a value with the default TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.store.Set(key, value, c.defaultTTL)
}

// SetWithTTL stores a value with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.store.Set(key, value, ttl+jitter(ttl))
}

// Get retrieves a value. Returns (value, true) on hit, (nil, false) on miss.
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

// Delete removes a single key.
func (c *Cache) Delete(key string) {
	c.store.Delete(key)
}

// FlushByPrefix removes all keys that start with prefix.
func (c *Cache) FlushByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := c.store.Items()
	for key := range items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.store.Delete(key)
		}
	}
}

// Stats returns cache statistics for observability.
func (c *Cache) Stats() (items int, evictions int) {
	return c.store.ItemCount(), 0
}

// jitter adds ±10% random variation to a TTL to prevent stampede.
func jitter(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return 0
	}
	j := float64(ttl) * 0.1
	return time.Duration(rand.Float64()*2*j - j)
}
