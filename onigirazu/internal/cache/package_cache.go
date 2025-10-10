package cache

import (
	"sync"
	"time"
)

// PackageInfo represents cached package information
type PackageInfo struct {
	Name      string
	Version   string
	Installed bool
	CachedAt  time.Time
	ExpiresAt time.Time
}

// PackageCache provides caching for package information
type PackageCache struct {
	mu      sync.RWMutex
	cache   map[string]map[string]*PackageInfo // host -> package -> info
	ttl     time.Duration
	enabled bool
}

// PackageCacheConfig holds configuration for package cache
type PackageCacheConfig struct {
	TTL     time.Duration
	Enabled bool
}

// DefaultPackageCacheConfig returns default configuration
func DefaultPackageCacheConfig() PackageCacheConfig {
	return PackageCacheConfig{
		TTL:     5 * time.Minute, // Cache for 5 minutes
		Enabled: true,
	}
}

// NewPackageCache creates a new package cache
func NewPackageCache(config PackageCacheConfig) *PackageCache {
	cache := &PackageCache{
		cache:   make(map[string]map[string]*PackageInfo),
		ttl:     config.TTL,
		enabled: config.Enabled,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves package info from cache
func (c *PackageCache) Get(host, packageName string) (*PackageInfo, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	hostCache, exists := c.cache[host]
	if !exists {
		return nil, false
	}

	info, exists := hostCache[packageName]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(info.ExpiresAt) {
		return nil, false
	}

	return info, true
}

// Set stores package info in cache
func (c *PackageCache) Set(host, packageName string, installed bool, version string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[host]; !exists {
		c.cache[host] = make(map[string]*PackageInfo)
	}

	now := time.Now()
	c.cache[host][packageName] = &PackageInfo{
		Name:      packageName,
		Version:   version,
		Installed: installed,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// Invalidate removes package info from cache
func (c *PackageCache) Invalidate(host, packageName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if hostCache, exists := c.cache[host]; exists {
		delete(hostCache, packageName)
	}
}

// InvalidateHost removes all cached info for a host
func (c *PackageCache) InvalidateHost(host string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, host)
}

// Clear removes all cached data
func (c *PackageCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]map[string]*PackageInfo)
}

// GetStats returns cache statistics
func (c *PackageCache) GetStats() PackageCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalEntries := 0
	expiredEntries := 0
	now := time.Now()

	for _, hostCache := range c.cache {
		for _, info := range hostCache {
			totalEntries++
			if now.After(info.ExpiresAt) {
				expiredEntries++
			}
		}
	}

	return PackageCacheStats{
		TotalEntries:   totalEntries,
		ExpiredEntries: expiredEntries,
		HostCount:      len(c.cache),
	}
}

// cleanupLoop periodically removes expired entries
func (c *PackageCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *PackageCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for host, hostCache := range c.cache {
		for packageName, info := range hostCache {
			if now.After(info.ExpiresAt) {
				delete(hostCache, packageName)
			}
		}

		// Remove empty host caches
		if len(hostCache) == 0 {
			delete(c.cache, host)
		}
	}
}

// PackageCacheStats holds cache statistics
type PackageCacheStats struct {
	TotalEntries   int
	ExpiredEntries int
	HostCount      int
}

// Global package cache instance
var globalPackageCache *PackageCache
var packageCacheOnce sync.Once

// GetGlobalPackageCache returns the global package cache instance
func GetGlobalPackageCache() *PackageCache {
	packageCacheOnce.Do(func() {
		globalPackageCache = NewPackageCache(DefaultPackageCacheConfig())
	})
	return globalPackageCache
}

// SetGlobalPackageCache sets the global package cache instance
func SetGlobalPackageCache(cache *PackageCache) {
	globalPackageCache = cache
}
