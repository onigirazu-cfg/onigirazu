package modules

import (
	"archive/tar"
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

func TestArchiveExecute_IdempotentAndExclude(t *testing.T) {
	module := NewArchiveModule()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "skip"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.txt", "b.txt", "skip/c.txt"} {
		if err := os.WriteFile(filepath.Join(src, f), []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dest := filepath.Join(dir, "out", "src.tar.gz")
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	args := map[string]interface{}{"path": src, "dest": dest, "format": "gz", "exclude_path": filepath.Join(src, "skip")}

	result, err := module.Execute(context.Background(), host, args)
	if err != nil || !result.Success || !result.Changed {
		t.Fatalf("first run: err=%v success=%v changed=%v error=%s", err, result.Success, result.Changed, result.Error)
	}
	if !verifyGzArchiveContents(t, dest, []string{"a.txt", "b.txt"}) {
		t.Error("archive is missing files")
	}
	if verifyGzArchiveContents(t, dest, []string{"skip/c.txt"}) {
		t.Error("excluded file is in the archive")
	}

	result, err = module.Execute(context.Background(), host, args)
	if err != nil || !result.Success || result.Changed {
		t.Errorf("second run should change nothing: err=%v changed=%v error=%s", err, result.Changed, result.Error)
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
