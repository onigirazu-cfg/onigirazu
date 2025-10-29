package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Pool manages parallel execution of tasks
type Pool struct {
	maxWorkers int
	semaphore  chan struct{}
	logger     interfaces.Logger
	results    chan TaskExecution
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// TaskExecution represents a task execution request
type TaskExecution struct {
	Task      *types.Task
	Host      types.Host
	Variables map[string]interface{}
	Module    interfaces.ModuleExecutor
	Result    chan PoolExecutionResult
}

// PoolExecutionResult represents the result of task execution
type PoolExecutionResult struct {
	TaskResult types.TaskResult
	Error      error
	Duration   time.Duration
}

// NewPool creates a new execution pool
func NewPool(maxWorkers int, logger interfaces.Logger) *Pool {
	return NewPoolWithContext(context.Background(), maxWorkers, logger)
}

// NewPoolWithContext creates a new execution pool with an external context
func NewPoolWithContext(ctx context.Context, maxWorkers int, logger interfaces.Logger) *Pool {
	newCtx, cancel := context.WithCancel(ctx)

	pool := &Pool{
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
		logger:     logger,
		results:    make(chan TaskExecution, maxWorkers*2), // Buffer for better performance
		ctx:        newCtx,
		cancel:     cancel,
	}

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		go pool.worker(i)
	}

	return pool
}

// worker processes tasks from the results channel
func (p *Pool) worker(workerID int) {
	p.logger.Debug("Worker %d started", workerID)
	defer p.logger.Debug("Worker %d stopped", workerID)

	for {
		select {
		case <-p.ctx.Done():
			return
		case execution, ok := <-p.results:
			if !ok {
				return
			}

			p.processExecution(workerID, execution)
		}
	}
}

// processExecution executes a single task
func (p *Pool) processExecution(workerID int, execution TaskExecution) {
	defer p.wg.Done()

	startTime := time.Now()

	p.logger.Debug("Worker %d executing task '%s' on host '%s'",
		workerID, execution.Task.Name, execution.Host.Name)

	// Create task context with timeout
	taskCtx := p.ctx
	if execution.Task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(p.ctx, execution.Task.Timeout)
		defer cancel()
	}

	// Execute the task
	result, err := execution.Module.Execute(taskCtx, execution.Host, execution.Task.Args)
	duration := time.Since(startTime)

	// Send result back
	select {
	case execution.Result <- PoolExecutionResult{
		TaskResult: result,
		Error:      err,
		Duration:   duration,
	}:
	case <-p.ctx.Done():
		return
	}

	p.logger.Debug("Worker %d completed task '%s' on host '%s' in %v",
		workerID, execution.Task.Name, execution.Host.Name, duration)
}

// Execute submits a task for execution
func (p *Pool) Execute(task *types.Task, host types.Host, variables map[string]interface{}, module interfaces.ModuleExecutor) <-chan PoolExecutionResult {
	resultChan := make(chan PoolExecutionResult, 1)

	execution := TaskExecution{
		Task:      task,
		Host:      host,
		Variables: variables,
		Module:    module,
		Result:    resultChan,
	}

	p.wg.Add(1)

	select {
	case p.results <- execution:
		// Task submitted successfully
	case <-p.ctx.Done():
		// Pool is shutting down
		p.wg.Done()
		go func() {
			resultChan <- PoolExecutionResult{
				Error: p.ctx.Err(),
			}
		}()
	}

	return resultChan
}

// ExecuteParallel executes multiple tasks in parallel
func (p *Pool) ExecuteParallel(tasks []*types.Task, hosts []types.Host, variables map[string]interface{}, moduleRegistry interfaces.ModuleRegistry) ([]types.TaskResult, error) {
	if len(tasks) == 0 || len(hosts) == 0 {
		return nil, nil
	}

	totalTasks := len(tasks) * len(hosts)
	results := make([]types.TaskResult, 0, totalTasks)
	resultChans := make([]<-chan PoolExecutionResult, 0, totalTasks)

	// Submit all tasks
	for _, task := range tasks {
		module, err := moduleRegistry.Get(task.Module)
		if err != nil {
			return nil, fmt.Errorf("module '%s' not found: %w", task.Module, err)
		}

		// Determine which hosts to run this task on
		tasksHosts := hosts

		// Handle run_once: only run on the first host
		if task.RunOnce && len(hosts) > 0 {
			tasksHosts = hosts[:1]
			p.logger.Debug("Task '%s' has run_once enabled, will execute only on first host '%s'", task.Name, hosts[0].Name)
		}

		for _, host := range tasksHosts {
			// Handle delegate_to: execute on a different host
			executionHost := host
			if task.DelegateTo != "" {
				// Find the delegated host
				delegatedHost := findHostByName(hosts, task.DelegateTo)
				if delegatedHost != nil {
					executionHost = *delegatedHost
					p.logger.Debug("Task '%s' delegated from host '%s' to '%s'", task.Name, host.Name, executionHost.Name)
				} else {
					p.logger.Debug("Task '%s' delegate_to host '%s' not found, using original host '%s'", task.Name, task.DelegateTo, host.Name)
				}
			}

			resultChan := p.Execute(task, executionHost, variables, module)
			resultChans = append(resultChans, resultChan)
		}
	}

	// Collect results
	var errors []error
	for i, resultChan := range resultChans {
		select {
		case result := <-resultChan:
			if result.Error != nil {
				errors = append(errors, fmt.Errorf("task %d failed: %w", i, result.Error))
			} else {
				results = append(results, result.TaskResult)
			}
		case <-p.ctx.Done():
			return results, p.ctx.Err()
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("execution failed with %d errors: %v", len(errors), errors[0])
	}

	return results, nil
}

// findHostByName finds a host by name in the hosts slice
func findHostByName(hosts []types.Host, name string) *types.Host {
	for i := range hosts {
		if hosts[i].Name == name || hosts[i].Address == name {
			return &hosts[i]
		}
	}
	return nil
}

// ExecuteWithRetry executes a task with retry logic
func (p *Pool) ExecuteWithRetry(task *types.Task, host types.Host, variables map[string]interface{}, module interfaces.ModuleExecutor) (types.TaskResult, error) {
	maxRetries := task.Retries
	if maxRetries <= 0 {
		maxRetries = 1 // At least one attempt
	}

	var lastErr error
	var result types.TaskResult

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resultChan := p.Execute(task, host, variables, module)

		select {
		case execResult := <-resultChan:
			if execResult.Error == nil {
				return execResult.TaskResult, nil
			}

			lastErr = execResult.Error
			result = execResult.TaskResult

			// Log retry attempt
			if attempt < maxRetries {
				delay := task.Delay
				if delay <= 0 {
					delay = time.Second // Default delay
				}

				p.logger.Retry(task.Name, host.Name, attempt, maxRetries, delay, lastErr)

				// Wait before retry
				select {
				case <-time.After(delay):
				case <-p.ctx.Done():
					return result, p.ctx.Err()
				}
			}

		case <-p.ctx.Done():
			return result, p.ctx.Err()
		}
	}

	return result, fmt.Errorf("task failed after %d attempts: %w", maxRetries, lastErr)
}

// Wait waits for all submitted tasks to complete
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Close shuts down the pool and waits for all workers to finish
func (p *Pool) Close() error {
	p.cancel()
	close(p.results)
	p.wg.Wait()
	return nil
}

// StopAll gracefully stops all workers and prevents new task submissions
func (p *Pool) StopAll() {
	p.cancel()
}

// KillAll forcefully stops all workers and prevents new task submissions
func (p *Pool) KillAll() {
	p.cancel()
}

// Stats returns pool statistics
func (p *Pool) Stats() PoolStats {
	return PoolStats{
		MaxWorkers:    p.maxWorkers,
		ActiveWorkers: len(p.semaphore),
		QueuedTasks:   len(p.results),
	}
}

// PoolStats holds pool statistics
type PoolStats struct {
	MaxWorkers    int `json:"max_workers"`
	ActiveWorkers int `json:"active_workers"`
	QueuedTasks   int `json:"queued_tasks"`
}

// Submit submits a task for execution (interface method)
func (p *Pool) Submit(task func()) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		task()
	}()
}

// SubmitWithResult submits a task with result channel (interface method)
func (p *Pool) SubmitWithResult(task func() error) <-chan error {
	resultChan := make(chan error, 1)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := task()
		resultChan <- err
		close(resultChan)
	}()

	return resultChan
}

// Shutdown shuts down the pool (interface method)
func (p *Pool) Shutdown(ctx context.Context) error {
	return p.Close()
}

// GetStats returns pool statistics (interface method)
func (p *Pool) GetStats() map[string]interface{} {
	stats := p.Stats()
	return map[string]interface{}{
		"max_workers":    stats.MaxWorkers,
		"active_workers": stats.ActiveWorkers,
		"queued_tasks":   stats.QueuedTasks,
	}
}
