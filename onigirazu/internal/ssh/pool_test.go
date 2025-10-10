package ssh

import (
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
	pool.CloseAll()
}

func TestConnectionPoolGetStats(t *testing.T) {
	pool := NewConnectionPool(DefaultPoolConfig())
	defer pool.CloseAll()

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
	defer pool.CloseAll()

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
	defer pool.CloseAll()

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
	defer pool.CloseAll()

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
	defer pool.CloseAll()

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
	defer pool.CloseAll()

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
	customPool.CloseAll()
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
