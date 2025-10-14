package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestCopyModuleValidate(t *testing.T) {
	module := NewCopyModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with content",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
			},
			wantErr: false,
		},
		{
			name: "valid with src",
			args: map[string]interface{}{
				"dest": "/tmp/test.txt",
				"src":  "/etc/hosts", // assuming this exists
			},
			wantErr: false,
		},
		{
			name: "missing dest",
			args: map[string]interface{}{
				"content": "test content",
			},
			wantErr: true,
		},
		{
			name: "missing src and content",
			args: map[string]interface{}{
				"dest": "/tmp/test.txt",
			},
			wantErr: true,
		},
		{
			name: "both src and content",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"src":     "/etc/hosts",
				"content": "test content",
			},
			wantErr: true,
		},
		{
			name: "non-existent src",
			args: map[string]interface{}{
				"dest": "/tmp/test.txt",
				"src":  "/non/existent/file",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyModule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCopyModuleExecuteWithContent(t *testing.T) {
	module := NewCopyModule()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "copy_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}
	args := map[string]interface{}{
		"dest":    destPath,
		"content": testContent,
	}

	result, err := module.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true for new file")
	}

	// Verify file was created with correct content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", testContent, string(content))
	}
}

func TestCopyModuleExecuteIdempotency(t *testing.T) {
	module := NewCopyModule()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "copy_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}
	args := map[string]interface{}{
		"dest":    destPath,
		"content": testContent,
	}

	// First execution - should create file
	result1, err := module.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatalf("First execute failed: %v", err)
	}

	if !result1.Success || !result1.Changed {
		t.Errorf("First execution should succeed and change")
	}

	// Second execution - should be idempotent
	result2, err := module.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatalf("Second execute failed: %v", err)
	}

	if !result2.Success {
		t.Errorf("Second execution should succeed")
	}

	if result2.Changed {
		t.Errorf("Second execution should not change (idempotent)")
	}
}

func TestCopyModuleExecuteWithSrc(t *testing.T) {
	module := NewCopyModule()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "copy_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	testContent := "Source file content"
	if err := os.WriteFile(srcPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	destPath := filepath.Join(tmpDir, "dest.txt")

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}
	args := map[string]interface{}{
		"src":  srcPath,
		"dest": destPath,
	}

	result, err := module.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true for new file")
	}

	// Verify file was copied with correct content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", testContent, string(content))
	}
}

func TestCopyModuleExecuteWithBackup(t *testing.T) {
	module := NewCopyModule()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "copy_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test.txt")
	originalContent := "Original content"
	newContent := "New content"

	// Create original file
	if err := os.WriteFile(destPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}
	args := map[string]interface{}{
		"dest":    destPath,
		"content": newContent,
		"backup":  true,
	}

	result, err := module.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true for updated file")
	}

	// Check if backup was created
	backupFile, ok := result.Output["backup_file"].(string)
	if !ok {
		t.Errorf("Expected backup_file in output")
	} else {
		// Verify backup contains original content
		backupContent, err := os.ReadFile(backupFile)
		if err != nil {
			t.Errorf("Failed to read backup file: %v", err)
		} else if string(backupContent) != originalContent {
			t.Errorf("Backup content mismatch. Expected: %s, Got: %s", originalContent, string(backupContent))
		}
	}

	// Verify main file has new content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	if string(content) != newContent {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", newContent, string(content))
	}
}
