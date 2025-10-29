package folderstructure

import (
	"os"
	"sync"
	"time"
)

// ProjectStructure represents the detected Ansible-style folder structure
type ProjectStructure struct {
	// RootPath is the root directory of the project
	RootPath string

	// HasDefaults indicates if defaults/ directory exists
	HasDefaults bool

	// HasVars indicates if vars/ directory exists
	HasVars bool

	// HasTemplates indicates if templates/ directory exists
	HasTemplates bool

	// HasFiles indicates if files/ directory exists
	HasFiles bool

	// HasHandlers indicates if handlers/ directory exists
	HasHandlers bool

	// HasTasks indicates if tasks/ directory exists
	HasTasks bool

	// DetectedAt is when the structure was detected
	DetectedAt time.Time

	// Metadata contains directory metadata for quick access
	Metadata map[string]os.FileInfo
}

// ResolutionResult represents the result of a path resolution
type ResolutionResult struct {
	// Path is the resolved path
	Path string

	// Found indicates if the path was found
	Found bool

	// Source indicates where the path was resolved from (e.g., "defaults", "vars", "templates", "files")
	Source string

	// Error is any error that occurred during resolution
	Error error
}

// VariableSet represents a set of resolved variables
type VariableSet struct {
	// Variables is the map of resolved variables
	Variables map[string]interface{}

	// Precedence contains the sources in order of precedence
	Precedence []string

	// Metadata contains information about variable sources
	Metadata map[string]VariableSource
}

// VariableSource represents information about a variable source
type VariableSource struct {
	// Source is the source of the variable (e.g., "defaults", "vars", "playbook")
	Source string

	// File is the file the variable came from
	File string

	// Priority is the priority of this source (higher = more important)
	Priority int
}

// Cache represents a caching layer with TTL
type Cache struct {
	mu       sync.RWMutex
	data     map[string]cacheEntry
	maxSize  int
	ttl      time.Duration
	created  time.Time
	accessed time.Time
}

type cacheEntry struct {
	value     interface{}
	timestamp time.Time
}

// NewCache creates a new cache with the given TTL and max size
func NewCache(ttl time.Duration, maxSize int) *Cache {
	now := time.Now()
	return &Cache{
		data:     make(map[string]cacheEntry),
		ttl:      ttl,
		maxSize:  maxSize,
		created:  now,
		accessed: now,
	}
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.data) >= c.maxSize {
		// Simple LRU: remove the first entry
		for k := range c.data {
			delete(c.data, k)
			break
		}
	}

	c.data[key] = cacheEntry{
		value:     value,
		timestamp: time.Now(),
	}
	c.accessed = time.Now()
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.value, true
}

// Clear clears the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]cacheEntry)
	c.accessed = time.Now()
}

// IsExpired checks if the cache is expired
func (c *Cache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.created) > c.ttl
}
