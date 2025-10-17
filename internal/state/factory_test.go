package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
)

func TestNewBackendFactory(t *testing.T) {
	cfg := NewDefaultConfig()
	factory := NewBackendFactory(cfg)

	if factory == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestNewBackendFactory_NilConfig(t *testing.T) {
	factory := NewBackendFactory(nil)

	if factory == nil {
		t.Fatal("expected non-nil factory")
	}

	// Should use default config
	backend, err := factory.CreateBackend("")
	if err != nil {
		t.Errorf("failed to create backend with default config: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestFactoryCreateBackend_FileBackend(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backend, err := factory.CreateBackend("")

	if err != nil {
		t.Fatalf("failed to create file backend: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}

	// Verify it's a FileBackend
	_, ok := backend.(*FileBackend)
	if !ok {
		t.Error("expected FileBackend type")
	}
}

func TestFactoryCreateBackend_FileBackendWithStatePath(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "custom-state.json")
	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backend, err := factory.CreateBackend(stateFile)

	if err != nil {
		t.Fatalf("failed to create file backend: %v", err)
	}

	if backend.GetPath() != stateFile {
		t.Errorf("expected path %s, got %s", stateFile, backend.GetPath())
	}
}

func TestFactoryCreateBackend_SQLiteBackend(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &Config{
		Backend: BackendTypeSQLite,
		SQLite: &SQLiteConfig{
			Database:       dbPath,
			AutoVacuum:     true,
			JournalMode:    "wal",
			BusyTimeout:    5000,
			MaxConnections: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backend, err := factory.CreateBackend("")

	if err != nil {
		t.Fatalf("failed to create sqlite backend: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}

	// Verify it's a SQLiteBackend
	_, ok := backend.(*SQLiteBackend)
	if !ok {
		t.Error("expected SQLiteBackend type")
	}

	// Verify database file exists
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("database file not created: %v", err)
	}

	// Clean up
	backend.DeleteState(context.Background())
}

func TestFactoryCreateBackend_Remote_NotImplemented(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeRemote,
		Remote: &RemoteConfig{
			APIURL:    "https://example.com",
			AuthToken: "token",
		},
	}

	factory := NewBackendFactory(cfg)
	_, err := factory.CreateBackend("")

	if err == nil {
		t.Fatal("expected error for unimplemented remote backend")
	}
}

func TestFactoryCreateBackend_InvalidConfig(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeFile,
		File:    nil, // Invalid
	}

	factory := NewBackendFactory(cfg)
	_, err := factory.CreateBackend("")

	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestFactoryCreateBackend_UnknownBackend(t *testing.T) {
	cfg := &Config{
		Backend: BackendType("unknown"),
	}

	factory := NewBackendFactory(cfg)
	_, err := factory.CreateBackend("")

	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestFactoryCreateFileBackend(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backend, err := factory.CreateFileBackend("")

	if err != nil {
		t.Fatalf("failed to create file backend: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestFactoryCreateSQLiteBackend(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &Config{
		Backend: BackendTypeSQLite,
		SQLite: &SQLiteConfig{
			Database:       dbPath,
			AutoVacuum:     true,
			JournalMode:    "wal",
			BusyTimeout:    5000,
			MaxConnections: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backend, err := factory.CreateSQLiteBackend()

	if err != nil {
		t.Fatalf("failed to create sqlite backend: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}

	// Clean up
	backend.DeleteState(context.Background())
}

func TestFactoryGetBackendType(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   "/tmp/test",
			BackupCount: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backendType := factory.GetBackendType()

	if backendType != BackendTypeFile {
		t.Errorf("expected backend type 'file', got %s", backendType)
	}
}

func TestFactorySetBackendType(t *testing.T) {
	cfg := NewDefaultConfig()
	factory := NewBackendFactory(cfg)

	if factory.GetBackendType() != BackendTypeFile {
		t.Fatal("expected initial backend type to be file")
	}

	factory.SetBackendType(BackendTypeSQLite)

	if factory.GetBackendType() != BackendTypeSQLite {
		t.Errorf("expected backend type to be changed to 'sqlite'")
	}
}

func TestFactoryBackendImplementsInterface(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
		},
	}

	factory := NewBackendFactory(cfg)
	backend, err := factory.CreateBackend("")

	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Verify it implements StateBackend interface
	var _ interfaces.StateBackend = backend
}

func TestFactoryMultipleBackendCreation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
		},
	}

	factory := NewBackendFactory(cfg)

	// Create multiple backends
	backend1, err := factory.CreateBackend("")
	if err != nil {
		t.Fatalf("failed to create first backend: %v", err)
	}

	backend2, err := factory.CreateBackend("")
	if err != nil {
		t.Fatalf("failed to create second backend: %v", err)
	}

	if backend1 == backend2 {
		t.Error("expected different backend instances")
	}
}

func TestFactoryBackendTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		backend  BackendType
		expected string
	}{
		{"File backend type", BackendTypeFile, "file"},
		{"SQLite backend type", BackendTypeSQLite, "sqlite"},
		{"Remote backend type", BackendTypeRemote, "remote"},
	}

	for _, tt := range tests {
		if string(tt.backend) != tt.expected {
			t.Errorf("%s: expected %s, got %s", tt.name, tt.expected, tt.backend)
		}
	}
}

func TestFactoryDefaultConfig_HasAllBackends(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.File == nil {
		t.Error("default config missing file configuration")
	}

	if cfg.SQLite == nil {
		t.Error("default config missing sqlite configuration")
	}

	if cfg.Remote == nil {
		t.Error("default config missing remote configuration")
	}
}
