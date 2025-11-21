package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	ttl := 5 * time.Minute
	m := NewManager(ttl)
	defer m.Close()

	if m == nil {
		t.Fatal("Expected manager to be created")
	}

	if m.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, m.ttl)
	}

	if m.maxSize != 1000 {
		t.Errorf("Expected default maxSize 1000, got %d", m.maxSize)
	}

	if m.Size() != 0 {
		t.Errorf("Expected empty cache, got size %d", m.Size())
	}
}

// TestNewManagerWithSize tests manager creation with custom size
func TestNewManagerWithSize(t *testing.T) {
	ttl := 5 * time.Minute
	maxSize := 100
	m := NewManagerWithSize(ttl, maxSize)
	defer m.Close()

	if m.maxSize != maxSize {
		t.Errorf("Expected maxSize %d, got %d", maxSize, m.maxSize)
	}
}

// TestManager_SetAndGet tests basic set and get operations
func TestManager_SetAndGet(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value
	err := m.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Get value
	retrieved, found := m.Get(ctx, key)
	if !found {
		t.Fatal("Expected to find value")
	}

	if retrieved != value {
		t.Errorf("Expected value %v, got %v", value, retrieved)
	}
}

// TestManager_GetNonExistent tests getting non-existent key
func TestManager_GetNonExistent(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	_, found := m.Get(ctx, "non-existent")
	if found {
		t.Error("Expected not to find non-existent key")
	}
}

// TestManager_SetWithTTL tests setting value with custom TTL
func TestManager_SetWithTTL(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"
	ttl := 100 * time.Millisecond

	// Set value with short TTL
	err := m.SetWithTTL(ctx, key, value, ttl)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Should be found immediately
	_, found := m.Get(ctx, key)
	if !found {
		t.Error("Expected to find value immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = m.Get(ctx, key)
	if found {
		t.Error("Expected value to be expired")
	}
}

// TestManager_Delete tests deleting entries
func TestManager_Delete(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set and verify
	_ = m.Set(ctx, key, value)
	if _, found := m.Get(ctx, key); !found {
		t.Fatal("Expected to find value before delete")
	}

	// Delete
	err := m.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deleted
	if _, found := m.Get(ctx, key); found {
		t.Error("Expected value to be deleted")
	}
}

// TestManager_Clear tests clearing all entries
func TestManager_Clear(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 10; i++ {
		_ = m.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	if m.Size() != 10 {
		t.Errorf("Expected size 10, got %d", m.Size())
	}

	// Clear
	err := m.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	if m.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", m.Size())
	}
}

// TestManager_Size tests size tracking
func TestManager_Size(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Initially empty
	if m.Size() != 0 {
		t.Errorf("Expected size 0, got %d", m.Size())
	}

	// Add entries
	for i := 0; i < 5; i++ {
		_ = m.Set(ctx, fmt.Sprintf("key-%d", i), i)
	}

	if m.Size() != 5 {
		t.Errorf("Expected size 5, got %d", m.Size())
	}

	// Delete one
	_ = m.Delete(ctx, "key-0")

	if m.Size() != 4 {
		t.Errorf("Expected size 4 after delete, got %d", m.Size())
	}
}

// TestManager_Keys tests retrieving all keys
func TestManager_Keys(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Add entries
	expected := map[string]bool{
		"key-1": true,
		"key-2": true,
		"key-3": true,
	}

	for key := range expected {
		_ = m.Set(ctx, key, "value")
	}

	// Get keys
	keys := m.Keys()

	if len(keys) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(keys))
	}

	for _, key := range keys {
		if !expected[key] {
			t.Errorf("Unexpected key: %s", key)
		}
	}
}

// TestManager_Stats tests statistics tracking
func TestManager_Stats(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Add entries
	_ = m.Set(ctx, "key-1", "value-1")
	_ = m.Set(ctx, "key-2", "value-2")

	// Generate hits
	m.Get(ctx, "key-1")
	m.Get(ctx, "key-1")

	// Generate misses
	m.Get(ctx, "non-existent")

	stats := m.Stats()

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", stats.TotalEntries)
	}

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := float64(2) / float64(3) * 100
	if stats.HitRate < expectedHitRate-0.1 || stats.HitRate > expectedHitRate+0.1 {
		t.Errorf("Expected hit rate ~%.2f%%, got %.2f%%", expectedHitRate, stats.HitRate)
	}
}

// TestManager_GetOrSet tests get-or-set functionality
func TestManager_GetOrSet(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	expectedValue := "computed-value"

	callCount := 0
	factory := func() (interface{}, error) {
		callCount++
		return expectedValue, nil
	}

	// First call should invoke factory
	value, err := m.GetOrSet(ctx, key, factory)
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if value != expectedValue {
		t.Errorf("Expected value %v, got %v", expectedValue, value)
	}

	if callCount != 1 {
		t.Errorf("Expected factory to be called once, got %d", callCount)
	}

	// Second call should use cached value
	value, err = m.GetOrSet(ctx, key, factory)
	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if value != expectedValue {
		t.Errorf("Expected cached value %v, got %v", expectedValue, value)
	}

	if callCount != 1 {
		t.Errorf("Expected factory not to be called again, got %d calls", callCount)
	}
}

// TestManager_GetOrSetWithError tests get-or-set with factory error
func TestManager_GetOrSetWithError(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	expectedError := fmt.Errorf("factory error")

	factory := func() (interface{}, error) {
		return nil, expectedError
	}

	_, err := m.GetOrSet(ctx, key, factory)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}

// TestManager_GetOrSetWithTTL tests get-or-set with custom TTL
func TestManager_GetOrSetWithTTL(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"
	ttl := 100 * time.Millisecond

	factory := func() (interface{}, error) {
		return value, nil
	}

	// Set with short TTL
	_, err := m.GetOrSetWithTTL(ctx, key, ttl, factory)
	if err != nil {
		t.Fatalf("GetOrSetWithTTL failed: %v", err)
	}

	// Should be found immediately
	if _, found := m.Get(ctx, key); !found {
		t.Error("Expected to find value immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	if _, found := m.Get(ctx, key); found {
		t.Error("Expected value to be expired")
	}
}

// TestManager_Exists tests existence checking
func TestManager_Exists(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// Should not exist initially
	if m.Exists(ctx, key) {
		t.Error("Expected key not to exist")
	}

	// Set value
	_ = m.Set(ctx, key, "value")

	// Should exist now
	if !m.Exists(ctx, key) {
		t.Error("Expected key to exist")
	}

	// Delete
	_ = m.Delete(ctx, key)

	// Should not exist after delete
	if m.Exists(ctx, key) {
		t.Error("Expected key not to exist after delete")
	}
}

// TestManager_GetTTL tests TTL retrieval
func TestManager_GetTTL(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	ttl := 1 * time.Second

	// Non-existent key
	_, found := m.GetTTL(ctx, key)
	if found {
		t.Error("Expected TTL not found for non-existent key")
	}

	// Set with TTL
	_ = m.SetWithTTL(ctx, key, "value", ttl)

	// Get TTL
	remainingTTL, found := m.GetTTL(ctx, key)
	if !found {
		t.Fatal("Expected to find TTL")
	}

	if remainingTTL > ttl || remainingTTL < ttl-100*time.Millisecond {
		t.Errorf("Expected TTL around %v, got %v", ttl, remainingTTL)
	}
}

// TestManager_Extend tests TTL extension
func TestManager_Extend(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	key := "test-key"
	initialTTL := 200 * time.Millisecond
	extension := 500 * time.Millisecond

	// Set with short TTL
	_ = m.SetWithTTL(ctx, key, "value", initialTTL)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Extend TTL
	extended := m.Extend(ctx, key, extension)
	if !extended {
		t.Fatal("Expected TTL to be extended")
	}

	// Wait for original TTL to pass
	time.Sleep(150 * time.Millisecond)

	// Should still exist due to extension
	if !m.Exists(ctx, key) {
		t.Error("Expected key to still exist after extension")
	}
}

// TestManager_ExtendNonExistent tests extending non-existent key
func TestManager_ExtendNonExistent(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	extended := m.Extend(ctx, "non-existent", 1*time.Second)
	if extended {
		t.Error("Expected extension to fail for non-existent key")
	}
}

// TestManager_GetMultiple tests batch retrieval
func TestManager_GetMultiple(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Set multiple values
	_ = m.Set(ctx, "key-1", "value-1")
	_ = m.Set(ctx, "key-2", "value-2")
	_ = m.Set(ctx, "key-3", "value-3")

	// Get multiple
	keys := []string{"key-1", "key-2", "non-existent"}
	result := m.GetMultiple(ctx, keys)

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}

	if result["key-1"] != "value-1" {
		t.Errorf("Expected value-1, got %v", result["key-1"])
	}

	if result["key-2"] != "value-2" {
		t.Errorf("Expected value-2, got %v", result["key-2"])
	}

	if _, exists := result["non-existent"]; exists {
		t.Error("Expected non-existent key not to be in result")
	}
}

// TestManager_SetMultiple tests batch storage
func TestManager_SetMultiple(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	items := map[string]interface{}{
		"key-1": "value-1",
		"key-2": "value-2",
		"key-3": "value-3",
	}

	err := m.SetMultiple(ctx, items)
	if err != nil {
		t.Fatalf("SetMultiple failed: %v", err)
	}

	// Verify all items were set
	for key, expectedValue := range items {
		value, found := m.Get(ctx, key)
		if !found {
			t.Errorf("Expected to find key %s", key)
		}
		if value != expectedValue {
			t.Errorf("Expected value %v for key %s, got %v", expectedValue, key, value)
		}
	}
}

// TestManager_LRUEviction tests LRU eviction
func TestManager_LRUEviction(t *testing.T) {
	maxSize := 3
	m := NewManagerWithSize(5*time.Minute, maxSize)
	defer m.Close()

	ctx := context.Background()

	// Fill cache to max
	_ = m.Set(ctx, "key-1", "value-1")
	_ = m.Set(ctx, "key-2", "value-2")
	_ = m.Set(ctx, "key-3", "value-3")

	if m.Size() != maxSize {
		t.Errorf("Expected size %d, got %d", maxSize, m.Size())
	}

	// Add one more - should evict key-1 (least recently used)
	_ = m.Set(ctx, "key-4", "value-4")

	if m.Size() != maxSize {
		t.Errorf("Expected size to remain %d, got %d", maxSize, m.Size())
	}

	// key-1 should be evicted
	if _, found := m.Get(ctx, "key-1"); found {
		t.Error("Expected key-1 to be evicted")
	}

	// Others should still exist
	if _, found := m.Get(ctx, "key-2"); !found {
		t.Error("Expected key-2 to exist")
	}
	if _, found := m.Get(ctx, "key-3"); !found {
		t.Error("Expected key-3 to exist")
	}
	if _, found := m.Get(ctx, "key-4"); !found {
		t.Error("Expected key-4 to exist")
	}

	// Check eviction counter
	stats := m.Stats()
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
}

// TestManager_LRUAccessOrder tests LRU access order tracking
func TestManager_LRUAccessOrder(t *testing.T) {
	maxSize := 3
	m := NewManagerWithSize(5*time.Minute, maxSize)
	defer m.Close()

	ctx := context.Background()

	// Fill cache
	_ = m.Set(ctx, "key-1", "value-1")
	_ = m.Set(ctx, "key-2", "value-2")
	_ = m.Set(ctx, "key-3", "value-3")

	// Access key-1 to make it most recently used
	m.Get(ctx, "key-1")

	// Add new key - should evict key-2 (now least recently used)
	_ = m.Set(ctx, "key-4", "value-4")

	// key-2 should be evicted
	if _, found := m.Get(ctx, "key-2"); found {
		t.Error("Expected key-2 to be evicted")
	}

	// key-1 should still exist (was accessed)
	if _, found := m.Get(ctx, "key-1"); !found {
		t.Error("Expected key-1 to exist (was recently accessed)")
	}
}

// TestManager_ConcurrentAccess tests thread safety
func TestManager_ConcurrentAccess(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_ = m.Set(ctx, key, j)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				m.Get(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional
	_ = m.Set(ctx, "test", "value")
	if _, found := m.Get(ctx, "test"); !found {
		t.Error("Cache not functional after concurrent access")
	}
}

// TestManager_CleanupExpired tests automatic cleanup of expired entries
func TestManager_CleanupExpired(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Set entries with short TTL
	_ = m.SetWithTTL(ctx, "key-1", "value-1", 50*time.Millisecond)
	_ = m.SetWithTTL(ctx, "key-2", "value-2", 50*time.Millisecond)
	_ = m.SetWithTTL(ctx, "key-3", "value-3", 5*time.Minute) // Long TTL

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	m.removeExpiredEntries()

	// Expired entries should be removed
	if _, found := m.Get(ctx, "key-1"); found {
		t.Error("Expected key-1 to be cleaned up")
	}
	if _, found := m.Get(ctx, "key-2"); found {
		t.Error("Expected key-2 to be cleaned up")
	}

	// Non-expired entry should remain
	if _, found := m.Get(ctx, "key-3"); !found {
		t.Error("Expected key-3 to remain")
	}
}

// TestManager_Close tests graceful shutdown
func TestManager_Close(t *testing.T) {
	m := NewManager(5 * time.Minute)

	ctx := context.Background()
	_ = m.Set(ctx, "key", "value")

	// Close should not panic
	err := m.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Cache should be empty after close
	if m.Size() != 0 {
		t.Errorf("Expected cache to be empty after close, got size %d", m.Size())
	}
}

// TestManager_StatsWithExpired tests stats with expired entries
func TestManager_StatsWithExpired(t *testing.T) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Add entries with different TTLs
	_ = m.SetWithTTL(ctx, "key-1", "value-1", 50*time.Millisecond)
	_ = m.SetWithTTL(ctx, "key-2", "value-2", 5*time.Minute)

	// Wait for one to expire
	time.Sleep(100 * time.Millisecond)

	stats := m.Stats()

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", stats.TotalEntries)
	}

	if stats.ExpiredEntries != 1 {
		t.Errorf("Expected 1 expired entry, got %d", stats.ExpiredEntries)
	}

	if stats.ActiveEntries != 1 {
		t.Errorf("Expected 1 active entry, got %d", stats.ActiveEntries)
	}
}

// Benchmark tests

// BenchmarkManager_Set benchmarks set operations
func BenchmarkManager_Set(b *testing.B) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		_ = m.Set(ctx, key, i)
	}
}

// BenchmarkManager_Get benchmarks get operations
func BenchmarkManager_Get(b *testing.B) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		_ = m.Set(ctx, key, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		m.Get(ctx, key)
	}
}

// BenchmarkManager_GetOrSet benchmarks get-or-set operations
func BenchmarkManager_GetOrSet(b *testing.B) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()
	var counter int64

	factory := func() (interface{}, error) {
		return atomic.AddInt64(&counter, 1), nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100) // Reuse keys to test caching
		m.GetOrSet(ctx, key, factory)
	}
}

// BenchmarkManager_ConcurrentAccess benchmarks concurrent access
func BenchmarkManager_ConcurrentAccess(b *testing.B) {
	m := NewManager(5 * time.Minute)
	defer m.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		_ = m.Set(ctx, fmt.Sprintf("key-%d", i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%100)
			if i%2 == 0 {
				m.Get(ctx, key)
			} else {
				m.Set(ctx, key, i)
			}
			i++
		}
	})
}
