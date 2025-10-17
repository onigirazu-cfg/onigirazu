package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FileBackend implements StateBackend interface for file-based storage
type FileBackend struct {
	config    *FileConfig
	stateFile string
	backupDir string
}

// NewFileBackend creates a new file-based state backend
func NewFileBackend(config *FileConfig, stateFile string) (*FileBackend, error) {
	if config == nil {
		config = &FileConfig{
			Compression: false,
			BackupCount: 5,
		}
	}

	if config.Directory == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		config.Directory = filepath.Join(homeDir, ".onigirazu", "states")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(config.Directory, 0750); err != nil {
		return nil, fmt.Errorf("cannot create state directory: %w", err)
	}

	// If no state file path provided, use default
	if stateFile == "" {
		stateFile = filepath.Join(config.Directory, ".onigirazu-state")
	}

	backupDir := filepath.Join(config.Directory, "backups")
	if config.BackupCount > 0 {
		if err := os.MkdirAll(backupDir, 0750); err != nil {
			return nil, fmt.Errorf("cannot create backup directory: %w", err)
		}
	}

	return &FileBackend{
		config:    config,
		stateFile: stateFile,
		backupDir: backupDir,
	}, nil
}

// LoadState loads state from file
func (fb *FileBackend) LoadState(ctx context.Context) (*types.State, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if _, err := os.Stat(fb.stateFile); os.IsNotExist(err) {
		return &types.State{
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(fb.stateFile) // #nosec G304 -- stateFile is constructed from fixed state file path
	if err != nil {
		return nil, fmt.Errorf("error reading state file: %w", err)
	}

	// Check if data is compressed
	if fb.config.Compression {
		// TODO: Add decompression logic if needed
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("error parsing state: %w", err)
	}

	// Initialize maps if nil
	if state.Variables == nil {
		state.Variables = make(map[string]interface{})
	}
	if state.Checksums == nil {
		state.Checksums = make(map[string]string)
	}

	return &state, nil
}

// SaveState saves state to file with automatic backup rotation
func (fb *FileBackend) SaveState(ctx context.Context, state *types.State) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create backup if file exists and backup count > 0
	if fb.config.BackupCount > 0 {
		if _, err := os.Stat(fb.stateFile); err == nil {
			if err := fb.createBackup(); err != nil {
				// Log but don't fail - backup is not critical
				fmt.Fprintf(os.Stderr, "warning: failed to create backup: %v\n", err)
			}
		}
	}

	// Marshal state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing state: %w", err)
	}

	// Compress if enabled
	if fb.config.Compression {
		// TODO: Add compression logic if needed
	}

	// Write to temporary file first, then rename (atomic)
	tmpFile := fb.stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("error writing temporary state file: %w", err)
	}

	if err := os.Rename(tmpFile, fb.stateFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("error renaming state file: %w", err)
	}

	// Clean up old backups if exceeding limit
	if fb.config.BackupCount > 0 {
		if err := fb.cleanupOldBackups(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to cleanup old backups: %v\n", err)
		}
	}

	return nil
}

// DeleteState removes the state file
func (fb *FileBackend) DeleteState(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.Remove(fb.stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error deleting state file: %w", err)
	}
	return nil
}

// GetPath returns the state file path
func (fb *FileBackend) GetPath() string {
	return fb.stateFile
}

// GetStats returns backend statistics
func (fb *FileBackend) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	info, err := os.Stat(fb.stateFile)
	if err == nil {
		stats["file_size"] = info.Size()
		stats["last_modified"] = info.ModTime()
	}

	stats["backup_dir"] = fb.backupDir
	stats["compression_enabled"] = fb.config.Compression
	stats["backup_count"] = fb.config.BackupCount

	// Count backups
	if entries, err := os.ReadDir(fb.backupDir); err == nil {
		stats["existing_backups"] = len(entries)
	}

	return stats
}

// Migrate performs necessary migrations (file backend doesn't need migrations)
func (fb *FileBackend) Migrate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Ensure directories exist
	if err := os.MkdirAll(fb.config.Directory, 0750); err != nil {
		return fmt.Errorf("cannot create state directory: %w", err)
	}

	if fb.config.BackupCount > 0 {
		if err := os.MkdirAll(fb.backupDir, 0750); err != nil {
			return fmt.Errorf("cannot create backup directory: %w", err)
		}
	}

	return nil
}

// createBackup creates a backup of the current state file
func (fb *FileBackend) createBackup() error {
	if _, err := os.Stat(fb.stateFile); err != nil {
		return nil // Nothing to backup
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(fb.backupDir, filepath.Base(fb.stateFile)+"."+timestamp)

	data, err := os.ReadFile(fb.stateFile)
	if err != nil {
		return err
	}

	return os.WriteFile(backupFile, data, 0600)
}

// cleanupOldBackups removes old backups exceeding the backup count limit
func (fb *FileBackend) cleanupOldBackups() error {
	entries, err := os.ReadDir(fb.backupDir)
	if err != nil {
		return err
	}

	// If we're within the limit, nothing to do
	if len(entries) <= fb.config.BackupCount {
		return nil
	}

	// Sort by modification time and delete oldest
	// For simplicity, just delete until we're at the limit
	toDelete := len(entries) - fb.config.BackupCount

	for i := 0; i < toDelete && i < len(entries); i++ {
		backupPath := filepath.Join(fb.backupDir, entries[i].Name())
		if err := os.Remove(backupPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete old backup %s: %v\n", backupPath, err)
		}
	}

	return nil
}
