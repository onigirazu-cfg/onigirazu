package folderstructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileResolverResolveFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files directory
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.Mkdir(filesDir, 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}

	// Create a test file in files/
	testFile := filepath.Join(filesDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewDetector()
	resolver := NewFileResolver(detector)

	// Test resolving from files/ directory
	result := resolver.ResolveFile("test.txt", tmpDir)

	if !result.Found {
		t.Error("expected file to be found")
	}

	if result.Source != "files" {
		t.Errorf("expected source 'files', got %s", result.Source)
	}

	if !fileExists(result.Path) {
		t.Errorf("resolved path does not exist: %s", result.Path)
	}
}

func TestFileResolverResolveAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewDetector()
	resolver := NewFileResolver(detector)

	// Test resolving with absolute path
	result := resolver.ResolveFile(testFile, tmpDir)

	if !result.Found {
		t.Error("expected file to be found")
	}

	if result.Source != "absolute" {
		t.Errorf("expected source 'absolute', got %s", result.Source)
	}
}

func TestFileResolverFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector()
	resolver := NewFileResolver(detector)

	// Test with non-existent file
	result := resolver.ResolveFile("nonexistent.txt", tmpDir)

	if result.Found {
		t.Error("expected file to not be found")
	}

	if result.Error == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFileResolverValidateFilePath(t *testing.T) {
	basePath := "/home/user/project"
	validPath := "files/test.txt"
	invalidPath := "../../../etc/passwd"

	detector := NewDetector()
	resolver := NewFileResolver(detector)

	// Valid path should not error
	if err := resolver.ValidateFilePath(basePath, validPath); err != nil {
		t.Errorf("unexpected error for valid path: %v", err)
	}

	// Invalid path should error
	if err := resolver.ValidateFilePath(basePath, invalidPath); err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestFileResolverCaching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files directory
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.Mkdir(filesDir, 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}

	// Create test file
	testFile := filepath.Join(filesDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewDetector()
	resolver := NewFileResolver(detector)

	// First resolution
	result1 := resolver.ResolveFile("test.txt", tmpDir)
	if !result1.Found {
		t.Fatal("first resolution failed")
	}

	// Second resolution should come from cache
	result2 := resolver.ResolveFile("test.txt", tmpDir)
	if !result2.Found {
		t.Fatal("second resolution failed")
	}

	if result1.Path != result2.Path {
		t.Error("cached result should match original")
	}
}

func TestFileResolverClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files directory
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.Mkdir(filesDir, 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}

	// Create test file
	testFile := filepath.Join(filesDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	detector := NewDetector()
	resolver := NewFileResolver(detector)

	// Populate cache
	result1 := resolver.ResolveFile("test.txt", tmpDir)
	if !result1.Found {
		t.Fatal("resolution failed")
	}

	// Clear cache
	resolver.ClearCache()

	// Should still be able to resolve after cache clear
	result2 := resolver.ResolveFile("test.txt", tmpDir)
	if !result2.Found {
		t.Error("resolution after cache clear failed")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
