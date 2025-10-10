package cache

import (
	"testing"
	"time"
)

func TestNewFactsCache(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	if cache.ttl != 5*time.Minute {
		t.Errorf("Expected TTL to be 5 minutes, got %v", cache.ttl)
	}

	if !cache.enabled {
		t.Error("Expected cache to be enabled by default")
	}
}

func TestFactsCache_SetAndGet(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	facts := &SystemFacts{
		OSFamily:      "Debian",
		Distribution:  "Ubuntu",
		OSVersion:     "24.04",
		Architecture:  "x86_64",
		Kernel:        "Linux",
		KernelVersion: "6.8.0-45-generic",
		Hostname:      "test-host",
		CPUCores:      4,
		MemoryTotal:   "8GB",
		FQDN:          "test-host.example.com",
		DefaultIPv4:   "192.168.1.100",
	}

	// Set facts
	cache.Set("test-host", facts)

	// Get facts
	retrieved, found := cache.Get("test-host")
	if !found {
		t.Fatal("Expected to find cached facts")
	}

	if retrieved.OSFamily != "Debian" {
		t.Errorf("Expected OS family 'Debian', got '%s'", retrieved.OSFamily)
	}

	if retrieved.Distribution != "Ubuntu" {
		t.Errorf("Expected distribution 'Ubuntu', got '%s'", retrieved.Distribution)
	}

	if retrieved.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", retrieved.Hostname)
	}
}

func TestFactsCache_GetNonExistent(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	_, found := cache.Get("non-existent-host")
	if found {
		t.Error("Expected not to find non-existent host")
	}
}

func TestFactsCache_Expiration(t *testing.T) {
	cache := NewFactsCache(100 * time.Millisecond)

	facts := &SystemFacts{
		OSFamily:     "Debian",
		Distribution: "Ubuntu",
		Hostname:     "test-host",
	}

	cache.Set("test-host", facts)

	// Should be found immediately
	_, found := cache.Get("test-host")
	if !found {
		t.Error("Expected to find facts immediately after setting")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get("test-host")
	if found {
		t.Error("Expected facts to be expired")
	}
}

func TestFactsCache_Invalidate(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	facts := &SystemFacts{
		OSFamily: "Debian",
		Hostname: "test-host",
	}

	cache.Set("test-host", facts)

	// Should be found
	_, found := cache.Get("test-host")
	if !found {
		t.Error("Expected to find facts before invalidation")
	}

	// Invalidate
	cache.Invalidate("test-host")

	// Should not be found after invalidation
	_, found = cache.Get("test-host")
	if found {
		t.Error("Expected facts to be invalidated")
	}
}

func TestFactsCache_Clear(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	// Add multiple hosts
	for i := 1; i <= 3; i++ {
		facts := &SystemFacts{
			OSFamily: "Debian",
			Hostname: "test-host",
		}
		cache.Set("host"+string(rune('0'+i)), facts)
	}

	// Verify all are cached
	if cache.GetEntryCount() != 3 {
		t.Errorf("Expected 3 entries, got %d", cache.GetEntryCount())
	}

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	if cache.GetEntryCount() != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", cache.GetEntryCount())
	}
}

func TestFactsCache_Stats(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	facts := &SystemFacts{
		OSFamily: "Debian",
		Hostname: "test-host",
	}

	cache.Set("test-host", facts)

	// Generate some hits and misses
	cache.Get("test-host")    // hit
	cache.Get("test-host")    // hit
	cache.Get("non-existent") // miss

	stats := cache.GetStats()

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.Entries)
	}

	expectedHitRate := 66.66666666666666 // 2/3 * 100
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f%%, got %.2f%%", expectedHitRate, stats.HitRate)
	}
}

func TestFactsCache_Disable(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	facts := &SystemFacts{
		OSFamily: "Debian",
		Hostname: "test-host",
	}

	// Set while enabled
	cache.Set("test-host", facts)

	// Disable cache
	cache.Disable()

	if cache.IsEnabled() {
		t.Error("Expected cache to be disabled")
	}

	// Get should return false when disabled
	_, found := cache.Get("test-host")
	if found {
		t.Error("Expected Get to return false when cache is disabled")
	}

	// Set should not work when disabled
	cache.Set("another-host", facts)

	// Re-enable
	cache.Enable()

	// Original entry should still be there
	_, found = cache.Get("test-host")
	if !found {
		t.Error("Expected original entry to still exist after re-enabling")
	}

	// New entry should not be there (wasn't set while disabled)
	_, found = cache.Get("another-host")
	if found {
		t.Error("Expected entry set while disabled to not exist")
	}
}

func TestFactsCache_MultipleHosts(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	hosts := []string{"host1", "host2", "host3"}

	for _, hostname := range hosts {
		facts := &SystemFacts{
			OSFamily: "Debian",
			Hostname: hostname,
		}
		cache.Set(hostname, facts)
	}

	// Verify all hosts are cached
	for _, hostname := range hosts {
		retrieved, found := cache.Get(hostname)
		if !found {
			t.Errorf("Expected to find facts for %s", hostname)
		}
		if retrieved.Hostname != hostname {
			t.Errorf("Expected hostname %s, got %s", hostname, retrieved.Hostname)
		}
	}

	// Verify entry count
	if cache.GetEntryCount() != 3 {
		t.Errorf("Expected 3 entries, got %d", cache.GetEntryCount())
	}
}

func TestFactsCache_Cleanup(t *testing.T) {
	cache := NewFactsCache(100 * time.Millisecond)

	// Add some facts
	facts := &SystemFacts{
		OSFamily: "Debian",
		Hostname: "test-host",
	}
	cache.Set("test-host", facts)

	// Verify it's there
	if cache.GetEntryCount() != 1 {
		t.Error("Expected 1 entry before cleanup")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	cache.cleanup()

	// Verify it's gone
	if cache.GetEntryCount() != 0 {
		t.Error("Expected 0 entries after cleanup")
	}
}

func TestFactsCache_GetAllHosts(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	expectedHosts := []string{"host1", "host2", "host3"}

	for _, hostname := range expectedHosts {
		facts := &SystemFacts{
			OSFamily: "Debian",
			Hostname: hostname,
		}
		cache.Set(hostname, facts)
	}

	allHosts := cache.GetAllHosts()

	if len(allHosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(allHosts))
	}

	// Verify all expected hosts are present
	hostMap := make(map[string]bool)
	for _, host := range allHosts {
		hostMap[host] = true
	}

	for _, expected := range expectedHosts {
		if !hostMap[expected] {
			t.Errorf("Expected to find host %s in results", expected)
		}
	}
}

func TestFactsCache_TTL(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	if cache.GetTTL() != 5*time.Minute {
		t.Errorf("Expected TTL to be 5 minutes, got %v", cache.GetTTL())
	}

	// Update TTL
	cache.SetTTL(10 * time.Minute)

	if cache.GetTTL() != 10*time.Minute {
		t.Errorf("Expected TTL to be 10 minutes after update, got %v", cache.GetTTL())
	}
}

func TestGetGlobalFactsCache(t *testing.T) {
	cache1 := GetGlobalFactsCache()
	cache2 := GetGlobalFactsCache()

	if cache1 != cache2 {
		t.Error("Expected GetGlobalFactsCache to return the same instance")
	}

	// Test that it works
	facts := &SystemFacts{
		OSFamily: "Debian",
		Hostname: "test-host",
	}

	cache1.Set("test-host", facts)

	retrieved, found := cache2.Get("test-host")
	if !found {
		t.Error("Expected to find facts in global cache")
	}

	if retrieved.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", retrieved.Hostname)
	}
}

func TestFactsCache_ConcurrentAccess(t *testing.T) {
	cache := NewFactsCache(5 * time.Minute)

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			facts := &SystemFacts{
				OSFamily: "Debian",
				Hostname: "test-host",
			}
			cache.Set("test-host", facts)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("test-host")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we get here without panic, concurrent access is safe
}

func TestFactsCache_DefaultTTL(t *testing.T) {
	cache := NewFactsCache(0) // Should use default

	if cache.GetTTL() != 10*time.Minute {
		t.Errorf("Expected default TTL to be 10 minutes, got %v", cache.GetTTL())
	}
}
