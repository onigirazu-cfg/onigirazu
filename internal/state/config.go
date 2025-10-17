package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackendType defines the type of state storage backend
type BackendType string

const (
	// BackendTypeFile stores state in JSON files
	BackendTypeFile BackendType = "file"

	// BackendTypeSQLite stores state in SQLite database
	BackendTypeSQLite BackendType = "sqlite"

	// BackendTypeRemote stores state on remote server (future)
	BackendTypeRemote BackendType = "remote"
)

// FileConfig holds configuration for file-based state storage
type FileConfig struct {
	// Directory where state files are stored
	Directory string `yaml:"directory" json:"directory"`

	// Compression enables gzip compression for state files
	Compression bool `yaml:"compression" json:"compression"`

	// BackupCount is the number of backups to keep
	BackupCount int `yaml:"backup_count" json:"backup_count"`

	// RotationSize is the max size in bytes before rotation (0 = disabled)
	RotationSize int64 `yaml:"rotation_size" json:"rotation_size"`
}

// SQLiteConfig holds configuration for SQLite state storage
type SQLiteConfig struct {
	// Database path
	Database string `yaml:"database" json:"database"`

	// AutoVacuum enables automatic vacuuming
	AutoVacuum bool `yaml:"auto_vacuum" json:"auto_vacuum"`

	// JournalMode sets the journal mode (wal, delete, etc.)
	JournalMode string `yaml:"journal_mode" json:"journal_mode"`

	// BusyTimeout in milliseconds
	BusyTimeout int `yaml:"busy_timeout" json:"busy_timeout"`

	// MaxConnections for connection pool
	MaxConnections int `yaml:"max_connections" json:"max_connections"`

	// RetentionDays for automatic cleanup (0 = disabled)
	RetentionDays int `yaml:"retention_days" json:"retention_days"`
}

// RemoteConfig holds configuration for remote state storage (future)
type RemoteConfig struct {
	// API URL
	APIURL string `yaml:"api_url" json:"api_url"`

	// Authentication token
	AuthToken string `yaml:"auth_token" json:"auth_token"`

	// SyncInterval for periodic sync
	SyncInterval time.Duration `yaml:"sync_interval" json:"sync_interval"`

	// CacheLocal enables local caching
	CacheLocal bool `yaml:"cache_local" json:"cache_local"`
}

// Config holds state backend configuration
type Config struct {
	// Backend type to use
	Backend BackendType `yaml:"backend" json:"backend"`

	// File backend configuration
	File *FileConfig `yaml:"file" json:"file"`

	// SQLite backend configuration
	SQLite *SQLiteConfig `yaml:"sqlite" json:"sqlite"`

	// Remote backend configuration
	Remote *RemoteConfig `yaml:"remote" json:"remote"`
}

// NewDefaultConfig returns a default configuration
func NewDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Backend: BackendTypeFile,
		File: &FileConfig{
			Directory:    filepath.Join(homeDir, ".onigirazu", "states"),
			Compression:  false,
			BackupCount:  5,
			RotationSize: 0,
		},
		SQLite: &SQLiteConfig{
			Database:       filepath.Join(homeDir, ".onigirazu", "state.db"),
			AutoVacuum:     true,
			JournalMode:    "wal",
			BusyTimeout:    5000,
			MaxConnections: 5,
			RetentionDays:  90,
		},
		Remote: &RemoteConfig{
			SyncInterval: 5 * time.Minute,
			CacheLocal:   true,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Backend == "" {
		return fmt.Errorf("backend type must be specified")
	}

	switch c.Backend {
	case BackendTypeFile:
		if c.File == nil {
			return fmt.Errorf("file backend configuration is required")
		}
		if c.File.Directory == "" {
			return fmt.Errorf("file backend directory must be specified")
		}
		if c.File.BackupCount < 0 {
			return fmt.Errorf("backup count cannot be negative")
		}

	case BackendTypeSQLite:
		if c.SQLite == nil {
			return fmt.Errorf("sqlite backend configuration is required")
		}
		if c.SQLite.Database == "" {
			return fmt.Errorf("sqlite database path must be specified")
		}
		if c.SQLite.MaxConnections < 1 {
			c.SQLite.MaxConnections = 5
		}
		if c.SQLite.BusyTimeout < 1000 {
			c.SQLite.BusyTimeout = 5000
		}

	case BackendTypeRemote:
		if c.Remote == nil {
			return fmt.Errorf("remote backend configuration is required")
		}
		if c.Remote.APIURL == "" {
			return fmt.Errorf("remote API URL must be specified")
		}
		if c.Remote.AuthToken == "" {
			return fmt.Errorf("remote authentication token must be specified")
		}

	default:
		return fmt.Errorf("unknown backend type: %s", c.Backend)
	}

	return nil
}

// GetDirectory returns the directory for file storage (if applicable)
func (c *Config) GetDirectory() string {
	if c.File != nil {
		return c.File.Directory
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".onigirazu", "states")
}
