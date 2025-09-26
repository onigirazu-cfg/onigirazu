package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/metrics"
	"github.com/onigirazu-cfg/onigirazu/internal/security"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ExecutionEngine is the main engine for executing playbooks
type ExecutionEngine struct {
	config            interfaces.Config
	logger            interfaces.Logger
	stateManager      interfaces.StateManager
	inventoryMgr      interfaces.InventoryManager
	moduleRegistry    interfaces.ModuleRegistry
	templateEngine    interfaces.TemplateEngine
	progressTracker   interfaces.ProgressTracker
	executionPool     interfaces.ExecutionPool
	cacheManager      interfaces.CacheManager
	metricsManager    *metrics.Metrics
	securityValidator *security.SecurityValidator

	// Execution context
	variables map[string]interface{}
	facts     map[string]map[string]interface{} // host -> facts
	mutex     sync.RWMutex

	// Statistics
	stats *ExecutionStats
}

// ExecutionStats holds execution statistics
type ExecutionStats struct {
	StartTime       time.Time
	EndTime         time.Time
	TotalTasks      int
	CompletedTasks  int
	SuccessfulTasks int
	FailedTasks     int
	ChangedTasks    int
	SkippedTasks    int
	HostStats       map[string]*HostStats
	mutex           sync.RWMutex
}

// HostStats holds per-host statistics
type HostStats struct {
	TotalTasks      int
	CompletedTasks  int
	SuccessfulTasks int
	FailedTasks     int
	ChangedTasks    int
	SkippedTasks    int
	LastTaskTime    time.Time
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(
	config interfaces.Config,
	logger interfaces.Logger,
	stateManager interfaces.StateManager,
	inventoryMgr interfaces.InventoryManager,
	moduleRegistry interfaces.ModuleRegistry,
	templateEngine interfaces.TemplateEngine,
	progressTracker interfaces.ProgressTracker,
	executionPool interfaces.ExecutionPool,
	cacheManager interfaces.CacheManager,
) *ExecutionEngine {
	return &ExecutionEngine{
		config:            config,
		logger:            logger,
		stateManager:      stateManager,
		inventoryMgr:      inventoryMgr,
		moduleRegistry:    moduleRegistry,
		templateEngine:    templateEngine,
		progressTracker:   progressTracker,
		executionPool:     executionPool,
		cacheManager:      cacheManager,
		metricsManager:    metrics.NewMetrics(),
		securityValidator: security.NewSecurityValidator(),
		variables:         make(map[string]interface{}),
		facts:             make(map[string]map[string]interface{}),
		stats: &ExecutionStats{
			HostStats: make(map[string]*HostStats),
		},
	}
}

// ExecutePlaybook executes a complete playbook
func (e *ExecutionEngine) ExecutePlaybook(ctx context.Context, playbook *types.Playbook) (*types.PlaybookResult, error) {
	e.logger.Info("Starting playbook execution: %s", playbook.Name)

	// Record playbook execution start
	e.metricsManager.IncrementPlaybooksExecuted()
	playbookStartTime := time.Now()

	// Initialize execution
	e.initializeExecution()

	// Set playbook variables
	if playbook.Vars != nil {
		e.setVariables(playbook.Vars)
	}

	result := &types.PlaybookResult{
		Name:      playbook.Name,
		StartTime: time.Now(),
		Plays:     make([]types.PlayResult, 0, len(playbook.Plays)),
	}

	// Execute each play
	for i, play := range playbook.Plays {
		playStartTime := time.Now()
		e.logger.PlayStart(play.Name, i+1, len(playbook.Plays))

		// Record play execution start
		e.metricsManager.IncrementPlaysExecuted()

		playResult, err := e.executePlay(ctx, &play)
		if err != nil {
			e.logger.Error("Play '%s' failed: %v", play.Name, err)
			result.Failed = true
			result.Error = err.Error()
			e.metricsManager.IncrementTasksFailed()

			// Check if we should continue on failure
			if !play.IgnoreErrors {
				break
			}
		} else if playResult.Success {
			e.metricsManager.IncrementTasksSucceeded()
		} else {
			e.metricsManager.IncrementTasksFailed()
		}

		result.Plays = append(result.Plays, *playResult)

		// Update overall success status
		if !playResult.Success {
			result.Failed = true
		}

		e.logger.PlayEnd(play.Name, "", playResult.Success, time.Since(playStartTime))
	}

	// Finalize execution
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Stats = e.getExecutionStats()

	// Record playbook execution metrics
	e.metricsManager.AddExecutionTime(time.Since(playbookStartTime))
	e.metricsManager.IncrementPlaybooksExecuted()

	e.logger.Info("Playbook execution completed: %s (duration: %v, success: %t)",
		playbook.Name, result.Duration, !result.Failed)

	return result, nil
}

// executePlay executes a single play
func (e *ExecutionEngine) executePlay(ctx context.Context, play *types.Play) (*types.PlayResult, error) {
	// Get target hosts
	hosts, err := e.getPlayHosts(play)
	if err != nil {
		return nil, fmt.Errorf("failed to get hosts for play '%s': %w", play.Name, err)
	}

	if len(hosts) == 0 {
		e.logger.Warn("No hosts found for play '%s'", play.Name)
		return &types.PlayResult{
			Name:    play.Name,
			Success: true,
			Hosts:   []types.HostResult{},
		}, nil
	}

	e.logger.Debug("Executing play '%s' on %d hosts", play.Name, len(hosts))

	result := &types.PlayResult{
		Name:      play.Name,
		StartTime: time.Now(),
		Hosts:     []types.HostResult{},
		Success:   true,
	}

	// Set play variables
	playVars := e.mergeVariables(e.variables, play.Vars)

	// Gather facts if enabled
	if play.GatherFacts {
		if err := e.gatherFacts(ctx, hosts); err != nil {
			e.logger.Warn("Failed to gather facts: %v", err)
		}
	}

	// Execute pre-tasks
	if len(play.PreTasks) > 0 {
		e.logger.Debug("Executing %d pre-tasks for play '%s'", len(play.PreTasks), play.Name)
		if err := e.executeTaskList(ctx, play.PreTasks, hosts, playVars, result); err != nil {
			return result, fmt.Errorf("pre-tasks failed: %w", err)
		}
	}

	// Execute main tasks
	if len(play.Tasks) > 0 {
		e.logger.Debug("Executing %d main tasks for play '%s'", len(play.Tasks), play.Name)
		if err := e.executeTaskListWithRetry(ctx, play.Tasks, hosts, playVars, result); err != nil {
			if !play.IgnoreErrors && !play.AnyErrorsFatal {
				return result, fmt.Errorf("main tasks failed: %w", err)
			}
			result.Success = false

			// Handle any_errors_fatal
			if play.AnyErrorsFatal {
				e.logger.Error("Task failed with any_errors_fatal=true, stopping execution")
				return result, fmt.Errorf("task failed with any_errors_fatal: %w", err)
			}
		}
	}

	// Execute post-tasks
	if len(play.PostTasks) > 0 {
		e.logger.Debug("Executing %d post-tasks for play '%s'", len(play.PostTasks), play.Name)
		if err := e.executeTaskList(ctx, play.PostTasks, hosts, playVars, result); err != nil {
			e.logger.Warn("Post-tasks failed: %v", err)
			// Post-task failures don't fail the play unless explicitly configured
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// executeTaskList executes a list of tasks
func (e *ExecutionEngine) executeTaskList(ctx context.Context, tasks []types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	for i, task := range tasks {
		e.logger.Debug("Executing task %d/%d: %s", i+1, len(tasks), task.Name)

		// Check if task should be skipped based on conditions
		if task.When != "" {
			skip, err := e.evaluateCondition(ctx, task.When, variables)
			if err != nil {
				e.logger.Warn("Failed to evaluate condition for task '%s': %v", task.Name, err)
			} else if skip {
				e.logger.Debug("Skipping task '%s' due to condition", task.Name)
				e.updateTaskStats("", &task, types.TaskResult{Skipped: true})
				continue
			}
		}

		// Execute task on all hosts
		if err := e.executeTask(ctx, &task, hosts, variables, playResult); err != nil {
			return fmt.Errorf("task '%s' failed: %w", task.Name, err)
		}
	}

	return nil
}

// executeTask executes a single task on multiple hosts
func (e *ExecutionEngine) executeTask(ctx context.Context, task *types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	// Handle loops
	if task.Loop != nil {
		return e.executeTaskWithLoop(ctx, task, hosts, variables, playResult)
	}

	// Execute on hosts (serial or parallel based on configuration)
	if task.Serial || len(hosts) == 1 {
		return e.executeTaskSerial(ctx, task, hosts, variables, playResult)
	} else {
		return e.executeTaskParallel(ctx, task, hosts, variables, playResult)
	}
}

// executeTaskSerial executes a task on hosts serially
func (e *ExecutionEngine) executeTaskSerial(ctx context.Context, task *types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	for _, host := range hosts {
		if err := e.executeTaskOnHost(ctx, task, &host, variables, playResult); err != nil {
			if !task.IgnoreErrors {
				return err
			}
			e.logger.Warn("Task '%s' failed on host '%s' (ignored): %v", task.Name, host.Name, err)
		}
	}

	return nil
}

// executeTaskParallel executes a task on hosts in parallel
func (e *ExecutionEngine) executeTaskParallel(ctx context.Context, task *types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	var wg sync.WaitGroup
	var mutex sync.Mutex
	var firstError error

	for _, host := range hosts {
		wg.Add(1)

		// Submit to execution pool
		e.executionPool.Submit(func() {
			defer wg.Done()

			if err := e.executeTaskOnHost(ctx, task, &host, variables, playResult); err != nil {
				mutex.Lock()
				if firstError == nil && !task.IgnoreErrors {
					firstError = err
				}
				mutex.Unlock()

				if task.IgnoreErrors {
					e.logger.Warn("Task '%s' failed on host '%s' (ignored): %v", task.Name, host.Name, err)
				}
			}
		})
	}

	wg.Wait()
	return firstError
}

// executeTaskOnHost executes a task on a single host
func (e *ExecutionEngine) executeTaskOnHost(ctx context.Context, task *types.Task, host *types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	e.logger.TaskStart(task.Name, host.Name)

	// Merge variables (host vars take precedence)
	taskVars := e.mergeVariables(variables, host.Vars)

	// Add host facts
	if hostFacts, exists := e.facts[host.Name]; exists {
		taskVars = e.mergeVariables(taskVars, map[string]interface{}{"ansible_facts": hostFacts})
	}

	// Render task arguments with templates
	renderedArgs, err := e.templateEngine.RenderTaskArgs(ctx, task.Args, taskVars)
	if err != nil {
		return fmt.Errorf("failed to render task arguments: %w", err)
	}

	// Perform security validation
	taskForValidation := &types.Task{
		Name:   task.Name,
		Module: task.Module,
		Args:   renderedArgs,
	}
	if err := e.securityValidator.ValidateTask(taskForValidation); err != nil {
		e.metricsManager.IncrementErrorByType("security_validation")
		return fmt.Errorf("security validation failed: %w", err)
	}

	// Create execution context (for future use)
	_ = &types.ExecutionContext{
		Host:      host,
		Task:      task,
		Variables: taskVars,
		Facts:     e.facts[host.Name],
	}

	// Execute with retry logic
	var result types.TaskResult
	maxAttempts := task.Retries + 1
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	// Record task execution start
	e.metricsManager.IncrementTasksExecuted()
	e.metricsManager.IncrementModuleUsage(task.Module)
	taskStartTime := time.Now()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check if we're in dry-run mode
		if e.config.GetDryRun() {
			// In dry-run mode, simulate the task execution
			result = types.TaskResult{
				TaskName: task.Name,
				Host:     host.Name,
				Module:   task.Module,
				Success:  true,
				Changed:  false, // Assume no changes in dry-run
				Skipped:  false,
				Failed:   false,
				Output: map[string]interface{}{
					"message": "Task would be executed (dry-run mode)",
					"args":    renderedArgs,
				},
				Duration:  time.Since(taskStartTime),
				Timestamp: taskStartTime,
			}
			err = nil
			break
		}

		// Execute the task
		result, err = e.moduleRegistry.ExecuteTask(ctx, &types.Task{
			Name:   task.Name,
			Module: task.Module,
			Args:   renderedArgs,
		}, *host, taskVars)

		if err == nil && !result.Failed {
			break // Success
		}

		if attempt < maxAttempts {
			delay := time.Duration(task.RetryDelay) * time.Second
			if delay <= 0 {
				delay = 1 * time.Second
			}

			e.logger.Retry(task.Name, host.Name, attempt, maxAttempts, delay, err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	// Handle final result
	if err != nil {
		result.Failed = true
		result.Error = err.Error()
	}

	// Record task execution metrics
	e.metricsManager.AddExecutionTime(time.Since(taskStartTime))
	if result.Failed {
		e.metricsManager.IncrementTasksFailed()
		e.metricsManager.IncrementErrorByType("task_execution")
	} else if result.Skipped {
		e.metricsManager.IncrementTasksSkipped()
	} else {
		e.metricsManager.IncrementTasksSucceeded()
		if result.Changed {
			e.metricsManager.IncrementTasksChanged()
		}
	}

	// Update statistics
	e.updateTaskStats(host.Name, task, result)

	// Update play result
	if playResult.Hosts == nil {
		playResult.Hosts = make([]types.HostResult, 0)
	}

	// Find or create host result
	var hostResult *types.HostResult
	for i := range playResult.Hosts {
		if playResult.Hosts[i].Host == host.Name {
			hostResult = &playResult.Hosts[i]
			break
		}
	}

	if hostResult == nil {
		playResult.Hosts = append(playResult.Hosts, types.HostResult{
			Host:    host.Name,
			Tasks:   make([]types.TaskResult, 0),
			Success: true,
			Failed:  false,
		})
		hostResult = &playResult.Hosts[len(playResult.Hosts)-1]
	}

	hostResult.Tasks = append(hostResult.Tasks, result)
	if result.Failed {
		hostResult.Failed = true
		hostResult.Success = false
		playResult.Success = false
	}

	// Log result
	e.logger.TaskEnd(task.Name, host.Name, result.Changed, !result.Failed)

	// Update progress
	e.progressTracker.UpdateTask(host.Name, task.Name, !result.Failed)

	if result.Failed && !task.IgnoreErrors {
		return fmt.Errorf("task failed: %s", result.Error)
	}

	return nil
}

// executeTaskWithLoop executes a task with loop
func (e *ExecutionEngine) executeTaskWithLoop(ctx context.Context, task *types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	// Get loop items
	items, err := e.getLoopItems(task.Loop, variables)
	if err != nil {
		return fmt.Errorf("failed to get loop items: %w", err)
	}

	e.logger.Debug("Executing task '%s' with loop (%d items)", task.Name, len(items))

	// Execute task for each item
	for i, item := range items {
		// Create task copy with loop variables
		taskCopy := *task
		taskCopy.Name = fmt.Sprintf("%s (item %d)", task.Name, i+1)

		// Add loop variables
		loopVars := e.mergeVariables(variables, map[string]interface{}{
			"item":       item,
			"item_index": i,
		})

		// Execute task
		if err := e.executeTask(ctx, &taskCopy, hosts, loopVars, playResult); err != nil {
			if !task.IgnoreErrors {
				return err
			}
		}
	}

	return nil
}

// Helper methods

// initializeExecution initializes execution state
func (e *ExecutionEngine) initializeExecution() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.stats = &ExecutionStats{
		StartTime: time.Now(),
		HostStats: make(map[string]*HostStats),
	}

	// Load state
	if _, err := e.stateManager.LoadState(context.Background()); err != nil {
		e.logger.Warn("Failed to load state: %v", err)
	}
}

// getPlayHosts gets target hosts for a play
func (e *ExecutionEngine) getPlayHosts(play *types.Play) ([]types.Host, error) {
	hosts, err := e.inventoryMgr.GetHosts(play.Hosts)
	if err != nil {
		return nil, fmt.Errorf("failed to get hosts for pattern '%s': %w", play.Hosts, err)
	}
	return hosts, nil
}

// gatherFacts gathers facts from hosts
func (e *ExecutionEngine) gatherFacts(ctx context.Context, hosts []types.Host) error {
	e.logger.Debug("Gathering facts from %d hosts", len(hosts))

	// This would typically execute a setup module to gather system facts
	// For now, we'll add basic host information
	for _, host := range hosts {
		e.facts[host.Name] = map[string]interface{}{
			"ansible_hostname": host.Name,
			"ansible_host":     host.Address,
			"ansible_port":     host.Port,
			"ansible_user":     host.User,
		}
	}

	return nil
}

// evaluateCondition evaluates a conditional expression
func (e *ExecutionEngine) evaluateCondition(ctx context.Context, condition string, variables map[string]interface{}) (bool, error) {
	// Render the condition as a template
	result, err := e.templateEngine.Render(ctx, condition, variables)
	if err != nil {
		return false, err
	}

	// Simple boolean evaluation
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "true", "yes", "1":
		return false, nil // Don't skip
	case "false", "no", "0", "":
		return true, nil // Skip
	default:
		// Try to evaluate as a boolean expression
		return result == "", nil
	}
}

// getLoopItems gets items for loop execution
func (e *ExecutionEngine) getLoopItems(loop *types.Loop, variables map[string]interface{}) ([]interface{}, error) {
	if loop.Items != nil {
		return loop.Items, nil
	}

	if loop.Range != "" {
		// Parse range string (e.g., "1-10" or "1-10:2")
		items := make([]interface{}, 0)
		// For now, return a simple range - this would need proper parsing
		for i := 1; i <= 10; i++ {
			items = append(items, i)
		}
		return items, nil
	}

	return nil, fmt.Errorf("loop must specify either items or range")
}

// mergeVariables merges multiple variable maps (later maps take precedence)
func (e *ExecutionEngine) mergeVariables(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}

// setVariables sets global variables
func (e *ExecutionEngine) setVariables(vars map[string]interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for k, v := range vars {
		e.variables[k] = v
	}
}

// updateTaskStats updates task execution statistics
func (e *ExecutionEngine) updateTaskStats(hostName string, task *types.Task, result types.TaskResult) {
	e.stats.mutex.Lock()
	defer e.stats.mutex.Unlock()

	e.stats.TotalTasks++
	e.stats.CompletedTasks++

	if result.Failed {
		e.stats.FailedTasks++
	} else if result.Skipped {
		e.stats.SkippedTasks++
	} else {
		e.stats.SuccessfulTasks++
		if result.Changed {
			e.stats.ChangedTasks++
		}
	}

	// Update host stats
	if hostName != "" {
		if e.stats.HostStats[hostName] == nil {
			e.stats.HostStats[hostName] = &HostStats{}
		}

		hostStats := e.stats.HostStats[hostName]
		hostStats.TotalTasks++
		hostStats.CompletedTasks++
		hostStats.LastTaskTime = time.Now()

		if result.Failed {
			hostStats.FailedTasks++
		} else if result.Skipped {
			hostStats.SkippedTasks++
		} else {
			hostStats.SuccessfulTasks++
			if result.Changed {
				hostStats.ChangedTasks++
			}
		}
	}
}

// getExecutionStats returns current execution statistics
func (e *ExecutionEngine) getExecutionStats() map[string]interface{} {
	e.stats.mutex.RLock()
	defer e.stats.mutex.RUnlock()

	e.stats.EndTime = time.Now()

	return map[string]interface{}{
		"start_time":       e.stats.StartTime,
		"end_time":         e.stats.EndTime,
		"duration":         e.stats.EndTime.Sub(e.stats.StartTime),
		"total_tasks":      e.stats.TotalTasks,
		"completed_tasks":  e.stats.CompletedTasks,
		"successful_tasks": e.stats.SuccessfulTasks,
		"failed_tasks":     e.stats.FailedTasks,
		"changed_tasks":    e.stats.ChangedTasks,
		"skipped_tasks":    e.stats.SkippedTasks,
		"host_stats":       e.stats.HostStats,
	}
}

// GetMetrics returns current execution metrics
func (e *ExecutionEngine) GetMetrics() *metrics.Metrics {
	return e.metricsManager.GetSnapshot()
}

// executeTaskListWithRetry executes a list of tasks with enhanced retry logic
func (e *ExecutionEngine) executeTaskListWithRetry(ctx context.Context, tasks []types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	for i, task := range tasks {
		e.logger.Debug("Executing task %d/%d: %s", i+1, len(tasks), task.Name)

		// Check if task should be skipped based on conditions
		if task.When != "" {
			skip, err := e.evaluateCondition(ctx, task.When, variables)
			if err != nil {
				e.logger.Warn("Failed to evaluate condition for task '%s': %v", task.Name, err)
				continue
			}
			if skip {
				e.logger.Debug("Skipping task '%s' due to condition", task.Name)
				e.metricsManager.IncrementTasksSkipped()
				continue
			}
		}

		// Execute task with retry logic
		if err := e.executeTaskWithRetryLogic(ctx, &task, hosts, variables, playResult); err != nil {
			if !task.IgnoreErrors {
				return fmt.Errorf("task '%s' failed: %w", task.Name, err)
			}
			e.logger.Warn("Task '%s' failed but continuing due to ignore_errors", task.Name)
		}
	}

	return nil
}

// executeTaskWithRetryLogic executes a single task with retry logic
func (e *ExecutionEngine) executeTaskWithRetryLogic(ctx context.Context, task *types.Task, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {

	maxRetries := task.Retries
	if maxRetries <= 0 {
		maxRetries = 1 // At least one attempt
	}

	retryDelay := task.RetryDelay
	if retryDelay <= 0 {
		retryDelay = time.Second // Default 1 second delay
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			e.logger.Debug("Retrying task '%s' (attempt %d/%d)", task.Name, attempt+1, maxRetries)

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		// Execute the task
		if task.Loop != nil {
			lastErr = e.executeTaskWithLoop(ctx, task, hosts, variables, playResult)
		} else {
			lastErr = e.executeTask(ctx, task, hosts, variables, playResult)
		}

		// Check if task succeeded
		if lastErr == nil {
			return nil
		}

		// Check until condition if specified
		if task.Until != "" {
			success, err := e.evaluateCondition(ctx, task.Until, variables)
			if err != nil {
				e.logger.Warn("Failed to evaluate until condition: %v", err)
			} else if success {
				e.logger.Debug("Task '%s' succeeded based on until condition", task.Name)
				return nil
			}
		}

		e.logger.Debug("Task '%s' failed on attempt %d: %v", task.Name, attempt+1, lastErr)
	}

	return fmt.Errorf("task failed after %d attempts: %w", maxRetries, lastErr)
}

// executeTaskWithTimeout executes a task with timeout handling
func (e *ExecutionEngine) executeTaskWithTimeout(ctx context.Context, task *types.Task, host *types.Host,
	variables map[string]interface{}) (types.TaskResult, error) {

	// Create context with timeout if specified
	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// Execute task with timeout
	resultChan := make(chan types.TaskResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := e.executeTaskOnHostInternal(taskCtx, task, host, variables)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return types.TaskResult{}, err
	case <-taskCtx.Done():
		if taskCtx.Err() == context.DeadlineExceeded {
			return types.TaskResult{
				TaskName:  task.Name,
				Host:      host.Name,
				Module:    task.Module,
				Failed:    true,
				Error:     fmt.Sprintf("task timed out after %v", task.Timeout),
				Duration:  task.Timeout,
				Timestamp: time.Now(),
			}, fmt.Errorf("task timed out after %v", task.Timeout)
		}
		return types.TaskResult{}, taskCtx.Err()
	}
}

// executeTaskOnHostInternal is the internal implementation for task execution
func (e *ExecutionEngine) executeTaskOnHostInternal(ctx context.Context, task *types.Task, host *types.Host,
	variables map[string]interface{}) (types.TaskResult, error) {

	startTime := time.Now()

	// Get module
	module, err := e.moduleRegistry.GetModule(task.Module)
	if err != nil {
		return types.TaskResult{
			TaskName:  task.Name,
			Host:      host.Name,
			Module:    task.Module,
			Failed:    true,
			Error:     fmt.Sprintf("module not found: %s", task.Module),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	// Validate task arguments
	if err := module.Validate(task.Args); err != nil {
		return types.TaskResult{
			TaskName:  task.Name,
			Host:      host.Name,
			Module:    task.Module,
			Failed:    true,
			Error:     fmt.Sprintf("validation failed: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	// Security validation
	if err := e.securityValidator.ValidateTask(task, *host); err != nil {
		return types.TaskResult{
			TaskName:  task.Name,
			Host:      host.Name,
			Module:    task.Module,
			Failed:    true,
			Error:     fmt.Sprintf("security validation failed: %v", err),
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	// Execute module
	result, err := module.Execute(ctx, *host, task.Args)
	if err != nil {
		result.Failed = true
		result.Error = err.Error()
	}

	// Update metrics
	e.metricsManager.IncrementModuleUsage(task.Module)
	e.metricsManager.AddTaskExecutionTime(task.Module, host.Name, result.Duration)

	if result.Failed {
		e.metricsManager.IncrementErrorByModule(task.Module)
	}

	// Evaluate changed_when condition if specified
	if task.ChangedWhen != "" {
		changed, err := e.evaluateCondition(ctx, task.ChangedWhen, variables)
		if err != nil {
			e.logger.Warn("Failed to evaluate changed_when condition: %v", err)
		} else {
			result.Changed = changed
		}
	}

	// Evaluate failed_when condition if specified
	if task.FailedWhen != "" {
		failed, err := e.evaluateCondition(ctx, task.FailedWhen, variables)
		if err != nil {
			e.logger.Warn("Failed to evaluate failed_when condition: %v", err)
		} else {
			result.Failed = failed
			if failed && result.Error == "" {
				result.Error = "Task failed due to failed_when condition"
			}
		}
	}

	return result, nil
}

// GetExecutionSummary returns a comprehensive execution summary
func (e *ExecutionEngine) GetExecutionSummary() map[string]interface{} {
	stats := e.getExecutionStats()
	metrics := e.metricsManager.GetSummary()

	return map[string]interface{}{
		"execution_stats": stats,
		"metrics":         metrics,
		"performance": map[string]interface{}{
			"total_execution_time": metrics.Performance.TotalExecutionTime,
			"average_task_time":    metrics.Performance.AverageTaskTime,
			"concurrent_tasks":     metrics.Performance.CurrentConcurrentTasks,
		},
		"errors": map[string]interface{}{
			"total_errors":     metrics.Errors.TotalErrors,
			"errors_by_module": metrics.Errors.ErrorsByModule,
			"errors_by_type":   metrics.Errors.ErrorsByType,
		},
		"hosts": map[string]interface{}{
			"connected":   metrics.Hosts.Connected,
			"unreachable": metrics.Hosts.Unreachable,
		},
		"cache": map[string]interface{}{
			"hits":     metrics.Cache.Hits,
			"misses":   metrics.Cache.Misses,
			"hit_rate": metrics.Cache.HitRate,
		},
	}
}
