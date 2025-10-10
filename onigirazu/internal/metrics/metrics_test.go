package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()

	assert.NotNil(t, m)
	assert.Equal(t, int64(0), m.PlaybooksExecuted)
	assert.Equal(t, int64(0), m.TasksExecuted)
	assert.NotNil(t, m.ModuleUsage)
	assert.NotNil(t, m.ErrorsByModule)
	assert.NotNil(t, m.ErrorsByType)
	assert.False(t, m.StartTime.IsZero())
}

func TestMetrics_IncrementCounters(t *testing.T) {
	m := NewMetrics()

	// Test playbook counters
	m.IncrementPlaybooksExecuted()
	assert.Equal(t, int64(1), m.PlaybooksExecuted)

	// Test play counters
	m.IncrementPlaysExecuted()
	assert.Equal(t, int64(1), m.PlaysExecuted)

	// Test task counters
	m.IncrementTasksExecuted()
	m.IncrementTasksSucceeded()
	m.IncrementTasksFailed()
	m.IncrementTasksSkipped()
	m.IncrementTasksChanged()

	assert.Equal(t, int64(1), m.TasksExecuted)
	assert.Equal(t, int64(1), m.TasksSucceeded)
	assert.Equal(t, int64(1), m.TasksFailed)
	assert.Equal(t, int64(1), m.TasksSkipped)
	assert.Equal(t, int64(1), m.TasksChanged)
}

func TestMetrics_AddExecutionTime(t *testing.T) {
	m := NewMetrics()

	// Add some execution time
	duration1 := 100 * time.Millisecond
	duration2 := 200 * time.Millisecond

	m.IncrementTasksExecuted()
	m.AddExecutionTime(duration1)

	assert.Equal(t, duration1, m.TotalExecutionTime)
	assert.Equal(t, duration1, m.AverageTaskTime)

	m.IncrementTasksExecuted()
	m.AddExecutionTime(duration2)

	expectedTotal := duration1 + duration2
	expectedAverage := expectedTotal / 2

	assert.Equal(t, expectedTotal, m.TotalExecutionTime)
	assert.Equal(t, expectedAverage, m.AverageTaskTime)
}

func TestMetrics_ModuleUsage(t *testing.T) {
	m := NewMetrics()

	// Test module usage tracking
	m.IncrementModuleUsage("file")
	m.IncrementModuleUsage("command")
	m.IncrementModuleUsage("file") // Increment file again

	assert.Equal(t, int64(2), m.ModuleUsage["file"])
	assert.Equal(t, int64(1), m.ModuleUsage["command"])
}

func TestMetrics_ErrorTracking(t *testing.T) {
	m := NewMetrics()

	// Test error tracking by module
	m.IncrementErrorByModule("file")
	m.IncrementErrorByModule("command")
	m.IncrementErrorByModule("file") // Increment file again

	assert.Equal(t, int64(2), m.ErrorsByModule["file"])
	assert.Equal(t, int64(1), m.ErrorsByModule["command"])

	// Test error tracking by type
	m.IncrementErrorByType("validation_error")
	m.IncrementErrorByType("execution_error")
	m.IncrementErrorByType("validation_error") // Increment validation again

	assert.Equal(t, int64(2), m.ErrorsByType["validation_error"])
	assert.Equal(t, int64(1), m.ErrorsByType["execution_error"])
}

func TestMetrics_CacheMetrics(t *testing.T) {
	m := NewMetrics()

	// Test cache metrics
	m.IncrementCacheHits()
	m.IncrementCacheMisses()
	m.IncrementCacheHits() // Increment hits again

	assert.Equal(t, int64(2), m.CacheHits)
	assert.Equal(t, int64(1), m.CacheMisses)
}

func TestMetrics_GetSnapshot(t *testing.T) {
	m := NewMetrics()

	// Add some data
	m.IncrementPlaybooksExecuted()
	m.IncrementTasksExecuted()
	m.IncrementModuleUsage("file")
	m.IncrementErrorByModule("command")

	// Get snapshot
	snapshot := m.GetSnapshot()

	// Verify snapshot has same data
	assert.Equal(t, m.PlaybooksExecuted, snapshot.PlaybooksExecuted)
	assert.Equal(t, m.TasksExecuted, snapshot.TasksExecuted)
	assert.Equal(t, m.ModuleUsage["file"], snapshot.ModuleUsage["file"])
	assert.Equal(t, m.ErrorsByModule["command"], snapshot.ErrorsByModule["command"])

	// Modify original - snapshot should be unchanged
	m.IncrementPlaybooksExecuted()
	assert.NotEqual(t, m.PlaybooksExecuted, snapshot.PlaybooksExecuted)
}

func TestMetrics_Reset(t *testing.T) {
	m := NewMetrics()

	// Add some data
	m.IncrementPlaybooksExecuted()
	m.IncrementTasksExecuted()
	m.IncrementModuleUsage("file")
	m.AddExecutionTime(100 * time.Millisecond)

	// Verify data exists
	assert.Equal(t, int64(1), m.PlaybooksExecuted)
	assert.Equal(t, int64(1), m.TasksExecuted)
	assert.Equal(t, int64(1), m.ModuleUsage["file"])
	assert.Greater(t, m.TotalExecutionTime, time.Duration(0))

	// Reset
	oldStartTime := m.StartTime
	time.Sleep(1 * time.Millisecond) // Ensure different start time
	m.Reset()

	// Verify everything is reset
	assert.Equal(t, int64(0), m.PlaybooksExecuted)
	assert.Equal(t, int64(0), m.TasksExecuted)
	assert.Equal(t, 0, len(m.ModuleUsage))
	assert.Equal(t, time.Duration(0), m.TotalExecutionTime)
	assert.True(t, m.StartTime.After(oldStartTime))
}

func TestMetrics_GetSuccessRate(t *testing.T) {
	m := NewMetrics()

	// No tasks executed - should return 0
	assert.Equal(t, 0.0, m.GetSuccessRate())

	// Add some tasks
	m.IncrementTasksExecuted()
	m.IncrementTasksExecuted()
	m.IncrementTasksSucceeded()

	// 1 success out of 2 tasks = 50%
	assert.Equal(t, 50.0, m.GetSuccessRate())

	// Add another success
	m.IncrementTasksSucceeded()

	// 2 successes out of 2 tasks = 100%
	assert.Equal(t, 100.0, m.GetSuccessRate())
}

func TestMetrics_GetCacheHitRate(t *testing.T) {
	m := NewMetrics()

	// No cache operations - should return 0
	assert.Equal(t, 0.0, m.GetCacheHitRate())

	// Add some cache operations
	m.IncrementCacheHits()
	m.IncrementCacheMisses()

	// 1 hit out of 2 operations = 50%
	assert.Equal(t, 50.0, m.GetCacheHitRate())

	// Add another hit
	m.IncrementCacheHits()

	// 2 hits out of 3 operations = 66.67%
	assert.InDelta(t, 66.67, m.GetCacheHitRate(), 0.01)
}

func TestMetrics_GetUptime(t *testing.T) {
	m := NewMetrics()

	// Sleep a bit to ensure uptime > 0
	time.Sleep(1 * time.Millisecond)

	uptime := m.GetUptime()
	assert.Greater(t, uptime, time.Duration(0))
	assert.Less(t, uptime, 1*time.Second) // Should be very small for test
}
