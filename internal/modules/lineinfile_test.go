package modules

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestLineinfileModule_Validate tests lineinfile module argument validation
func TestLineinfileModule_Validate(t *testing.T) {
	module := NewLineinfileModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid minimal args",
			args: map[string]interface{}{
				"name": "test_lineinfile",
				"path": "/tmp/test.txt",
				"line": "test line",
			},
			wantErr: false,
		},
		{
			name: "valid with state present",
			args: map[string]interface{}{
				"name":  "test_lineinfile",
				"path":  "/tmp/test.txt",
				"line":  "test line",
				"state": "present",
			},
			wantErr: false,
		},
		{
			name: "valid with state absent",
			args: map[string]interface{}{
				"name":  "test_lineinfile",
				"path":  "/tmp/test.txt",
				"line":  "test line",
				"state": "absent",
			},
			wantErr: false,
		},
		{
			name: "missing path",
			args: map[string]interface{}{
				"line": "test line",
			},
			wantErr: true,
		},
		{
			name: "missing line",
			args: map[string]interface{}{
				"path": "/tmp/test.txt",
			},
			wantErr: true,
		},
		{
			name: "path not string",
			args: map[string]interface{}{
				"path": 123,
				"line": "test line",
			},
			wantErr: true,
		},
		{
			name: "line not string",
			args: map[string]interface{}{
				"path": "/tmp/test.txt",
				"line": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid state",
			args: map[string]interface{}{
				"path":  "/tmp/test.txt",
				"line":  "test line",
				"state": "invalid",
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

// TestLineinfileModule_GetName tests module name retrieval
func TestLineinfileModule_GetName(t *testing.T) {
	module := NewLineinfileModule()

	if module.GetName() != "lineinfile" {
		t.Errorf("Expected name 'lineinfile', got '%s'", module.GetName())
	}
}

// TestLineinfileModule_GetDescription tests module description retrieval
func TestLineinfileModule_GetDescription(t *testing.T) {
	module := NewLineinfileModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	if desc != "Manages lines in text files" {
		t.Errorf("Expected description 'Manages lines in text files', got '%s'", desc)
	}
}

// TestLineinfileModule_Execute_AddLine tests adding a line to a file
func TestLineinfileModule_Execute_AddLine(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "line1\nline2\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "test add line",
		"path":  tmpFile.Name(),
		"line":  "new line",
		"state": "present",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Verify file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "new line") {
		t.Errorf("Expected file to contain 'new line', got: %s", string(content))
	}
}

// TestLineinfileModule_Execute_LineExists tests when line already exists
func TestLineinfileModule_Execute_LineExists(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content with the line we want to add
	initialContent := "line1\nexisting line\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "test line exists",
		"path":  tmpFile.Name(),
		"line":  "existing line",
		"state": "present",
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
		t.Errorf("Expected changed=false (line already exists), got true")
	}
}

// TestLineinfileModule_Execute_RemoveLine tests removing a line from a file
func TestLineinfileModule_Execute_RemoveLine(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "line1\nline to remove\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "test remove line",
		"path":  tmpFile.Name(),
		"line":  "line to remove",
		"state": "absent",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Verify file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if strings.Contains(string(content), "line to remove") {
		t.Errorf("Expected file to not contain 'line to remove', got: %s", string(content))
	}
}

// TestLineinfileModule_Execute_WithRegexp tests using regexp to match lines
func TestLineinfileModule_Execute_WithRegexp(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "line1\nold value: 123\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":   "test regexp",
		"path":   tmpFile.Name(),
		"line":   "new value: 456",
		"regexp": "^old value:",
		"state":  "present",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Verify file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "new value: 456") {
		t.Errorf("Expected file to contain 'new value: 456', got: %s", string(content))
	}

	if strings.Contains(string(content), "old value: 123") {
		t.Errorf("Expected file to not contain 'old value: 123', got: %s", string(content))
	}
}

// TestLineinfileModule_Execute_CreateFile tests creating a new file
func TestLineinfileModule_Execute_CreateFile(t *testing.T) {
	module := NewLineinfileModule()

	// Use a non-existent file path
	tmpPath := os.TempDir() + "/lineinfile_test_create_" + time.Now().Format("20060102150405") + ".txt"
	defer os.Remove(tmpPath)

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":   "test create file",
		"path":   tmpPath,
		"line":   "new line",
		"create": true,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Verify file was created
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		t.Errorf("Expected file to be created, but it doesn't exist")
	}

	// Verify file content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "new line") {
		t.Errorf("Expected file to contain 'new line', got: %s", string(content))
	}
}

// TestLineinfileModule_Execute_NoCreate tests error when file doesn't exist and create=false
func TestLineinfileModule_Execute_NoCreate(t *testing.T) {
	module := NewLineinfileModule()

	// Use a non-existent file path
	tmpPath := os.TempDir() + "/lineinfile_test_nocreate_" + time.Now().Format("20060102150405") + ".txt"

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":   "test no create",
		"path":   tmpPath,
		"line":   "new line",
		"create": false,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Errorf("Expected error for non-existent file with create=false")
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}
}

// TestLineinfileModule_Execute_InsertAfter tests insertafter parameter
func TestLineinfileModule_Execute_InsertAfter(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "line1\nmarker line\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":        "test insertafter",
		"path":        tmpFile.Name(),
		"line":        "inserted line",
		"insertafter": "^marker",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Verify file content and order
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	foundMarker := false
	foundInserted := false
	for i, line := range lines {
		if strings.Contains(line, "marker line") {
			foundMarker = true
			// Check if next line is the inserted line
			if i+1 < len(lines) && strings.Contains(lines[i+1], "inserted line") {
				foundInserted = true
			}
		}
	}

	if !foundMarker {
		t.Errorf("Expected to find marker line")
	}
	if !foundInserted {
		t.Errorf("Expected to find inserted line after marker")
	}
}

// TestLineinfileModule_Execute_InsertBefore tests insertbefore parameter
func TestLineinfileModule_Execute_InsertBefore(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "line1\nmarker line\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":         "test insertbefore",
		"path":         tmpFile.Name(),
		"line":         "inserted line",
		"insertbefore": "^marker",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Verify file content and order
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	foundMarker := false
	foundInserted := false
	for i, line := range lines {
		if strings.Contains(line, "inserted line") {
			foundInserted = true
			// Check if next line is the marker line
			if i+1 < len(lines) && strings.Contains(lines[i+1], "marker line") {
				foundMarker = true
			}
		}
	}

	if !foundMarker {
		t.Errorf("Expected to find marker line")
	}
	if !foundInserted {
		t.Errorf("Expected to find inserted line before marker")
	}
}

// TestLineinfileModule_Execute_InvalidRegexp tests error with invalid regexp
func TestLineinfileModule_Execute_InvalidRegexp(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":   "test invalid regexp",
		"path":   tmpFile.Name(),
		"line":   "new line",
		"regexp": "[invalid(regexp",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Errorf("Expected error for invalid regexp")
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}

	if !strings.Contains(result.Error, "invalid regexp") {
		t.Errorf("Expected error message to contain 'invalid regexp', got: %s", result.Error)
	}
}

// TestLineinfileModule_Execute_WithTimeout tests execution with context timeout
func TestLineinfileModule_Execute_WithTimeout(t *testing.T) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test with timeout",
		"path": tmpFile.Name(),
		"line": "new line",
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

// BenchmarkLineinfileModule_Validate benchmarks argument validation
func BenchmarkLineinfileModule_Validate(b *testing.B) {
	module := NewLineinfileModule()

	args := map[string]interface{}{
		"path": "/tmp/test.txt",
		"line": "test line",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkLineinfileModule_Execute benchmarks lineinfile execution
func BenchmarkLineinfileModule_Execute(b *testing.B) {
	module := NewLineinfileModule()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "lineinfile_bench_*.txt")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "line1\nline2\nline3\n"
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "benchmark lineinfile",
		"path": tmpFile.Name(),
		"line": "benchmark line",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}
