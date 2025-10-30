package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheManager(t *testing.T) {
	manager, err := NewCacheManager()
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotEmpty(t, manager.cacheDir)
}

func TestCacheManager_Save(t *testing.T) {
	manager, err := NewCacheManager()
	require.NoError(t, err)

	result := &ExecutionResult{
		ExecutionID:  "test-exec-123",
		Timestamp:    time.Now(),
		PlaybookPath: "/tmp/test.yaml",
		PlaybookName: "Test Playbook",
		TotalHosts:   2,
		Status:       "success",
		TotalSuccess: 2,
		Duration:     5 * time.Second,
		StartTime:    time.Now().Add(-5 * time.Second),
		EndTime:      time.Now(),
	}

	err = manager.Save(result)
	assert.NoError(t, err)

	// Verify file was created
	expectedFile := filepath.Join(manager.cacheDir, "test-exec-123.json")
	_, err = os.Stat(expectedFile)
	assert.NoError(t, err)

	// Clean up
	defer os.Remove(expectedFile)
}

func TestCacheManager_Load(t *testing.T) {
	manager, err := NewCacheManager()
	require.NoError(t, err)

	// Create and save a result
	original := &ExecutionResult{
		ExecutionID:  "test-exec-456",
		Timestamp:    time.Now(),
		PlaybookPath: "/tmp/test.yaml",
		PlaybookName: "Test Playbook",
		TotalHosts:   2,
		Status:       "success",
		TotalSuccess: 2,
		Duration:     5 * time.Second,
		StartTime:    time.Now().Add(-5 * time.Second),
		EndTime:      time.Now(),
	}

	err = manager.Save(original)
	require.NoError(t, err)
	defer os.Remove(filepath.Join(manager.cacheDir, "test-exec-456.json"))

	// Load and verify
	loaded, err := manager.Load("test-exec-456")
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, original.ExecutionID, loaded.ExecutionID)
	assert.Equal(t, original.PlaybookName, loaded.PlaybookName)
	assert.Equal(t, original.TotalHosts, loaded.TotalHosts)
}

func TestCacheManager_Load_NotFound(t *testing.T) {
	manager, err := NewCacheManager()
	require.NoError(t, err)

	loaded, err := manager.Load("nonexistent-exec-id")
	assert.Error(t, err)
	assert.Nil(t, loaded)
}

func TestCacheManager_ListExecutions(t *testing.T) {
	manager, err := NewCacheManager()
	require.NoError(t, err)

	// Save multiple results
	for i := 1; i <= 3; i++ {
		result := &ExecutionResult{
			ExecutionID:  "test-exec-list-" + string(rune('0'+i)),
			Timestamp:    time.Now(),
			PlaybookName: "Test Playbook",
			Status:       "success",
		}
		err = manager.Save(result)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(manager.cacheDir, result.ExecutionID+".json"))
	}

	// List results
	results, err := manager.ListExecutions(10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 3)
}

func TestCacheManager_LoadLatest(t *testing.T) {
	manager, err := NewCacheManager()
	require.NoError(t, err)

	// Create and save a result
	result := &ExecutionResult{
		ExecutionID:  "test-exec-latest",
		Timestamp:    time.Now(),
		PlaybookName: "Test Playbook",
		Status:       "success",
	}

	err = manager.Save(result)
	require.NoError(t, err)

	filePath := filepath.Join(manager.cacheDir, "test-exec-latest.json")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Load latest
	latest, err := manager.LoadLatest()
	assert.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, "test-exec-latest", latest.ExecutionID)

	// Clean up
	defer os.Remove(filePath)
}

func TestHostResult_Structure(t *testing.T) {
	hostResult := HostResult{
		Hostname:  "test-host",
		Status:    "success",
		ExitCode:  0,
		Timestamp: time.Now(),
		Output:    "test output",
	}

	assert.Equal(t, "test-host", hostResult.Hostname)
	assert.Equal(t, "success", hostResult.Status)
	assert.Equal(t, 0, hostResult.ExitCode)

	// Test JSON marshaling
	data, err := json.Marshal(hostResult)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test JSON unmarshaling
	var unmarshaled HostResult
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, hostResult.Hostname, unmarshaled.Hostname)
	assert.Equal(t, hostResult.Status, unmarshaled.Status)
}

func TestTaskResult_Structure(t *testing.T) {
	taskResult := TaskResult{
		Name:     "test-task",
		Total:    2,
		Success:  2,
		Failed:   0,
		Changed:  1,
		Skipped:  0,
		Duration: 1 * time.Second,
		HostResults: map[string]HostResult{
			"host1": {
				Hostname: "host1",
				Status:   "success",
			},
			"host2": {
				Hostname: "host2",
				Status:   "success",
			},
		},
	}

	assert.Equal(t, "test-task", taskResult.Name)
	assert.Equal(t, 2, taskResult.Total)
	assert.Equal(t, 2, taskResult.Success)
	assert.Equal(t, 1, taskResult.Changed)
}

func TestExecutionResult_Structure(t *testing.T) {
	now := time.Now()
	result := &ExecutionResult{
		ExecutionID:  "test-exec-789",
		Timestamp:    now,
		PlaybookPath: "/tmp/test.yaml",
		PlaybookName: "Test Playbook",
		TotalHosts:   2,
		Tasks:        []TaskResult{},
		Status:       "success",
		TotalSuccess: 2,
		TotalFailed:  0,
		TotalChanged: 1,
		TotalSkipped: 0,
		Duration:     10 * time.Second,
		StartTime:    now.Add(-10 * time.Second),
		EndTime:      now,
	}

	assert.Equal(t, "test-exec-789", result.ExecutionID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 2, result.TotalHosts)
	assert.Equal(t, 2, result.TotalSuccess)
}
