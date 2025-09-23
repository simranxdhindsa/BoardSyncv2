package cache

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"
)

// MemoryCache implements an in-memory cache with TTL support
type MemoryCache struct {
    items map[string]*CacheItem
    mutex sync.RWMutex
}

// CacheItem represents a cached item with expiration
type CacheItem struct {
    Value     interface{}
    ExpiresAt time.Time
}

// Cache interface for different cache implementations
type Cache interface {
    Set(key string, value interface{}, ttl time.Duration) error
    Get(key string, dest interface{}) error
    Delete(key string) error
    Clear() error
    Exists(key string) bool
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
    cache := &MemoryCache{
        items: make(map[string]*CacheItem),
    }
    
    // Start cleanup goroutine
    go cache.cleanup()
    
    return cache
}

// Set stores a value in the cache with TTL
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
    mc.mutex.Lock()
    defer mc.mutex.Unlock()

    expiresAt := time.Now().Add(ttl)
    if ttl <= 0 {
        // No expiration
        expiresAt = time.Time{}
    }

    mc.items[key] = &CacheItem{
        Value:     value,
        ExpiresAt: expiresAt,
    }

    return nil
}

// Get retrieves a value from the cache
func (mc *MemoryCache) Get(key string, dest interface{}) error {
    mc.mutex.RLock()
    defer mc.mutex.RUnlock()

    item, exists := mc.items[key]
    if !exists {
        return ErrCacheNotFound
    }

    // Check if item has expired
    if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
        // Remove expired item
        delete(mc.items, key)
        return ErrCacheNotFound
    }

    // Use JSON marshaling/unmarshaling for deep copy
    data, err := json.Marshal(item.Value)
    if err != nil {
        return err
    }

    return json.Unmarshal(data, dest)
}

// Delete removes a value from the cache
func (mc *MemoryCache) Delete(key string) error {
    mc.mutex.Lock()
    defer mc.mutex.Unlock()

    delete(mc.items, key)
    return nil
}

// Clear removes all items from the cache
func (mc *MemoryCache) Clear() error {
    mc.mutex.Lock()
    defer mc.mutex.Unlock()

    mc.items = make(map[string]*CacheItem)
    return nil
}

// Exists checks if a key exists in the cache
func (mc *MemoryCache) Exists(key string) bool {
    mc.mutex.RLock()
    defer mc.mutex.RUnlock()

    item, exists := mc.items[key]
    if !exists {
        return false
    }

    // Check if item has expired
    if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
        // Remove expired item
        delete(mc.items, key)
        return false
    }

    return true
}

// GetString retrieves a string value from the cache
func (mc *MemoryCache) GetString(key string) (string, error) {
    var value string
    err := mc.Get(key, &value)
    return value, err
}

// SetString stores a string value in the cache
func (mc *MemoryCache) SetString(key, value string, ttl time.Duration) error {
    return mc.Set(key, value, ttl)
}

// GetInt retrieves an int value from the cache
func (mc *MemoryCache) GetInt(key string) (int, error) {
    var value int
    err := mc.Get(key, &value)
    return value, err
}

// SetInt stores an int value in the cache
func (mc *MemoryCache) SetInt(key string, value int, ttl time.Duration) error {
    return mc.Set(key, value, ttl)
}

// GetKeys returns all keys in the cache
func (mc *MemoryCache) GetKeys() []string {
    mc.mutex.RLock()
    defer mc.mutex.RUnlock()

    keys := make([]string, 0, len(mc.items))
    now := time.Now()

    for key, item := range mc.items {
        // Skip expired items
        if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
            continue
        }
        keys = append(keys, key)
    }

    return keys
}

// Size returns the number of items in the cache
func (mc *MemoryCache) Size() int {
    mc.mutex.RLock()
    defer mc.mutex.RUnlock()

    count := 0
    now := time.Now()

    for _, item := range mc.items {
        // Skip expired items
        if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
            continue
        }
        count++
    }

    return count
}

// cleanup periodically removes expired items
func (mc *MemoryCache) cleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            mc.removeExpired()
        }
    }
}

// removeExpired removes all expired items from the cache
func (mc *MemoryCache) removeExpired() {
    mc.mutex.Lock()
    defer mc.mutex.Unlock()

    now := time.Now()
    for key, item := range mc.items {
        if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
            delete(mc.items, key)
        }
    }
}

// Cache errors
var (
    ErrCacheNotFound = fmt.Errorf("cache: key not found")
    ErrCacheExpired  = fmt.Errorf("cache: key expired")
)

// CacheManager manages multiple cache instances
type CacheManager struct {
    defaultCache Cache
    caches       map[string]Cache
    mutex        sync.RWMutex
}

// NewCacheManager creates a new cache manager
func NewCacheManager() *CacheManager {
    return &CacheManager{
        defaultCache: NewMemoryCache(),
        caches:       make(map[string]Cache),
    }
}

// GetCache returns a cache instance by name
func (cm *CacheManager) GetCache(name string) Cache {
    if name == "" {
        return cm.defaultCache
    }

    cm.mutex.RLock()
    defer cm.mutex.RUnlock()

    if cache, exists := cm.caches[name]; exists {
        return cache
    }

    return cm.defaultCache
}

// AddCache adds a new cache instance
func (cm *CacheManager) AddCache(name string, cache Cache) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()

    cm.caches[name] = cache
}

// RemoveCache removes a cache instance
func (cm *CacheManager) RemoveCache(name string) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()

    delete(cm.caches, name)
}

// ClearAll clears all cache instances
func (cm *CacheManager) ClearAll() error {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()

    if err := cm.defaultCache.Clear(); err != nil {
        return err
    }

    for _, cache := range cm.caches {
        if err := cache.Clear(); err != nil {
            return err
        }
    }

    return nil
}