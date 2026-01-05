// cache/cache.go
package cache

import (
	"sync"
	"time"
)

type Cache struct {
    mu    sync.RWMutex
    items map[string]*Item
}

func NewCache() *Cache {
    c := &Cache{
        items: make(map[string]*Item),
    }
    go c.cleanupExpired()
    return c
}

func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    var expiration int64
    if ttl > 0 {
        expiration = time.Now().Add(ttl).UnixNano()
    }

    c.items[key] = &Item{
        Value:      value,
        Expiration: expiration,
    }
}

func (c *Cache) Get(key string) ([]byte, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, found := c.items[key]
    if !found {
        return nil, false
    }

    if item.IsExpired() {
        return nil, false
    }

    return item.Value, true
}

func (c *Cache) Delete(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.items, key)
}

func (c *Cache) cleanupExpired() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        c.mu.Lock()
        for key, item := range c.items {
            if item.IsExpired() {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}