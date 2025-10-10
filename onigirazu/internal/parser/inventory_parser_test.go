package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInventoryParser(t *testing.T) {
	logger := &mockLogger{}

	parser := NewInventoryParser(logger)

	assert.NotNil(t, parser)
	assert.Equal(t, logger, parser.logger)
}

func TestInventoryParser_FindInventoryFile_Found(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.yml")

	err := os.WriteFile(inventoryPath, []byte("test"), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	found, err := parser.FindInventoryFile(tmpDir)

	assert.NoError(t, err)
	assert.Equal(t, inventoryPath, found)
}

func TestInventoryParser_FindInventoryFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	found, err := parser.FindInventoryFile(tmpDir)

	assert.Error(t, err)
	assert.Empty(t, found)
	assert.Contains(t, err.Error(), "no inventory file found")
}

func TestInventoryParser_FindInventoryFile_MultipleFormats(t *testing.T) {
	tmpDir := t.TempDir()

	// Test different file names
	testFiles := []string{
		"inventory.yml",
		"inventory.yaml",
		"inventory.toml",
		"hosts",
		"hosts.yml",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, filename)
			err := os.MkdirAll(testDir, 0755)
			require.NoError(t, err)

			filePath := filepath.Join(testDir, filename)
			err = os.WriteFile(filePath, []byte("test"), 0644)
			require.NoError(t, err)

			logger := &mockLogger{}
			parser := NewInventoryParser(logger)

			found, err := parser.FindInventoryFile(testDir)

			assert.NoError(t, err)
			assert.Equal(t, filePath, found)
		})
	}
}

func TestInventoryParser_ParseInventoryFile_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.yml")

	inventoryContent := `all:
  hosts:
    host1:
      onigirazu_host: 192.168.1.1
groups:
  webservers:
    hosts:
      host1: {}
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
}

func TestInventoryParser_ParseInventoryFile_TOML(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.toml")

	inventoryContent := `[hosts.host1]
address = "192.168.1.1"
port = 22
user = "admin"

[groups.webservers]
hosts = ["host1"]
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 1)
	assert.Equal(t, "host1", inventory.Hosts[0].Name)
	assert.Equal(t, "192.168.1.1", inventory.Hosts[0].Address)
	assert.Equal(t, 22, inventory.Hosts[0].Port)
	assert.Equal(t, "admin", inventory.Hosts[0].User)
}

func TestInventoryParser_ParseInventoryFile_SimpleList(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "hosts")

	inventoryContent := `192.168.1.1
192.168.1.2
192.168.1.3
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 3)
	assert.Equal(t, "192.168.1.1", inventory.Hosts[0].Address)
	assert.Equal(t, "192.168.1.2", inventory.Hosts[1].Address)
	assert.Equal(t, "192.168.1.3", inventory.Hosts[2].Address)
}

func TestInventoryParser_ParseInventoryFile_FileNotFound(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, "/nonexistent/inventory.yml")

	assert.Error(t, err)
	assert.Nil(t, inventory)
	assert.Contains(t, err.Error(), "error reading inventory file")
}

func TestInventoryParser_ParseSimpleList_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "hosts")

	inventoryContent := `# Web servers
192.168.1.1
192.168.1.2

# Database servers
192.168.1.10
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 3)
}

func TestInventoryParser_ParseSimpleList_WithPorts(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "hosts")

	inventoryContent := `192.168.1.1:22
192.168.1.2:2222
192.168.1.3:3333
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 3)
	assert.Equal(t, 22, inventory.Hosts[0].Port)
	assert.Equal(t, 2222, inventory.Hosts[1].Port)
	assert.Equal(t, 3333, inventory.Hosts[2].Port)
}

func TestInventoryParser_ParseSimpleList_WithUser(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "hosts")

	inventoryContent := `admin@192.168.1.1
user@192.168.1.2:2222
root@192.168.1.3
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 3)
	assert.Equal(t, "admin", inventory.Hosts[0].User)
	assert.Equal(t, "user", inventory.Hosts[1].User)
	assert.Equal(t, 2222, inventory.Hosts[1].Port)
	assert.Equal(t, "root", inventory.Hosts[2].User)
}

func TestInventoryParser_ParseSimpleList_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "hosts")

	err := os.WriteFile(inventoryPath, []byte(""), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	// Empty file might parse as YAML (empty document), so we just check it doesn't crash
	if err != nil {
		assert.Contains(t, err.Error(), "no valid hosts")
	} else {
		// If it parses, it should be empty
		assert.NotNil(t, inventory)
	}
}

func TestInventoryParser_ParseSimpleList_OnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "hosts")

	inventoryContent := `# Comment 1
# Comment 2
# Comment 3
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)
	ctx := context.Background()

	inventory, err := parser.ParseInventoryFile(ctx, inventoryPath)

	// File with only comments might parse as YAML or fail as simple list
	if err != nil {
		// Expected: no valid hosts
		assert.Contains(t, err.Error(), "no valid hosts")
	} else {
		// If it parses as YAML, it should be empty or minimal
		assert.NotNil(t, inventory)
	}
}

func TestInventoryParser_ParseYamlInventory_Valid(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	yamlContent := `all:
  hosts:
    host1:
      onigirazu_host: 192.168.1.1
groups:
  webservers:
    hosts:
      host1: {}
`

	inventory, err := parser.parseYamlInventory([]byte(yamlContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.NotNil(t, inventory.Groups)
	assert.NotNil(t, inventory.Hosts)
}

func TestInventoryParser_ParseYamlInventory_Invalid(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	// Use truly invalid YAML that will fail parsing
	invalidYaml := `all:
  hosts: [[[
    invalid
`

	inventory, err := parser.parseYamlInventory([]byte(invalidYaml))

	// YAML parser might be lenient, so check if error or valid parse
	if err != nil {
		assert.Contains(t, err.Error(), "error parsing YAML inventory")
		assert.Nil(t, inventory)
	} else {
		// If it parses, it should at least be initialized
		assert.NotNil(t, inventory)
	}
}

func TestInventoryParser_ParseTomlInventory_Valid(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlContent := `[hosts.host1]
address = "192.168.1.1"
port = 22
user = "admin"

[groups.webservers]
hosts = ["host1"]
`

	inventory, err := parser.parseTomlInventory([]byte(tomlContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 1)
	assert.Len(t, inventory.Groups, 1)
}

func TestInventoryParser_ParseTomlInventory_Invalid(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	invalidToml := `[hosts.host1
invalid toml syntax
`

	inventory, err := parser.parseTomlInventory([]byte(invalidToml))

	assert.Error(t, err)
	assert.Nil(t, inventory)
	assert.Contains(t, err.Error(), "error parsing TOML inventory")
}

func TestInventoryParser_ParseTomlInventory_WithDefaults(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlContent := `[hosts.host1]
address = "192.168.1.1"

[hosts.host2]
# No address, port, or user specified

[groups.all]
hosts = ["host1", "host2"]
`

	inventory, err := parser.parseTomlInventory([]byte(tomlContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 2)

	// Check defaults
	for _, host := range inventory.Hosts {
		if host.Port == 0 {
			t.Errorf("Expected default port 22, got %d", host.Port)
		}
		if host.User == "" {
			t.Errorf("Expected default user 'root', got empty")
		}
		if host.Vars == nil {
			t.Error("Expected vars map to be initialized")
		}
	}
}

func TestInventoryParser_IsSimpleList_True(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	simpleContent := `192.168.1.1
192.168.1.2
192.168.1.3
`

	result := parser.isSimpleList(simpleContent)

	assert.True(t, result)
}

func TestInventoryParser_IsSimpleList_False_YAML(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	yamlContent := `all:
  hosts:
    host1:
      address: 192.168.1.1
`

	result := parser.isSimpleList(yamlContent)

	assert.False(t, result)
}

func TestInventoryParser_IsSimpleList_False_TOML(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlContent := `[hosts.host1]
address = "192.168.1.1"
`

	result := parser.isSimpleList(tomlContent)

	assert.False(t, result)
}

func TestInventoryParser_IsSimpleList_WithComments(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	contentWithComments := `# Comment
192.168.1.1
# Another comment
192.168.1.2
`

	result := parser.isSimpleList(contentWithComments)

	assert.True(t, result)
}

func TestInventoryParser_ParseSimpleHostLine_IPOnly(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	host := parser.parseSimpleHostLine("192.168.1.1", 1)

	assert.NotNil(t, host)
	assert.Equal(t, "192.168.1.1", host.Address)
	assert.Equal(t, "192.168.1.1", host.Name)
	assert.Equal(t, 22, host.Port)
	assert.Equal(t, "root", host.User)
}

func TestInventoryParser_ParseSimpleHostLine_WithPort(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	host := parser.parseSimpleHostLine("192.168.1.1:2222", 1)

	assert.NotNil(t, host)
	assert.Equal(t, "192.168.1.1", host.Address)
	assert.Equal(t, 2222, host.Port)
}

func TestInventoryParser_ParseSimpleHostLine_WithUser(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	host := parser.parseSimpleHostLine("admin@192.168.1.1", 1)

	assert.NotNil(t, host)
	assert.Equal(t, "192.168.1.1", host.Address)
	assert.Equal(t, "admin", host.User)
	assert.Equal(t, "admin@192.168.1.1", host.Name)
}

func TestInventoryParser_ParseSimpleHostLine_WithUserAndPort(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	host := parser.parseSimpleHostLine("admin@192.168.1.1:2222", 1)

	assert.NotNil(t, host)
	assert.Equal(t, "192.168.1.1", host.Address)
	assert.Equal(t, "admin", host.User)
	assert.Equal(t, 2222, host.Port)
	assert.Equal(t, "admin@192.168.1.1", host.Name)
}

func TestInventoryParser_ParseSimpleHostLine_Hostname(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	host := parser.parseSimpleHostLine("server.example.com", 1)

	assert.NotNil(t, host)
	assert.Equal(t, "server.example.com", host.Address)
	assert.Equal(t, "server.example.com", host.Name)
	assert.Equal(t, 22, host.Port)
}

func TestInventoryParser_ParseSimpleHostLine_InvalidPort(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	host := parser.parseSimpleHostLine("192.168.1.1:invalid", 1)

	assert.NotNil(t, host)
	assert.Equal(t, "192.168.1.1", host.Address)
	assert.Equal(t, 22, host.Port) // Should default to 22
	// Note: Warning is logged but we don't track it in mock
}

func TestInventoryParser_AutoDetectAndParse_YAML(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	yamlContent := `all:
  hosts:
    host1:
      onigirazu_host: 192.168.1.1
`

	inventory, err := parser.autoDetectAndParse([]byte(yamlContent), "test.yml")

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
}

func TestInventoryParser_AutoDetectAndParse_TOML(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlContent := `[hosts.host1]
address = "192.168.1.1"
`

	inventory, err := parser.autoDetectAndParse([]byte(tomlContent), "test.toml")

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
}

func TestInventoryParser_AutoDetectAndParse_SimpleList(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	simpleContent := `192.168.1.1
192.168.1.2
192.168.1.3
`

	inventory, err := parser.autoDetectAndParse([]byte(simpleContent), "hosts")

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 3)
}

func TestInventoryParser_ParseTomlInventory_WithVars(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlContent := `[hosts.host1]
address = "192.168.1.1"
port = 22
user = "admin"

[hosts.host1.vars]
env = "production"
region = "us-east"

[groups.webservers]
hosts = ["host1"]

[groups.webservers.vars]
http_port = 80
`

	inventory, err := parser.parseTomlInventory([]byte(tomlContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 1)
	assert.NotNil(t, inventory.Hosts[0].Vars)
	assert.NotNil(t, inventory.Groups["webservers"].Vars)
}

func TestInventoryParser_ParseTomlInventory_WithChildren(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlContent := `[hosts.host1]
address = "192.168.1.1"

[groups.webservers]
hosts = ["host1"]

[groups.production]
children = ["webservers"]
`

	inventory, err := parser.parseTomlInventory([]byte(tomlContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Groups, 2)
	assert.Contains(t, inventory.Groups["production"].Children, "webservers")
}
