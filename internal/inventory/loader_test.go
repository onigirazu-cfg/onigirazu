package inventory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestMultiSourceLoaderBasic tests basic loading from multiple files
func TestMultiSourceLoaderBasic(t *testing.T) {
	// Create temporary files for testing
	tmpDir := t.TempDir()

	// Create first inventory file with ansible format
	inv1Path := filepath.Join(tmpDir, "inventory1.yml")
	inv1Content := `all:
  hosts:
    host1:
      ansible_host: 192.168.1.1
      ansible_port: 22
      env: prod
  children:
    webservers:
      hosts:
        host1: null
      vars:
        role: web
`
	if err := os.WriteFile(inv1Path, []byte(inv1Content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create second inventory file (should override host1 vars)
	inv2Path := filepath.Join(tmpDir, "inventory2.yml")
	inv2Content := `all:
  hosts:
    host1:
      ansible_host: 192.168.1.1
      ansible_port: 22
      env: staging
    host2:
      ansible_host: 192.168.1.2
      ansible_port: 22
  children:
    webservers:
      hosts:
        host1: null
        host2: null
      vars:
        role: app
`
	if err := os.WriteFile(inv2Path, []byte(inv2Content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup loader
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 10*time.Minute)

	// Load from multiple sources
	ctx := context.Background()
	inventory, err := loader.LoadFromMultipleSources(ctx, []string{inv1Path, inv2Path})
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Verify inventory
	if inventory == nil {
		t.Fatal("Inventory should not be nil")
	}

	if len(inventory.Hosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(inventory.Hosts))
	}

	// Check that host1 has env=staging (from inv2, last-occurrence-wins)
	var host1 *types.Host
	for i := range inventory.Hosts {
		if inventory.Hosts[i].Name == "host1" {
			host1 = &inventory.Hosts[i]
			break
		}
	}

	if host1 == nil {
		t.Fatal("host1 not found in inventory")
	}

	if env, ok := host1.Vars["env"].(string); !ok || env != "staging" {
		t.Errorf("Expected host1 env=staging, got %v", host1.Vars["env"])
	}
}

// TestMultiSourceLoaderDirectory tests loading from a directory
func TestMultiSourceLoaderDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	invDir := filepath.Join(tmpDir, "inventory")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create multiple inventory files in directory
	inv1Path := filepath.Join(invDir, "1-servers.yml")
	inv1Content := `all:
  hosts:
    server1:
      ansible_host: 10.0.0.1
  children:
    production:
      hosts:
        server1: null
`
	if err := os.WriteFile(inv1Path, []byte(inv1Content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	inv2Path := filepath.Join(invDir, "2-apps.yml")
	inv2Content := `all:
  hosts:
    app1:
      ansible_host: 10.0.0.2
  children:
    applications:
      hosts:
        app1: null
`
	if err := os.WriteFile(inv2Path, []byte(inv2Content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup loader
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 10*time.Minute)

	// Load from directory
	ctx := context.Background()
	inventory, err := loader.LoadFromMultipleSources(ctx, []string{invDir})
	if err != nil {
		t.Fatalf("Failed to load inventory from directory: %v", err)
	}

	// Verify inventory
	if len(inventory.Hosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(inventory.Hosts))
	}

	if len(inventory.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(inventory.Groups))
	}
}

// TestMultiSourceLoaderLastOccurrenceWins tests that last-occurrence strategy works
func TestMultiSourceLoaderLastOccurrenceWins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create three inventory files
	files := []struct {
		name    string
		content string
	}{
		{
			"inv1.yml",
			`all:
  hosts:
    host1:
      ansible_host: 1.1.1.1
      version: v1
      status: active
  children:
    ungrouped:
      hosts:
        host1: null
`,
		},
		{
			"inv2.yml",
			`all:
  hosts:
    host1:
      ansible_host: 1.1.1.1
      version: v2
  children:
    ungrouped:
      hosts:
        host1: null
`,
		},
		{
			"inv3.yml",
			`all:
  hosts:
    host1:
      ansible_host: 1.1.1.1
      status: inactive
  children:
    ungrouped:
      hosts:
        host1: null
`,
		},
	}

	paths := make([]string, 0)
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		paths = append(paths, path)
	}

	// Setup loader
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 10*time.Minute)

	// Load from multiple sources
	ctx := context.Background()
	inventory, err := loader.LoadFromMultipleSources(ctx, paths)
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	if len(inventory.Hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(inventory.Hosts))
	}

	host := inventory.Hosts[0]

	// version should be v2 (from inv2, which is second)
	if version, ok := host.Vars["version"].(string); !ok || version != "v2" {
		t.Errorf("Expected version=v2, got %v", host.Vars["version"])
	}

	// status should be inactive (from inv3, which is last)
	if status, ok := host.Vars["status"].(string); !ok || status != "inactive" {
		t.Errorf("Expected status=inactive, got %v", host.Vars["status"])
	}
}

// TestMultiSourceLoaderCacheTTL tests that dynamic inventory cache respects TTL
func TestMultiSourceLoaderCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple inventory file
	invPath := filepath.Join(tmpDir, "inventory.yml")
	invContent := `all:
  hosts:
    host1:
      ansible_host: 192.168.1.1
  children:
    ungrouped:
      hosts:
        host1: null
`
	if err := os.WriteFile(invPath, []byte(invContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup loader with short TTL
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 100*time.Millisecond)

	// Load inventory
	ctx := context.Background()
	inventory1, err := loader.LoadFromMultipleSources(ctx, []string{invPath})
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	if inventory1 == nil {
		t.Fatal("inventory1 should not be nil")
	}

	// Verify TTL getter
	if ttl := loader.GetCacheTTL(); ttl != 100*time.Millisecond {
		t.Errorf("Expected TTL 100ms, got %v", ttl)
	}

	// Set different TTL
	loader.SetCacheTTL(200 * time.Millisecond)
	if ttl := loader.GetCacheTTL(); ttl != 200*time.Millisecond {
		t.Errorf("Expected TTL 200ms, got %v", ttl)
	}
}

// TestMultiSourceLoaderInsertionOrder tests that insertion order is maintained
func TestMultiSourceLoaderInsertionOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create inventory files with specific host order
	invPath := filepath.Join(tmpDir, "inventory.yml")
	invContent := `all:
  hosts:
    host3:
      ansible_host: 192.168.1.3
    host1:
      ansible_host: 192.168.1.1
    host2:
      ansible_host: 192.168.1.2
  children:
    ungrouped:
      hosts:
        host1: null
        host2: null
        host3: null
`
	if err := os.WriteFile(invPath, []byte(invContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup loader
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 10*time.Minute)

	// Load inventory
	ctx := context.Background()
	inventory, err := loader.LoadFromMultipleSources(ctx, []string{invPath})
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Verify order is maintained (first occurrence)
	expectedOrder := []string{"host3", "host1", "host2"}
	if len(inventory.Hosts) != len(expectedOrder) {
		t.Errorf("Expected %d hosts, got %d", len(expectedOrder), len(inventory.Hosts))
	}

	for i, hostName := range expectedOrder {
		if i >= len(inventory.Hosts) {
			break
		}
		if inventory.Hosts[i].Name != hostName {
			t.Errorf("Expected host %d to be %s, got %s", i, hostName, inventory.Hosts[i].Name)
		}
	}
}

// TestMultiSourceLoaderErrorHandling tests error handling
func TestMultiSourceLoaderErrorHandling(t *testing.T) {
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 10*time.Minute)

	// Test with non-existent file
	ctx := context.Background()
	_, err := loader.LoadFromMultipleSources(ctx, []string{"/non/existent/path.yml"})
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}

	// Test with empty paths
	_, err = loader.LoadFromMultipleSources(ctx, []string{})
	if err == nil {
		t.Fatal("Expected error for empty paths")
	}
}

// TestMultiSourceLoaderGroupMerge tests that groups are merged correctly
func TestMultiSourceLoaderGroupMerge(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first inventory with a group
	inv1Path := filepath.Join(tmpDir, "inventory1.yml")
	inv1Content := `all:
  hosts:
    host1:
      ansible_host: 192.168.1.1
  children:
    webservers:
      hosts:
        host1: null
      vars:
        role: web
`
	if err := os.WriteFile(inv1Path, []byte(inv1Content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create second inventory modifying the same group
	inv2Path := filepath.Join(tmpDir, "inventory2.yml")
	inv2Content := `all:
  hosts:
    host1:
      ansible_host: 192.168.1.1
    host2:
      ansible_host: 192.168.1.2
  children:
    webservers:
      hosts:
        host1: null
        host2: null
      vars:
        role: api
`
	if err := os.WriteFile(inv2Path, []byte(inv2Content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup loader
	log := logger.New(false)
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute)

	loader := NewMultiSourceLoader(enhancedParser, log, cacheManager, 10*time.Minute)

	// Load from multiple sources
	ctx := context.Background()
	inventory, err := loader.LoadFromMultipleSources(ctx, []string{inv1Path, inv2Path})
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Verify group was merged
	webserversGroup := inventory.Groups["webservers"]
	if webserversGroup == nil {
		t.Fatal("webservers group not found")
	}

	// Role should be api (from inv2, last-occurrence-wins)
	if role, ok := webserversGroup.Vars["role"].(string); !ok || role != "api" {
		t.Errorf("Expected role=api, got %v", webserversGroup.Vars["role"])
	}
}
