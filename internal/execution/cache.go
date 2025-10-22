package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ExecutionID generates unique execution IDs
type ExecutionID string

// HostResult stores result for a single host
type HostResult struct {
	Hostname  string    `json:"hostname"`
	Status    string    `json:"status"` // success, failed, skipped, changed
	Error     string    `json:"error,omitempty"`
	ExitCode  int       `json:"exit_code,omitempty"`
	Stderr    string    `json:"stderr,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Output    string    `json:"output,omitempty"`
}

// TaskResult stores aggregated result for a task
type TaskResult struct {
	Name         string                `json:"name"`
	Total        int                   `json:"total"`
	Success      int                   `json:"success"`
	Failed       int                   `json:"failed"`
	Changed      int                   `json:"changed"`
	Skipped      int                   `json:"skipped"`
	Duration     time.Duration         `json:"duration"`
	HostResults  map[string]HostResult `json:"host_results"`   // hostname -> result
	ErrorsByType map[string][]string   `json:"errors_by_type"` // error type -> hosts
	StartTime    time.Time             `json:"start_time"`
	EndTime      time.Time             `json:"end_time"`
}

// ExecutionResult stores complete execution data
type ExecutionResult struct {
	ExecutionID    string                 `json:"execution_id"`
	Timestamp      time.Time              `json:"timestamp"`
	PlaybookPath   string                 `json:"playbook_path"`
	PlaybookName   string                 `json:"playbook_name"`
	TotalHosts     int                    `json:"total_hosts"`
	Tasks          []TaskResult           `json:"tasks"`
	Status         string                 `json:"status"` // success, partial_success, failed
	TotalSuccess   int                    `json:"total_success"`
	TotalFailed    int                    `json:"total_failed"`
	TotalChanged   int                    `json:"total_changed"`
	TotalSkipped   int                    `json:"total_skipped"`
	Duration       time.Duration          `json:"duration"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	HostResults    map[string]*HostResult `json:"host_results,omitempty"`
	PlaybookResult interface{}            `json:"playbook_result,omitempty"` // Complete playbook result
	CacheFile      string                 `json:"-"`
}

// CacheManager handles execution result storage and retrieval
type CacheManager struct {
	cacheDir string
}

// NewCacheManager creates a new cache manager
func NewCacheManager() (*CacheManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".onigirazu", "cache", "executions")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CacheManager{cacheDir: cacheDir}, nil
}

// NewCacheManagerWithPath creates a new cache manager with a specific path
func NewCacheManagerWithPath(cacheDir string) (*CacheManager, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CacheManager{cacheDir: cacheDir}, nil
}

// Save persists an execution result to cache
func (cm *CacheManager) Save(result *ExecutionResult) error {
	if result.ExecutionID == "" {
		result.ExecutionID = generateExecutionID()
	}

	filename := filepath.Join(cm.cacheDir, result.ExecutionID+".json")
	result.CacheFile = filename

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Also update "current.json" symlink
	currentFile := filepath.Join(cm.cacheDir, "current.json")
	os.Remove(currentFile) // Ignore error if file doesn't exist
	if err := os.WriteFile(currentFile, data, 0644); err != nil {
		// Non-critical error
		fmt.Fprintf(os.Stderr, "Warning: failed to update current.json: %v\n", err)
	}

	return nil
}

// LoadLatest loads the most recent execution result
func (cm *CacheManager) LoadLatest() (*ExecutionResult, error) {
	entries, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no cached executions found")
	}

	// Find most recent .json file (excluding current.json)
	var latestFile string
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "current.json" {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = filepath.Join(cm.cacheDir, entry.Name())
		}
	}

	if latestFile == "" {
		return nil, fmt.Errorf("no valid cached executions found")
	}

	return cm.Load(filepath.Base(latestFile))
}

// Load loads a specific execution result by ID (without .json extension)
func (cm *CacheManager) Load(executionID string) (*ExecutionResult, error) {
	// Add .json extension if not present
	if filepath.Ext(executionID) != ".json" {
		executionID = executionID + ".json"
	}

	filename := filepath.Join(cm.cacheDir, executionID)

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var result ExecutionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution result: %w", err)
	}

	result.CacheFile = filename
	return &result, nil
}

// ListExecutions lists all cached executions
func (cm *CacheManager) ListExecutions(limit int) ([]ExecutionResult, error) {
	entries, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var results []ExecutionResult

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "current.json" {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		result, err := cm.Load(entry.Name())
		if err != nil {
			continue
		}

		results = append(results, *result)
	}

	// Sort by timestamp descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// generateExecutionID creates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec-%d", time.Now().UnixNano())
}
