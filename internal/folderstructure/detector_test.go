package folderstructure

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectorDetect(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create standard directories
	dirs := []string{"defaults", "vars", "templates", "files", "handlers", "tasks"}
	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	detector := NewDetector()
	structure, err := detector.Detect(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if structure.RootPath != tmpDir {
		t.Errorf("expected RootPath %s, got %s", tmpDir, structure.RootPath)
	}

	if !structure.HasDefaults || !structure.HasVars || !structure.HasTemplates ||
		!structure.HasFiles || !structure.HasHandlers || !structure.HasTasks {
		t.Error("expected all standard directories to be detected")
	}

	if time.Since(structure.DetectedAt) > 5*time.Second {
		t.Error("expected recent DetectedAt timestamp")
	}
}

func TestDetectorDetectPartial(t *testing.T) {
	// Create temporary directory with only some directories
	tmpDir := t.TempDir()

	// Create only some directories
	dirs := []string{"defaults", "vars"}
	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	detector := NewDetector()
	structure, err := detector.Detect(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !structure.HasDefaults || !structure.HasVars {
		t.Error("expected defaults and vars to be detected")
	}

	if structure.HasTemplates || structure.HasFiles || structure.HasHandlers || structure.HasTasks {
		t.Error("expected other directories to not be detected")
	}
}

func TestDetectorDetectProjectRoot(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create standard directories in project
	if err := os.Mkdir(filepath.Join(projectDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	detector := NewDetector()
	root, err := detector.DetectProjectRoot(projectDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if root != projectDir {
		t.Errorf("expected root %s, got %s", projectDir, root)
	}
}

func TestDetectorCaching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create standard directories
	if err := os.Mkdir(filepath.Join(tmpDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	detector := NewDetector()

	// First detection
	structure1, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("first detection failed: %v", err)
	}

	// Second detection should come from cache
	structure2, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("second detection failed: %v", err)
	}

	// Both should have same root path
	if structure1.RootPath != structure2.RootPath {
		t.Error("cached structure should match original")
	}
}

func TestDetectorIsStructured(t *testing.T) {
	tmpDir := t.TempDir()

	// Create standard directories
	if err := os.Mkdir(filepath.Join(tmpDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	detector := NewDetector()
	isStructured, err := detector.IsStructured(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !isStructured {
		t.Error("expected path to be structured")
	}
}

func TestDetectorEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector()
	isStructured, err := detector.IsStructured(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if isStructured {
		t.Error("expected empty directory to not be structured")
	}
}

func TestDetectorGetStandardDirs(t *testing.T) {
	detector := NewDetector()
	dirs := detector.GetStandardDirs()

	expectedDirs := []string{"defaults", "vars", "templates", "files", "handlers", "tasks"}
	if len(dirs) != len(expectedDirs) {
		t.Errorf("expected %d standard dirs, got %d", len(expectedDirs), len(dirs))
	}

	// Check all expected dirs are present
	dirMap := make(map[string]bool)
	for _, dir := range dirs {
		dirMap[dir] = true
	}

	for _, expected := range expectedDirs {
		if !dirMap[expected] {
			t.Errorf("expected dir %s not found", expected)
		}
	}
}

func TestDetectorClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create standard directories
	if err := os.Mkdir(filepath.Join(tmpDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	detector := NewDetector()

	// Populate cache
	_, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("detection failed: %v", err)
	}

	// Clear cache
	detector.ClearCache()

	// Should still be able to detect after clearing cache
	structure, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("detection after cache clear failed: %v", err)
	}

	if !structure.HasDefaults {
		t.Error("expected structure to be detected after cache clear")
	}
}
