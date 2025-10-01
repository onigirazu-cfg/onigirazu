package cache

import (
	"sync"
	"time"
)

// SystemFacts represents cached system facts for a host
type SystemFacts struct {
	// Operating System
	OSFamily      string `json:"os_family"`
	Distribution  string `json:"distribution"`
	OSVersion     string `json:"os_version"`
	Architecture  string `json:"architecture"`
	Kernel        string `json:"kernel"`
	KernelVersion string `json:"kernel_version"`

	// Hardware
	Hostname    string `json:"hostname"`
	CPUCores    int    `json:"cpu_cores"`
	MemoryTotal string `json:"memory_total"`

	// Network
	FQDN        string `json:"fqdn"`
	DefaultIPv4 string `json:"default_ipv4"`

	// Timestamps
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// FactsCacheEntry represents a cached facts entry
type FactsCacheEntry struct {
	Facts     *SystemFacts
	ExpiresAt time.Time
}

// FactsCache provides thread-safe caching for system facts
type FactsCache struct {
	mu      sync.RWMutex
	entries map[string]*FactsCacheEntry // key: hostname
	ttl     time.Duration
	enabled bool

	// Statistics
	hits   uint64
	misses uint64
}

// FactsCacheStats contains cache statistics
type FactsCacheStats struct {
	Hits    uint64
	Misses  uint64
	Entries int
	HitRate float64
	Enabled bool
}

var (
	globalFactsCache     *FactsCache
	globalFactsCacheOnce sync.Once
)

// NewFactsCache creates a new facts cache
func NewFactsCache(ttl time.Duration) *FactsCache {
	if ttl == 0 {
		ttl = 10 * time.Minute // Default: 10 minutes (facts change rarely)
	}

	fc := &FactsCache{
		entries: make(map[string]*FactsCacheEntry),
		ttl:     ttl,
		enabled: true,
	}

	// Start cleanup goroutine
	go fc.cleanupLoop()

	return fc
}

// GetGlobalFactsCache returns the global facts cache singleton
func GetGlobalFactsCache() *FactsCache {
	globalFactsCacheOnce.Do(func() {
		globalFactsCache = NewFactsCache(10 * time.Minute)
	})
	return globalFactsCache
}

// Get retrieves cached facts for a host
func (fc *FactsCache) Get(hostname string) (*SystemFacts, bool) {
	if !fc.enabled {
		return nil, false
	}

	fc.mu.RLock()
	defer fc.mu.RUnlock()

	entry, exists := fc.entries[hostname]
	if !exists {
		fc.misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		fc.misses++
		return nil, false
	}

	fc.hits++
	return entry.Facts, true
}

// Set stores facts for a host
func (fc *FactsCache) Set(hostname string, facts *SystemFacts) {
	if !fc.enabled {
		return
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(fc.ttl)

	// Update facts timestamps
	facts.CachedAt = now
	facts.ExpiresAt = expiresAt

	fc.entries[hostname] = &FactsCacheEntry{
		Facts:     facts,
		ExpiresAt: expiresAt,
	}
}

// Invalidate removes cached facts for a specific host
func (fc *FactsCache) Invalidate(hostname string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	delete(fc.entries, hostname)
}

// Clear removes all cached facts
func (fc *FactsCache) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.entries = make(map[string]*FactsCacheEntry)
	fc.hits = 0
	fc.misses = 0
}

// GetStats returns cache statistics
func (fc *FactsCache) GetStats() FactsCacheStats {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	total := fc.hits + fc.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(fc.hits) / float64(total) * 100.0
	}

	return FactsCacheStats{
		Hits:    fc.hits,
		Misses:  fc.misses,
		Entries: len(fc.entries),
		HitRate: hitRate,
		Enabled: fc.enabled,
	}
}

// Enable enables the cache
func (fc *FactsCache) Enable() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.enabled = true
}

// Disable disables the cache
func (fc *FactsCache) Disable() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.enabled = false
}

// IsEnabled returns whether the cache is enabled
func (fc *FactsCache) IsEnabled() bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.enabled
}

// cleanupLoop periodically removes expired entries
func (fc *FactsCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fc.cleanup()
	}
}

// cleanup removes expired entries
func (fc *FactsCache) cleanup() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	now := time.Now()
	for hostname, entry := range fc.entries {
		if now.After(entry.ExpiresAt) {
			delete(fc.entries, hostname)
		}
	}
}

// GetTTL returns the cache TTL
func (fc *FactsCache) GetTTL() time.Duration {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.ttl
}

// SetTTL updates the cache TTL
func (fc *FactsCache) SetTTL(ttl time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.ttl = ttl
}

// GetEntryCount returns the number of cached entries
func (fc *FactsCache) GetEntryCount() int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return len(fc.entries)
}

// GetAllHosts returns all cached hostnames
func (fc *FactsCache) GetAllHosts() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	hosts := make([]string, 0, len(fc.entries))
	for hostname := range fc.entries {
		hosts = append(hosts, hostname)
	}
	return hosts
}
