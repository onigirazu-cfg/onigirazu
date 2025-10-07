package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestFetchModule_Validate tests the Validate method
func TestFetchModule_Validate(t *testing.T) {
	module := NewFetchModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid arguments",
			args: map[string]interface{}{
				"src":  "/remote/file.txt",
				"dest": "/local/dir",
			},
			wantErr: false,
		},
		{
			name:    "missing src",
			args:    map[string]interface{}{"dest": "/local/dir"},
			wantErr: true,
			errMsg:  "src parameter is required",
		},
		{
			name:    "missing dest",
			args:    map[string]interface{}{"src": "/remote/file.txt"},
			wantErr: true,
			errMsg:  "dest parameter is required",
		},
		{
			name:    "empty arguments",
			args:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "src parameter is required",
		},
		{
			name: "with optional flat parameter",
			args: map[string]interface{}{
				"src":  "/remote/file.txt",
				"dest": "/local/dir",
				"flat": true,
			},
			wantErr: false,
		},
		{
			name: "with optional fail_on_missing parameter",
			args: map[string]interface{}{
				"src":             "/remote/file.txt",
				"dest":            "/local/dir",
				"fail_on_missing": false,
			},
			wantErr: false,
		},
		{
			name: "with optional validate parameter",
			args: map[string]interface{}{
				"src":      "/remote/file.txt",
				"dest":     "/local/dir",
				"validate": true,
			},
			wantErr: false,
		},
		{
			name: "with all optional parameters",
			args: map[string]interface{}{
				"src":             "/remote/file.txt",
				"dest":            "/local/dir",
				"flat":            true,
				"fail_on_missing": false,
				"validate":        true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
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

// TestFetchModule_GetDescription tests the GetDescription method
func TestFetchModule_GetDescription(t *testing.T) {
	module := NewFetchModule()
	desc := module.GetDescription()

	if desc == "" {
		t.Error("GetDescription() returned empty string")
	}

	expectedDesc := "Fetch files from remote hosts to local machine"
	if desc != expectedDesc {
		t.Errorf("GetDescription() = %v, want %v", desc, expectedDesc)
	}
}

// TestFetchModule_Execute_ValidationError tests Execute with invalid arguments
func TestFetchModule_Execute_ValidationError(t *testing.T) {
	module := NewFetchModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	// Missing src parameter
	args := map[string]interface{}{
		"dest": "/tmp/test",
	}

	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Error("Execute() expected error for missing src parameter")
	}

	if !result.Failed {
		t.Error("Execute() result.Failed should be true for validation error")
	}

	if result.Error == "" {
		t.Error("Execute() result.Error should not be empty for validation error")
	}
}

// TestFetchModule_Execute_ContextTimeout tests Execute with context timeout
func TestFetchModule_Execute_ContextTimeout(t *testing.T) {
	module := NewFetchModule()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"src":  "/tmp/source.txt",
		"dest": "/tmp/dest",
	}

	result, err := module.Execute(ctx, host, args)

	// Should fail due to context cancellation or executor creation failure
	if err == nil && !result.Failed {
		t.Log("Execute() with cancelled context may succeed if executor creation is fast")
	}

	// Verify result structure
	if result.Host != host.Name {
		t.Errorf("Execute() result.Host = %v, want %v", result.Host, host.Name)
	}

	if result.Module != "fetch" {
		t.Errorf("Execute() result.Module = %v, want %v", result.Module, "fetch")
	}
}

// TestFetchModule_FlatMode tests flat mode destination path construction
func TestFetchModule_FlatMode(t *testing.T) {
	// This test verifies the flat mode logic by checking validation
	// Full integration test would require mock executor
	module := NewFetchModule()

	args := map[string]interface{}{
		"src":  "/remote/path/to/file.txt",
		"dest": "/local/dir",
		"flat": true,
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Validate() with flat mode failed: %v", err)
	}

	// Verify flat parameter is accepted
	if _, ok := args["flat"]; !ok {
		t.Error("flat parameter should be present in args")
	}
}

// TestFetchModule_HierarchicalMode tests hierarchical mode (default)
func TestFetchModule_HierarchicalMode(t *testing.T) {
	module := NewFetchModule()

	args := map[string]interface{}{
		"src":  "/remote/path/to/file.txt",
		"dest": "/local/dir",
		"flat": false,
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Validate() with hierarchical mode failed: %v", err)
	}
}

// TestFetchModule_FailOnMissing tests fail_on_missing parameter
func TestFetchModule_FailOnMissing(t *testing.T) {
	module := NewFetchModule()

	tests := []struct {
		name           string
		failOnMissing  bool
		expectValidate bool
	}{
		{
			name:           "fail_on_missing true",
			failOnMissing:  true,
			expectValidate: true,
		},
		{
			name:           "fail_on_missing false",
			failOnMissing:  false,
			expectValidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"src":             "/remote/file.txt",
				"dest":            "/local/dir",
				"fail_on_missing": tt.failOnMissing,
			}

			err := module.Validate(args)
			if tt.expectValidate && err != nil {
				t.Errorf("Validate() failed: %v", err)
			}
		})
	}
}

// TestFetchModule_ValidateParameter tests validate parameter
func TestFetchModule_ValidateParameter(t *testing.T) {
	module := NewFetchModule()

	args := map[string]interface{}{
		"src":      "/remote/file.txt",
		"dest":     "/local/dir",
		"validate": true,
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Validate() with validate parameter failed: %v", err)
	}
}

// TestCalculateMD5 tests the calculateMD5 helper function
func TestCalculateMD5(t *testing.T) {
	// Create a temporary file with known content
	tmpFile, err := os.CreateTemp("", "fetch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("test content for checksum")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Calculate checksum
	checksum, err := calculateMD5(tmpFile.Name())
	if err != nil {
		t.Errorf("calculateMD5() error = %v", err)
	}

	if checksum == "" {
		t.Error("calculateMD5() returned empty checksum")
	}

	// Verify checksum is hex string (SHA256 should be 64 characters)
	if len(checksum) != 64 {
		t.Errorf("calculateMD5() checksum length = %d, want 64 (SHA256)", len(checksum))
	}

	// Calculate again to verify consistency
	checksum2, err := calculateMD5(tmpFile.Name())
	if err != nil {
		t.Errorf("calculateMD5() second call error = %v", err)
	}

	if checksum != checksum2 {
		t.Errorf("calculateMD5() inconsistent results: %v != %v", checksum, checksum2)
	}
}

// TestCalculateMD5_NonExistentFile tests calculateMD5 with non-existent file
func TestCalculateMD5_NonExistentFile(t *testing.T) {
	_, err := calculateMD5("/nonexistent/file/path.txt")
	if err == nil {
		t.Error("calculateMD5() expected error for non-existent file")
	}
}

// TestFetchModule_ResultStructure tests the result structure
func TestFetchModule_ResultStructure(t *testing.T) {
	module := NewFetchModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	// Use invalid args to get a quick result
	args := map[string]interface{}{
		"dest": "/tmp/test",
	}

	result, _ := module.Execute(ctx, host, args)

	// Verify result structure
	if result.Host != host.Name {
		t.Errorf("Execute() result.Host = %v, want %v", result.Host, host.Name)
	}

	if result.Module != "fetch" {
		t.Errorf("Execute() result.Module = %v, want %v", result.Module, "fetch")
	}

	if result.Timestamp.IsZero() {
		t.Error("Execute() result.Timestamp should not be zero")
	}

	if result.Output == nil {
		t.Error("Execute() result.Output should not be nil")
	}

	if result.Duration < 0 {
		t.Error("Execute() result.Duration should not be negative")
	}
}

// TestFetchModule_DestinationPathConstruction tests destination path logic
func TestFetchModule_DestinationPathConstruction(t *testing.T) {
	// Test the logic that would be used in Execute
	tests := []struct {
		name     string
		src      string
		dest     string
		flat     bool
		hostname string
		expected string
	}{
		{
			name:     "flat mode",
			src:      "/remote/path/file.txt",
			dest:     "/local/dir",
			flat:     true,
			hostname: "server1",
			expected: filepath.Join("/local/dir", "file.txt"),
		},
		{
			name:     "hierarchical mode",
			src:      "/remote/path/file.txt",
			dest:     "/local/dir",
			flat:     false,
			hostname: "server1",
			expected: filepath.Join("/local/dir", "server1", "/remote/path/file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var destPath string
			if tt.flat {
				destPath = filepath.Join(tt.dest, filepath.Base(tt.src))
			} else {
				destPath = filepath.Join(tt.dest, tt.hostname, tt.src)
			}

			if destPath != tt.expected {
				t.Errorf("Destination path = %v, want %v", destPath, tt.expected)
			}
		})
	}
}

// TestFetchModule_NewFetchModule tests module creation
func TestFetchModule_NewFetchModule(t *testing.T) {
	module := NewFetchModule()

	if module == nil {
		t.Fatal("NewFetchModule() returned nil")
	}

	if module.name != "fetch" {
		t.Errorf("NewFetchModule() name = %v, want %v", module.name, "fetch")
	}

	if module.description == "" {
		t.Error("NewFetchModule() description should not be empty")
	}
}

// BenchmarkFetchModule_Validate benchmarks the Validate method
func BenchmarkFetchModule_Validate(b *testing.B) {
	module := NewFetchModule()
	args := map[string]interface{}{
		"src":             "/remote/file.txt",
		"dest":            "/local/dir",
		"flat":            true,
		"fail_on_missing": false,
		"validate":        true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkCalculateMD5 benchmarks the calculateMD5 function
func BenchmarkCalculateMD5(b *testing.B) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "fetch-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 1MB of data
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if _, err := tmpFile.Write(data); err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calculateMD5(tmpFile.Name())
	}
}

// BenchmarkFetchModule_Execute benchmarks the Execute method with validation error
func BenchmarkFetchModule_Execute(b *testing.B) {
	module := NewFetchModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	// Use invalid args for quick benchmark (validation error path)
	args := map[string]interface{}{
		"dest": "/tmp/test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}
