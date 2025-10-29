package folderstructure

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestManagerIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create complete project structure
	dirs := []string{"defaults", "vars", "templates", "files", "handlers", "tasks"}
	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	// Create defaults/main.yml
	defaultsData := map[string]interface{}{
		"app_port": 8080,
		"env":      "dev",
	}
	if data, err := yaml.Marshal(defaultsData); err != nil {
		t.Fatalf("failed to marshal defaults: %v", err)
	} else if err := os.WriteFile(
		filepath.Join(tmpDir, "defaults", "main.yml"),
		data,
		0644,
	); err != nil {
		t.Fatalf("failed to write defaults: %v", err)
	}

	// Create vars/main.yml
	varsData := map[string]interface{}{
		"env": "prod",
	}
	if data, err := yaml.Marshal(varsData); err != nil {
		t.Fatalf("failed to marshal vars: %v", err)
	} else if err := os.WriteFile(
		filepath.Join(tmpDir, "vars", "main.yml"),
		data,
		0644,
	); err != nil {
		t.Fatalf("failed to write vars: %v", err)
	}

	// Create test file
	if err := os.WriteFile(
		filepath.Join(tmpDir, "files", "config.txt"),
		[]byte("config content"),
		0644,
	); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create test template
	if err := os.WriteFile(
		filepath.Join(tmpDir, "templates", "config.j2"),
		[]byte("{{ app_port }}"),
		0644,
	); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Create handlers
	handlersData := []map[string]interface{}{
		{
			"name":   "restart app",
			"listen": "app restart",
		},
	}
	if data, err := yaml.Marshal(handlersData); err != nil {
		t.Fatalf("failed to marshal handlers: %v", err)
	} else if err := os.WriteFile(
		filepath.Join(tmpDir, "handlers", "main.yml"),
		data,
		0644,
	); err != nil {
		t.Fatalf("failed to write handlers: %v", err)
	}

	manager := NewManager()

	// Test structure detection
	structure, err := manager.DetectStructure(tmpDir)
	if err != nil {
		t.Fatalf("failed to detect structure: %v", err)
	}

	if !structure.HasDefaults || !structure.HasVars || !structure.HasTemplates ||
		!structure.HasFiles || !structure.HasHandlers {
		t.Error("expected all directories to be detected")
	}

	// Test variables loading
	varSet, err := manager.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("failed to load variables: %v", err)
	}

	if varSet.Variables["app_port"] != 8080 {
		t.Error("expected app_port from defaults")
	}

	if varSet.Variables["env"] != "prod" {
		t.Error("expected env to be overridden by vars")
	}

	// Test file resolution
	fileResult := manager.ResolveFile("config.txt", tmpDir)
	if !fileResult.Found {
		t.Error("expected file to be found")
	}

	// Test template resolution
	templateResult := manager.ResolveTemplate("config.j2", tmpDir)
	if !templateResult.Found {
		t.Error("expected template to be found")
	}

	// Test handler loading
	handlers, err := manager.LoadHandlers(tmpDir)
	if err != nil {
		t.Fatalf("failed to load handlers: %v", err)
	}

	if len(handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(handlers))
	}

	// Test cache clear
	manager.ClearCache()

	// Should still work after cache clear
	structure2, err := manager.DetectStructure(tmpDir)
	if err != nil {
		t.Fatalf("failed to detect structure after cache clear: %v", err)
	}

	if structure2.RootPath != structure.RootPath {
		t.Error("structure should be same after cache clear")
	}
}

func TestManagerStatistics(t *testing.T) {
	manager := NewManager()
	stats := manager.GetStatistics()

	if stats["feature"] != "ansible_folder_structure" {
		t.Error("expected correct feature name in statistics")
	}

	if stats["version"] != "1.0.0" {
		t.Error("expected correct version in statistics")
	}

	if stats["components"] != 5 {
		t.Error("expected 5 components in statistics")
	}
}

func TestManagerDetectProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project structure
	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create defaults directory
	if err := os.Mkdir(filepath.Join(projectDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create defaults: %v", err)
	}

	manager := NewManager()

	root, err := manager.DetectProjectRoot(projectDir)
	if err != nil {
		t.Fatalf("failed to detect project root: %v", err)
	}

	if root != projectDir {
		t.Errorf("expected root %s, got %s", projectDir, root)
	}
}

func TestManagerIsStructured(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project structure
	if err := os.Mkdir(filepath.Join(tmpDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create defaults: %v", err)
	}

	manager := NewManager()

	isStructured, err := manager.IsStructured(tmpDir)
	if err != nil {
		t.Fatalf("failed to check if structured: %v", err)
	}

	if !isStructured {
		t.Error("expected path to be structured")
	}

	// Test with non-structured directory
	tmpDir2 := t.TempDir()
	isStructured2, err := manager.IsStructured(tmpDir2)
	if err != nil {
		t.Fatalf("failed to check if structured: %v", err)
	}

	if isStructured2 {
		t.Error("expected empty directory to not be structured")
	}
}

func TestManagerValidatePath(t *testing.T) {
	basePath := "/home/user/project"

	manager := NewManager()

	// Valid path
	err := manager.ValidatePath(basePath, "files/test.txt")
	if err != nil {
		t.Errorf("unexpected error for valid path: %v", err)
	}

	// Invalid path (traversal)
	err = manager.ValidatePath(basePath, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}
