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

func TestNewMetricsWithPrometheus(t *testing.T) {
	m := NewMetricsWithPrometheus()

	assert.NotNil(t, m)
	assert.NotNil(t, m.promRegistry)
	assert.NotNil(t, m.promMetrics)
	assert.NotNil(t, m.promMetrics.TasksTotal)
	assert.NotNil(t, m.promMetrics.TaskDuration)
	assert.NotNil(t, m.promMetrics.PlaybooksTotal)
	assert.NotNil(t, m.promMetrics.PlaysTotal)
	assert.NotNil(t, m.promMetrics.CacheHitRate)
	assert.NotNil(t, m.promMetrics.HostsConnected)
	assert.NotNil(t, m.promMetrics.ConcurrentTasks)
	assert.NotNil(t, m.promMetrics.ErrorsTotal)
	assert.NotNil(t, m.promMetrics.ModuleUsage)
	assert.Equal(t, int64(0), m.PlaybooksExecuted)
	assert.NotNil(t, m.ModuleUsage)
	assert.False(t, m.StartTime.IsZero())
}

func TestMetrics_AddTaskExecutionTime(t *testing.T) {
	m := NewMetrics()

	duration := 150 * time.Millisecond
	m.AddTaskExecutionTime("test-module", "host1", duration)

	assert.Equal(t, duration, m.TotalExecutionTime)
	assert.Equal(t, duration, m.MinTaskTime)
	assert.Equal(t, duration, m.MaxTaskTime)

	// Add another task with different duration
	duration2 := 250 * time.Millisecond
	m.AddTaskExecutionTime("test-module", "host2", duration2)

	assert.Equal(t, duration+duration2, m.TotalExecutionTime)
	assert.Equal(t, duration, m.MinTaskTime)
	assert.Equal(t, duration2, m.MaxTaskTime)
}

func TestMetrics_RecordHostExecutionTime(t *testing.T) {
	m := NewMetrics()

	m.RecordHostExecutionTime("host1", 100*time.Millisecond)
	m.RecordHostExecutionTime("host2", 200*time.Millisecond)
	m.RecordHostExecutionTime("host1", 150*time.Millisecond)

	assert.Equal(t, int64(250), m.HostExecutionTime["host1"]) // 100 + 150
	assert.Equal(t, int64(200), m.HostExecutionTime["host2"])
}

func TestMetrics_IncrementHostsConnected(t *testing.T) {
	m := NewMetrics()

	m.IncrementHostsConnected()
	assert.Equal(t, int64(1), m.HostsConnected)

	m.IncrementHostsConnected()
	assert.Equal(t, int64(2), m.HostsConnected)
}

func TestMetrics_IncrementHostsUnreachable(t *testing.T) {
	m := NewMetrics()

	m.IncrementHostsUnreachable()
	assert.Equal(t, int64(1), m.HostsUnreachable)

	m.IncrementHostsUnreachable()
	assert.Equal(t, int64(2), m.HostsUnreachable)
}

func TestMetrics_SetConcurrentTasks(t *testing.T) {
	m := NewMetrics()

	m.SetConcurrentTasks(5)
	assert.Equal(t, int64(5), m.CurrentConcurrentTasks)

	m.SetConcurrentTasks(10)
	assert.Equal(t, int64(10), m.CurrentConcurrentTasks)
	assert.Equal(t, int64(10), m.MaxConcurrentTasks)

	// Test with Prometheus metrics
	mProm := NewMetricsWithPrometheus()
	mProm.SetConcurrentTasks(3)
	assert.Equal(t, int64(3), mProm.CurrentConcurrentTasks)
}

func TestMetrics_UpdateResourceUsage(t *testing.T) {
	m := NewMetrics()

	m.UpdateResourceUsage(1024*1024, 50.5, 1000, 2000) // 1MB memory, 50.5% CPU, 1000 network, 2000 disk

	assert.Equal(t, int64(1024*1024), m.MemoryUsage)
	assert.Equal(t, 50.5, m.CPUUsage)
	assert.Equal(t, int64(1000), m.NetworkBytes)
	assert.Equal(t, int64(2000), m.DiskIOBytes)
}

func TestMetrics_GetSummary(t *testing.T) {
	m := NewMetrics()

	// Add some data
	m.IncrementPlaybooksExecuted()
	m.IncrementTasksExecuted()
	m.IncrementTasksSucceeded()
	m.IncrementModuleUsage("file")
	m.IncrementModuleUsage("command")
	m.IncrementModuleUsage("file")
	m.AddExecutionTime(100 * time.Millisecond)

	summary := m.GetSummary()

	assert.NotNil(t, summary)
	assert.NotNil(t, summary.Overview)
	assert.Equal(t, int64(1), summary.Overview.PlaybooksExecuted)
	assert.Equal(t, int64(1), summary.Overview.TasksExecuted)
	assert.Equal(t, int64(1), summary.Overview.TasksSucceeded)
	assert.Equal(t, 100.0, summary.Overview.SuccessRate)
	assert.NotNil(t, summary.Modules)
	assert.Equal(t, int64(2), summary.Modules.Usage["file"])
	assert.Equal(t, int64(1), summary.Modules.Usage["command"])
}

func TestMetrics_GetPrometheusHandler(t *testing.T) {
	m := NewMetricsWithPrometheus()

	handler := m.GetPrometheusHandler()
	assert.NotNil(t, handler)
}

func TestMetrics_RecordTaskResult(t *testing.T) {
	m := NewMetrics()

	// Test successful task
	m.RecordTaskResult("file", "host1", "success", 100*time.Millisecond, false)
	assert.Equal(t, int64(1), m.TasksExecuted)
	assert.Equal(t, int64(1), m.TasksSucceeded)
	assert.Equal(t, int64(1), m.ModuleUsage["file"])

	// Test successful task with change
	m.RecordTaskResult("file", "host1", "success", 100*time.Millisecond, true)
	assert.Equal(t, int64(2), m.TasksExecuted)
	assert.Equal(t, int64(2), m.TasksSucceeded)
	assert.Equal(t, int64(1), m.TasksChanged)

	// Test failed task
	m.RecordTaskResult("command", "host2", "failed", 50*time.Millisecond, false)
	assert.Equal(t, int64(3), m.TasksExecuted)
	assert.Equal(t, int64(1), m.TasksFailed)
	assert.Equal(t, int64(1), m.ModuleUsage["command"])

	// Test skipped task
	m.RecordTaskResult("copy", "host3", "skipped", 10*time.Millisecond, false)
	assert.Equal(t, int64(4), m.TasksExecuted)
	assert.Equal(t, int64(1), m.TasksSkipped)
}

func TestMetrics_GetFormattedSummary(t *testing.T) {
	m := NewMetrics()

	// Add some data
	m.IncrementPlaybooksExecuted()
	m.IncrementTasksExecuted()
	m.IncrementTasksSucceeded()
	m.IncrementCacheHits()
	m.IncrementCacheMisses()
	m.AddExecutionTime(500 * time.Millisecond)

	summary := m.GetFormattedSummary()

	assert.Contains(t, summary, "=== Onigirazu Metrics Summary ===")
	assert.Contains(t, summary, "📊 Overview:")
	assert.Contains(t, summary, "Playbooks: 1")
	assert.Contains(t, summary, "Tasks: 1")
	assert.Contains(t, summary, "Success: 1 (100.0%)")
	assert.Contains(t, summary, "⚡ Performance:")
	assert.Contains(t, summary, "🖥️  Hosts:")
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	m := NewMetrics()

	// Test concurrent access to metrics
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.IncrementTasksExecuted()
				m.IncrementModuleUsage("test")
				m.AddExecutionTime(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, int64(1000), m.TasksExecuted)
	assert.Equal(t, int64(1000), m.ModuleUsage["test"])
}

func TestMetrics_PrometheusIntegration(t *testing.T) {
	m := NewMetricsWithPrometheus()

	// Test that Prometheus metrics are updated
	m.IncrementPlaybooksExecuted()
	m.IncrementTasksExecuted()
	m.IncrementTasksSucceeded()
	m.IncrementModuleUsage("file")
	m.IncrementCacheHits()
	m.IncrementCacheMisses()

	// Verify basic metrics are tracked
	assert.Equal(t, int64(1), m.PlaybooksExecuted)
	assert.Equal(t, int64(1), m.TasksExecuted)
	assert.Equal(t, int64(1), m.TasksSucceeded)
	assert.Equal(t, int64(1), m.ModuleUsage["file"])
}
