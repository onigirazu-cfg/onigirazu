package inventory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing

type mockParser struct {
	parseInventoryFunc func(ctx context.Context, filePath string) (*types.Inventory, error)
}

func (m *mockParser) ParsePlaybook(ctx context.Context, filePath string) (*types.Playbook, error) {
	return nil, nil
}

func (m *mockParser) ParseInventory(ctx context.Context, filePath string) (*types.Inventory, error) {
	if m.parseInventoryFunc != nil {
		return m.parseInventoryFunc(ctx, filePath)
	}
	return &types.Inventory{}, nil
}

func (m *mockParser) ValidatePlaybook(playbook *types.Playbook) error {
	return nil
}

func (m *mockParser) SetVariables(variables map[string]interface{}) {}

func (m *mockParser) AddVariable(key string, value interface{}) {}

type mockLogger struct {
	debugCalls int32
	infoCalls  int32
	warnCalls  int32
	errorCalls int32
}

func (m *mockLogger) Debug(format string, args ...interface{})                                { atomic.AddInt32(&m.debugCalls, 1) }
func (m *mockLogger) Info(format string, args ...interface{})                                 { atomic.AddInt32(&m.infoCalls, 1) }
func (m *mockLogger) Warn(format string, args ...interface{})                                 { atomic.AddInt32(&m.warnCalls, 1) }
func (m *mockLogger) Error(format string, args ...interface{})                                { atomic.AddInt32(&m.errorCalls, 1) }
func (m *mockLogger) Fatal(format string, args ...interface{})                                {}
func (m *mockLogger) SetLevel(level string)                                                   {}
func (m *mockLogger) TaskStart(taskName, hostName string)                                     {}
func (m *mockLogger) TaskEnd(taskName, hostName string, changed, success bool)                {}
func (m *mockLogger) PlayStart(playName string, playIndex, totalPlays int)                    {}
func (m *mockLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {}
func (m *mockLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
}
func (m *mockLogger) Progress(completed, total int, currentTask, currentHost string) {}

type mockCache struct {
	data map[string]interface{}
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]interface{}),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (interface{}, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Clear(ctx context.Context) error {
	m.data = make(map[string]interface{})
	return nil
}

func (m *mockCache) GetStats() map[string]interface{} {
	return map[string]interface{}{"size": len(m.data)}
}

func (m *mockCache) Size() int {
	return len(m.data)
}

// Test functions

func TestNewManager(t *testing.T) {
	parser := &mockParser{}
	logger := &mockLogger{}
	cache := newMockCache()

	manager := NewManager(parser, logger, cache)

	assert.NotNil(t, manager, "Manager should not be nil")
	assert.NotNil(t, manager.parser, "Parser should be set")
	assert.NotNil(t, manager.logger, "Logger should be set")
	assert.NotNil(t, manager.cache, "Cache should be set")
	assert.Empty(t, manager.hostFilters, "Host filters should be empty")
	assert.Empty(t, manager.groupFilters, "Group filters should be empty")
}

func TestManager_LoadInventory(t *testing.T) {
	parser := &mockParser{
		parseInventoryFunc: func(ctx context.Context, filePath string) (*types.Inventory, error) {
			return &types.Inventory{
				Groups: map[string]*types.Group{
					"webservers": {
						Name: "webservers",
						Hosts: map[string]*types.Host{
							"web1": {Name: "web1", Address: "192.168.1.10", Port: 22, User: "admin"},
							"web2": {Name: "web2", Address: "192.168.1.11", Port: 22, User: "admin"},
						},
						Vars: map[string]interface{}{"http_port": 80},
					},
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	cache := newMockCache()

	manager := NewManager(parser, logger, cache)
	ctx := context.Background()

	err := manager.LoadInventory(ctx, "test_inventory.yml")

	assert.NoError(t, err, "LoadInventory should not return error")
	assert.NotNil(t, manager.inventory, "Inventory should be loaded")
	assert.Equal(t, 1, len(manager.inventory.Groups), "Should have 1 group")
	assert.True(t, atomic.LoadInt32(&logger.infoCalls) > 0, "Should log info messages")
}

func TestManager_LoadInventory_WithCache(t *testing.T) {
	callCount := int32(0)
	parser := &mockParser{
		parseInventoryFunc: func(ctx context.Context, filePath string) (*types.Inventory, error) {
			atomic.AddInt32(&callCount, 1)
			return &types.Inventory{
				Groups: map[string]*types.Group{
					"test": {Name: "test", Hosts: map[string]*types.Host{}},
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	cache := newMockCache()

	manager := NewManager(parser, logger, cache)
	ctx := context.Background()

	// First load - should parse
	err := manager.LoadInventory(ctx, "test.yml")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "Should parse once")

	// Second load - should use cache
	err = manager.LoadInventory(ctx, "test.yml")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "Should not parse again (cached)")
}

func TestManager_GetHosts_All(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	hosts, err := manager.GetHosts("all")

	assert.NoError(t, err, "GetHosts should not return error")
	assert.Equal(t, 3, len(hosts), "Should return all 3 hosts")
}

func TestManager_GetHosts_Localhost(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	hosts, err := manager.GetHosts("localhost")

	assert.NoError(t, err, "GetHosts should not return error")
	assert.Equal(t, 1, len(hosts), "Should return 1 host")
	assert.Equal(t, "localhost", hosts[0].Name, "Should be localhost")
	assert.Equal(t, "127.0.0.1", hosts[0].Address, "Should have localhost address")
}

func TestManager_GetHosts_ByGroup(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	hosts, err := manager.GetHosts("webservers")

	assert.NoError(t, err, "GetHosts should not return error")
	assert.Equal(t, 2, len(hosts), "Should return 2 hosts from webservers group")
}

func TestManager_GetHosts_ByPattern(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	hosts, err := manager.GetHosts("web*")

	assert.NoError(t, err, "GetHosts should not return error")
	assert.Equal(t, 2, len(hosts), "Should return hosts matching web* pattern")
}

func TestManager_GetHosts_NoInventory(t *testing.T) {
	parser := &mockParser{}
	logger := &mockLogger{}
	cache := newMockCache()
	manager := NewManager(parser, logger, cache)

	hosts, err := manager.GetHosts("all")

	assert.Error(t, err, "Should return error when no inventory loaded")
	assert.Nil(t, hosts, "Hosts should be nil")
	assert.Contains(t, err.Error(), "no inventory loaded")
}

func TestManager_GetGroups_All(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	groups, err := manager.GetGroups("all")

	assert.NoError(t, err, "GetGroups should not return error")
	assert.Equal(t, 2, len(groups), "Should return all 2 groups")
}

func TestManager_GetGroups_ByPattern(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	groups, err := manager.GetGroups("web*")

	assert.NoError(t, err, "GetGroups should not return error")
	assert.Equal(t, 1, len(groups), "Should return 1 group matching pattern")
	assert.NotNil(t, groups["webservers"], "Should contain webservers group")
}

func TestManager_GetHostByName(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	host, err := manager.GetHostByName("web1")

	assert.NoError(t, err, "GetHostByName should not return error")
	assert.NotNil(t, host, "Host should not be nil")
	assert.Equal(t, "web1", host.Name, "Should return correct host")
}

func TestManager_GetHostByName_NotFound(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	host, err := manager.GetHostByName("nonexistent")

	assert.Error(t, err, "Should return error for nonexistent host")
	assert.Nil(t, host, "Host should be nil")
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_GetGroupByName(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	group, err := manager.GetGroupByName("webservers")

	assert.NoError(t, err, "GetGroupByName should not return error")
	assert.NotNil(t, group, "Group should not be nil")
	assert.Equal(t, "webservers", group.Name, "Should return correct group")
}

func TestManager_GetGroupByName_NotFound(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	group, err := manager.GetGroupByName("nonexistent")

	assert.Error(t, err, "Should return error for nonexistent group")
	assert.Nil(t, group, "Group should be nil")
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_AddHostFilter(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	// Add filter to only include hosts with "web1" name
	manager.AddHostFilter(func(h *types.Host) bool {
		return h.Name == "web1"
	})

	hosts, err := manager.GetHosts("all")

	assert.NoError(t, err, "GetHosts should not return error")
	assert.Equal(t, 1, len(hosts), "Should return only 1 host after filtering")
	assert.Equal(t, "web1", hosts[0].Name, "Should return web1")
}

func TestManager_AddGroupFilter(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	// Add filter to only include groups with "webservers" name
	manager.AddGroupFilter(func(g *types.Group) bool {
		return g.Name == "webservers"
	})

	groups, err := manager.GetGroups("all")

	assert.NoError(t, err, "GetGroups should not return error")
	assert.Equal(t, 1, len(groups), "Should return only 1 group after filtering")
	assert.NotNil(t, groups["webservers"], "Should contain webservers group")
}

func TestManager_ClearFilters(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	// Add filters
	manager.AddHostFilter(func(h *types.Host) bool { return false })
	manager.AddGroupFilter(func(g *types.Group) bool { return false })

	// Clear filters
	manager.ClearFilters()

	hosts, _ := manager.GetHosts("all")
	groups, _ := manager.GetGroups("all")

	assert.Equal(t, 3, len(hosts), "Should return all hosts after clearing filters")
	assert.Equal(t, 2, len(groups), "Should return all groups after clearing filters")
}

func TestManager_GetInventoryStats(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	stats := manager.GetInventoryStats()

	assert.True(t, stats["loaded"].(bool), "Inventory should be loaded")
	assert.Equal(t, 2, stats["groups"].(int), "Should have 2 groups")
	assert.Equal(t, 3, stats["total_hosts"].(int), "Should have 3 total hosts")
	assert.NotNil(t, stats["group_stats"], "Should have group stats")
}

func TestManager_GetInventoryStats_NoInventory(t *testing.T) {
	parser := &mockParser{}
	logger := &mockLogger{}
	cache := newMockCache()
	manager := NewManager(parser, logger, cache)

	stats := manager.GetInventoryStats()

	assert.False(t, stats["loaded"].(bool), "Inventory should not be loaded")
}

func TestManager_ListHosts(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	hosts := manager.ListHosts()

	assert.Equal(t, 3, len(hosts), "Should return 3 host names")
	assert.Equal(t, "db1", hosts[0], "Should be sorted alphabetically")
	assert.Equal(t, "web1", hosts[1], "Should be sorted alphabetically")
	assert.Equal(t, "web2", hosts[2], "Should be sorted alphabetically")
}

func TestManager_ListGroups(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	groups := manager.ListGroups()

	assert.Equal(t, 2, len(groups), "Should return 2 group names")
	// Groups should be sorted alphabetically
	assert.Contains(t, groups, "webservers")
	assert.Contains(t, groups, "databases")
}

func TestManager_MatchPattern(t *testing.T) {
	manager := createTestManager()

	tests := []struct {
		name     string
		str      string
		pattern  string
		expected bool
	}{
		{"Exact match", "web1", "web1", true},
		{"No match", "web1", "web2", false},
		{"Wildcard all", "anything", "*", true},
		{"Prefix wildcard", "web1", "web*", true},
		{"Suffix wildcard", "web1", "*1", true},
		{"Middle wildcard", "web1", "w*1", true},
		{"No wildcard match", "web1", "db*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.matchPattern(tt.str, tt.pattern)
			assert.Equal(t, tt.expected, result, "Pattern matching should work correctly")
		})
	}
}

func TestManager_IsValidHostname(t *testing.T) {
	manager := createTestManager()

	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{"Valid hostname", "web-server-01", true},
		{"Valid FQDN", "web.example.com", true},
		{"Valid with numbers", "server123", true},
		{"Empty string", "", false},
		{"Too long", string(make([]byte, 254)), false},
		{"Invalid characters", "web_server", false},
		{"Invalid characters 2", "web@server", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isValidHostname(tt.hostname)
			assert.Equal(t, tt.expected, result, "Hostname validation should work correctly")
		})
	}
}

func TestManager_ResolveGroupInheritance(t *testing.T) {
	parser := &mockParser{
		parseInventoryFunc: func(ctx context.Context, filePath string) (*types.Inventory, error) {
			return &types.Inventory{
				Groups: map[string]*types.Group{
					"parent": {
						Name:     "parent",
						Children: []string{"child"},
						Hosts:    map[string]*types.Host{},
						Vars:     map[string]interface{}{"parent_var": "parent_value"},
					},
					"child": {
						Name: "child",
						Hosts: map[string]*types.Host{
							"host1": {Name: "host1", Address: "192.168.1.1"},
						},
						Vars: map[string]interface{}{"child_var": "child_value"},
					},
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	cache := newMockCache()
	manager := NewManager(parser, logger, cache)
	ctx := context.Background()

	err := manager.LoadInventory(ctx, "test.yml")

	assert.NoError(t, err, "LoadInventory should not return error")

	// Parent should inherit child's hosts
	parentGroup, _ := manager.GetGroupByName("parent")
	assert.NotNil(t, parentGroup, "Parent group should exist")
	assert.Equal(t, 1, len(parentGroup.Hosts), "Parent should inherit child's hosts")
	assert.NotNil(t, parentGroup.Hosts["host1"], "Parent should have host1")
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = manager.GetHosts("all")
			_, _ = manager.GetGroups("all")
			_ = manager.ListHosts()
			_ = manager.ListGroups()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or cause race conditions
	assert.True(t, true, "Concurrent access should be safe")
}

// Benchmark tests

func BenchmarkManager_GetHosts(b *testing.B) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetHosts("all")
	}
}

func BenchmarkManager_GetGroups(b *testing.B) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetGroups("all")
	}
}

func BenchmarkManager_ListHosts(b *testing.B) {
	manager := createTestManager()
	ctx := context.Background()
	_ = manager.LoadInventory(ctx, "test.yml")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ListHosts()
	}
}

// Helper functions

func createTestManager() *Manager {
	parser := &mockParser{
		parseInventoryFunc: func(ctx context.Context, filePath string) (*types.Inventory, error) {
			return &types.Inventory{
				Hosts: []types.Host{
					{Name: "web1", Address: "192.168.1.10", Port: 22, User: "admin"},
					{Name: "web2", Address: "192.168.1.11", Port: 22, User: "admin"},
					{Name: "db1", Address: "192.168.1.20", Port: 22, User: "admin"},
				},
				Groups: map[string]*types.Group{
					"webservers": {
						Name: "webservers",
						Hosts: map[string]*types.Host{
							"web1": {Name: "web1", Address: "192.168.1.10", Port: 22, User: "admin"},
							"web2": {Name: "web2", Address: "192.168.1.11", Port: 22, User: "admin"},
						},
						Vars: map[string]interface{}{"http_port": 80},
					},
					"databases": {
						Name: "databases",
						Hosts: map[string]*types.Host{
							"db1": {Name: "db1", Address: "192.168.1.20", Port: 22, User: "admin"},
						},
						Vars: map[string]interface{}{"db_port": 5432},
					},
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	cache := newMockCache()

	return NewManager(parser, logger, cache)
}

func TestManager_GroupVarsAppliedToHostFields(t *testing.T) {
	logger := &mockLogger{}
	parser := &mockParser{
		parseInventoryFunc: func(ctx context.Context, filePath string) (*types.Inventory, error) {
			return &types.Inventory{
				Groups: map[string]*types.Group{
					"test-group": {
						Name: "test-group",
						Hosts: map[string]*types.Host{
							"host1": {
								Name: "host1",
							},
							"host2": {
								Name:                  "host2",
								Address:               "192.168.1.2",
								User:                  "custom-user",
								Password:              "custom-pass",
								InsecureIgnoreHostKey: true,
							},
						},
						Vars: map[string]interface{}{
							"address":                  "192.168.1.100",
							"user":                     "group-user",
							"port":                     2222,
							"password":                 "group-pass",
							"key_file":                 "/path/to/group/key",
							"insecure_ignore_host_key": true,
							"custom_var":               "group-value",
						},
					},
				},
			}, nil
		},
	}

	cache := newMockCache()
	manager := NewManager(parser, logger, cache)
	err := manager.LoadInventory(context.Background(), "test.yml")
	assert.NoError(t, err)

	hosts, err := manager.GetHosts("test-group")
	assert.NoError(t, err)
	assert.Len(t, hosts, 2)

	// host1 should inherit all group settings
	host1Found := false
	for _, h := range hosts {
		if h.Name == "host1" {
			host1Found = true
			assert.Equal(t, "192.168.1.100", h.Address, "host1 should inherit address from group")
			assert.Equal(t, "group-user", h.User, "host1 should inherit user from group")
			assert.Equal(t, 2222, h.Port, "host1 should inherit port from group")
			assert.Equal(t, "group-pass", h.Password, "host1 should inherit password from group")
			assert.Equal(t, "/path/to/group/key", h.KeyFile, "host1 should inherit key_file from group")
			assert.True(t, h.InsecureIgnoreHostKey, "host1 should inherit insecure_ignore_host_key from group")
			assert.Equal(t, "group-value", h.Vars["custom_var"], "host1 should inherit custom vars from group")
		}
	}
	assert.True(t, host1Found, "host1 should be found in results")

	// host2 should keep its own settings (host settings take precedence)
	host2Found := false
	for _, h := range hosts {
		if h.Name == "host2" {
			host2Found = true
			assert.Equal(t, "192.168.1.2", h.Address, "host2 should keep its own address")
			assert.Equal(t, "custom-user", h.User, "host2 should keep its own user")
			assert.Equal(t, "custom-pass", h.Password, "host2 should keep its own password")
			assert.Equal(t, 2222, h.Port, "host2 should inherit port from group (not set on host)")
			assert.Equal(t, "/path/to/group/key", h.KeyFile, "host2 should inherit key_file from group (not set on host)")
			assert.True(t, h.InsecureIgnoreHostKey, "host2 already has insecure_ignore_host_key=true")
		}
	}
	assert.True(t, host2Found, "host2 should be found in results")
}
