package ssh

import (
	"fmt"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ConnectionWrapper wraps an SSH client with metadata
type ConnectionWrapper struct {
	client     *Client
	lastUsed   time.Time
	inUse      bool
	host       types.Host
	createdAt  time.Time
	usageCount int64
}

// ConnectionPool manages a pool of SSH connections
type ConnectionPool struct {
	connections map[string]*ConnectionWrapper
	mutex       sync.RWMutex
	maxIdle     time.Duration
	maxLifetime time.Duration
	cleanupTick time.Duration
	stopCleanup chan struct{}
	hostKeyMgr  *HostKeyManager
}

// PoolConfig holds configuration for the connection pool
type PoolConfig struct {
	MaxIdle     time.Duration // Maximum idle time before closing connection
	MaxLifetime time.Duration // Maximum lifetime of a connection
	CleanupTick time.Duration // How often to run cleanup
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdle:     5 * time.Minute,
		MaxLifetime: 30 * time.Minute,
		CleanupTick: 1 * time.Minute,
	}
}

// NewConnectionPool creates a new SSH connection pool
func NewConnectionPool(config PoolConfig) *ConnectionPool {
	if config.MaxIdle == 0 {
		config.MaxIdle = 5 * time.Minute
	}
	if config.MaxLifetime == 0 {
		config.MaxLifetime = 30 * time.Minute
	}
	if config.CleanupTick == 0 {
		config.CleanupTick = 1 * time.Minute
	}

	pool := &ConnectionPool{
		connections: make(map[string]*ConnectionWrapper),
		maxIdle:     config.MaxIdle,
		maxLifetime: config.MaxLifetime,
		cleanupTick: config.CleanupTick,
		stopCleanup: make(chan struct{}),
		hostKeyMgr:  NewHostKeyManager("", false),
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// GetConnection gets or creates a connection for the given host
func (p *ConnectionPool) GetConnection(host types.Host) (*Client, error) {
	key := p.getConnectionKey(host)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if we have an existing connection
	if wrapper, exists := p.connections[key]; exists {
		// Check if connection is still valid
		if p.isConnectionValid(wrapper) {
			wrapper.lastUsed = time.Now()
			wrapper.inUse = true
			wrapper.usageCount++
			return wrapper.client, nil
		}

		// Connection is invalid, close and remove it
		_ = wrapper.client.Close() // Ignore close error, connection is already invalid
		delete(p.connections, key)
	}

	// Create new connection
	client, err := NewClientWithHostKeyManager(host, p.hostKeyMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}

	// Store in pool
	wrapper := &ConnectionWrapper{
		client:     client,
		lastUsed:   time.Now(),
		inUse:      true,
		host:       host,
		createdAt:  time.Now(),
		usageCount: 1,
	}
	p.connections[key] = wrapper

	return client, nil
}

// ReleaseConnection marks a connection as no longer in use
func (p *ConnectionPool) ReleaseConnection(host types.Host) {
	key := p.getConnectionKey(host)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if wrapper, exists := p.connections[key]; exists {
		wrapper.inUse = false
		wrapper.lastUsed = time.Now()
	}
}

// CloseConnection closes and removes a specific connection
func (p *ConnectionPool) CloseConnection(host types.Host) error {
	key := p.getConnectionKey(host)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if wrapper, exists := p.connections[key]; exists {
		err := wrapper.client.Close()
		delete(p.connections, key)
		return err
	}

	return nil
}

// CloseAll closes all connections in the pool
func (p *ConnectionPool) CloseAll() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Stop cleanup goroutine
	close(p.stopCleanup)

	var lastErr error
	for key, wrapper := range p.connections {
		if err := wrapper.client.Close(); err != nil {
			lastErr = err
		}
		delete(p.connections, key)
	}

	return lastErr
}

// GetStats returns statistics about the connection pool
func (p *ConnectionPool) GetStats() PoolStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := PoolStats{
		TotalConnections:  len(p.connections),
		ActiveConnections: 0,
		IdleConnections:   0,
	}

	for _, wrapper := range p.connections {
		if wrapper.inUse {
			stats.ActiveConnections++
		} else {
			stats.IdleConnections++
		}
		stats.TotalUsageCount += wrapper.usageCount
	}

	return stats
}

// PoolStats holds statistics about the connection pool
type PoolStats struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	TotalUsageCount   int64
}

// getConnectionKey generates a unique key for a host
func (p *ConnectionPool) getConnectionKey(host types.Host) string {
	port := host.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s@%s:%d", host.User, host.Address, port)
}

// isConnectionValid checks if a connection is still valid
func (p *ConnectionPool) isConnectionValid(wrapper *ConnectionWrapper) bool {
	now := time.Now()

	// Check if connection exceeded max lifetime
	if now.Sub(wrapper.createdAt) > p.maxLifetime {
		return false
	}

	// Check if connection is idle for too long
	if !wrapper.inUse && now.Sub(wrapper.lastUsed) > p.maxIdle {
		return false
	}

	// TODO: Add actual connection health check (ping)
	// For now, we assume the connection is valid if it passes time checks

	return true
}

// cleanupLoop periodically cleans up stale connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(p.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.stopCleanup:
			return
		}
	}
}

// cleanup removes stale connections from the pool
func (p *ConnectionPool) cleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for key, wrapper := range p.connections {
		// Skip connections that are currently in use
		if wrapper.inUse {
			continue
		}

		// Check if connection should be removed
		shouldRemove := false

		// Remove if exceeded max lifetime
		if now.Sub(wrapper.createdAt) > p.maxLifetime {
			shouldRemove = true
		}

		// Remove if idle for too long
		if now.Sub(wrapper.lastUsed) > p.maxIdle {
			shouldRemove = true
		}

		if shouldRemove {
			toRemove = append(toRemove, key)
		}
	}

	// Close and remove stale connections
	for _, key := range toRemove {
		if wrapper, exists := p.connections[key]; exists {
			_ = wrapper.client.Close() // Ignore close error during cleanup
			delete(p.connections, key)
		}
	}
}

// Global connection pool instance
var globalPool *ConnectionPool
var poolOnce sync.Once

// GetGlobalPool returns the global connection pool instance
func GetGlobalPool() *ConnectionPool {
	poolOnce.Do(func() {
		globalPool = NewConnectionPool(DefaultPoolConfig())
	})
	return globalPool
}

// SetGlobalPool sets a custom global connection pool
func SetGlobalPool(pool *ConnectionPool) {
	globalPool = pool
}
