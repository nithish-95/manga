package mangadex

import (
	"sync"
)

// Cache is a simple in-memory cache with a mutex for concurrent access.

type Cache[T any] struct {
	data map[string]T
	mu   sync.RWMutex
}

func NewCache[T any]() *Cache[T] {
	return &Cache[T]{
		data: make(map[string]T),
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}
