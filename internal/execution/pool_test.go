package execution

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

// Mock logger for testing
type mockLogger struct {
	debugCalls int32
	infoCalls  int32
	warnCalls  int32
	errorCalls int32
	retryCalls int32
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	atomic.AddInt32(&m.debugCalls, 1)
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	atomic.AddInt32(&m.infoCalls, 1)
}

func (m *mockLogger) Warn(format string, args ...interface{}) {
	atomic.AddInt32(&m.warnCalls, 1)
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	atomic.AddInt32(&m.errorCalls, 1)
}

func (m *mockLogger) Fatal(format string, args ...interface{}) {
	atomic.AddInt32(&m.errorCalls, 1)
}

func (m *mockLogger) SetLevel(level string) {}

func (m *mockLogger) Retry(taskName, hostName string, attempt, maxRetries int, delay time.Duration, err error) {
	atomic.AddInt32(&m.retryCalls, 1)
}

func (m *mockLogger) TaskStart(taskName, hostName string)                                     {}
func (m *mockLogger) TaskEnd(taskName, hostName string, changed, success bool)                {}
func (m *mockLogger) PlayStart(playName string, current, total int)                           {}
func (m *mockLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {}
func (m *mockLogger) Progress(completed, total int, currentTask, currentHost string)          {}

// Mock module executor for testing
type mockModuleExecutor struct {
	executeFunc  func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
	validateFunc func(args map[string]interface{}) error
}

func (m *mockModuleExecutor) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, host, args)
	}
	return types.TaskResult{Success: true}, nil
}

func (m *mockModuleExecutor) Validate(args map[string]interface{}) error {
	if m.validateFunc != nil {
		return m.validateFunc(args)
	}
	return nil
}

func (m *mockModuleExecutor) GetName() string {
	return "mock"
}

func (m *mockModuleExecutor) GetDescription() string {
	return "Mock module for testing"
}

// Mock module registry for testing
type mockModuleRegistry struct {
	modules map[string]*mockModuleExecutor
}

func (m *mockModuleRegistry) Get(name string) (interfaces.ModuleExecutor, error) {
	if module, ok := m.modules[name]; ok {
		return module, nil
	}
	return nil, errors.New("module not found")
}

func (m *mockModuleRegistry) Register(name string, module interfaces.ModuleExecutor) error {
	return nil
}

func (m *mockModuleRegistry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error) {
	if module, ok := m.modules[task.Module]; ok {
		return module.Execute(ctx, host, task.Args)
	}
	return types.TaskResult{Success: false}, errors.New("module not found")
}

func (m *mockModuleRegistry) List() []string {
	return []string{}
}

func (m *mockModuleRegistry) Unregister(name string) error {
	return nil
}

// TestNewPool tests pool creation
func TestNewPool(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(4, logger)

	assert.NotNil(t, pool, "Pool should not be nil")
	assert.Equal(t, 4, pool.maxWorkers, "Max workers should be 4")
	assert.NotNil(t, pool.semaphore, "Semaphore should not be nil")
	assert.NotNil(t, pool.results, "Results channel should not be nil")
	assert.NotNil(t, pool.ctx, "Context should not be nil")

	// Clean up
	err := pool.Close()
	assert.NoError(t, err, "Close should not return error")
}

// TestPool_Execute tests single task execution
func TestPool_Execute(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	executed := false
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			executed = true
			return types.TaskResult{
				Success: true,
				Changed: false,
				Output:  map[string]interface{}{"status": "ok"},
			}, nil
		},
	}

	task := &types.Task{
		Name:   "Test Task",
		Module: "mock",
		Args:   map[string]interface{}{},
	}

	host := types.Host{
		Name: "testhost",
		Vars: map[string]interface{}{},
	}

	resultChan := pool.Execute(task, host, map[string]interface{}{}, module)

	select {
	case result := <-resultChan:
		assert.NoError(t, result.Error, "Execution should not return error")
		assert.True(t, result.TaskResult.Success, "Task should succeed")
		assert.True(t, executed, "Module execute should be called")
	case <-time.After(2 * time.Second):
		t.Fatal("Execution timed out")
	}
}

// TestPool_Execute_WithError tests task execution with error
func TestPool_Execute_WithError(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	expectedError := errors.New("execution failed")
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: false}, expectedError
		},
	}

	task := &types.Task{
		Name:   "Failing Task",
		Module: "mock",
		Args:   map[string]interface{}{},
	}

	host := types.Host{Name: "testhost"}

	resultChan := pool.Execute(task, host, map[string]interface{}{}, module)

	select {
	case result := <-resultChan:
		assert.Error(t, result.Error, "Should return error")
		assert.Equal(t, expectedError, result.Error, "Error should match")
	case <-time.After(2 * time.Second):
		t.Fatal("Execution timed out")
	}
}

// TestPool_Execute_WithTimeout tests task execution with timeout
func TestPool_Execute_WithTimeout(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			// Simulate long-running task
			select {
			case <-time.After(5 * time.Second):
				return types.TaskResult{Success: true}, nil
			case <-ctx.Done():
				return types.TaskResult{Success: false}, ctx.Err()
			}
		},
	}

	task := &types.Task{
		Name:    "Timeout Task",
		Module:  "mock",
		Args:    map[string]interface{}{},
		Timeout: 100 * time.Millisecond,
	}

	host := types.Host{Name: "testhost"}

	resultChan := pool.Execute(task, host, map[string]interface{}{}, module)

	select {
	case result := <-resultChan:
		assert.Error(t, result.Error, "Should return timeout error")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

// TestPool_ExecuteParallel tests parallel execution of multiple tasks
func TestPool_ExecuteParallel(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(4, logger)
	defer pool.Close()

	executionCount := int32(0)
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			atomic.AddInt32(&executionCount, 1)
			time.Sleep(50 * time.Millisecond) // Simulate work
			return types.TaskResult{Success: true}, nil
		},
	}

	registry := &mockModuleRegistry{
		modules: map[string]*mockModuleExecutor{
			"mock": module,
		},
	}

	tasks := []*types.Task{
		{Name: "Task 1", Module: "mock", Args: map[string]interface{}{}},
		{Name: "Task 2", Module: "mock", Args: map[string]interface{}{}},
		{Name: "Task 3", Module: "mock", Args: map[string]interface{}{}},
	}

	hosts := []types.Host{
		{Name: "host1"},
		{Name: "host2"},
	}

	results, err := pool.ExecuteParallel(tasks, hosts, map[string]interface{}{}, registry)

	assert.NoError(t, err, "Parallel execution should not return error")
	assert.Equal(t, 6, len(results), "Should have 6 results (3 tasks × 2 hosts)")
	assert.Equal(t, int32(6), atomic.LoadInt32(&executionCount), "Should execute 6 times")
}

// TestPool_ExecuteParallel_EmptyTasks tests parallel execution with empty tasks
func TestPool_ExecuteParallel_EmptyTasks(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	registry := &mockModuleRegistry{modules: map[string]*mockModuleExecutor{}}

	results, err := pool.ExecuteParallel([]*types.Task{}, []types.Host{}, map[string]interface{}{}, registry)

	assert.NoError(t, err, "Should not return error for empty tasks")
	assert.Nil(t, results, "Results should be nil for empty tasks")
}

// TestPool_ExecuteWithRetry tests retry logic
func TestPool_ExecuteWithRetry(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	attemptCount := int32(0)
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				return types.TaskResult{Success: false}, errors.New("temporary failure")
			}
			return types.TaskResult{Success: true}, nil
		},
	}

	task := &types.Task{
		Name:    "Retry Task",
		Module:  "mock",
		Args:    map[string]interface{}{},
		Retries: 3,
		Delay:   10 * time.Millisecond,
	}

	host := types.Host{Name: "testhost"}

	result, err := pool.ExecuteWithRetry(task, host, map[string]interface{}{}, module)

	assert.NoError(t, err, "Should succeed after retries")
	assert.True(t, result.Success, "Task should succeed")
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "Should attempt 3 times")
	assert.True(t, atomic.LoadInt32(&logger.retryCalls) >= 2, "Should log retry attempts")
}

// TestPool_ExecuteWithRetry_AllFail tests retry logic when all attempts fail
func TestPool_ExecuteWithRetry_AllFail(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	attemptCount := int32(0)
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			atomic.AddInt32(&attemptCount, 1)
			return types.TaskResult{Success: false}, errors.New("persistent failure")
		},
	}

	task := &types.Task{
		Name:    "Always Fail Task",
		Module:  "mock",
		Args:    map[string]interface{}{},
		Retries: 3,
		Delay:   10 * time.Millisecond,
	}

	host := types.Host{Name: "testhost"}

	_, err := pool.ExecuteWithRetry(task, host, map[string]interface{}{}, module)

	assert.Error(t, err, "Should return error after all retries")
	assert.Contains(t, err.Error(), "failed after 3 attempts", "Error should mention retry count")
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "Should attempt 3 times")
}

// TestPool_Wait tests waiting for all tasks to complete
func TestPool_Wait(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	completed := int32(0)
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return types.TaskResult{Success: true}, nil
		},
	}

	task := &types.Task{
		Name:   "Wait Task",
		Module: "mock",
		Args:   map[string]interface{}{},
	}

	host := types.Host{Name: "testhost"}

	// Submit multiple tasks
	for i := 0; i < 5; i++ {
		pool.Execute(task, host, map[string]interface{}{}, module)
	}

	// Wait for all to complete
	pool.Wait()

	assert.Equal(t, int32(5), atomic.LoadInt32(&completed), "All tasks should complete")
}

// TestPool_Close tests pool shutdown
func TestPool_Close(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)

	// Submit a task before closing
	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: true}, nil
		},
	}
	task := &types.Task{Name: "Before Close", Module: "mock"}
	host := types.Host{Name: "testhost"}

	resultChan := pool.Execute(task, host, map[string]interface{}{}, module)

	// Wait for task to complete
	select {
	case result := <-resultChan:
		assert.NoError(t, result.Error, "Task should complete successfully")
	case <-time.After(1 * time.Second):
		t.Fatal("Task should complete before close")
	}

	// Now close the pool
	err := pool.Close()
	assert.NoError(t, err, "Close should not return error")
}

// TestPool_Stats tests pool statistics
func TestPool_Stats(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(4, logger)
	defer pool.Close()

	stats := pool.Stats()

	assert.Equal(t, 4, stats.MaxWorkers, "Max workers should be 4")
	assert.GreaterOrEqual(t, stats.ActiveWorkers, 0, "Active workers should be non-negative")
	assert.GreaterOrEqual(t, stats.QueuedTasks, 0, "Queued tasks should be non-negative")
}

// TestPool_GetStats tests GetStats interface method
func TestPool_GetStats(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(4, logger)
	defer pool.Close()

	stats := pool.GetStats()

	assert.NotNil(t, stats, "Stats should not be nil")
	assert.Equal(t, 4, stats["max_workers"], "Max workers should be 4")
	assert.Contains(t, stats, "active_workers", "Should contain active_workers")
	assert.Contains(t, stats, "queued_tasks", "Should contain queued_tasks")
}

// TestPool_Submit tests Submit interface method
func TestPool_Submit(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	executed := false
	pool.Submit(func() {
		executed = true
	})

	pool.Wait()
	assert.True(t, executed, "Submitted task should execute")
}

// TestPool_SubmitWithResult tests SubmitWithResult interface method
func TestPool_SubmitWithResult(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)
	defer pool.Close()

	expectedError := errors.New("task error")
	resultChan := pool.SubmitWithResult(func() error {
		return expectedError
	})

	select {
	case err := <-resultChan:
		assert.Equal(t, expectedError, err, "Should return expected error")
	case <-time.After(1 * time.Second):
		t.Fatal("Should receive result")
	}
}

// TestPool_Shutdown tests Shutdown interface method
func TestPool_Shutdown(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(2, logger)

	ctx := context.Background()
	err := pool.Shutdown(ctx)
	assert.NoError(t, err, "Shutdown should not return error")
}

// TestPool_ConcurrentExecution tests concurrent task execution
func TestPool_ConcurrentExecution(t *testing.T) {
	logger := &mockLogger{}
	pool := NewPool(4, logger)
	defer pool.Close()

	concurrentCount := int32(0)
	maxConcurrent := int32(0)
	var mu sync.Mutex

	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			current := atomic.AddInt32(&concurrentCount, 1)

			mu.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&concurrentCount, -1)

			return types.TaskResult{Success: true}, nil
		},
	}

	task := &types.Task{Name: "Concurrent Task", Module: "mock"}
	host := types.Host{Name: "testhost"}

	// Submit 10 tasks
	var resultChans []<-chan PoolExecutionResult
	for i := 0; i < 10; i++ {
		resultChans = append(resultChans, pool.Execute(task, host, map[string]interface{}{}, module))
	}

	// Wait for all results
	for _, ch := range resultChans {
		<-ch
	}

	mu.Lock()
	max := maxConcurrent
	mu.Unlock()

	assert.LessOrEqual(t, max, int32(4), "Should not exceed max workers")
	assert.Greater(t, max, int32(1), "Should have concurrent execution")
}

// Benchmark tests
func BenchmarkPool_Execute(b *testing.B) {
	logger := &mockLogger{}
	pool := NewPool(4, logger)
	defer pool.Close()

	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: true}, nil
		},
	}

	task := &types.Task{Name: "Benchmark Task", Module: "mock"}
	host := types.Host{Name: "testhost"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := pool.Execute(task, host, map[string]interface{}{}, module)
		<-resultChan
	}
}

func BenchmarkPool_ExecuteParallel(b *testing.B) {
	logger := &mockLogger{}
	pool := NewPool(8, logger)
	defer pool.Close()

	module := &mockModuleExecutor{
		executeFunc: func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			return types.TaskResult{Success: true}, nil
		},
	}

	registry := &mockModuleRegistry{
		modules: map[string]*mockModuleExecutor{"mock": module},
	}

	tasks := []*types.Task{
		{Name: "Task 1", Module: "mock"},
		{Name: "Task 2", Module: "mock"},
		{Name: "Task 3", Module: "mock"},
	}

	hosts := []types.Host{
		{Name: "host1"},
		{Name: "host2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pool.ExecuteParallel(tasks, hosts, map[string]interface{}{}, registry)
	}
}
