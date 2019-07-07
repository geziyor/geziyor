package memorycache

import (
	"sync"
)

// Cache is an implementation of Cache that stores responses in an in-memory map.
type Cache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

// Get returns the []byte representation of the response and true if present, false if not
func (c *Cache) Get(key string) (resp []byte, ok bool) {
	c.mu.RLock()
	resp, ok = c.items[key]
	c.mu.RUnlock()
	return resp, ok
}

// Set saves response resp to the cache with key
func (c *Cache) Set(key string, resp []byte) {
	c.mu.Lock()
	c.items[key] = resp
	c.mu.Unlock()
}

// Delete removes key from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// New returns a new Cache that will store items in an in-memory map
func New() *Cache {
	c := &Cache{items: map[string][]byte{}}
	return c
}
