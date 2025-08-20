package cache

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"

	"vault-docker-proxy/pkg/auth"
)

const (
	DefaultCacheTTL     = 5 * time.Minute
	DefaultCleanupInterval = 10 * time.Minute
)

// CredentialCache provides caching for registry credentials
type CredentialCache struct {
	cache *cache.Cache
}

// NewCredentialCache creates a new credential cache with default TTL
func NewCredentialCache() *CredentialCache {
	return &CredentialCache{
		cache: cache.New(DefaultCacheTTL, DefaultCleanupInterval),
	}
}

// NewCredentialCacheWithTTL creates a new credential cache with custom TTL
func NewCredentialCacheWithTTL(ttl, cleanupInterval time.Duration) *CredentialCache {
	return &CredentialCache{
		cache: cache.New(ttl, cleanupInterval),
	}
}

// generateCacheKey creates a unique cache key from vault token and path
func (c *CredentialCache) generateCacheKey(vaultToken, vaultPath string) string {
	// Hash the token and path for security and consistency
	h := sha256.New()
	h.Write([]byte(vaultToken + ":" + vaultPath))
	return fmt.Sprintf("creds:%x", h.Sum(nil))
}

// Get retrieves cached credentials if available
func (c *CredentialCache) Get(vaultToken, vaultPath string) (*auth.Credentials, bool) {
	key := c.generateCacheKey(vaultToken, vaultPath)
	
	if item, found := c.cache.Get(key); found {
		if creds, ok := item.(*auth.Credentials); ok {
			return creds, true
		}
	}
	
	return nil, false
}

// Set stores credentials in cache with default TTL
func (c *CredentialCache) Set(vaultToken, vaultPath string, credentials *auth.Credentials) {
	key := c.generateCacheKey(vaultToken, vaultPath)
	c.cache.Set(key, credentials, cache.DefaultExpiration)
}

// SetWithTTL stores credentials in cache with custom TTL
func (c *CredentialCache) SetWithTTL(vaultToken, vaultPath string, credentials *auth.Credentials, ttl time.Duration) {
	key := c.generateCacheKey(vaultToken, vaultPath)
	c.cache.Set(key, credentials, ttl)
}

// Delete removes credentials from cache
func (c *CredentialCache) Delete(vaultToken, vaultPath string) {
	key := c.generateCacheKey(vaultToken, vaultPath)
	c.cache.Delete(key)
}

// Clear removes all cached credentials
func (c *CredentialCache) Clear() {
	c.cache.Flush()
}

// Stats returns cache statistics
func (c *CredentialCache) Stats() (itemCount int, evictedCount int64, hitCount uint64, missCount uint64) {
	itemCount = c.cache.ItemCount()
	// Note: go-cache doesn't provide hit/miss/eviction stats by default
	// These would need to be tracked separately if detailed metrics are needed
	return itemCount, 0, 0, 0
}

// CachedCredentialGetter interface for objects that can retrieve and cache credentials
type CachedCredentialGetter interface {
	GetCredentials(vaultToken, vaultPath string) (*auth.Credentials, error)
}

// CacheWrapper wraps a credential getter with caching functionality
type CacheWrapper struct {
	cache  *CredentialCache
	getter CachedCredentialGetter
}

// NewCacheWrapper creates a new cache wrapper
func NewCacheWrapper(getter CachedCredentialGetter) *CacheWrapper {
	return &CacheWrapper{
		cache:  NewCredentialCache(),
		getter: getter,
	}
}

// GetCredentials attempts to get credentials from cache first, then from the underlying getter
func (w *CacheWrapper) GetCredentials(vaultToken, vaultPath string) (*auth.Credentials, error) {
	// Try cache first
	if creds, found := w.cache.Get(vaultToken, vaultPath); found {
		return creds, nil
	}

	// Not in cache, get from underlying getter
	creds, err := w.getter.GetCredentials(vaultToken, vaultPath)
	if err != nil {
		return nil, err
	}

	// Store in cache
	w.cache.Set(vaultToken, vaultPath, creds)

	return creds, nil
}