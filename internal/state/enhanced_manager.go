package state

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// EnhancedManager provides thread-safe state management with advanced features
type EnhancedManager struct {
	stateFile   string
	mutex       sync.RWMutex
	state       *types.State
	taskStates  map[string]*types.TaskState
	autoSave    bool
	backupCount int
}

// NewEnhanced creates a new enhanced state manager
func NewEnhanced(stateFile string, autoSave bool, backupCount int) *EnhancedManager {
	return &EnhancedManager{
		stateFile:   stateFile,
		taskStates:  make(map[string]*types.TaskState),
		autoSave:    autoSave,
		backupCount: backupCount,
	}
}

// NewEnhancedManager creates a new enhanced state manager with logger
func NewEnhancedManager(stateFile string, logger interface{}) *EnhancedManager {
	return &EnhancedManager{
		stateFile:   stateFile,
		taskStates:  make(map[string]*types.TaskState),
		autoSave:    true,
		backupCount: 5,
	}
}

// LoadState loads state from file with context support
func (m *EnhancedManager) LoadState(ctx context.Context) (*types.State, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if context is canceled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if _, err := os.Stat(m.stateFile); os.IsNotExist(err) {
		m.state = &types.State{
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
			LastRun:   time.Now(),
		}
		return m.state, nil
	}

	data, err := os.ReadFile(m.stateFile) // #nosec G304 -- stateFile is constructed from fixed state file path
	if err != nil {
		return nil, fmt.Errorf("error reading state file: %w", err)
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

	m.state = &state
	return m.state, nil
}

// SaveState saves state to file with context support and backup
func (m *EnhancedManager) SaveState(ctx context.Context, state *types.State) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if context is canceled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create backup if file exists
	if err := m.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Update timestamp
	state.LastRun = time.Now()

	// Serialize state
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing state: %w", err)
	}

	// Write to temporary file first
	tempFile := m.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("error writing temporary state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, m.stateFile); err != nil {
		// Clean up temp file, ignore error as we're already in error state
		_ = os.Remove(tempFile)
		return fmt.Errorf("error renaming state file: %w", err)
	}

	m.state = state

	// Cleanup old backups
	if err := m.cleanupOldBackups(); err != nil {
		// Log error but don't fail the save operation
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return nil
}

// SaveCurrentState saves current state (convenience method without parameters)
func (m *EnhancedManager) SaveCurrentState() error {
	if m.state == nil {
		m.state = &types.State{
			LastRun:   time.Now(),
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
		}
	}
	return m.SaveState(context.Background(), m.state)
}

// HasChanged checks if a file has changed since last run
func (m *EnhancedManager) HasChanged(filePath string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.state == nil {
		return true, nil // No state loaded, assume changed
	}

	checksum, err := m.calculateChecksum(filePath)
	if err != nil {
		return false, err
	}

	lastChecksum, exists := m.state.Checksums[filePath]
	if !exists {
		// New file
		return true, nil
	}

	return checksum != lastChecksum, nil
}

// GetTaskState retrieves task state by ID
func (m *EnhancedManager) GetTaskState(taskID string) (*types.TaskState, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	taskState, exists := m.taskStates[taskID]
	return taskState, exists
}

// SetTaskState stores task state
func (m *EnhancedManager) SetTaskState(taskID string, taskState *types.TaskState) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.taskStates[taskID] = taskState

	// Auto-save if enabled
	if m.autoSave && m.state != nil {
		// This is a simplified auto-save, in production you might want to batch these
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Ignore error in background save, as this is best-effort
			_ = m.SaveState(ctx, m.state)
		}()
	}
}

// Clear clears all state data
func (m *EnhancedManager) Clear() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state = &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
		LastRun:   time.Now(),
	}
	m.taskStates = make(map[string]*types.TaskState)

	return nil
}

// UpdateChecksum updates the checksum for a file
func (m *EnhancedManager) UpdateChecksum(filePath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.state == nil {
		return fmt.Errorf("state not loaded")
	}

	checksum, err := m.calculateChecksum(filePath)
	if err != nil {
		return err
	}

	m.state.Checksums[filePath] = checksum
	return nil
}

// GetStats returns state statistics
func (m *EnhancedManager) GetStats() StateStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := StateStats{
		TaskStates:    len(m.taskStates),
		Checksums:     0,
		Variables:     0,
		LastRun:       time.Time{},
		StateFileSize: 0,
	}

	if m.state != nil {
		stats.Checksums = len(m.state.Checksums)
		stats.Variables = len(m.state.Variables)
		stats.LastRun = m.state.LastRun

		// Get file size
		if info, err := os.Stat(m.stateFile); err == nil {
			stats.StateFileSize = info.Size()
		}
	}

	return stats
}

// StateStats holds state statistics
type StateStats struct {
	TaskStates    int       `json:"task_states"`
	Checksums     int       `json:"checksums"`
	Variables     int       `json:"variables"`
	LastRun       time.Time `json:"last_run"`
	StateFileSize int64     `json:"state_file_size"`
}

// IsTaskUpToDate checks if a task is up to date based on its state
func (m *EnhancedManager) IsTaskUpToDate(task types.Task, host types.Host) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	taskID := m.generateTaskID(task, host)
	taskState, exists := m.taskStates[taskID]
	if !exists {
		return false
	}

	// Check if task arguments have changed
	currentChecksum := m.calculateTaskChecksum(task, host)
	return taskState.Checksum == currentChecksum
}

// generateTaskID generates a unique ID for a task on a specific host
func (m *EnhancedManager) generateTaskID(task types.Task, host types.Host) string {
	return fmt.Sprintf("%s-%s-%s", host.Name, task.Module, task.Name)
}

// calculateTaskChecksum calculates checksum for task arguments
func (m *EnhancedManager) calculateTaskChecksum(task types.Task, host types.Host) string {
	data := fmt.Sprintf("%s-%s-%v", task.Module, task.Name, task.Args)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// calculateChecksum calculates file checksum
func (m *EnhancedManager) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath) // #nosec G304 -- filePath is state file path
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// createBackup creates a backup of the current state file
func (m *EnhancedManager) createBackup() error {
	if _, err := os.Stat(m.stateFile); os.IsNotExist(err) {
		return nil // No file to backup
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := fmt.Sprintf("%s.backup.%s", m.stateFile, timestamp)

	// Copy file
	src, err := os.Open(m.stateFile) // #nosec G304 -- stateFile is constructed from fixed state file path
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupFile) // #nosec G304 -- backupFile is constructed from state file path
	if err != nil {
		return err
	}
	defer func() {
		// Best-effort cleanup in case of panic
		_ = dst.Close()
	}()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Ensure data is flushed to disk before closing
	if err = dst.Sync(); err != nil {
		return err
	}

	// Explicitly close and handle any errors
	return dst.Close()
}

// cleanupOldBackups removes old backup files, keeping only the specified number
func (m *EnhancedManager) cleanupOldBackups() error {
	if m.backupCount <= 0 {
		return nil
	}

	dir := filepath.Dir(m.stateFile)
	baseName := filepath.Base(m.stateFile)
	pattern := baseName + ".backup.*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	if len(matches) <= m.backupCount {
		return nil
	}

	// Sort by modification time and remove oldest
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: match, modTime: info.ModTime()})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.Before(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Remove oldest files
	for i := m.backupCount; i < len(files); i++ {
		if err := os.Remove(files[i].path); err != nil {
			return err
		}
	}

	return nil
}

// Restore restores state from a backup file
func (m *EnhancedManager) Restore(backupFile string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Verify backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	// Copy backup to state file
	src, err := os.Open(backupFile) // #nosec G304 -- backupFile is constructed from state file path
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(m.stateFile)
	if err != nil {
		return err
	}
	defer func() {
		// Best-effort cleanup in case of panic
		_ = dst.Close()
	}()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Ensure data is flushed to disk before closing
	if err = dst.Sync(); err != nil {
		return err
	}

	// Explicitly close and handle any errors
	if err = dst.Close(); err != nil {
		return err
	}

	// Reload state
	ctx := context.Background()
	_, err = m.LoadState(ctx)
	return err
}
