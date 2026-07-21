package research

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value      interface{}
	expiresAt  time.Time
}

// Cache is a simple in-memory TTL cache.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]cacheEntry
	defaultTTL time.Duration
}

func NewCache(defaultTTL time.Duration) *Cache {
	return &Cache{
		items:      make(map[string]cacheEntry),
		defaultTTL: defaultTTL,
	}
}

func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return entry.value, true
}

