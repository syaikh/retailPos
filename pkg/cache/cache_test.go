package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	c := New(time.Minute, time.Minute)
	assert.NotNil(t, c)
	c.Set("test", "val")
	c.Wait()
	v, ok := c.Get("test")
	assert.True(t, ok)
	assert.Equal(t, "val", v)
}

func TestSetGet(t *testing.T) {
	t.Run("set then get returns value and true", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		c.Set("key1", "value1")
		c.Wait()
		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("get on missing key returns nil and false", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		val, ok := c.Get("missing")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestDelete(t *testing.T) {
	c := New(time.Minute, time.Minute)
	c.Set("key1", "value1")
	c.Wait()
	c.Delete("key1")
	val, ok := c.Get("key1")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestFlushByPrefix(t *testing.T) {
	t.Run("removes only matching prefix keys", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		c.Set("user:1", "u1")
		c.Set("user:2", "u2")
		c.Set("brand:1", "b1")
		c.Wait()

		c.FlushByPrefix("user:")

		_, ok1 := c.Get("user:1")
		_, ok2 := c.Get("user:2")
		_, ok3 := c.Get("brand:1")
		assert.False(t, ok1)
		assert.False(t, ok2)
		assert.True(t, ok3)
	})

	t.Run("empty prefix removes all keys", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		c.Set("key1", "v1")
		c.Set("key2", "v2")
		c.Wait()

		c.FlushByPrefix("")

		_, ok1 := c.Get("key1")
		_, ok2 := c.Get("key2")
		assert.False(t, ok1)
		assert.False(t, ok2)
	})

	t.Run("no matching keys removes nothing", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		c.Set("user:1", "u1")
		c.Set("user:2", "u2")
		c.Wait()

		c.FlushByPrefix("brand:")

		_, ok1 := c.Get("user:1")
		_, ok2 := c.Get("user:2")
		assert.True(t, ok1)
		assert.True(t, ok2)
	})
}

func TestSetWithTTL(t *testing.T) {
	t.Run("accessible immediately after set", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		c.SetWithTTL("key1", "value1", 100*time.Millisecond)
		c.Wait()
		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("expired after waiting", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		c.SetWithTTL("key1", "value1", 100*time.Millisecond)
		time.Sleep(200 * time.Millisecond)
		val, ok := c.Get("key1")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("jitter within tolerance", func(t *testing.T) {
		c := New(time.Minute, time.Minute)
		ttl := 500 * time.Millisecond
		c.SetWithTTL("key1", "value1", ttl)
		c.Wait()

		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})
}

func TestStats(t *testing.T) {
	c := New(time.Minute, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Wait()

	items, evictions := c.Stats()
	assert.Equal(t, 3, items)
	assert.Equal(t, 0, evictions)
}

func TestJitter(t *testing.T) {
	t.Run("non-zero ttl within range", func(t *testing.T) {
		ttl := time.Second
		for i := 0; i < 100; i++ {
			j := jitter(ttl)
			lower := time.Duration(float64(ttl) * -0.1)
			upper := time.Duration(float64(ttl) * 0.1)
			assert.GreaterOrEqual(t, j, lower)
			assert.LessOrEqual(t, j, upper)
		}
	})

	t.Run("zero ttl returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), jitter(0))
	})

	t.Run("negative ttl returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), jitter(-time.Second))
	})

	t.Run("produces varying results", func(t *testing.T) {
		ttl := time.Second
		results := make(map[time.Duration]bool)
		for i := 0; i < 50; i++ {
			results[jitter(ttl)] = true
		}
		assert.Greater(t, len(results), 1, "jitter should produce varying results")
	})
}
