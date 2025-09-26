package cache

import (
	"context"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Manager implements a thread-safe in-memory cache with TTL support
type Manager struct {
	entries map[string]*types.CacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration

	// Cleanup goroutine control
	stopCleanup chan struct{}
	cleanupDone chan struct{}

	// Performance metrics
	hits        int64
	misses      int64
	evictions   int64
	maxSize     int

	// LRU tracking
	accessOrder []string
	accessMutex sync.Mutex
}

// NewManager creates a new cache manager
func NewManager(defaultTTL time.Duration) *Manager {
	return NewManagerWithSize(defaultTTL, 1000) // Default max size of 1000 entries
}

// NewManagerWithSize creates a new cache manager with specified max size
func NewManagerWithSize(defaultTTL time.Duration, maxSize int) *Manager {
	m := &Manager{
		entries:     make(map[string]*types.CacheEntry),
		ttl:         defaultTTL,
		maxSize:     maxSize,
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
		accessOrder: make([]string, 0),
	}

	// Start cleanup goroutine
	go m.cleanupExpired()

	return m
}

// Get retrieves a value from the cache
func (m *Manager) Get(ctx context.Context, key string) (interface{}, bool) {
	m.mutex.RLock()
	entry, exists := m.entries[key]
	m.mutex.RUnlock()

	if !exists {
		m.recordMiss()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		m.mutex.Lock()
		delete(m.entries, key)
		m.mutex.Unlock()
		m.recordMiss()
		return nil, false
	}

	// Update access order for LRU
	m.updateAccessOrder(key)
	m.recordHit()

	return entry.Value, true
}

// Set stores a value in the cache with default TTL
func (m *Manager) Set(ctx context.Context, key string, value interface{}) error {
	return m.SetWithTTL(ctx, key, value, m.ttl)
}

// SetWithTTL stores a value in the cache with custom TTL
func (m *Manager) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if we need to evict entries to make room
	if len(m.entries) >= m.maxSize {
		m.evictLRU()
	}

	now := time.Now()
	entry := &types.CacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		TTL:       ttl,
	}

	m.entries[key] = entry
	m.updateAccessOrder(key)
	return nil
}

// Delete removes a value from the cache
func (m *Manager) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.entries, key)
	return nil
}

// Clear removes all entries from the cache
func (m *Manager) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.entries = make(map[string]*types.CacheEntry)
	return nil
}

// Size returns the number of entries in the cache
func (m *Manager) Size() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.entries)
}

// Keys returns all keys in the cache
func (m *Manager) Keys() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	keys := make([]string, 0, len(m.entries))
	for key := range m.entries {
		keys = append(keys, key)
	}

	return keys
}

// Stats returns cache statistics
func (m *Manager) Stats() CacheStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now()
	expired := 0

	for _, entry := range m.entries {
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	hitRate := float64(0)
	if m.hits+m.misses > 0 {
		hitRate = float64(m.hits) / float64(m.hits+m.misses) * 100
	}

	return CacheStats{
		TotalEntries:   len(m.entries),
		ExpiredEntries: expired,
		ActiveEntries:  len(m.entries) - expired,
		Hits:           m.hits,
		Misses:         m.misses,
		Evictions:      m.evictions,
		HitRate:        hitRate,
		MaxSize:        m.maxSize,
	}
}

// CacheStats holds cache statistics
type CacheStats struct {
	TotalEntries   int     `json:"total_entries"`
	ExpiredEntries int     `json:"expired_entries"`
	ActiveEntries  int     `json:"active_entries"`
	Hits           int64   `json:"hits"`
	Misses         int64   `json:"misses"`
	Evictions      int64   `json:"evictions"`
	HitRate        float64 `json:"hit_rate"`
	MaxSize        int     `json:"max_size"`
}

// cleanupExpired runs periodically to remove expired entries
func (m *Manager) cleanupExpired() {
	ticker := time.NewTicker(time.Minute) // Cleanup every minute
	defer ticker.Stop()
	defer close(m.cleanupDone)

	for {
		select {
		case <-ticker.C:
			m.removeExpiredEntries()
		case <-m.stopCleanup:
			return
		}
	}
}

// removeExpiredEntries removes all expired entries
func (m *Manager) removeExpiredEntries() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for key, entry := range m.entries {
		if now.After(entry.ExpiresAt) {
			delete(m.entries, key)
		}
	}
}

// Close stops the cleanup goroutine and clears the cache
func (m *Manager) Close() error {
	close(m.stopCleanup)
	<-m.cleanupDone

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.entries = make(map[string]*types.CacheEntry)

	return nil
}

// GetOrSet retrieves a value from cache or sets it if not found
func (m *Manager) GetOrSet(ctx context.Context, key string, factory func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := m.Get(ctx, key); found {
		return value, nil
	}

	// Not found, create new value
	value, err := factory()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := m.Set(ctx, key, value); err != nil {
		return value, err // Return value even if caching failed
	}

	return value, nil
}

// GetOrSetWithTTL retrieves a value from cache or sets it with custom TTL if not found
func (m *Manager) GetOrSetWithTTL(ctx context.Context, key string, ttl time.Duration, factory func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := m.Get(ctx, key); found {
		return value, nil
	}

	// Not found, create new value
	value, err := factory()
	if err != nil {
		return nil, err
	}

	// Store in cache with custom TTL
	if err := m.SetWithTTL(ctx, key, value, ttl); err != nil {
		return value, err // Return value even if caching failed
	}

	return value, nil
}

// recordHit increments the hit counter
func (m *Manager) recordHit() {
	m.accessMutex.Lock()
	defer m.accessMutex.Unlock()
	m.hits++
}

// recordMiss increments the miss counter
func (m *Manager) recordMiss() {
	m.accessMutex.Lock()
	defer m.accessMutex.Unlock()
	m.misses++
}

// updateAccessOrder updates the LRU access order
func (m *Manager) updateAccessOrder(key string) {
	m.accessMutex.Lock()
	defer m.accessMutex.Unlock()

	// Remove key from current position
	for i, k := range m.accessOrder {
		if k == key {
			m.accessOrder = append(m.accessOrder[:i], m.accessOrder[i+1:]...)
			break
		}
	}

	// Add key to the end (most recently used)
	m.accessOrder = append(m.accessOrder, key)
}

// evictLRU evicts the least recently used entry
func (m *Manager) evictLRU() {
	if len(m.accessOrder) == 0 {
		return
	}

	// Get the least recently used key
	lruKey := m.accessOrder[0]

	// Remove from entries
	delete(m.entries, lruKey)

	// Remove from access order
	m.accessOrder = m.accessOrder[1:]

	// Update eviction counter
	m.evictions++
}

// Exists checks if a key exists in the cache (without updating access order)
func (m *Manager) Exists(ctx context.Context, key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return false
	}

	// Check if expired
	return !time.Now().After(entry.ExpiresAt)
}

// GetTTL returns the remaining TTL for a key
func (m *Manager) GetTTL(ctx context.Context, key string) (time.Duration, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.entries[key]
	if !exists {
		return 0, false
	}

	now := time.Now()
	if now.After(entry.ExpiresAt) {
		return 0, false
	}

	return entry.ExpiresAt.Sub(now), true
}

// Extend extends the TTL of an existing entry
func (m *Manager) Extend(ctx context.Context, key string, additionalTTL time.Duration) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	entry, exists := m.entries[key]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(m.entries, key)
		return false
	}

	// Extend the expiration time
	entry.ExpiresAt = entry.ExpiresAt.Add(additionalTTL)
	return true
}

// GetMultiple retrieves multiple values from the cache
func (m *Manager) GetMultiple(ctx context.Context, keys []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, key := range keys {
		if value, found := m.Get(ctx, key); found {
			result[key] = value
		}
	}

	return result
}

// SetMultiple stores multiple key-value pairs in the cache
func (m *Manager) SetMultiple(ctx context.Context, items map[string]interface{}) error {
	for key, value := range items {
		if err := m.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMultiple removes multiple keys from the cache
func (m *Manager) DeleteMultiple(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := m.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}
