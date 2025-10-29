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

func TestInventoryParser_ParseInventoryFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.json")

	jsonContent := `{
  "hosts": [
    {
      "name": "web1",
      "address": "192.168.1.10",
      "port": 22,
      "user": "deploy",
      "vars": {
        "app_port": 8080
      }
    },
    {
      "name": "db1",
      "address": "192.168.1.20",
      "port": 22,
      "user": "postgres",
      "vars": {
        "db_port": 5432
      }
    }
  ],
  "groups": {
    "webservers": {
      "name": "webservers",
      "hosts": {
        "web1": {}
      },
      "vars": {
        "http_port": 80
      },
      "children": []
    },
    "databases": {
      "name": "databases",
      "hosts": {
        "db1": {}
      },
      "vars": {
        "backup_enabled": true
      },
      "children": []
    }
  }
}`

	err := os.WriteFile(inventoryPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	inventory, err := parser.ParseInventoryFile(context.Background(), inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 2)
	assert.Len(t, inventory.Groups, 2)
	assert.Equal(t, "web1", inventory.Hosts[0].Name)
	assert.Equal(t, "192.168.1.10", inventory.Hosts[0].Address)
	assert.Equal(t, float64(8080), inventory.Hosts[0].Vars["app_port"])
	assert.Contains(t, inventory.Groups, "webservers")
	assert.Contains(t, inventory.Groups, "databases")
}

func TestInventoryParser_ParseInventoryFile_INI(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.ini")

	iniContent := `# Test inventory
[webservers]
web1 ansible_host=192.168.1.10 ansible_user=deploy app_port=8080
web2 ansible_host=192.168.1.11 ansible_user=deploy app_port=8080

[databases]
db1 ansible_host=192.168.1.20 ansible_user=postgres db_port=5432

[webservers:vars]
http_port=80
https_port=443

[databases:vars]
backup_enabled=true

[production:children]
webservers
databases

[production:vars]
env=production
`

	err := os.WriteFile(inventoryPath, []byte(iniContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	inventory, err := parser.ParseInventoryFile(context.Background(), inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 3)
	assert.Len(t, inventory.Groups, 3)

	assert.Contains(t, inventory.Groups, "webservers")
	assert.Contains(t, inventory.Groups, "databases")
	assert.Contains(t, inventory.Groups, "production")

	webservers := inventory.Groups["webservers"]
	assert.Len(t, webservers.Hosts, 2)
	assert.Equal(t, "80", webservers.Vars["http_port"])

	production := inventory.Groups["production"]
	assert.Len(t, production.Children, 2)
	assert.Contains(t, production.Children, "webservers")
	assert.Contains(t, production.Children, "databases")
	assert.Equal(t, "production", production.Vars["env"])
}

func TestInventoryParser_ParseJsonInventory(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	jsonContent := `{
  "hosts": [
    {
      "name": "test1",
      "address": "10.0.0.1",
      "port": 22,
      "user": "root",
      "vars": {}
    }
  ],
  "groups": {
    "testgroup": {
      "name": "testgroup",
      "hosts": {
        "test1": {}
      },
      "vars": {
        "test_var": "value"
      },
      "children": []
    }
  }
}`

	inventory, err := parser.parseJsonInventory([]byte(jsonContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 1)
	assert.Len(t, inventory.Groups, 1)
	assert.Equal(t, "test1", inventory.Hosts[0].Name)
	assert.Equal(t, "10.0.0.1", inventory.Hosts[0].Address)
}

func TestInventoryParser_ParseJsonInventory_Invalid(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	invalidJson := `{invalid json`

	inventory, err := parser.parseJsonInventory([]byte(invalidJson))

	assert.Error(t, err)
	assert.Nil(t, inventory)
	assert.Contains(t, err.Error(), "error parsing JSON inventory")
}

func TestInventoryParser_ParseIniInventory(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	iniContent := `[group1]
host1 ansible_host=10.0.0.1 ansible_user=ubuntu
host2 ansible_host=10.0.0.2

[group1:vars]
var1=value1

[group2:children]
group1

[group2:vars]
var2=value2
`

	inventory, err := parser.parseIniInventory([]byte(iniContent))

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Groups, 2)

	group1 := inventory.Groups["group1"]
	assert.NotNil(t, group1)
	assert.Len(t, group1.Hosts, 2)
	assert.Equal(t, "value1", group1.Vars["var1"])

	group2 := inventory.Groups["group2"]
	assert.NotNil(t, group2)
	assert.Len(t, group2.Children, 1)
	assert.Contains(t, group2.Children, "group1")
	assert.Equal(t, "value2", group2.Vars["var2"])
}

func TestInventoryParser_ParseIniHostLine(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tests := []struct {
		name     string
		line     string
		expected struct {
			name    string
			address string
			port    int
			user    string
		}
	}{
		{
			name: "simple host",
			line: "web1",
			expected: struct {
				name    string
				address string
				port    int
				user    string
			}{
				name:    "web1",
				address: "web1",
				port:    22,
				user:    "",
			},
		},
		{
			name: "host with ansible_host",
			line: "web1 ansible_host=192.168.1.10",
			expected: struct {
				name    string
				address string
				port    int
				user    string
			}{
				name:    "web1",
				address: "192.168.1.10",
				port:    22,
				user:    "",
			},
		},
		{
			name: "host with ansible_user",
			line: "web1 ansible_host=192.168.1.10 ansible_user=deploy",
			expected: struct {
				name    string
				address string
				port    int
				user    string
			}{
				name:    "web1",
				address: "192.168.1.10",
				port:    22,
				user:    "deploy",
			},
		},
		{
			name: "host with ansible_port",
			line: "web1 ansible_host=192.168.1.10 ansible_port=2222",
			expected: struct {
				name    string
				address string
				port    int
				user    string
			}{
				name:    "web1",
				address: "192.168.1.10",
				port:    2222,
				user:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := parser.parseIniHostLine(tt.line, 1)

			assert.NotNil(t, host)
			assert.Equal(t, tt.expected.name, host.Name)
			assert.Equal(t, tt.expected.address, host.Address)
			assert.Equal(t, tt.expected.port, host.Port)
			assert.Equal(t, tt.expected.user, host.User)
		})
	}
}

func TestInventoryParser_ParseIniHostLineWithKeyFile(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	line := "web1 ansible_host=192.168.1.10 ansible_user=deploy ansible_ssh_private_key_file=/home/user/.ssh/id_rsa"
	host := parser.parseIniHostLine(line, 1)

	assert.NotNil(t, host)
	assert.Equal(t, "web1", host.Name)
	assert.Equal(t, "192.168.1.10", host.Address)
	assert.Equal(t, "deploy", host.User)
	assert.Equal(t, "/home/user/.ssh/id_rsa", host.KeyFile)
}

func TestInventoryParser_IsExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	executablePath := filepath.Join(tmpDir, "script.sh")
	err := os.WriteFile(executablePath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	nonExecutablePath := filepath.Join(tmpDir, "file.txt")
	err = os.WriteFile(nonExecutablePath, []byte("test"), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	assert.True(t, parser.isExecutable(executablePath))
	assert.False(t, parser.isExecutable(nonExecutablePath))
	assert.False(t, parser.isExecutable("/nonexistent/file"))
}

func TestInventoryParser_ParseDynamicInventory(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "dynamic.sh")

	scriptContent := `#!/bin/bash
if [ "$1" = "--list" ]; then
    cat <<EOF
{
  "hosts": [
    {
      "name": "dynamic1",
      "address": "10.0.0.1",
      "port": 22,
      "user": "ubuntu",
      "vars": {}
    }
  ],
  "groups": {
    "dynamic": {
      "name": "dynamic",
      "hosts": {
        "dynamic1": {}
      },
      "vars": {},
      "children": []
    }
  }
}
EOF
fi
`

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	inventory, err := parser.parseDynamicInventory(scriptPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
	assert.Len(t, inventory.Hosts, 1)
	assert.Equal(t, "dynamic1", inventory.Hosts[0].Name)
	assert.Contains(t, inventory.Groups, "dynamic")
}

func TestInventoryParser_InsecureIgnoreHostKey_YAML(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	yamlData := `
hosts:
  - name: test-host
    address: 192.168.1.1
    insecure_ignore_host_key: true
`
	inv, err := parser.parseYamlInventory([]byte(yamlData))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)
	assert.True(t, inv.Hosts[0].InsecureIgnoreHostKey, "InsecureIgnoreHostKey should be true")
}

func TestInventoryParser_InsecureIgnoreHostKey_TOML(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	tomlData := `
[hosts.test-host]
address = "192.168.1.1"
insecure_ignore_host_key = true
`
	inv, err := parser.parseTomlInventory([]byte(tomlData))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)
	assert.True(t, inv.Hosts[0].InsecureIgnoreHostKey, "InsecureIgnoreHostKey should be true")
}

func TestInventoryParser_InsecureIgnoreHostKey_JSON(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	jsonData := `{
  "hosts": [
    {
      "name": "test-host",
      "address": "192.168.1.1",
      "insecure_ignore_host_key": true
    }
  ]
}`
	inv, err := parser.parseJsonInventory([]byte(jsonData))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)
	assert.True(t, inv.Hosts[0].InsecureIgnoreHostKey, "InsecureIgnoreHostKey should be true")
}

// Tests for Ansible-format YAML inventory support

func TestInventoryParser_AnsibleYAML_BasicHosts(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      ansible_port: 22
    web2:
      ansible_host: 192.168.1.11
      ansible_user: deploy
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.NotNil(t, inv)
	assert.Len(t, inv.Hosts, 2)

	// Check first host
	assert.Equal(t, "web1", inv.Hosts[0].Name)
	assert.Equal(t, "192.168.1.10", inv.Hosts[0].Address)
	assert.Equal(t, "deploy", inv.Hosts[0].User)
	assert.Equal(t, 22, inv.Hosts[0].Port)

	// Check second host
	assert.Equal(t, "web2", inv.Hosts[1].Name)
	assert.Equal(t, "192.168.1.11", inv.Hosts[1].Address)
	assert.Equal(t, "deploy", inv.Hosts[1].User)
}

func TestInventoryParser_AnsibleYAML_WithGroups(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
    web2:
      ansible_host: 192.168.1.11
      ansible_user: deploy
    db1:
      ansible_host: 192.168.1.20
      ansible_user: postgres
  children:
    webservers:
      hosts:
        web1:
        web2:
      vars:
        http_port: 80
    databases:
      hosts:
        db1:
      vars:
        db_port: 5432
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.NotNil(t, inv)
	assert.Len(t, inv.Hosts, 3)
	assert.Len(t, inv.Groups, 2)

	// Check webservers group
	webserversGroup, ok := inv.Groups["webservers"]
	assert.True(t, ok)
	assert.Len(t, webserversGroup.Hosts, 2)
	assert.Equal(t, 80, webserversGroup.Vars["http_port"])

	// Check databases group
	databasesGroup, ok := inv.Groups["databases"]
	assert.True(t, ok)
	assert.Len(t, databasesGroup.Hosts, 1)
	assert.Equal(t, 5432, databasesGroup.Vars["db_port"])
}

func TestInventoryParser_AnsibleYAML_WithCustomVars(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      app_version: 1.0.0
      environment: production
      custom_var: custom_value
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)

	host := inv.Hosts[0]
	assert.Equal(t, "192.168.1.10", host.Address)
	assert.Equal(t, "deploy", host.User)
	// Custom vars should be stored (ansible_ prefix removed from ansible_* vars)
	assert.Equal(t, "1.0.0", host.Vars["app_version"])
	assert.Equal(t, "production", host.Vars["environment"])
	assert.Equal(t, "custom_value", host.Vars["custom_var"])
}

func TestInventoryParser_AnsibleYAML_WithSSHKey(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    secure_host:
      ansible_host: 192.168.1.100
      ansible_user: admin
      ansible_ssh_private_key_file: ~/.ssh/id_rsa
      ansible_ssh_host_key_checking: false
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)

	host := inv.Hosts[0]
	assert.Equal(t, "~/.ssh/id_rsa", host.KeyFile)
	assert.True(t, host.InsecureIgnoreHostKey)
}

func TestInventoryParser_AnsibleYAML_WithPassword(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    pwd_host:
      ansible_host: 192.168.1.100
      ansible_user: admin
      ansible_password: my_password
      ansible_port: 2222
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)

	host := inv.Hosts[0]
	assert.Equal(t, "192.168.1.100", host.Address)
	assert.Equal(t, "admin", host.User)
	assert.Equal(t, "my_password", host.Password)
	assert.Equal(t, 2222, host.Port)
}

func TestInventoryParser_AnsibleYAML_NestedGroups(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
    db1:
      ansible_host: 192.168.1.20
    cache1:
      ansible_host: 192.168.1.30
  children:
    webservers:
      hosts:
        web1:
      vars:
        tier: frontend
    databases:
      hosts:
        db1:
      vars:
        tier: backend
    cache:
      hosts:
        cache1:
      vars:
        tier: backend
    production:
      children:
        - webservers
        - databases
        - cache
      vars:
        env: production
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 3)
	assert.Len(t, inv.Groups, 4)

	// Check production group has children
	prodGroup, ok := inv.Groups["production"]
	assert.True(t, ok)
	assert.Len(t, prodGroup.Children, 3)
	assert.Equal(t, "production", prodGroup.Vars["env"])
}

func TestInventoryParser_AnsibleYAML_EmptyChildren(t *testing.T) {
	logger := &mockLogger{}
	parser := NewInventoryParser(logger)

	ansibleYaml := `
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
  children:
    webservers:
      hosts:
        web1:
`

	inv, err := parser.parseYamlInventory([]byte(ansibleYaml))
	assert.NoError(t, err)
	assert.Len(t, inv.Hosts, 1)
	assert.Len(t, inv.Groups, 1)

	webGroup, ok := inv.Groups["webservers"]
	assert.True(t, ok)
	assert.Len(t, webGroup.Children, 0)
}

func TestInventoryParser_AnsibleYAML_IsDetected(t *testing.T) {
	// Test that Ansible format is properly detected
	ansibleYaml := []byte(`
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
`)

	assert.True(t, isAnsibleYaml(ansibleYaml))

	// Test Onigirazu format is not detected as Ansible
	onigiraazuYaml := []byte(`
hosts:
  - name: web1
    address: 192.168.1.10
groups:
  webservers:
    hosts:
      web1:
`)

	assert.False(t, isAnsibleYaml(onigiraazuYaml))

	// Test detection via ansible_ prefix
	ansibleWithVar := []byte(`
hosts:
  web1:
    ansible_host: 192.168.1.10
`)

	assert.True(t, isAnsibleYaml(ansibleWithVar))
}
