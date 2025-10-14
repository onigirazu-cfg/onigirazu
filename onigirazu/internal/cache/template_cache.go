package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"text/template"
	"time"
)

// TemplateCache provides caching for parsed templates
type TemplateCache struct {
	cache       map[string]*CachedTemplate
	mutex       sync.RWMutex
	ttl         time.Duration
	maxSize     int
	accessOrder []string
	accessMutex sync.Mutex

	// Performance metrics
	hits      int64
	misses    int64
	evictions int64

	// Cleanup goroutine control
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// CachedTemplate represents a cached parsed template
type CachedTemplate struct {
	Template  *template.Template
	Hash      string
	CreatedAt time.Time
	ExpiresAt time.Time
	AccessAt  time.Time
}

// NewTemplateCache creates a new template cache
func NewTemplateCache(ttl time.Duration, maxSize int) *TemplateCache {
	tc := &TemplateCache{
		cache:       make(map[string]*CachedTemplate),
		ttl:         ttl,
		maxSize:     maxSize,
		accessOrder: make([]string, 0),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup goroutine
	go tc.cleanupExpired()

	return tc
}

// Get retrieves a cached template by its hash
func (tc *TemplateCache) Get(ctx context.Context, templateStr string) (*template.Template, bool) {
	hash := tc.hashTemplate(templateStr)

	tc.mutex.RLock()
	cached, exists := tc.cache[hash]
	tc.mutex.RUnlock()

	if !exists {
		tc.recordMiss()
		return nil, false
	}

	// Check if expired
	now := time.Now()
	if now.After(cached.ExpiresAt) {
		tc.mutex.Lock()
		delete(tc.cache, hash)
		tc.mutex.Unlock()
		tc.recordMiss()
		return nil, false
	}

	// Update access time and order
	tc.mutex.Lock()
	cached.AccessAt = now
	tc.mutex.Unlock()
	tc.updateAccessOrder(hash)
	tc.recordHit()

	return cached.Template, true
}

// Set stores a parsed template in the cache
func (tc *TemplateCache) Set(ctx context.Context, templateStr string, tmpl *template.Template) error {
	hash := tc.hashTemplate(templateStr)

	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Check if we need to evict entries to make room
	if len(tc.cache) >= tc.maxSize {
		tc.evictLRU()
	}

	now := time.Now()
	cached := &CachedTemplate{
		Template:  tmpl,
		Hash:      hash,
		CreatedAt: now,
		ExpiresAt: now.Add(tc.ttl),
		AccessAt:  now,
	}

	tc.cache[hash] = cached
	tc.updateAccessOrder(hash)

	return nil
}

// GetOrParse retrieves a template from cache or parses it if not found
func (tc *TemplateCache) GetOrParse(ctx context.Context, templateStr string, funcMap template.FuncMap) (*template.Template, error) {
	// Try to get from cache first
	if tmpl, found := tc.Get(ctx, templateStr); found {
		return tmpl, nil
	}

	// Not found, parse the template
	tmpl, err := template.New("template").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Store in cache
	if err := tc.Set(ctx, templateStr, tmpl); err != nil {
		// Log error but return the template anyway
		return tmpl, nil
	}

	return tmpl, nil
}

// Clear removes all entries from the cache
func (tc *TemplateCache) Clear(ctx context.Context) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.cache = make(map[string]*CachedTemplate)
	tc.accessOrder = make([]string, 0)

	return nil
}

// Size returns the number of entries in the cache
func (tc *TemplateCache) Size() int {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	return len(tc.cache)
}

// Stats returns cache statistics
func (tc *TemplateCache) Stats() TemplateCacheStats {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	now := time.Now()
	expired := 0

	for _, cached := range tc.cache {
		if now.After(cached.ExpiresAt) {
			expired++
		}
	}

	hitRate := float64(0)
	if tc.hits+tc.misses > 0 {
		hitRate = float64(tc.hits) / float64(tc.hits+tc.misses) * 100
	}

	return TemplateCacheStats{
		TotalEntries:   len(tc.cache),
		ExpiredEntries: expired,
		ActiveEntries:  len(tc.cache) - expired,
		Hits:           tc.hits,
		Misses:         tc.misses,
		Evictions:      tc.evictions,
		HitRate:        hitRate,
		MaxSize:        tc.maxSize,
	}
}

// TemplateCacheStats holds template cache statistics
type TemplateCacheStats struct {
	TotalEntries   int     `json:"total_entries"`
	ExpiredEntries int     `json:"expired_entries"`
	ActiveEntries  int     `json:"active_entries"`
	Hits           int64   `json:"hits"`
	Misses         int64   `json:"misses"`
	Evictions      int64   `json:"evictions"`
	HitRate        float64 `json:"hit_rate"`
	MaxSize        int     `json:"max_size"`
}

// Close stops the cleanup goroutine and clears the cache
func (tc *TemplateCache) Close() error {
	close(tc.stopCleanup)
	<-tc.cleanupDone

	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.cache = make(map[string]*CachedTemplate)

	return nil
}

// hashTemplate creates a hash of the template string for cache key
func (tc *TemplateCache) hashTemplate(templateStr string) string {
	hash := sha256.Sum256([]byte(templateStr))
	return fmt.Sprintf("%x", hash)
}

// cleanupExpired runs periodically to remove expired entries
func (tc *TemplateCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()
	defer close(tc.cleanupDone)

	for {
		select {
		case <-ticker.C:
			tc.removeExpiredEntries()
		case <-tc.stopCleanup:
			return
		}
	}
}

// removeExpiredEntries removes all expired entries
func (tc *TemplateCache) removeExpiredEntries() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	now := time.Now()
	for hash, cached := range tc.cache {
		if now.After(cached.ExpiresAt) {
			delete(tc.cache, hash)
		}
	}
}

// recordHit increments the hit counter
func (tc *TemplateCache) recordHit() {
	tc.accessMutex.Lock()
	defer tc.accessMutex.Unlock()
	tc.hits++
}

// recordMiss increments the miss counter
func (tc *TemplateCache) recordMiss() {
	tc.accessMutex.Lock()
	defer tc.accessMutex.Unlock()
	tc.misses++
}

// updateAccessOrder updates the LRU access order
func (tc *TemplateCache) updateAccessOrder(hash string) {
	tc.accessMutex.Lock()
	defer tc.accessMutex.Unlock()

	// Remove hash from current position
	for i, h := range tc.accessOrder {
		if h == hash {
			tc.accessOrder = append(tc.accessOrder[:i], tc.accessOrder[i+1:]...)
			break
		}
	}

	// Add hash to the end (most recently used)
	tc.accessOrder = append(tc.accessOrder, hash)
}

// evictLRU evicts the least recently used entry
func (tc *TemplateCache) evictLRU() {
	if len(tc.accessOrder) == 0 {
		return
	}

	// Get the least recently used hash
	lruHash := tc.accessOrder[0]

	// Remove from cache
	delete(tc.cache, lruHash)

	// Remove from access order
	tc.accessOrder = tc.accessOrder[1:]

	// Update eviction counter
	tc.evictions++
}
