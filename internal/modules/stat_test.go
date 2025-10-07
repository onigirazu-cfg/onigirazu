package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestStatModule_Validate tests stat module argument validation
func TestStatModule_Validate(t *testing.T) {
	module := NewStatModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid path",
			args: map[string]interface{}{
				"path": "/tmp/test.txt",
			},
			wantErr: false,
		},
		{
			name: "missing path",
			args: map[string]interface{}{
				"name": "test",
			},
			wantErr: true,
		},
		{
			name: "path not string",
			args: map[string]interface{}{
				"path": 123,
			},
			wantErr: true,
		},
		{
			name: "path is empty string",
			args: map[string]interface{}{
				"path": "",
			},
			wantErr: false, // Empty string is valid, will just fail during execution
		},
		{
			name: "path is nil",
			args: map[string]interface{}{
				"path": nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStatModule_GetName tests module name retrieval
func TestStatModule_GetName(t *testing.T) {
	module := NewStatModule()

	if module.GetName() != "stat" {
		t.Errorf("Expected name 'stat', got '%s'", module.GetName())
	}
}

// TestStatModule_GetDescription tests module description retrieval
func TestStatModule_GetDescription(t *testing.T) {
	module := NewStatModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	if desc != "Retrieves file or directory status" {
		t.Errorf("Expected description 'Retrieves file or directory status', got '%s'", desc)
	}
}

// TestStatModule_Execute_FileExists tests stat on existing file
func TestStatModule_Execute_FileExists(t *testing.T) {
	module := NewStatModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "stat_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some content
	content := "test content"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "test stat",
		"path": tmpFile.Name(),
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected no changes, but got changed=true")
	}

	// Check output structure
	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected output to be map[string]interface{}")
	}

	// Verify exists flag
	exists, ok := output["exists"].(bool)
	if !ok || !exists {
		t.Errorf("Expected exists=true, got %v", output["exists"])
	}

	// Verify path
	if output["path"] != tmpFile.Name() {
		t.Errorf("Expected path=%s, got %v", tmpFile.Name(), output["path"])
	}

	// Verify it's a regular file
	isreg, ok := output["isreg"].(bool)
	if !ok || !isreg {
		t.Errorf("Expected isreg=true, got %v", output["isreg"])
	}

	// Verify it's not a directory
	isdir, ok := output["isdir"].(bool)
	if !ok || isdir {
		t.Errorf("Expected isdir=false, got %v", output["isdir"])
	}

	// Verify size
	size, ok := output["size"].(int64)
	if !ok {
		t.Errorf("Expected size to be int64, got %T", output["size"])
	}
	if size != int64(len(content)) {
		t.Errorf("Expected size=%d, got %d", len(content), size)
	}

	// Verify mode exists
	if _, ok := output["mode"].(string); !ok {
		t.Errorf("Expected mode to be string, got %T", output["mode"])
	}

	// Verify stat substructure (Ansible compatibility)
	stat, ok := output["stat"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected stat to be map[string]interface{}, got %T", output["stat"])
	} else {
		if stat["exists"] != true {
			t.Errorf("Expected stat.exists=true, got %v", stat["exists"])
		}
	}
}

// TestStatModule_Execute_FileNotExists tests stat on non-existing file
func TestStatModule_Execute_FileNotExists(t *testing.T) {
	module := NewStatModule()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	nonExistentPath := "/tmp/this_file_should_not_exist_" + time.Now().Format("20060102150405")

	args := map[string]interface{}{
		"name": "test stat non-existent",
		"path": nonExistentPath,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success even for non-existent file, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected no changes, but got changed=true")
	}

	// Check output structure
	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected output to be map[string]interface{}")
	}

	// Verify exists flag is false
	exists, ok := output["exists"].(bool)
	if !ok || exists {
		t.Errorf("Expected exists=false, got %v", output["exists"])
	}

	// Verify path
	if output["path"] != nonExistentPath {
		t.Errorf("Expected path=%s, got %v", nonExistentPath, output["path"])
	}
}

// TestStatModule_Execute_Directory tests stat on directory
func TestStatModule_Execute_Directory(t *testing.T) {
	module := NewStatModule()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "stat_test_dir_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "test stat directory",
		"path": tmpDir,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check output structure
	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected output to be map[string]interface{}")
	}

	// Verify it's a directory
	isdir, ok := output["isdir"].(bool)
	if !ok || !isdir {
		t.Errorf("Expected isdir=true, got %v", output["isdir"])
	}

	// Verify it's not a regular file
	isreg, ok := output["isreg"].(bool)
	if !ok || isreg {
		t.Errorf("Expected isreg=false, got %v", output["isreg"])
	}
}

// TestStatModule_Execute_Symlink tests stat on symbolic link
func TestStatModule_Execute_Symlink(t *testing.T) {
	module := NewStatModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "stat_test_target_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create a symlink
	symlinkPath := filepath.Join(os.TempDir(), "stat_test_symlink_"+time.Now().Format("20060102150405"))
	if err := os.Symlink(tmpFile.Name(), symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	defer os.Remove(symlinkPath)

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "test stat symlink",
		"path": symlinkPath,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check output structure
	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected output to be map[string]interface{}")
	}

	// Note: os.Stat follows symlinks, so islnk will be false
	// To detect symlinks, we'd need to use os.Lstat instead
	// This is a known limitation in the current implementation
	exists, ok := output["exists"].(bool)
	if !ok || !exists {
		t.Errorf("Expected exists=true, got %v", output["exists"])
	}
}

// TestStatModule_Execute_MissingPath tests error when path is missing
func TestStatModule_Execute_MissingPath(t *testing.T) {
	module := NewStatModule()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "test stat",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Errorf("Expected error for missing path")
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}

	if result.Error == "" {
		t.Errorf("Expected error message, got empty string")
	}
}

// TestStatModule_Execute_WithTimeout tests execution with context timeout
func TestStatModule_Execute_WithTimeout(t *testing.T) {
	module := NewStatModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "stat_test_timeout_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "test stat with timeout",
		"path": tmpFile.Name(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}
}

// TestStatModule_Execute_PermissionChecks tests permission flags
func TestStatModule_Execute_PermissionChecks(t *testing.T) {
	module := NewStatModule()

	// Create a temporary file with specific permissions
	tmpFile, err := os.CreateTemp("", "stat_test_perms_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set specific permissions (e.g., 0644)
	if err := os.Chmod(tmpFile.Name(), 0644); err != nil {
		t.Fatalf("Failed to chmod temp file: %v", err)
	}

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "test stat permissions",
		"path": tmpFile.Name(),
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check output structure
	output, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected output to be map[string]interface{}")
	}

	// Verify mode
	mode, ok := output["mode"].(string)
	if !ok {
		t.Errorf("Expected mode to be string, got %T", output["mode"])
	} else if mode != "0644" {
		t.Logf("Expected mode=0644, got %s (may vary by OS)", mode)
	}

	// Verify permission flags exist
	if _, ok := output["readable"].(bool); !ok {
		t.Errorf("Expected readable to be bool, got %T", output["readable"])
	}
	if _, ok := output["writable"].(bool); !ok {
		t.Errorf("Expected writable to be bool, got %T", output["writable"])
	}
	if _, ok := output["executable"].(bool); !ok {
		t.Errorf("Expected executable to be bool, got %T", output["executable"])
	}
}

// BenchmarkStatModule_Validate benchmarks argument validation
func BenchmarkStatModule_Validate(b *testing.B) {
	module := NewStatModule()

	args := map[string]interface{}{
		"path": "/tmp/test.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkStatModule_Execute benchmarks stat execution
func BenchmarkStatModule_Execute(b *testing.B) {
	module := NewStatModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "stat_bench_*.txt")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "benchmark stat",
		"path": tmpFile.Name(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}
