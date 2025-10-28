package modules

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestArchiveModuleValidate(t *testing.T) {
	module := NewArchiveModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid archive with gz format",
			args: map[string]interface{}{
				"name":   "archive test",
				"path":   "/tmp/test.txt",
				"dest":   "/tmp/test.tar.gz",
				"format": "gz",
			},
			wantErr: false,
		},
		{
			name: "valid archive with tar format",
			args: map[string]interface{}{
				"name":   "archive test",
				"path":   "/tmp/test.txt",
				"dest":   "/tmp/test.tar",
				"format": "tar",
			},
			wantErr: false,
		},
		{
			name: "valid archive with zip format",
			args: map[string]interface{}{
				"name":   "archive test",
				"path":   "/tmp/test.txt",
				"dest":   "/tmp/test.zip",
				"format": "zip",
			},
			wantErr: false,
		},
		{
			name: "missing path",
			args: map[string]interface{}{
				"name":   "archive test",
				"dest":   "/tmp/test.tar.gz",
				"format": "gz",
			},
			wantErr: true,
		},
		{
			name: "missing dest",
			args: map[string]interface{}{
				"name":   "archive test",
				"path":   "/tmp/test.txt",
				"format": "gz",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			args: map[string]interface{}{
				"name":   "archive test",
				"path":   "/tmp/test.txt",
				"dest":   "/tmp/test.tar",
				"format": "invalid",
			},
			wantErr: true,
		},
		{
			name: "path as list",
			args: map[string]interface{}{
				"name":   "archive test",
				"path":   []interface{}{"/tmp/test1.txt", "/tmp/test2.txt"},
				"dest":   "/tmp/test.tar.gz",
				"format": "gz",
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

func TestArchiveCreateGzArchive(t *testing.T) {
	module := NewArchiveModule()
	tmpDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	destArchive := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(testFile1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create archive
	size, err := module.createArchive(destArchive, "gz", []string{testFile1, testFile2})
	if err != nil {
		t.Fatalf("createArchive() error = %v", err)
	}

	if size <= 0 {
		t.Errorf("archive size should be > 0, got %d", size)
	}

	// Verify archive exists
	if _, err := os.Stat(destArchive); err != nil {
		t.Errorf("archive file not created: %v", err)
	}

	// Verify archive contents
	if !verifyGzArchiveContents(t, destArchive, []string{"test1.txt", "test2.txt"}) {
		t.Error("archive contents verification failed")
	}
}

func TestArchiveCreateZipArchive(t *testing.T) {
	module := NewArchiveModule()
	tmpDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	destArchive := filepath.Join(tmpDir, "test.zip")

	if err := os.WriteFile(testFile1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create archive
	size, err := module.createArchive(destArchive, "zip", []string{testFile1, testFile2})
	if err != nil {
		t.Fatalf("createArchive() error = %v", err)
	}

	if size <= 0 {
		t.Errorf("archive size should be > 0, got %d", size)
	}

	// Verify archive exists
	if _, err := os.Stat(destArchive); err != nil {
		t.Errorf("archive file not created: %v", err)
	}

	// Verify archive contents
	if !verifyZipArchiveContents(t, destArchive, []string{"test1.txt", "test2.txt"}) {
		t.Error("archive contents verification failed")
	}
}

func TestArchiveCollectFiles(t *testing.T) {
	module := NewArchiveModule()
	tmpDir := t.TempDir()

	// Create test directory structure
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	testDir := filepath.Join(tmpDir, "subdir")
	testFile3 := filepath.Join(testDir, "test3.txt")

	os.MkdirAll(testDir, 0o755)
	os.WriteFile(testFile1, []byte("content1"), 0o644)
	os.WriteFile(testFile2, []byte("content2"), 0o644)
	os.WriteFile(testFile3, []byte("content3"), 0o644)

	// Test collecting files with glob
	pattern := filepath.Join(tmpDir, "*.txt")
	files, err := module.collectFiles([]string{pattern}, nil)
	if err != nil {
		t.Fatalf("collectFiles() error = %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	// Test collecting with exclusion
	files, err = module.collectFiles(
		[]string{filepath.Join(tmpDir, "test*.txt")},
		[]string{testFile2},
	)
	if err != nil {
		t.Fatalf("collectFiles() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file after exclusion, got %d", len(files))
	}
}

func TestArchiveGetPaths(t *testing.T) {
	module := NewArchiveModule()

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected int
	}{
		{
			name: "string path",
			args: map[string]interface{}{
				"path": "/tmp/test.txt",
			},
			expected: 1,
		},
		{
			name: "list of paths",
			args: map[string]interface{}{
				"path": []interface{}{"/tmp/test1.txt", "/tmp/test2.txt"},
			},
			expected: 2,
		},
		{
			name: "empty path",
			args: map[string]interface{}{
				"path": "",
			},
			expected: 0,
		},
		{
			name:     "no path",
			args:     map[string]interface{}{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := module.getPaths(tt.args)
			if len(paths) != tt.expected {
				t.Errorf("getPaths() got %d paths, expected %d", len(paths), tt.expected)
			}
		})
	}
}

func TestArchiveExecuteWithLocalHost(t *testing.T) {
	module := NewArchiveModule()
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	destArchive := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create local host
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Execute archive
	args := map[string]interface{}{
		"name":   "test archive",
		"path":   testFile,
		"dest":   destArchive,
		"format": "gz",
	}

	result, err := module.Execute(context.Background(), host, args)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if result.Failed {
		t.Errorf("Execute() failed: %s", result.Error)
	}

	if !result.Changed {
		t.Error("Execute() expected Changed to be true")
	}

	// Verify archive was created
	if _, err := os.Stat(destArchive); err != nil {
		t.Errorf("archive file not created: %v", err)
	}
}

// Helper functions

func verifyGzArchiveContents(t *testing.T, archivePath string, expectedFiles []string) bool {
	file, err := os.Open(archivePath)
	if err != nil {
		t.Logf("failed to open archive: %v", err)
		return false
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		t.Logf("failed to create gzip reader: %v", err)
		return false
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	found := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Logf("failed to read tar header: %v", err)
			return false
		}

		// Extract just the filename
		name := filepath.Base(header.Name)
		found[name] = true
	}

	// Check if all expected files were found
	for _, expectedFile := range expectedFiles {
		if !found[expectedFile] {
			t.Logf("expected file %s not found in archive", expectedFile)
			return false
		}
	}

	return true
}

func verifyZipArchiveContents(t *testing.T, archivePath string, expectedFiles []string) bool {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Logf("failed to open zip archive: %v", err)
		return false
	}
	defer reader.Close()

	found := make(map[string]bool)

	for _, file := range reader.File {
		// Extract just the filename
		name := filepath.Base(file.Name)
		if name != "" { // skip directories
			found[name] = true
		}
	}

	// Check if all expected files were found
	for _, expectedFile := range expectedFiles {
		if !found[expectedFile] {
			t.Logf("expected file %s not found in archive", expectedFile)
			return false
		}
	}

	return true
}
