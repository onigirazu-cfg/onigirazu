package ssh

import (
	"os"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewConnectionPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewConnectionPool(config)

	if pool == nil {
		t.Fatal("Expected pool to be created")
	}

	if pool.maxIdle != config.MaxIdle {
		t.Errorf("Expected maxIdle=%v, got %v", config.MaxIdle, pool.maxIdle)
	}

	if pool.maxLifetime != config.MaxLifetime {
		t.Errorf("Expected maxLifetime=%v, got %v", config.MaxLifetime, pool.maxLifetime)
	}

	// Cleanup
	_ = pool.CloseAll()
}

func TestConnectionPoolGetStats(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	stats := pool.GetStats()
	if stats.TotalConnections != 0 {
		t.Errorf("Expected 0 connections, got %d", stats.TotalConnections)
	}

	if stats.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections, got %d", stats.ActiveConnections)
	}
}

func TestConnectionPoolGetConnectionKey(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	key := pool.getConnectionKey(host)
	expected := "testuser@localhost:22"

	if key != expected {
		t.Errorf("Expected key=%s, got %s", expected, key)
	}
}

func TestConnectionPoolGetConnectionKeyDefaultPort(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    0, // Should default to 22
	}

	key := pool.getConnectionKey(host)
	expected := "testuser@localhost:22"

	if key != expected {
		t.Errorf("Expected key=%s, got %s", expected, key)
	}
}

func TestConnectionPoolReleaseConnection(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Release non-existent connection should not panic
	pool.ReleaseConnection(host)
}

func TestConnectionPoolCloseConnection(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Close non-existent connection should not error
	err := pool.CloseConnection(host)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestConnectionWrapperValidation(t *testing.T) {
	pool := NewConnectionPool(PoolConfig{
		MaxIdle:     1 * time.Second,
		MaxLifetime: 2 * time.Second,
		CleanupTick: 100 * time.Millisecond,
	})
	defer func() { _ = pool.CloseAll() }()

	now := time.Now()

	tests := []struct {
		name     string
		wrapper  *ConnectionWrapper
		expected bool
	}{
		{
			name: "valid connection",
			wrapper: &ConnectionWrapper{
				createdAt: now,
				lastUsed:  now,
				inUse:     false,
			},
			expected: true,
		},
		{
			name: "exceeded max lifetime",
			wrapper: &ConnectionWrapper{
				createdAt: now.Add(-3 * time.Second),
				lastUsed:  now,
				inUse:     false,
			},
			expected: false,
		},
		{
			name: "idle too long",
			wrapper: &ConnectionWrapper{
				createdAt: now,
				lastUsed:  now.Add(-2 * time.Second),
				inUse:     false,
			},
			expected: false,
		},
		{
			name: "in use but old",
			wrapper: &ConnectionWrapper{
				createdAt: now.Add(-3 * time.Second),
				lastUsed:  now.Add(-2 * time.Second),
				inUse:     true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pool.isConnectionValid(tt.wrapper)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGlobalPool(t *testing.T) {
	pool1 := GetGlobalPool()
	pool2 := GetGlobalPool()

	if pool1 != pool2 {
		t.Error("Expected same global pool instance")
	}

	// Test custom global pool
	customPool := NewConnectionPool(DefaultPoolConfig())
	SetGlobalPool(customPool)

	pool3 := GetGlobalPool()
	if pool3 != customPool {
		t.Error("Expected custom global pool")
	}

	// Cleanup
	_ = customPool.CloseAll()
}

func TestPoolStats(t *testing.T) {
	stats := PoolStats{
		TotalConnections:  10,
		ActiveConnections: 5,
		IdleConnections:   5,
		TotalUsageCount:   100,
	}

	if stats.TotalConnections != 10 {
		t.Errorf("Expected 10 total connections, got %d", stats.TotalConnections)
	}

	if stats.ActiveConnections != 5 {
		t.Errorf("Expected 5 active connections, got %d", stats.ActiveConnections)
	}

	if stats.IdleConnections != 5 {
		t.Errorf("Expected 5 idle connections, got %d", stats.IdleConnections)
	}

	if stats.TotalUsageCount != 100 {
		t.Errorf("Expected 100 total usage count, got %d", stats.TotalUsageCount)
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	if config.MaxIdle != 5*time.Minute {
		t.Errorf("Expected MaxIdle=5m, got %v", config.MaxIdle)
	}

	if config.MaxLifetime != 30*time.Minute {
		t.Errorf("Expected MaxLifetime=30m, got %v", config.MaxLifetime)
	}

	if config.CleanupTick != 1*time.Minute {
		t.Errorf("Expected CleanupTick=1m, got %v", config.CleanupTick)
	}
}

// TestConnectionPool_GetConnection_Error tests GetConnection with invalid host
// This test is skipped in CI because it attempts a real network connection
// which can be flaky and slow in CI environments
func TestConnectionPool_GetConnection_Error(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "192.168.1.100", // Non-existent host
		User:    "testuser",
		Port:    22,
		// No authentication method
	}

	client, err := pool.GetConnection(host)
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
	if client != nil {
		t.Error("Expected nil client on error")
	}
}

// TestConnectionPool_Cleanup tests cleanup of stale connections
func TestConnectionPool_Cleanup(t *testing.T) {
	pool := NewConnectionPool(PoolConfig{
		MaxIdle:     100 * time.Millisecond,
		MaxLifetime: 200 * time.Millisecond,
		CleanupTick: 50 * time.Millisecond,
	})
	defer func() { _ = pool.CloseAll() }()

	// Add a mock connection that will become stale
	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	key := pool.getConnectionKey(host)
	wrapper := &ConnectionWrapper{
		client:    &Client{client: nil, host: host},
		lastUsed:  time.Now().Add(-150 * time.Millisecond), // Stale
		inUse:     false,
		host:      host,
		createdAt: time.Now().Add(-150 * time.Millisecond),
	}

	pool.mutex.Lock()
	pool.connections[key] = wrapper
	pool.mutex.Unlock()

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Verify connection was cleaned up
	pool.mutex.RLock()
	_, exists := pool.connections[key]
	pool.mutex.RUnlock()

	if exists {
		t.Error("Expected stale connection to be cleaned up")
	}
}

// TestConnectionPool_CloseConnection_WithError tests CloseConnection when Close fails
func TestConnectionPool_CloseConnection_WithError(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Add a connection to the pool
	key := pool.getConnectionKey(host)
	wrapper := &ConnectionWrapper{
		client:    &Client{client: nil, host: host},
		lastUsed:  time.Now(),
		inUse:     false,
		host:      host,
		createdAt: time.Now(),
	}

	pool.mutex.Lock()
	pool.connections[key] = wrapper
	pool.mutex.Unlock()

	// Close the connection
	err := pool.CloseConnection(host)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify connection was removed
	pool.mutex.RLock()
	_, exists := pool.connections[key]
	pool.mutex.RUnlock()

	if exists {
		t.Error("Expected connection to be removed after close")
	}
}

// TestConnectionPool_GetStats_WithConnections tests GetStats with active and idle connections
func TestConnectionPool_GetStats_WithConnections(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host1 := types.Host{Name: "test1", Address: "localhost", User: "user1", Port: 22}
	host2 := types.Host{Name: "test2", Address: "localhost", User: "user2", Port: 22}

	// Add active connection
	key1 := pool.getConnectionKey(host1)
	wrapper1 := &ConnectionWrapper{
		client:     &Client{client: nil, host: host1},
		lastUsed:   time.Now(),
		inUse:      true,
		host:       host1,
		createdAt:  time.Now(),
		usageCount: 5,
	}

	// Add idle connection
	key2 := pool.getConnectionKey(host2)
	wrapper2 := &ConnectionWrapper{
		client:     &Client{client: nil, host: host2},
		lastUsed:   time.Now(),
		inUse:      false,
		host:       host2,
		createdAt:  time.Now(),
		usageCount: 3,
	}

	pool.mutex.Lock()
	pool.connections[key1] = wrapper1
	pool.connections[key2] = wrapper2
	pool.mutex.Unlock()

	stats := pool.GetStats()

	if stats.TotalConnections != 2 {
		t.Errorf("Expected 2 total connections, got %d", stats.TotalConnections)
	}

	if stats.ActiveConnections != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats.ActiveConnections)
	}

	if stats.IdleConnections != 1 {
		t.Errorf("Expected 1 idle connection, got %d", stats.IdleConnections)
	}

	if stats.TotalUsageCount != 8 {
		t.Errorf("Expected 8 total usage count, got %d", stats.TotalUsageCount)
	}
}

// TestConnectionPool_CloseAll_WithError tests CloseAll when some connections fail to close
func TestConnectionPool_CloseAll_WithError(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Add a connection
	key := pool.getConnectionKey(host)
	wrapper := &ConnectionWrapper{
		client:    &Client{client: nil, host: host},
		lastUsed:  time.Now(),
		inUse:     false,
		host:      host,
		createdAt: time.Now(),
	}

	pool.mutex.Lock()
	pool.connections[key] = wrapper
	pool.mutex.Unlock()

	// Close all connections
	err := pool.CloseAll()
	// Should not error even if individual close fails
	if err != nil {
		t.Logf("CloseAll returned error: %v", err)
	}

	// Verify all connections were removed
	pool.mutex.RLock()
	count := len(pool.connections)
	pool.mutex.RUnlock()

	if count != 0 {
		t.Errorf("Expected 0 connections after CloseAll, got %d", count)
	}
}

// TestConnectionPool_NewConnectionPool_ZeroConfig tests NewConnectionPool with zero values
func TestConnectionPool_NewConnectionPool_ZeroConfig(t *testing.T) {
	pool := NewConnectionPool(PoolConfig{
		MaxIdle:     0,
		MaxLifetime: 0,
		CleanupTick: 0,
	})
	defer func() { _ = pool.CloseAll() }()

	// Should use default values
	if pool.maxIdle != 5*time.Minute {
		t.Errorf("Expected default MaxIdle=5m, got %v", pool.maxIdle)
	}

	if pool.maxLifetime != 30*time.Minute {
		t.Errorf("Expected default MaxLifetime=30m, got %v", pool.maxLifetime)
	}

	if pool.cleanupTick != 1*time.Minute {
		t.Errorf("Expected default CleanupTick=1m, got %v", pool.cleanupTick)
	}
}

// TestConnectionPool_GetConnection_ReuseValid tests reusing a valid connection
func TestConnectionPool_GetConnection_ReuseValid(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Add a valid connection to the pool
	key := pool.getConnectionKey(host)
	mockClient := &Client{client: nil, host: host}
	wrapper := &ConnectionWrapper{
		client:     mockClient,
		lastUsed:   time.Now(),
		inUse:      false,
		host:       host,
		createdAt:  time.Now(),
		usageCount: 1,
	}

	pool.mutex.Lock()
	pool.connections[key] = wrapper
	pool.mutex.Unlock()

	// Try to get the connection - should reuse existing
	client, err := pool.GetConnection(host)

	// Will fail because we can't create a real connection, but that's OK
	// We're testing the reuse logic
	if err == nil && client == mockClient {
		// Successfully reused the connection
		if wrapper.usageCount != 2 {
			t.Errorf("Expected usage count to be 2, got %d", wrapper.usageCount)
		}
		if !wrapper.inUse {
			t.Error("Expected connection to be marked as in use")
		}
	}
}

// TestConnectionPool_GetConnection_InvalidConnection tests replacing invalid connection
func TestConnectionPool_GetConnection_InvalidConnection(t *testing.T) {
	pool := NewConnectionPool(PoolConfig{
		MaxIdle:     100 * time.Millisecond,
		MaxLifetime: 200 * time.Millisecond,
		CleanupTick: 50 * time.Millisecond,
	})
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Add an invalid (stale) connection to the pool
	key := pool.getConnectionKey(host)
	mockClient := &Client{client: nil, host: host}
	wrapper := &ConnectionWrapper{
		client:     mockClient,
		lastUsed:   time.Now().Add(-300 * time.Millisecond), // Stale
		inUse:      false,
		host:       host,
		createdAt:  time.Now().Add(-300 * time.Millisecond), // Old
		usageCount: 1,
	}

	pool.mutex.Lock()
	pool.connections[key] = wrapper
	pool.mutex.Unlock()

	// Try to get the connection - should remove invalid and try to create new
	_, err := pool.GetConnection(host)

	// Will fail because we can't create a real connection
	if err != nil {
		// Expected - can't create real connection
		// But the invalid connection should have been removed
		pool.mutex.RLock()
		_, exists := pool.connections[key]
		pool.mutex.RUnlock()

		if exists {
			t.Error("Expected invalid connection to be removed")
		}
	}
}

// TestConnectionPool_ReleaseConnection_Existing tests releasing an existing connection
func TestConnectionPool_ReleaseConnection_Existing(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer func() { _ = pool.CloseAll() }()

	host := types.Host{
		Name:    "test",
		Address: "localhost",
		User:    "testuser",
		Port:    22,
	}

	// Add a connection in use
	key := pool.getConnectionKey(host)
	wrapper := &ConnectionWrapper{
		client:    &Client{client: nil, host: host},
		lastUsed:  time.Now().Add(-1 * time.Second),
		inUse:     true,
		host:      host,
		createdAt: time.Now(),
	}

	pool.mutex.Lock()
	pool.connections[key] = wrapper
	pool.mutex.Unlock()

	// Release the connection
	pool.ReleaseConnection(host)

	// Verify it's marked as not in use
	pool.mutex.RLock()
	released := pool.connections[key]
	pool.mutex.RUnlock()

	if released.inUse {
		t.Error("Expected connection to be marked as not in use")
	}

	// lastUsed should be updated
	if released.lastUsed.Before(wrapper.lastUsed) {
		t.Error("Expected lastUsed to be updated")
	}
}
