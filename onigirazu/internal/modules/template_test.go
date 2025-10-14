package modules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestTemplateModule_Validate tests template module argument validation
func TestTemplateModule_Validate(t *testing.T) {
	module := NewTemplateModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid minimal args",
			args: map[string]interface{}{
				"src":  "/tmp/template.j2",
				"dest": "/tmp/output.txt",
			},
			wantErr: false,
		},
		{
			name: "valid with all options",
			args: map[string]interface{}{
				"src":    "/tmp/template.j2",
				"dest":   "/tmp/output.txt",
				"mode":   "0644",
				"owner":  "root",
				"group":  "root",
				"backup": true,
				"vars": map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "missing src",
			args: map[string]interface{}{
				"dest": "/tmp/output.txt",
			},
			wantErr: true,
		},
		{
			name: "missing dest",
			args: map[string]interface{}{
				"src": "/tmp/template.j2",
			},
			wantErr: true,
		},
		{
			name: "src not string",
			args: map[string]interface{}{
				"src":  123,
				"dest": "/tmp/output.txt",
			},
			wantErr: true,
		},
		{
			name: "dest not string",
			args: map[string]interface{}{
				"src":  "/tmp/template.j2",
				"dest": 123,
			},
			wantErr: true,
		},
		{
			name: "empty src",
			args: map[string]interface{}{
				"src":  "",
				"dest": "/tmp/output.txt",
			},
			wantErr: true,
		},
		{
			name: "empty dest",
			args: map[string]interface{}{
				"src":  "/tmp/template.j2",
				"dest": "",
			},
			wantErr: true,
		},
		{
			name: "mode not string",
			args: map[string]interface{}{
				"src":  "/tmp/template.j2",
				"dest": "/tmp/output.txt",
				"mode": 644,
			},
			wantErr: true,
		},
		{
			name: "owner not string",
			args: map[string]interface{}{
				"src":   "/tmp/template.j2",
				"dest":  "/tmp/output.txt",
				"owner": 123,
			},
			wantErr: true,
		},
		{
			name: "group not string",
			args: map[string]interface{}{
				"src":   "/tmp/template.j2",
				"dest":  "/tmp/output.txt",
				"group": 123,
			},
			wantErr: true,
		},
		{
			name: "backup not boolean",
			args: map[string]interface{}{
				"src":    "/tmp/template.j2",
				"dest":   "/tmp/output.txt",
				"backup": "true",
			},
			wantErr: true,
		},
		{
			name: "vars not map",
			args: map[string]interface{}{
				"src":  "/tmp/template.j2",
				"dest": "/tmp/output.txt",
				"vars": "invalid",
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

// TestTemplateModule_GetName tests module name retrieval
func TestTemplateModule_GetName(t *testing.T) {
	module := NewTemplateModule()

	if module.GetName() != "template" {
		t.Errorf("Expected name 'template', got '%s'", module.GetName())
	}
}

// TestTemplateModule_GetDescription tests module description retrieval
func TestTemplateModule_GetDescription(t *testing.T) {
	module := NewTemplateModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	expectedDesc := "Processes Jinja2-like templates with advanced features and creates files"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

// TestTemplateModule_Execute_SimpleTemplate tests rendering a simple template
func TestTemplateModule_Execute_SimpleTemplate(t *testing.T) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := "Hello {{ name }}!"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test template",
		"src":  templatePath,
		"dest": outputPath,
		"vars": map[string]interface{}{
			"name": "World",
		},
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

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Expected output file to be created")
	}

	// Verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedContent := "Hello World!"
	if strings.TrimSpace(string(content)) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, strings.TrimSpace(string(content)))
	}
}

// TestTemplateModule_Execute_TemplateAlreadyUpToDate tests when template is already rendered
func TestTemplateModule_Execute_TemplateAlreadyUpToDate(t *testing.T) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := "Static content"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Pre-create output file with same content
	if err := os.WriteFile(outputPath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test template up to date",
		"src":  templatePath,
		"dest": outputPath,
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
		t.Errorf("Expected changed=false (file already up to date), got true")
	}
}

// TestTemplateModule_Execute_MissingSourceFile tests error when source file doesn't exist
func TestTemplateModule_Execute_MissingSourceFile(t *testing.T) {
	module := NewTemplateModule()

	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test missing source",
		"src":  nonExistentPath,
		"dest": outputPath,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Errorf("Expected error for missing source file")
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}

	if !strings.Contains(result.Error, "does not exist") {
		t.Errorf("Expected error message to contain 'does not exist', got: %s", result.Error)
	}
}

// TestTemplateModule_Execute_WithBackup tests backup functionality
func TestTemplateModule_Execute_WithBackup(t *testing.T) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := "New content"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Pre-create output file with different content
	originalContent := "Original content"
	if err := os.WriteFile(outputPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name":   "test backup",
		"src":    templatePath,
		"dest":   outputPath,
		"backup": true,
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

	// Verify backup file was created
	output := result.Output

	backupFile, ok := output["backup_file"].(string)
	if !ok || backupFile == "" {
		t.Errorf("Expected backup_file in output")
	} else {
		// Verify backup file exists and contains original content
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			t.Errorf("Expected backup file to exist: %s", backupFile)
		} else {
			backupContent, err := os.ReadFile(backupFile)
			if err != nil {
				t.Fatalf("Failed to read backup file: %v", err)
			}
			if string(backupContent) != originalContent {
				t.Errorf("Expected backup content '%s', got '%s'", originalContent, string(backupContent))
			}
		}
	}
}

// TestTemplateModule_Execute_CreateDirectory tests creating destination directory
func TestTemplateModule_Execute_CreateDirectory(t *testing.T) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "subdir", "nested", "output.txt")

	templateContent := "Test content"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test create directory",
		"src":  templatePath,
		"dest": outputPath,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Verify output file was created in nested directory
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Expected output file to be created in nested directory")
	}
}

// TestTemplateModule_Execute_WithHostVars tests template with host variables
func TestTemplateModule_Execute_WithHostVars(t *testing.T) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := "Host: {{ onigirazu_hostname }}, Port: {{ onigirazu_port }}"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    2222,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test host vars",
		"src":  templatePath,
		"dest": outputPath,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Verify content includes host variables
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "test-host") {
		t.Errorf("Expected content to contain hostname 'test-host', got: %s", string(content))
	}

	if !strings.Contains(string(content), "2222") {
		t.Errorf("Expected content to contain port '2222', got: %s", string(content))
	}
}

// TestTemplateModule_Execute_WithTimeout tests execution with context timeout
func TestTemplateModule_Execute_WithTimeout(t *testing.T) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := "Simple content"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test with timeout",
		"src":  templatePath,
		"dest": outputPath,
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

// TestTemplateModule_Execute_MissingSrc tests error when src is missing
func TestTemplateModule_Execute_MissingSrc(t *testing.T) {
	module := NewTemplateModule()

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test missing src",
		"dest": "/tmp/output.txt",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Errorf("Expected error for missing src")
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}
}

// TestTemplateModule_Execute_MissingDest tests error when dest is missing
func TestTemplateModule_Execute_MissingDest(t *testing.T) {
	module := NewTemplateModule()

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "test missing dest",
		"src":  "/tmp/template.j2",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Errorf("Expected error for missing dest")
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}
}

// BenchmarkTemplateModule_Validate benchmarks argument validation
func BenchmarkTemplateModule_Validate(b *testing.B) {
	module := NewTemplateModule()

	args := map[string]interface{}{
		"src":  "/tmp/template.j2",
		"dest": "/tmp/output.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkTemplateModule_Execute benchmarks template execution
func BenchmarkTemplateModule_Execute(b *testing.B) {
	module := NewTemplateModule()

	// Create a temporary template file
	tmpDir := b.TempDir()
	templatePath := filepath.Join(tmpDir, "template.j2")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := "Hello {{ name }}!"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		b.Fatalf("Failed to create template file: %v", err)
	}

	host := types.Host{
		Name:    "test-host",
		Port:    22,
		Address: "127.0.0.1",
		User:    "testuser",
		Vars:    make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "benchmark template",
		"src":  templatePath,
		"dest": outputPath,
		"vars": map[string]interface{}{
			"name": "World",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}
