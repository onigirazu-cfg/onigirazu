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

// ExecutionState holds state for a single execution
type ExecutionState struct {
	TaskStates map[string]*types.TaskState
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// EnhancedManager provides thread-safe state management with advanced features
type EnhancedManager struct {
	stateFile     string
	mutex         sync.RWMutex
	state         *types.State
	executions    map[string]*ExecutionState  // Per-execution isolation
	taskStates    map[string]*types.TaskState // Legacy compatibility
	currentExecID string                      // Track current execution
	autoSave      bool
	backupCount   int
	saveWg        sync.WaitGroup     // Track background save operations
	ctx           context.Context    // Parent context for cancellation
	cancel        context.CancelFunc // Cancel function
	checksumCache sync.Map           // Caches task checksums (task ID -> checksum)
	fileChecksum  map[string]string  // Caches file checksums
}

// NewEnhanced creates a new enhanced state manager
func NewEnhanced(stateFile string, autoSave bool, backupCount int) *EnhancedManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &EnhancedManager{
		stateFile:    stateFile,
		taskStates:   make(map[string]*types.TaskState), // Legacy compatibility
		executions:   make(map[string]*ExecutionState),  // Per-execution isolation
		autoSave:     autoSave,
		backupCount:  backupCount,
		ctx:          ctx,
		cancel:       cancel,
		fileChecksum: make(map[string]string), // Initialize checksum cache
	}
}

// NewEnhancedManager creates a new enhanced state manager with logger
func NewEnhancedManager(stateFile string, logger interface{}) *EnhancedManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &EnhancedManager{
		stateFile:    stateFile,
		taskStates:   make(map[string]*types.TaskState), // Legacy compatibility
		executions:   make(map[string]*ExecutionState),  // Per-execution isolation
		autoSave:     true,
		backupCount:  5,
		ctx:          ctx,
		cancel:       cancel,
		fileChecksum: make(map[string]string), // Initialize checksum cache
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

	// Cleanup old backups (don't fail on backup cleanup errors)
	if err := m.cleanupOldBackups(); err != nil {
		// Note: This is not critical for the main operation, so we skip it
		_ = err
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

// GetTaskState retrieves task state by ID from the current execution or legacy storage
func (m *EnhancedManager) GetTaskState(taskID string) (*types.TaskState, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if we have a current execution with isolated state
	if m.currentExecID != "" {
		if execState, exists := m.executions[m.currentExecID]; exists {
			if taskState, found := execState.TaskStates[taskID]; found {
				return taskState, true
			}
		}
	}

	// Fall back to legacy global task states for backwards compatibility
	taskState, exists := m.taskStates[taskID]
	return taskState, exists
}

// SetTaskState stores task state in the current execution or legacy storage
func (m *EnhancedManager) SetTaskState(taskID string, taskState *types.TaskState) {
	m.mutex.Lock()

	// Store in current execution's isolated state if available
	if m.currentExecID != "" {
		if execState, exists := m.executions[m.currentExecID]; exists {
			execState.TaskStates[taskID] = taskState
			execState.UpdatedAt = time.Now()
		}
	}

	// Also store in legacy global state for backwards compatibility
	m.taskStates[taskID] = taskState

	// Auto-save if enabled
	if m.autoSave && m.state != nil {
		// Capture state while holding lock to avoid race
		stateCopy := m.state
		// Add to WaitGroup while holding mutex
		m.saveWg.Add(1)
		// Release mutex before launching goroutine
		m.mutex.Unlock()

		// Launch goroutine for background save
		go func() {
			defer m.saveWg.Done()
			ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
			defer cancel()
			// Ignore error in background save, as this is best-effort
			_ = m.SaveState(ctx, stateCopy)
		}()
		return
	}

	m.mutex.Unlock()
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
	m.executions = make(map[string]*ExecutionState)
	m.currentExecID = ""

	return nil
}

// Shutdown gracefully shuts down the manager, waiting for all pending save operations
// It cancels the context, waits for goroutines to complete, and cleans up resources
func (m *EnhancedManager) Shutdown(timeout time.Duration) error {
	m.cancel() // Signal all goroutines to stop

	// Wait for background save operations with timeout
	done := make(chan struct{})
	go func() {
		m.saveWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout: background save operations did not complete within %v", timeout)
	}
}

// ==================== EXECUTION ISOLATION (Phase 1 Fix) ====================

// BeginExecution creates a new isolated execution context
// ExecutionID should be unique per playbook/deployment run
func (m *EnhancedManager) BeginExecution(executionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if executionID == "" {
		return fmt.Errorf("executionID cannot be empty")
	}

	// Check if this execution already exists
	if _, exists := m.executions[executionID]; exists {
		return fmt.Errorf("execution %s already exists", executionID)
	}

	// Create new isolated execution state
	m.executions[executionID] = &ExecutionState{
		TaskStates: make(map[string]*types.TaskState),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set as current execution
	m.currentExecID = executionID

	return nil
}

// EndExecution marks an execution as complete and keeps it for auditing
// If cleanup is true, removes the execution state to free resources
func (m *EnhancedManager) EndExecution(executionID string, cleanup bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if executionID == "" {
		return fmt.Errorf("executionID cannot be empty")
	}

	// Verify execution exists
	if _, exists := m.executions[executionID]; !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	// Clear current execution if it matches
	if m.currentExecID == executionID {
		m.currentExecID = ""
	}

	// Optionally cleanup
	if cleanup {
		delete(m.executions, executionID)
	}

	return nil
}

// GetExecutionTaskState retrieves task state from a specific execution
func (m *EnhancedManager) GetExecutionTaskState(executionID, taskID string) (*types.TaskState, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	execState, exists := m.executions[executionID]
	if !exists {
		return nil, false
	}

	taskState, found := execState.TaskStates[taskID]
	return taskState, found
}

// ListExecutions returns a list of all active executions
func (m *EnhancedManager) ListExecutions() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	execIDs := make([]string, 0, len(m.executions))
	for execID := range m.executions {
		execIDs = append(execIDs, execID)
	}
	return execIDs
}

// GetExecutionStats returns statistics for a specific execution
func (m *EnhancedManager) GetExecutionStats(executionID string) (int, time.Time, time.Time, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	execState, exists := m.executions[executionID]
	if !exists {
		return 0, time.Time{}, time.Time{}, fmt.Errorf("execution %s not found", executionID)
	}

	return len(execState.TaskStates), execState.CreatedAt, execState.UpdatedAt, nil
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

	// Check if task arguments have changed (using cached checksum)
	currentChecksum := m.getOrCalcTaskChecksum(taskID, task, host)
	return taskState.Checksum == currentChecksum
}

// generateTaskID generates a unique ID for a task on a specific host
func (m *EnhancedManager) generateTaskID(task types.Task, host types.Host) string {
	return fmt.Sprintf("%s-%s-%s", host.Name, task.Module, task.Name)
}

// getOrCalcTaskChecksum gets cached checksum or calculates it if not cached
func (m *EnhancedManager) getOrCalcTaskChecksum(taskID string, task types.Task, host types.Host) string {
	// Check cache first (lock-free read with sync.Map)
	if cached, ok := m.checksumCache.Load(taskID); ok {
		return cached.(string)
	}

	// Calculate and cache
	checksum := m.calculateTaskChecksum(task, host)
	m.checksumCache.Store(taskID, checksum)
	return checksum
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

	// Verify backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		m.mutex.Unlock()
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	// Copy backup to state file
	src, err := os.Open(backupFile) // #nosec G304 -- backupFile is constructed from state file path
	if err != nil {
		m.mutex.Unlock()
		return err
	}
	defer src.Close()

	dst, err := os.Create(m.stateFile)
	if err != nil {
		m.mutex.Unlock()
		return err
	}
	defer func() {
		// Best-effort cleanup in case of panic
		_ = dst.Close()
	}()

	if _, err = io.Copy(dst, src); err != nil {
		m.mutex.Unlock()
		return err
	}

	// Ensure data is flushed to disk before closing
	if err = dst.Sync(); err != nil {
		m.mutex.Unlock()
		return err
	}

	// Explicitly close and handle any errors
	if err = dst.Close(); err != nil {
		m.mutex.Unlock()
		return err
	}

	// Unlock before reloading state to avoid deadlock
	m.mutex.Unlock()

	// Reload state
	ctx := context.Background()
	_, err = m.LoadState(ctx)
	return err
}

// ==================== PHASE 1 FEATURES ====================

// LoadStateWithMigration loads state from file with automatic migration and validation
func (m *EnhancedManager) LoadStateWithMigration(ctx context.Context) (*types.State, error) {
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
			Version:   1, // Set default version
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
			LastRun:   time.Now(),
			Metadata:  &types.ExecutionMetadata{},
		}
		return m.state, nil
	}

	data, err := os.ReadFile(m.stateFile) // #nosec G304 -- stateFile is constructed from fixed state file path
	if err != nil {
		return nil, fmt.Errorf("error reading state file: %w", err)
	}

	// Use migrator to load and migrate state
	migrator := NewMigrator()
	state, err := migrator.MigrateJSON(data)
	if err != nil {
		return nil, fmt.Errorf("error migrating state: %w", err)
	}

	// Validate state
	validator := NewValidator(false) // Non-strict mode for backwards compatibility
	result := validator.Validate(state)
	if !result.Valid {
		// Try to repair
		_, repaired := validator.ValidateAndRepair(state)
		if repaired {
			fmt.Printf("Warning: State file had issues that were automatically repaired\n")
		}
	}

	m.state = state
	return m.state, nil
}

// SaveStateWithCompression saves state with optional compression
func (m *EnhancedManager) SaveStateWithCompression(ctx context.Context, state *types.State) error {
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

	// Update timestamp and version
	state.LastRun = time.Now()
	if state.Version == 0 {
		state.Version = 1 // Set to latest version if not set
	}

	// Use compression manager
	compMgr := NewCompressionManager(DefaultCompressionConfig())
	data, err := compMgr.CompressState(state)
	if err != nil {
		return fmt.Errorf("error compressing state: %w", err)
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

	// Cleanup old backups (don't fail on backup cleanup errors)
	if err := m.cleanupOldBackups(); err != nil {
		// Note: This is not critical for the main operation, so we skip it
		_ = err
	}

	return nil
}

// RotateResults applies rotation policy to state
func (m *EnhancedManager) RotateResults() (*RotationStats, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.state == nil {
		return nil, fmt.Errorf("state not loaded")
	}

	rotMgr := NewRotationManager(DefaultRotationPolicy())
	stats, err := rotMgr.RotateState(m.state)
	if err != nil {
		return stats, fmt.Errorf("error rotating results: %w", err)
	}

	return stats, nil
}

// ValidateState performs comprehensive validation on current state
func (m *EnhancedManager) ValidateState(strictMode bool) *ValidationResult {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.state == nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []ValidationError{{Field: "State", Message: "state not loaded"}},
		}
	}

	validator := NewValidator(strictMode)
	return validator.Validate(m.state)
}

// GetCompressionStats returns compression statistics for current state
func (m *EnhancedManager) GetCompressionStats() (*CompressionStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.state == nil {
		return nil, fmt.Errorf("state not loaded")
	}

	compMgr := NewCompressionManager(DefaultCompressionConfig())
	return compMgr.GetStats(m.state)
}

// DeduplicateResults removes duplicate results from state
func (m *EnhancedManager) DeduplicateResults() (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.state == nil {
		return 0, fmt.Errorf("state not loaded")
	}

	rotMgr := NewRotationManager(DefaultRotationPolicy())
	return rotMgr.DeduplicateResults(m.state), nil
}
