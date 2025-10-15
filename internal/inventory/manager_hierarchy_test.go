package inventory

import (
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type testLogger struct{}

func (l *testLogger) Debug(format string, args ...interface{})                                {}
func (l *testLogger) Info(format string, args ...interface{})                                 {}
func (l *testLogger) Warn(format string, args ...interface{})                                 {}
func (l *testLogger) Error(format string, args ...interface{})                                {}
func (l *testLogger) Fatal(format string, args ...interface{})                                {}
func (l *testLogger) SetLevel(level string)                                                   {}
func (l *testLogger) TaskStart(taskName, hostName string)                                     {}
func (l *testLogger) TaskEnd(taskName, hostName string, changed, success bool)                {}
func (l *testLogger) PlayStart(playName string, playIndex, totalPlays int)                    {}
func (l *testLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {}
func (l *testLogger) Progress(completed, total int, currentTask, currentHost string)          {}
func (l *testLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {}

func setupTestManager(t *testing.T) *Manager {
	log := &testLogger{}
	cacheManager := cache.NewManager(0)
	templateEngine := template.NewEngine()
	playbookParser := parser.NewEnhancedParser(templateEngine, log)
	
	return NewManager(playbookParser, log, cacheManager)
}

func createTestInventory() *types.Inventory {
	return &types.Inventory{
		Hosts: []types.Host{
			{Name: "web1", Address: "192.168.1.10", Port: 22, User: "deploy"},
			{Name: "web2", Address: "192.168.1.11", Port: 22, User: "deploy"},
			{Name: "db1", Address: "192.168.1.20", Port: 22, User: "postgres"},
			{Name: "db2", Address: "192.168.1.21", Port: 22, User: "postgres"},
			{Name: "cache1", Address: "192.168.1.30", Port: 22, User: "redis"},
		},
		Groups: map[string]*types.Group{
			"webservers": {
				Name: "webservers",
				Hosts: map[string]*types.Host{
					"web1": {Name: "web1", Address: "192.168.1.10"},
					"web2": {Name: "web2", Address: "192.168.1.11"},
				},
				Children: []string{},
				Vars:     map[string]interface{}{"tier": "frontend"},
			},
			"databases": {
				Name: "databases",
				Hosts: map[string]*types.Host{
					"db1": {Name: "db1", Address: "192.168.1.20"},
					"db2": {Name: "db2", Address: "192.168.1.21"},
				},
				Children: []string{},
				Vars:     map[string]interface{}{"tier": "backend"},
			},
			"cache": {
				Name: "cache",
				Hosts: map[string]*types.Host{
					"cache1": {Name: "cache1", Address: "192.168.1.30"},
				},
				Children: []string{},
				Vars:     map[string]interface{}{"tier": "backend"},
			},
			"production": {
				Name:     "production",
				Hosts:    map[string]*types.Host{},
				Children: []string{"webservers", "databases", "cache"},
				Vars:     map[string]interface{}{"env": "prod"},
			},
			"frontend": {
				Name:     "frontend",
				Hosts:    map[string]*types.Host{},
				Children: []string{"webservers"},
				Vars:     map[string]interface{}{"ssl": true},
			},
			"backend": {
				Name:     "backend",
				Hosts:    map[string]*types.Host{},
				Children: []string{"databases", "cache"},
				Vars:     map[string]interface{}{"internal": true},
			},
		},
	}
}

func TestGetHostGroups(t *testing.T) {
	manager := setupTestManager(t)
	manager.inventory = createTestInventory()

	tests := []struct {
		name     string
		hostName string
		expected []string
	}{
		{
			name:     "web server host",
			hostName: "web1",
			expected: []string{"webservers"},
		},
		{
			name:     "database host",
			hostName: "db1",
			expected: []string{"databases"},
		},
		{
			name:     "cache host",
			hostName: "cache1",
			expected: []string{"cache"},
		},
		{
			name:     "non-existent host",
			hostName: "nonexistent",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := manager.GetHostGroups(tt.hostName)
			
			if len(groups) != len(tt.expected) {
				t.Errorf("expected %d groups, got %d", len(tt.expected), len(groups))
				return
			}

			for i, group := range groups {
				if group != tt.expected[i] {
					t.Errorf("expected group %s, got %s", tt.expected[i], group)
				}
			}
		})
	}
}

func TestGetGroupHierarchy(t *testing.T) {
	manager := setupTestManager(t)
	manager.inventory = createTestInventory()

	tests := []struct {
		name           string
		groupName      string
		expectError    bool
		expectedHosts  int
		expectedChildren int
		expectedParents  int
	}{
		{
			name:             "leaf group with hosts",
			groupName:        "webservers",
			expectError:      false,
			expectedHosts:    2,
			expectedChildren: 0,
			expectedParents:  2, // production and frontend
		},
		{
			name:             "parent group",
			groupName:        "production",
			expectError:      false,
			expectedHosts:    0, // no direct hosts
			expectedChildren: 3, // webservers, databases, cache
			expectedParents:  0,
		},
		{
			name:             "middle group",
			groupName:        "frontend",
			expectError:      false,
			expectedHosts:    0,
			expectedChildren: 1, // webservers
			expectedParents:  0,
		},
		{
			name:        "non-existent group",
			groupName:   "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hierarchy, err := manager.GetGroupHierarchy(tt.groupName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(hierarchy.Hosts) != tt.expectedHosts {
				t.Errorf("expected %d hosts, got %d", tt.expectedHosts, len(hierarchy.Hosts))
			}

			if len(hierarchy.Children) != tt.expectedChildren {
				t.Errorf("expected %d children, got %d", tt.expectedChildren, len(hierarchy.Children))
			}

			if len(hierarchy.Parents) != tt.expectedParents {
				t.Errorf("expected %d parents, got %d", tt.expectedParents, len(hierarchy.Parents))
			}
		})
	}
}

func TestIsHostInGroup(t *testing.T) {
	manager := setupTestManager(t)
	manager.inventory = createTestInventory()

	tests := []struct {
		name      string
		hostName  string
		groupName string
		expected  bool
	}{
		{
			name:      "host in group",
			hostName:  "web1",
			groupName: "webservers",
			expected:  true,
		},
		{
			name:      "host not in group",
			hostName:  "web1",
			groupName: "databases",
			expected:  false,
		},
		{
			name:      "non-existent host",
			hostName:  "nonexistent",
			groupName: "webservers",
			expected:  false,
		},
		{
			name:      "non-existent group",
			hostName:  "web1",
			groupName: "nonexistent",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsHostInGroup(tt.hostName, tt.groupName)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetAllHostsInGroup(t *testing.T) {
	manager := setupTestManager(t)
	manager.inventory = createTestInventory()

	tests := []struct {
		name          string
		groupName     string
		expectError   bool
		expectedHosts int
	}{
		{
			name:          "leaf group",
			groupName:     "webservers",
			expectError:   false,
			expectedHosts: 2,
		},
		{
			name:          "parent group with children",
			groupName:     "production",
			expectError:   false,
			expectedHosts: 5, // web1, web2, db1, db2, cache1
		},
		{
			name:          "middle group",
			groupName:     "backend",
			expectError:   false,
			expectedHosts: 3, // db1, db2, cache1
		},
		{
			name:        "non-existent group",
			groupName:   "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hosts, err := manager.GetAllHostsInGroup(tt.groupName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(hosts) != tt.expectedHosts {
				t.Errorf("expected %d hosts, got %d: %v", tt.expectedHosts, len(hosts), hosts)
			}
		})
	}
}

func TestFindParentGroups(t *testing.T) {
	manager := setupTestManager(t)
	manager.inventory = createTestInventory()

	tests := []struct {
		name           string
		childGroupName string
		expectedCount  int
	}{
		{
			name:           "group with multiple parents",
			childGroupName: "webservers",
			expectedCount:  2, // production and frontend
		},
		{
			name:           "group with one parent",
			childGroupName: "cache",
			expectedCount:  2, // production and backend
		},
		{
			name:           "group with no parents",
			childGroupName: "production",
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parents := manager.findParentGroups(tt.childGroupName)
			if len(parents) != tt.expectedCount {
				t.Errorf("expected %d parents, got %d: %v", tt.expectedCount, len(parents), parents)
			}
		})
	}
}

func TestCollectHostsRecursive(t *testing.T) {
	manager := setupTestManager(t)
	manager.inventory = createTestInventory()

	tests := []struct {
		name          string
		groupName     string
		expectedHosts int
	}{
		{
			name:          "leaf group",
			groupName:     "webservers",
			expectedHosts: 2,
		},
		{
			name:          "parent group",
			groupName:     "production",
			expectedHosts: 5,
		},
		{
			name:          "nested parent",
			groupName:     "frontend",
			expectedHosts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := manager.inventory.Groups[tt.groupName]
			hostSet := make(map[string]bool)
			manager.collectHostsRecursive(group, hostSet)

			if len(hostSet) != tt.expectedHosts {
				t.Errorf("expected %d hosts, got %d", tt.expectedHosts, len(hostSet))
			}
		})
	}
}

func TestGroupHierarchyWithNoInventory(t *testing.T) {
	manager := setupTestManager(t)

	_, err := manager.GetGroupHierarchy("test")
	if err == nil {
		t.Error("expected error when no inventory loaded")
	}

	groups := manager.GetHostGroups("test")
	if len(groups) != 0 {
		t.Error("expected empty groups when no inventory loaded")
	}

	result := manager.IsHostInGroup("test", "test")
	if result {
		t.Error("expected false when no inventory loaded")
	}
}
