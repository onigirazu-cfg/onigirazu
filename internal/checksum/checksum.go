package checksum

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Algorithm defines the hashing algorithm to use
type Algorithm string

const (
	SHA256 Algorithm = "sha256"
)

// ChecksumResult contains the computed checksum and metadata
type ChecksumResult struct {
	Checksum  string    `json:"checksum"`
	Algorithm Algorithm `json:"algorithm"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	Computed  time.Time `json:"computed"`
}

// CacheEntry holds cached checksum data with TTL
type CacheEntry struct {
	Result    *ChecksumResult
	ExpiresAt time.Time
}

// Manager handles checksum computation, verification, and caching
type Manager struct {
	algorithm Algorithm
	cache     map[string]*CacheEntry
	cacheTTL  time.Duration
	mu        sync.RWMutex
}

// NewManager creates a new checksum manager
func NewManager(algorithm Algorithm, cacheTTL time.Duration) *Manager {
	if algorithm == "" {
		algorithm = SHA256
	}
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute // Default 5-minute TTL
	}
	return &Manager{
		algorithm: algorithm,
		cache:     make(map[string]*CacheEntry),
		cacheTTL:  cacheTTL,
	}
}

// ComputeFile computes the checksum of a file
func (m *Manager) ComputeFile(filepath string) (*ChecksumResult, error) {
	// Check cache first
	if cached := m.getFromCache(filepath); cached != nil {
		// Verify file hasn't changed
		info, err := os.Stat(filepath)
		if err == nil && info.ModTime().Equal(cached.Result.ModTime) && info.Size() == cached.Result.Size {
			return cached.Result, nil
		}
		// Cache miss - file was modified
		m.removeFromCache(filepath)
	}

	// Get file info
	info, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}

	// Open file
	file, err := os.Open(filepath) // #nosec G304 -- caller controls filepath
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Compute checksum
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("error reading file for checksum: %w", err)
	}

	result := &ChecksumResult{
		Checksum:  fmt.Sprintf("%x", hash.Sum(nil)),
		Algorithm: m.algorithm,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		Computed:  time.Now(),
	}

	// Cache result
	m.putInCache(filepath, result)

	return result, nil
}

// ComputeData computes the checksum of data in memory
func (m *Manager) ComputeData(data []byte) (*ChecksumResult, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return nil, fmt.Errorf("error computing hash: %w", err)
	}

	return &ChecksumResult{
		Checksum:  fmt.Sprintf("%x", hash.Sum(nil)),
		Algorithm: m.algorithm,
		Size:      int64(len(data)),
		Computed:  time.Now(),
	}, nil
}

// Verify checks if a file's checksum matches the expected value
func (m *Manager) Verify(filepath string, expectedChecksum string) (bool, error) {
	result, err := m.ComputeFile(filepath)
	if err != nil {
		return false, err
	}

	return result.Checksum == expectedChecksum, nil
}

// VerifyData checks if data's checksum matches the expected value
func (m *Manager) VerifyData(data []byte, expectedChecksum string) (bool, error) {
	result, err := m.ComputeData(data)
	if err != nil {
		return false, err
	}

	return result.Checksum == expectedChecksum, nil
}

// ClearCache removes all cache entries
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]*CacheEntry)
}

// ClearCacheEntry removes a specific cache entry
func (m *Manager) ClearCacheEntry(filepath string) {
	m.removeFromCache(filepath)
}

// CacheStats returns cache statistics
type CacheStats struct {
	Entries int
	Size    int64
}

// GetCacheStats returns statistics about the cache
func (m *Manager) GetCacheStats() *CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var size int64
	for _, entry := range m.cache {
		if entry.Result != nil {
			size += entry.Result.Size
		}
	}

	return &CacheStats{
		Entries: len(m.cache),
		Size:    size,
	}
}

// getFromCache retrieves a checksum from cache if valid
func (m *Manager) getFromCache(filepath string) *CacheEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.cache[filepath]
	if !exists {
		return nil
	}

	// Check if cache entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Return nil to indicate cache miss
		return nil
	}

	return entry
}

// putInCache stores a checksum in cache
func (m *Manager) putInCache(filepath string, result *ChecksumResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache[filepath] = &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(m.cacheTTL),
	}
}

// removeFromCache removes a cache entry
func (m *Manager) removeFromCache(filepath string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, filepath)
}

// CleanupExpiredCache removes all expired cache entries
func (m *Manager) CleanupExpiredCache() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0

	for filepath, entry := range m.cache {
		if now.After(entry.ExpiresAt) {
			delete(m.cache, filepath)
			removed++
		}
	}

	return removed
}
