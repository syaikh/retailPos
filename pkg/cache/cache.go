package cache

import (
	"math/rand"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
)

type Cache struct {
	store      *ristretto.Cache
	keys       map[string]struct{}
	mu         sync.RWMutex
	defaultTTL time.Duration
}

func New(defaultTTL, cleanupInterval time.Duration) *Cache {
	store, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}
	return &Cache{
		store:      store,
		keys:       make(map[string]struct{}),
		defaultTTL: defaultTTL,
	}
}

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	c.keys[key] = struct{}{}
	c.mu.Unlock()
	c.store.SetWithTTL(key, value, 1, c.defaultTTL+jitter(c.defaultTTL))
}

func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	c.keys[key] = struct{}{}
	c.mu.Unlock()
	c.store.SetWithTTL(key, value, 1, ttl+jitter(ttl))
}

func (c *Cache) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.keys, key)
	c.mu.Unlock()
	c.store.Del(key)
	c.store.Wait()
}

func (c *Cache) FlushByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.keys {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.keys, key)
			c.store.Del(key)
		}
	}
	c.store.Wait()
}

func (c *Cache) Wait() {
	c.store.Wait()
}

func (c *Cache) Stats() (items int, evictions int) {
	m := c.store.Metrics
	if m == nil {
		return len(c.keys), 0
	}
	return len(c.keys), int(m.KeysEvicted())
}

func jitter(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return 0
	}
	j := float64(ttl) * 0.1
	return time.Duration(rand.Float64()*2*j - j)
}
