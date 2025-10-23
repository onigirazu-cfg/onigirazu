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

// TestCopyModule_Validate tests the Validate method with comprehensive test cases
func TestCopyModule_Validate(t *testing.T) {
	module := NewCopyModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
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
			name: "valid with src - /etc/hosts",
			args: map[string]interface{}{
				"dest": "/tmp/test.txt",
				"src":  "/etc/hosts",
			},
			wantErr: false,
		},
		{
			name: "valid with content and mode",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"mode":    "0644",
			},
			wantErr: false,
		},
		{
			name: "valid with content and owner/group",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"owner":   "root",
				"group":   "wheel",
			},
			wantErr: false,
		},
		{
			name: "valid with backup option",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"backup":  true,
			},
			wantErr: false,
		},
		{
			name: "missing dest",
			args: map[string]interface{}{
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "'dest' is required and must be a non-empty string",
		},
		{
			name: "empty dest",
			args: map[string]interface{}{
				"dest":    "",
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "'dest' is required and must be a non-empty string",
		},
		{
			name: "dest not a string",
			args: map[string]interface{}{
				"dest":    123,
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "'dest' is required and must be a non-empty string",
		},
		{
			name: "missing src and content",
			args: map[string]interface{}{
				"dest": "/tmp/test.txt",
			},
			wantErr: true,
			errMsg:  "either 'src' or 'content' must be specified",
		},
		{
			name: "both src and content",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"src":     "/etc/hosts",
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "'src' and 'content' are mutually exclusive",
		},
		{
			name: "non-existent src file",
			args: map[string]interface{}{
				"dest": "/tmp/test.txt",
				"src":  "/non/existent/file/path/that/does/not/exist",
			},
			wantErr: true,
			errMsg:  "source file '/non/existent/file/path/that/does/not/exist' does not exist",
		},
		{
			name: "invalid mode - too short",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"mode":    "64",
			},
			wantErr: true,
			errMsg:  "invalid mode '64': mode should be 3-4 digit octal string",
		},
		{
			name: "invalid mode - too long",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"mode":    "06444",
			},
			wantErr: true,
			errMsg:  "invalid mode '06444': mode should be 3-4 digit octal string",
		},
		{
			name: "valid mode - 3 digits",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"mode":    "644",
			},
			wantErr: false,
		},
		{
			name: "valid mode - 4 digits",
			args: map[string]interface{}{
				"dest":    "/tmp/test.txt",
				"content": "test content",
				"mode":    "0755",
			},
			wantErr: false,
		},
		{
			name: "valid with remote_src and non-existent file (remote validation deferred)",
			args: map[string]interface{}{
				"dest":       "/tmp/test.txt",
				"src":        "/remote/non/existent/file",
				"remote_src": true,
			},
			wantErr: false,
		},
		{
			name: "valid with remote_src and content (no src needed)",
			args: map[string]interface{}{
				"dest":       "/tmp/test.txt",
				"content":    "test content",
				"remote_src": true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestCopyModule_GetDescription tests the GetDescription method
func TestCopyModule_GetDescription(t *testing.T) {
	module := NewCopyModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Error("GetDescription() should not return empty string")
	}

	expectedDesc := "Copy files to remote locations"
	if desc != expectedDesc {
		t.Errorf("GetDescription() = %q, want %q", desc, expectedDesc)
	}
}

// TestCopyModule_GetName tests the GetName method
func TestCopyModule_GetName(t *testing.T) {
	module := NewCopyModule()

	name := module.GetName()
	if name != "copy" {
		t.Errorf("GetName() = %q, want %q", name, "copy")
	}
}

// TestCopyModule_EdgeCases tests edge cases for copy operations
func TestCopyModule_EdgeCases(t *testing.T) {
	module := NewCopyModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "empty_content",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "",
				"dest":    "/tmp/test.txt",
			},
			wantErr: false,
		},
		{
			name: "content_with_special_chars",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "Line1\nLine2\n\tTabbed\nUnicode: 日本語",
				"dest":    "/tmp/test.txt",
			},
			wantErr: false,
		},
		{
			name: "mode_octal_string",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"mode":    "0755",
			},
			wantErr: false,
		},
		{
			name: "mode_numeric",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"mode":    0755,
			},
			wantErr: false,
		},
		{
			name: "with_owner",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"owner":   "root",
			},
			wantErr: false,
		},
		{
			name: "with_group",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"group":   "root",
			},
			wantErr: false,
		},
		{
			name: "with_both_owner_and_group",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"owner":   "root",
				"group":   "root",
			},
			wantErr: false,
		},
		{
			name: "backup_enabled",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"backup":  true,
			},
			wantErr: false,
		},
		{
			name: "force_disabled",
			args: map[string]interface{}{
				"name":    "copy_task",
				"content": "test",
				"dest":    "/tmp/test.txt",
				"force":   false,
			},
			wantErr: false,
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

// TestCopyModule_ValidateMissingRequired tests validation with missing required args
func TestCopyModule_ValidateMissingRequired(t *testing.T) {
	module := NewCopyModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "missing_dest",
			args: map[string]interface{}{
				"name":    "task",
				"content": "test",
			},
			wantErr: true,
		},
		{
			name: "missing_both_src_and_content",
			args: map[string]interface{}{
				"name": "task",
				"dest": "/tmp/test.txt",
			},
			wantErr: true,
		},
		{
			name: "src_and_content_both",
			args: map[string]interface{}{
				"name":    "task",
				"src":     "/tmp/source.txt",
				"content": "test",
				"dest":    "/tmp/test.txt",
			},
			wantErr: true,
		},
		{
			name: "valid_content_only",
			args: map[string]interface{}{
				"name":    "task",
				"content": "test",
				"dest":    "/tmp/test.txt",
			},
			wantErr: false,
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
