package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/core"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
)

func TestIntegrationSimplePlaybook(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "onigirazu-integration-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test playbook
	playbookContent := `name: "Integration Test Playbook"

plays:
  - name: "Test basic commands"
    hosts: "localhost"

    tasks:
      - name: "Check current directory"
        module: "command"
        args:
          command: "pwd"
          shell: false

      - name: "Create test file"
        module: "file"
        args:
          path: "` + filepath.Join(tempDir, "test.txt") + `"
          state: "present"
          content: "Hello from Onigirazu!"

      - name: "Verify file exists"
        module: "command"
        args:
          command: "ls -la ` + filepath.Join(tempDir, "test.txt") + `"
          shell: true
`

	playbookPath := filepath.Join(tempDir, "test-playbook.yml")
	err = os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	// Create simple inventory file
	inventoryContent := `hosts:
  - name: localhost
    address: localhost
    user: ` + os.Getenv("USER") + `
    port: 22
groups:
  localhost:
    hosts:
      localhost:
        name: localhost
        address: localhost
        user: ` + os.Getenv("USER") + `
        port: 22
`
	inventoryPath := filepath.Join(tempDir, "inventory.ini")
	err = os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	// Initialize components
	cfg := config.NewConfig()
	cfg.LogLevel = "info"
	cfg.StateFile = filepath.Join(tempDir, "state.json")

	logger, err := logger.NewEnhancedLogger(cfg.LogLevel, "text", os.Stdout)
	require.NoError(t, err)

	// Create CoreEngine instance
	coreEngine := core.NewCoreEngine(logger)

	// Run playbook
	err = coreEngine.Run(playbookPath, inventoryPath, false, cfg.StateFile)
	assert.NoError(t, err)

	// Verify test file was created
	testFilePath := filepath.Join(tempDir, "test.txt")
	assert.FileExists(t, testFilePath)

	// Verify file content
	content, err := os.ReadFile(testFilePath)
	require.NoError(t, err)
	assert.Equal(t, "Hello from Onigirazu!", string(content))

	// Verify state file was created
	assert.FileExists(t, cfg.StateFile)
}

func TestIntegrationIdempotency(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "onigirazu-idempotency-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test playbook
	playbookContent := `name: "Idempotency Test Playbook"

plays:
  - name: "Test idempotent operations"
    hosts: "localhost"

    tasks:
      - name: "Create test file"
        module: "file"
        args:
          path: "` + filepath.Join(tempDir, "idempotent.txt") + `"
          state: "present"
          content: "Idempotent content"
`

	playbookPath := filepath.Join(tempDir, "idempotent-playbook.yml")
	err = os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	// Create simple inventory file
	inventoryContent := `hosts:
  - name: localhost
    address: localhost
    user: ` + os.Getenv("USER") + `
    port: 22
groups:
  localhost:
    hosts:
      localhost:
        name: localhost
        address: localhost
        user: ` + os.Getenv("USER") + `
        port: 22
`
	inventoryPath := filepath.Join(tempDir, "inventory.ini")
	err = os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	// Initialize components
	cfg := config.NewConfig()
	cfg.LogLevel = "info"
	cfg.StateFile = filepath.Join(tempDir, "state.json")

	logger, err := logger.NewEnhancedLogger(cfg.LogLevel, "text", os.Stdout)
	require.NoError(t, err)

	// Create CoreEngine instance
	coreEngine := core.NewCoreEngine(logger)

	// Run playbook first time
	err = coreEngine.Run(playbookPath, inventoryPath, false, cfg.StateFile)
	assert.NoError(t, err)

	// Get file modification time
	testFilePath := filepath.Join(tempDir, "idempotent.txt")
	info1, err := os.Stat(testFilePath)
	require.NoError(t, err)
	modTime1 := info1.ModTime()

	// Wait a bit to ensure different timestamps if file is modified
	time.Sleep(100 * time.Millisecond)

	// Run playbook second time (should be idempotent)
	err = coreEngine.Run(playbookPath, inventoryPath, false, cfg.StateFile)
	assert.NoError(t, err)

	// Check that file wasn't modified (idempotent behavior)
	info2, err := os.Stat(testFilePath)
	require.NoError(t, err)
	modTime2 := info2.ModTime()

	// File should not have been modified on second run
	assert.Equal(t, modTime1, modTime2, "File should not be modified on idempotent run")
}

func TestIntegrationErrorHandling(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "onigirazu-error-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test playbook with failing command
	playbookContent := `name: "Error Handling Test Playbook"

plays:
  - name: "Test error handling"
    hosts: "localhost"

    tasks:
      - name: "This command should fail"
        module: "command"
        args:
          command: "nonexistent-command-that-should-fail"
          shell: false
`

	playbookPath := filepath.Join(tempDir, "error-playbook.yml")
	err = os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	// Create simple inventory file
	inventoryContent := `hosts:
  - name: localhost
    address: localhost
    user: ` + os.Getenv("USER") + `
    port: 22
groups:
  localhost:
    hosts:
      localhost:
        name: localhost
        address: localhost
        user: ` + os.Getenv("USER") + `
        port: 22
`
	inventoryPath := filepath.Join(tempDir, "inventory.ini")
	err = os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	// Initialize components
	cfg := config.NewConfig()
	cfg.LogLevel = "info"
	cfg.StateFile = filepath.Join(tempDir, "state.json")

	logger, err := logger.NewEnhancedLogger(cfg.LogLevel, "text", os.Stdout)
	require.NoError(t, err)

	// Create CoreEngine instance
	coreEngine := core.NewCoreEngine(logger)

	// Run playbook - should fail gracefully
	err = coreEngine.Run(playbookPath, inventoryPath, false, cfg.StateFile)
	assert.Error(t, err, "Playbook should fail when command doesn't exist")
}

func TestIntegrationMultipleModules(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "onigirazu-multi-module-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test playbook using multiple modules
	playbookContent := `name: "Multi-Module Test Playbook"

plays:
  - name: "Test multiple modules"
    hosts: "localhost"

    tasks:
      - name: "Create directory"
        module: "file"
        args:
          path: "` + filepath.Join(tempDir, "testdir") + `"
          state: "directory"

      - name: "Create file in directory"
        module: "file"
        args:
          path: "` + filepath.Join(tempDir, "testdir", "file.txt") + `"
          state: "present"
          content: "Multi-module test"

      - name: "List directory contents"
        module: "command"
        args:
          command: "ls -la ` + filepath.Join(tempDir, "testdir") + `"
          shell: true

      - name: "Echo using shell module"
        module: "shell"
        args:
          command: "echo 'Shell module works!'"
`

	playbookPath := filepath.Join(tempDir, "multi-module-playbook.yml")
	err = os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	// Create simple inventory file
	inventoryContent := `hosts:
  - name: localhost
    address: localhost
    user: ` + os.Getenv("USER") + `
    port: 22
groups:
  localhost:
    hosts:
      localhost:
        name: localhost
        address: localhost
        user: ` + os.Getenv("USER") + `
        port: 22
`
	inventoryPath := filepath.Join(tempDir, "inventory.ini")
	err = os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	// Initialize components
	cfg := config.NewConfig()
	cfg.LogLevel = "info"
	cfg.StateFile = filepath.Join(tempDir, "state.json")

	logger, err := logger.NewEnhancedLogger(cfg.LogLevel, "text", os.Stdout)
	require.NoError(t, err)

	// Create CoreEngine instance
	coreEngine := core.NewCoreEngine(logger)

	// Run playbook
	err = coreEngine.Run(playbookPath, inventoryPath, false, cfg.StateFile)
	assert.NoError(t, err)

	// Verify directory was created
	testDirPath := filepath.Join(tempDir, "testdir")
	assert.DirExists(t, testDirPath)

	// Verify file was created in directory
	testFilePath := filepath.Join(tempDir, "testdir", "file.txt")
	assert.FileExists(t, testFilePath)

	// Verify file content
	content, err := os.ReadFile(testFilePath)
	require.NoError(t, err)
	assert.Equal(t, "Multi-module test", string(content))
}
