package folderstructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateResolverResolveTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create templates directory
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.Mkdir(templatesDir, 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	// Create a test template
	testTemplate := filepath.Join(templatesDir, "test.j2")
	if err := os.WriteFile(testTemplate, []byte("{{ var }}"), 0644); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	detector := NewDetector()
	resolver := NewTemplateResolver(detector)

	// Test resolving from templates/ directory
	result := resolver.ResolveTemplate("test.j2", tmpDir)

	if !result.Found {
		t.Error("expected template to be found")
	}

	if result.Source != "templates" {
		t.Errorf("expected source 'templates', got %s", result.Source)
	}

	if !fileExists(result.Path) {
		t.Errorf("resolved path does not exist: %s", result.Path)
	}
}

func TestTemplateResolverResolveAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test template
	testTemplate := filepath.Join(tmpDir, "test.j2")
	if err := os.WriteFile(testTemplate, []byte("{{ var }}"), 0644); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	detector := NewDetector()
	resolver := NewTemplateResolver(detector)

	// Test resolving with absolute path
	result := resolver.ResolveTemplate(testTemplate, tmpDir)

	if !result.Found {
		t.Error("expected template to be found")
	}

	if result.Source != "absolute" {
		t.Errorf("expected source 'absolute', got %s", result.Source)
	}
}

func TestTemplateResolverTemplateNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector()
	resolver := NewTemplateResolver(detector)

	// Test with non-existent template
	result := resolver.ResolveTemplate("nonexistent.j2", tmpDir)

	if result.Found {
		t.Error("expected template to not be found")
	}

	if result.Error == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestTemplateResolverValidateTemplatePath(t *testing.T) {
	basePath := "/home/user/project"
	validPath := "templates/test.j2"
	invalidPath := "../../../etc/passwd"

	detector := NewDetector()
	resolver := NewTemplateResolver(detector)

	// Valid path should not error
	if err := resolver.ValidateTemplatePath(basePath, validPath); err != nil {
		t.Errorf("unexpected error for valid path: %v", err)
	}

	// Invalid path should error
	if err := resolver.ValidateTemplatePath(basePath, invalidPath); err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestTemplateResolverCaching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create templates directory
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.Mkdir(templatesDir, 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	// Create test template
	testTemplate := filepath.Join(templatesDir, "test.j2")
	if err := os.WriteFile(testTemplate, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	detector := NewDetector()
	resolver := NewTemplateResolver(detector)

	// First resolution
	result1 := resolver.ResolveTemplate("test.j2", tmpDir)
	if !result1.Found {
		t.Fatal("first resolution failed")
	}

	// Second resolution should come from cache
	result2 := resolver.ResolveTemplate("test.j2", tmpDir)
	if !result2.Found {
		t.Fatal("second resolution failed")
	}

	if result1.Path != result2.Path {
		t.Error("cached result should match original")
	}
}

func TestTemplateResolverClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create templates directory
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.Mkdir(templatesDir, 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	// Create test template
	testTemplate := filepath.Join(templatesDir, "test.j2")
	if err := os.WriteFile(testTemplate, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	detector := NewDetector()
	resolver := NewTemplateResolver(detector)

	// Populate cache
	result1 := resolver.ResolveTemplate("test.j2", tmpDir)
	if !result1.Found {
		t.Fatal("resolution failed")
	}

	// Clear cache
	resolver.ClearCache()

	// Should still be able to resolve after cache clear
	result2 := resolver.ResolveTemplate("test.j2", tmpDir)
	if !result2.Found {
		t.Error("resolution after cache clear failed")
	}
}
