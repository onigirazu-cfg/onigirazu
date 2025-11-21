package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.Backend != BackendTypeFile {
		t.Errorf("expected backend to be 'file', got %s", cfg.Backend)
	}

	if cfg.File == nil {
		t.Fatal("expected file config to be non-nil")
	}

	if cfg.File.BackupCount != 5 {
		t.Errorf("expected backup count 5, got %d", cfg.File.BackupCount)
	}

	if cfg.SQLite == nil {
		t.Fatal("expected sqlite config to be non-nil")
	}

	if cfg.SQLite.RetentionDays != 90 {
		t.Errorf("expected retention days 90, got %d", cfg.SQLite.RetentionDays)
	}
}

func TestConfigValidate_FileBackend(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Backend = BackendTypeFile

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestConfigValidate_FileBackendMissingConfig(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeFile,
		File:    nil,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing file config")
	}
}

func TestConfigValidate_FileBackendMissingDirectory(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeFile,
		File:    &FileConfig{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestConfigValidate_FileBackendNegativeBackupCount(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:   "/tmp/test",
			BackupCount: -1,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative backup count")
	}
}

func TestConfigValidate_SQLiteBackend(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	cfg := &Config{
		Backend: BackendTypeSQLite,
		SQLite: &SQLiteConfig{
			Database:       filepath.Join(homeDir, ".onigirazu", "test.db"),
			MaxConnections: 5,
			BusyTimeout:    5000,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestConfigValidate_SQLiteLowMaxConnections(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	cfg := &Config{
		Backend: BackendTypeSQLite,
		SQLite: &SQLiteConfig{
			Database:       filepath.Join(homeDir, ".onigirazu", "test.db"),
			MaxConnections: 0,
			BusyTimeout:    1000,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.SQLite.MaxConnections != 5 {
		t.Errorf("expected max connections to be set to 5, got %d", cfg.SQLite.MaxConnections)
	}
}

func TestConfigValidate_SQLiteLowBusyTimeout(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	cfg := &Config{
		Backend: BackendTypeSQLite,
		SQLite: &SQLiteConfig{
			Database:       filepath.Join(homeDir, ".onigirazu", "test.db"),
			MaxConnections: 5,
			BusyTimeout:    500,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.SQLite.BusyTimeout != 5000 {
		t.Errorf("expected busy timeout to be set to 5000, got %d", cfg.SQLite.BusyTimeout)
	}
}

func TestConfigValidate_SQLiteMissingDatabase(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeSQLite,
		SQLite: &SQLiteConfig{
			Database: "",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing database path")
	}
}

func TestConfigValidate_RemoteBackend(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeRemote,
		Remote: &RemoteConfig{
			APIURL:    "https://example.com/api",
			AuthToken: "token123",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestConfigValidate_RemoteMissingURL(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeRemote,
		Remote: &RemoteConfig{
			AuthToken: "token123",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing API URL")
	}
}

func TestConfigValidate_RemoteMissingToken(t *testing.T) {
	cfg := &Config{
		Backend: BackendTypeRemote,
		Remote: &RemoteConfig{
			APIURL: "https://example.com/api",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing auth token")
	}
}

func TestConfigValidate_EmptyBackend(t *testing.T) {
	cfg := &Config{
		Backend: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty backend")
	}
}

func TestConfigValidate_UnknownBackend(t *testing.T) {
	cfg := &Config{
		Backend: BackendType("unknown"),
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestGetDirectory(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".onigirazu", "states")

	cfg := NewDefaultConfig()
	dir := cfg.GetDirectory()

	if dir != expectedDir {
		t.Errorf("expected directory %s, got %s", expectedDir, dir)
	}
}

func TestGetDirectory_Nil(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".onigirazu", "states")

	cfg := &Config{
		File: nil,
	}

	dir := cfg.GetDirectory()
	if dir != expectedDir {
		t.Errorf("expected default directory %s, got %s", expectedDir, dir)
	}
}

func TestFileConfigDefaults(t *testing.T) {
	fc := &FileConfig{
		Directory:   "/tmp/test",
		BackupCount: 5,
	}

	if fc.Compression {
		t.Error("expected compression to be false by default")
	}

	if fc.RotationSize != 0 {
		t.Errorf("expected rotation size 0, got %d", fc.RotationSize)
	}
}

func TestSQLiteConfigDefaults(t *testing.T) {
	sc := &SQLiteConfig{
		Database:       "/tmp/test.db",
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	if !sc.AutoVacuum {
		t.Error("expected autovacuum to be true by default")
	}

	if sc.JournalMode != "wal" {
		t.Errorf("expected journal mode 'wal', got %s", sc.JournalMode)
	}
}

func TestRemoteConfigDefaults(t *testing.T) {
	rc := &RemoteConfig{
		SyncInterval: 5 * time.Minute,
		CacheLocal:   true,
	}

	if rc.SyncInterval != 5*time.Minute {
		t.Errorf("expected sync interval 5m, got %v", rc.SyncInterval)
	}

	if !rc.CacheLocal {
		t.Error("expected cache local to be true")
	}
}

func TestBackendTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		expected BackendType
	}{
		{"File", BackendTypeFile},
		{"SQLite", BackendTypeSQLite},
		{"Remote", BackendTypeRemote},
	}

	for _, tt := range tests {
		if tt.expected == "" {
			t.Errorf("backend type %s is empty", tt.name)
		}
	}
}
