package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPackageCache(t *testing.T) {
	config := PackageCacheConfig{
		TTL:     10 * time.Minute,
		Enabled: true,
	}

	cache := NewPackageCache(config)

	assert.NotNil(t, cache)
	assert.Equal(t, 10*time.Minute, cache.ttl)
	assert.True(t, cache.enabled)
	assert.NotNil(t, cache.cache)
}

func TestPackageCache_SetAndGet(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Set package info
	cache.Set("host1", "nginx", true, "1.18.0")

	// Get package info
	info, found := cache.Get("host1", "nginx")

	assert.True(t, found)
	assert.NotNil(t, info)
	assert.Equal(t, "nginx", info.Name)
	assert.Equal(t, "1.18.0", info.Version)
	assert.True(t, info.Installed)
}

func TestPackageCache_GetNonExistent(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Try to get non-existent package
	info, found := cache.Get("host1", "nonexistent")

	assert.False(t, found)
	assert.Nil(t, info)
}

func TestPackageCache_Expiration(t *testing.T) {
	config := PackageCacheConfig{
		TTL:     100 * time.Millisecond,
		Enabled: true,
	}
	cache := NewPackageCache(config)

	// Set package info
	cache.Set("host1", "nginx", true, "1.18.0")

	// Should be found immediately
	info, found := cache.Get("host1", "nginx")
	assert.True(t, found)
	assert.NotNil(t, info)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	info, found = cache.Get("host1", "nginx")
	assert.False(t, found)
	assert.Nil(t, info)
}

func TestPackageCache_Invalidate(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Set package info
	cache.Set("host1", "nginx", true, "1.18.0")

	// Verify it's cached
	_, found := cache.Get("host1", "nginx")
	assert.True(t, found)

	// Invalidate
	cache.Invalidate("host1", "nginx")

	// Should not be found after invalidation
	_, found = cache.Get("host1", "nginx")
	assert.False(t, found)
}

func TestPackageCache_InvalidateHost(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Set multiple packages for host
	cache.Set("host1", "nginx", true, "1.18.0")
	cache.Set("host1", "apache2", true, "2.4.41")
	cache.Set("host2", "mysql", true, "8.0.23")

	// Verify they're cached
	_, found1 := cache.Get("host1", "nginx")
	_, found2 := cache.Get("host1", "apache2")
	_, found3 := cache.Get("host2", "mysql")
	assert.True(t, found1)
	assert.True(t, found2)
	assert.True(t, found3)

	// Invalidate host1
	cache.InvalidateHost("host1")

	// host1 packages should not be found
	_, found1 = cache.Get("host1", "nginx")
	_, found2 = cache.Get("host1", "apache2")
	assert.False(t, found1)
	assert.False(t, found2)

	// host2 packages should still be found
	_, found3 = cache.Get("host2", "mysql")
	assert.True(t, found3)
}

func TestPackageCache_Clear(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Set multiple packages
	cache.Set("host1", "nginx", true, "1.18.0")
	cache.Set("host2", "apache2", true, "2.4.41")

	// Verify they're cached
	stats := cache.GetStats()
	assert.Equal(t, 2, stats.TotalEntries)

	// Clear cache
	cache.Clear()

	// Should be empty
	stats = cache.GetStats()
	assert.Equal(t, 0, stats.TotalEntries)
	assert.Equal(t, 0, stats.HostCount)
}

func TestPackageCache_GetStats(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Set packages for multiple hosts
	cache.Set("host1", "nginx", true, "1.18.0")
	cache.Set("host1", "apache2", true, "2.4.41")
	cache.Set("host2", "mysql", true, "8.0.23")

	stats := cache.GetStats()

	assert.Equal(t, 3, stats.TotalEntries)
	assert.Equal(t, 2, stats.HostCount)
	assert.Equal(t, 0, stats.ExpiredEntries)
}

func TestPackageCache_Disabled(t *testing.T) {
	config := PackageCacheConfig{
		TTL:     10 * time.Minute,
		Enabled: false,
	}
	cache := NewPackageCache(config)

	// Set package info
	cache.Set("host1", "nginx", true, "1.18.0")

	// Should not be found when cache is disabled
	info, found := cache.Get("host1", "nginx")
	assert.False(t, found)
	assert.Nil(t, info)
}

func TestPackageCache_MultipleHosts(t *testing.T) {
	cache := NewPackageCache(DefaultPackageCacheConfig())

	// Set same package for different hosts
	cache.Set("host1", "nginx", true, "1.18.0")
	cache.Set("host2", "nginx", true, "1.20.0")

	// Get from host1
	info1, found1 := cache.Get("host1", "nginx")
	assert.True(t, found1)
	assert.Equal(t, "1.18.0", info1.Version)

	// Get from host2
	info2, found2 := cache.Get("host2", "nginx")
	assert.True(t, found2)
	assert.Equal(t, "1.20.0", info2.Version)
}

func TestPackageCache_Cleanup(t *testing.T) {
	config := PackageCacheConfig{
		TTL:     100 * time.Millisecond,
		Enabled: true,
	}
	cache := NewPackageCache(config)

	// Set packages
	cache.Set("host1", "nginx", true, "1.18.0")
	cache.Set("host1", "apache2", true, "2.4.41")

	// Verify they're cached
	stats := cache.GetStats()
	assert.Equal(t, 2, stats.TotalEntries)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Run cleanup manually
	cache.cleanup()

	// Should be empty after cleanup
	stats = cache.GetStats()
	assert.Equal(t, 0, stats.TotalEntries)
	assert.Equal(t, 0, stats.HostCount)
}

func TestGlobalPackageCache(t *testing.T) {
	// Get global cache
	cache1 := GetGlobalPackageCache()
	assert.NotNil(t, cache1)

	// Should return same instance
	cache2 := GetGlobalPackageCache()
	assert.Equal(t, cache1, cache2)

	// Set custom cache
	customCache := NewPackageCache(PackageCacheConfig{
		TTL:     1 * time.Minute,
		Enabled: true,
	})
	SetGlobalPackageCache(customCache)

	// Should return custom cache
	cache3 := GetGlobalPackageCache()
	assert.Equal(t, customCache, cache3)
}

func TestDefaultPackageCacheConfig(t *testing.T) {
	config := DefaultPackageCacheConfig()

	assert.Equal(t, 5*time.Minute, config.TTL)
	assert.True(t, config.Enabled)
}
