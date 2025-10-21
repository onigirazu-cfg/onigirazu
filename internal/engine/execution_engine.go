package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/facts"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/metrics"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/security"
	"github.com/onigirazu-cfg/onigirazu/internal/tagfilter"
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
	factsGatherer     *facts.Gatherer
	roleLoader        *parser.RoleLoader
	tagFilter         *tagfilter.Filter

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
	// Create default tag filter (no filtering)
	defaultFilter, _ := tagfilter.New("", "")

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
		securityValidator: security.NewSecurityValidator(security.DefaultSecurityConfig()),
		factsGatherer:     facts.NewGatherer(),
		roleLoader:        parser.NewRoleLoader(logger, "playbooks/roles"),
		tagFilter:         defaultFilter,
		variables:         make(map[string]interface{}),
		facts:             make(map[string]map[string]interface{}),
		stats: &ExecutionStats{
			HostStats: make(map[string]*HostStats),
		},
	}
}

// SetTagFilter sets the tag filter for this execution engine
func (e *ExecutionEngine) SetTagFilter(filter *tagfilter.Filter) {
	if filter != nil {
		e.tagFilter = filter
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

	// Gather facts if enabled (before setting play variables so they can use facts)
	if play.GatherFacts {
		if err := e.gatherFacts(ctx, hosts); err != nil {
			e.logger.Warn("Failed to gather facts: %v", err)
		}
	}

	// Set play variables - merge with facts if available
	playVars := e.mergeVariables(e.variables, play.Vars)

	// If we have facts for the first host, merge them into playVars for template rendering
	// This allows play vars to use facts in their definitions
	if len(hosts) > 0 {
		if hostFacts, exists := e.facts[hosts[0].Name]; exists {
			playVars = e.mergeVariables(playVars, hostFacts)
		}
	}

	// Render play variables that contain templates
	if len(play.Vars) > 0 {
		renderedPlayVars := make(map[string]interface{})
		for key, value := range play.Vars {
			if strValue, ok := value.(string); ok {
				// Try to render the value as a template
				rendered, err := e.templateEngine.Render(ctx, strValue, playVars)
				if err == nil && rendered != strValue {
					renderedPlayVars[key] = rendered
				} else {
					renderedPlayVars[key] = value
				}
			} else {
				renderedPlayVars[key] = value
			}
		}
		// Merge rendered vars back
		playVars = e.mergeVariables(e.variables, renderedPlayVars)
	}

	// Execute roles (with conditional and dependency support)
	if len(play.Roles) > 0 || len(play.RoleObjects) > 0 {
		e.logger.Debug("Executing roles for play '%s'", play.Name)

		// Use Roles (RoleReference) if available, otherwise fallback to RoleObjects
		var roleRefs []types.RoleReference
		if len(play.Roles) > 0 {
			roleRefs = play.Roles
		} else {
			// Convert RoleObjects back to RoleReferences for consistency
			for _, role := range play.RoleObjects {
				roleRefs = append(roleRefs, types.RoleReference{Name: role.Name, Path: role.Path})
			}
		}

		for i, roleRef := range roleRefs {
			e.logger.Debug("Processing role %d/%d: %s", i+1, len(roleRefs), roleRef.Name)

			// Check conditional execution
			if roleRef.When != "" {
				skip, err := e.evaluateCondition(ctx, roleRef.When, playVars)
				if err != nil {
					e.logger.Warn("Failed to evaluate condition for role '%s': %v", roleRef.Name, err)
				} else if skip {
					e.logger.Debug("Skipping role '%s' due to condition", roleRef.Name)
					continue
				}
			}

			// Load role if using RoleReference
			var role *types.Role
			if len(play.Roles) > 0 {
				var err error
				role, err = e.roleLoader.LoadRole(ctx, roleRef)
				if err != nil {
					e.logger.Error("Failed to load role '%s': %v", roleRef.Name, err)
					if !play.IgnoreErrors {
						return result, fmt.Errorf("failed to load role '%s': %w", roleRef.Name, err)
					}
					result.Success = false
					continue
				}
			} else {
				// Use pre-loaded RoleObject
				role = play.RoleObjects[i]
			}

			// Execute role with dependencies
			if err := e.executeRoleWithDependencies(ctx, role, hosts, playVars, result); err != nil {
				e.logger.Error("Role '%s' failed: %v", role.Name, err)
				if !play.IgnoreErrors {
					return result, fmt.Errorf("role '%s' failed: %w", role.Name, err)
				}
				result.Success = false
			}
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

	// Execute handlers if any were triggered
	if len(play.Handlers) > 0 {
		triggeredHandlers := e.collectTriggeredHandlers(result)
		if len(triggeredHandlers) > 0 {
			if err := e.executeHandlers(ctx, play.Handlers, triggeredHandlers, hosts, playVars, result); err != nil {
				e.logger.Error("Handlers failed: %v", err)
				if !play.IgnoreErrors {
					result.Success = false
				}
			}
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
		e.logger.Debug("Executing task %d/%d: %s (tags: %v)", i+1, len(tasks), task.Name, task.Tags)

		// Check if task should be skipped based on tags
		if e.tagFilter != nil && !e.tagFilter.ShouldRun(task.Tags) {
			e.logger.Debug("Skipping task '%s' due to tag filter (%s), task tags: %v", task.Name, e.tagFilter.String(), task.Tags)
			e.updateTaskStats("", &task, types.TaskResult{Skipped: true})
			continue
		}

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

	// Merge variables in order of precedence (later overrides earlier):
	// 1. Play variables
	// 2. Global variables (from set_fact and register)
	// 3. Host vars (highest precedence)
	e.mutex.RLock()
	globalVars := make(map[string]interface{})
	for k, v := range e.variables {
		globalVars[k] = v
	}
	e.mutex.RUnlock()

	taskVars := e.mergeVariables(variables, globalVars, host.Vars)

	// Add host facts - both as onigirazu_facts and unpacked at top level
	if hostFacts, exists := e.facts[host.Name]; exists {
		// Add facts as onigirazu_facts for compatibility
		taskVars = e.mergeVariables(taskVars, map[string]interface{}{"onigirazu_facts": hostFacts})
		// Also unpack facts to top level so they can be accessed directly in templates
		taskVars = e.mergeVariables(taskVars, hostFacts)
	}

	// Debug: log available variables
	e.logger.Debug("Task '%s' variables: %v", task.Name, taskVars)

	// Render task arguments with templates
	renderedArgs, err := e.templateEngine.RenderTaskArgs(ctx, task.Args, taskVars)
	if err != nil {
		return fmt.Errorf("failed to render task arguments: %w", err)
	}

	// Perform security validation
	taskForValidation := types.Task{
		Name:   task.Name,
		Module: task.Module,
		Args:   renderedArgs,
	}
	validationResult := e.securityValidator.ValidateTask(taskForValidation)
	if !validationResult.Valid {
		e.metricsManager.IncrementErrorByType("security_validation")
		return fmt.Errorf("security validation failed: %s", validationResult.Error())
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
			Name:         task.Name,
			Module:       task.Module,
			Args:         renderedArgs,
			Become:       task.Become,
			BecomeUser:   task.BecomeUser,
			BecomeMethod: task.BecomeMethod,
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

	// Add notify handlers if task was successful
	if result.Success && !result.Skipped && len(task.Notify) > 0 {
		result.Notify = task.Notify
	}

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

	// Print debug output if this is a debug module
	if task.Module == "debug" && !result.Failed {
		if msg, ok := result.Output["msg"].(string); ok {
			fmt.Printf("    %s\n", msg)
		}
	}

	// Update progress
	e.progressTracker.UpdateTask(host.Name, task.Name, !result.Failed)

	// Handle register: store task result in variables
	if task.Register != "" && !result.Failed {
		e.mutex.Lock()
		// Store the result in a format compatible with Ansible
		registeredVar := map[string]interface{}{
			"changed": result.Changed,
			"failed":  result.Failed,
			"skipped": result.Skipped,
		}

		// Add stdout/stderr if available (for command/shell modules)
		if stdout, ok := result.Output["stdout"]; ok {
			registeredVar["stdout"] = stdout
		}
		if stderr, ok := result.Output["stderr"]; ok {
			registeredVar["stderr"] = stderr
		}
		if rc, ok := result.Output["rc"]; ok {
			registeredVar["rc"] = rc
		}

		// Add all output fields
		for key, value := range result.Output {
			if key != "stdout" && key != "stderr" && key != "rc" {
				registeredVar[key] = value
			}
		}

		// Store in global variables
		e.variables[task.Register] = registeredVar
		e.mutex.Unlock()

		e.logger.Debug("Registered variable '%s' with result from task '%s': %+v", task.Register, task.Name, registeredVar)
	}

	// Handle set_fact: merge facts into global variables
	if task.Module == "set_fact" && !result.Failed {
		e.mutex.Lock()
		if onigirazuFacts, ok := result.Output["onigirazu_facts"].(map[string]interface{}); ok {
			for key, value := range onigirazuFacts {
				e.variables[key] = value
				e.logger.Debug("Set fact '%s' = %v", key, value)
			}
		}
		e.mutex.Unlock()
	}

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

	// Gather facts from each host
	for _, host := range hosts {
		// Gather system facts using the facts gatherer (with caching)
		systemFacts, err := e.factsGatherer.GatherFacts(ctx, host)
		if err != nil {
			e.logger.Warn("Failed to gather facts from %s: %v", host.Name, err)
			// Continue with basic facts on error
			e.facts[host.Name] = map[string]interface{}{
				"onigirazu_hostname": host.Name,
				"onigirazu_host":     host.Address,
				"onigirazu_port":     host.Port,
				"onigirazu_user":     host.User,
			}
			continue
		}

		// Get current time for date_time facts
		now := time.Now()

		// Store facts in Onigirazu format
		e.facts[host.Name] = map[string]interface{}{
			// Basic host info
			"onigirazu_hostname": host.Name,
			"onigirazu_host":     host.Address,
			"onigirazu_port":     host.Port,
			"onigirazu_user":     host.User,

			// System facts
			"onigirazu_os_family":            systemFacts.OSFamily,
			"onigirazu_distribution":         systemFacts.Distribution,
			"onigirazu_distribution_version": systemFacts.OSVersion,
			"onigirazu_architecture":         systemFacts.Architecture,
			"onigirazu_kernel":               systemFacts.Kernel,
			"onigirazu_kernel_version":       systemFacts.KernelVersion,
			"onigirazu_fqdn":                 systemFacts.FQDN,
			"onigirazu_processor_cores":      systemFacts.CPUCores,
			"onigirazu_memtotal_mb":          systemFacts.MemoryTotal,
			"onigirazu_default_ipv4": map[string]interface{}{
				"address": systemFacts.DefaultIPv4,
			},

			// Date and time facts
			"onigirazu_date_time": map[string]interface{}{
				"iso8601":        now.Format(time.RFC3339),
				"date":           now.Format("2006-01-02"),
				"time":           now.Format("15:04:05"),
				"year":           now.Year(),
				"month":          int(now.Month()),
				"day":            now.Day(),
				"hour":           now.Hour(),
				"minute":         now.Minute(),
				"second":         now.Second(),
				"epoch":          now.Unix(),
				"weekday":        now.Weekday().String(),
				"weekday_number": int(now.Weekday()),
			},

			// User and environment facts
			"onigirazu_user_id": systemFacts.Username,
			"onigirazu_env": map[string]interface{}{
				"HOME": systemFacts.HomeDir,
				"PATH": systemFacts.Path,
			},
		}
	}

	// Log cache statistics
	stats := e.factsGatherer.GetCacheStats()
	if stats.Hits > 0 || stats.Misses > 0 {
		e.logger.Debug("Facts cache stats: %d hits, %d misses (%.1f%% hit rate)",
			stats.Hits, stats.Misses, stats.HitRate)
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
// Supports formats:
//   - "1-10" -> [1, 2, 3, ..., 10]
//   - "1-10:2" -> [1, 3, 5, 7, 9]
//   - "a-z" -> ['a', 'b', 'c', ..., 'z']
//   - "0-3" -> [0, 1, 2, 3]
func (e *ExecutionEngine) getLoopItems(loop *types.Loop, variables map[string]interface{}) ([]interface{}, error) {
	if loop.Items != nil {
		return loop.Items, nil
	}

	if loop.Range != "" {
		return e.parseRange(loop.Range)
	}

	return nil, fmt.Errorf("loop must specify either items or range")
}

// parseRange parses range string and returns items
// Supports formats:
//   - Numeric ranges: "1-10" or "start-end:step"
//   - Character ranges: "a-z", "A-Z"
func (e *ExecutionEngine) parseRange(rangeStr string) ([]interface{}, error) {
	items := make([]interface{}, 0)

	// Check if it's a step range (contains colon)
	step := 1
	rangeWithoutStep := rangeStr
	if idx := strings.Index(rangeStr, ":"); idx != -1 {
		stepStr := rangeStr[idx+1:]
		s, err := strconv.Atoi(stepStr)
		if err != nil {
			return nil, fmt.Errorf("invalid step in range '%s': %w", rangeStr, err)
		}
		if s <= 0 {
			return nil, fmt.Errorf("step must be positive, got: %d", s)
		}
		step = s
		rangeWithoutStep = rangeStr[:idx]
	}

	// Split by dash to get start and end
	parts := strings.Split(rangeWithoutStep, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format '%s', expected 'start-end' or 'start-end:step'", rangeStr)
	}

	start, end := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if start == "" || end == "" {
		return nil, fmt.Errorf("invalid range format '%s', start and end must not be empty", rangeStr)
	}

	// Try numeric range first
	startNum, errStart := strconv.Atoi(start)
	endNum, errEnd := strconv.Atoi(end)

	if errStart == nil && errEnd == nil {
		// Numeric range
		if startNum <= endNum {
			for i := startNum; i <= endNum; i += step {
				items = append(items, i)
			}
		} else {
			// Reverse order
			for i := startNum; i >= endNum; i -= step {
				items = append(items, i)
			}
		}
		return items, nil
	}

	// Try character range
	if len(start) == 1 && len(end) == 1 {
		startChar := rune(start[0])
		endChar := rune(end[0])

		// Validate character range consistency
		// Either both should be letters (a-z, A-Z) or both should be something else
		startIsLetter := (startChar >= 'a' && startChar <= 'z') || (startChar >= 'A' && startChar <= 'Z')
		endIsLetter := (endChar >= 'a' && endChar <= 'z') || (endChar >= 'A' && endChar <= 'Z')

		// Don't allow mixing letters with digits or other characters
		if startIsLetter != endIsLetter {
			return nil, fmt.Errorf("range '%s' cannot mix letters and non-letters (got '%s' and '%s')", rangeStr, start, end)
		}

		if startChar <= endChar {
			for i := startChar; i <= endChar; i += rune(step) {
				items = append(items, string(i))
			}
		} else {
			// Reverse order
			for i := startChar; i >= endChar; i -= rune(step) {
				items = append(items, string(i))
			}
		}
		return items, nil
	}

	return nil, fmt.Errorf("range '%s' must be numeric (e.g., '1-10') or character (e.g., 'a-z'), not mixed", rangeStr)
}

// executeRole executes a role with its tasks, handlers, and dependencies
func (e *ExecutionEngine) executeRole(ctx context.Context, role *types.Role, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {
	if role == nil {
		return fmt.Errorf("role is nil")
	}

	e.logger.Debug("Starting role execution: %s", role.Name)

	// Merge role variables with play variables
	// Priority: RoleVars > PlayVars > Defaults (handled by roleLoader)
	roleVars := e.mergeRoleVariables(role, variables)

	// Execute role pre_tasks if defined
	if len(role.PreTasks) > 0 {
		e.logger.Debug("Executing %d pre-tasks for role '%s'", len(role.PreTasks), role.Name)
		if err := e.executeTaskList(ctx, role.PreTasks, hosts, roleVars, playResult); err != nil {
			return fmt.Errorf("role pre-tasks failed: %w", err)
		}
	}

	// Execute role main tasks if defined
	if len(role.Tasks) > 0 {
		e.logger.Debug("Executing %d main tasks for role '%s'", len(role.Tasks), role.Name)
		if err := e.executeTaskListWithRetry(ctx, role.Tasks, hosts, roleVars, playResult); err != nil {
			e.logger.Warn("Role main tasks failed: %v", err)
			// Continue to handlers even if tasks fail
		}
	}

	// Execute role handlers if any were triggered
	if len(role.Handlers) > 0 {
		triggeredHandlers := e.collectTriggeredHandlers(playResult)
		if len(triggeredHandlers) > 0 {
			if err := e.executeHandlers(ctx, role.Handlers, triggeredHandlers, hosts, roleVars, playResult); err != nil {
				e.logger.Warn("Role handlers failed: %v", err)
			}
		}
	}

	// Execute role post_tasks if defined
	if len(role.PostTasks) > 0 {
		e.logger.Debug("Executing %d post-tasks for role '%s'", len(role.PostTasks), role.Name)
		if err := e.executeTaskList(ctx, role.PostTasks, hosts, roleVars, playResult); err != nil {
			e.logger.Warn("Role post-tasks failed: %v", err)
		}
	}

	e.logger.Debug("Completed role execution: %s", role.Name)
	return nil
}

// executeRoleWithDependencies executes a role and all its dependencies in correct order
func (e *ExecutionEngine) executeRoleWithDependencies(ctx context.Context, role *types.Role, hosts []types.Host,
	variables map[string]interface{}, playResult *types.PlayResult) error {
	if role == nil {
		return fmt.Errorf("role is nil")
	}

	// Resolve dependency execution order (topological sort)
	roleOrder, err := e.roleLoader.ResolveDependencyOrder(ctx, role)
	if err != nil {
		e.logger.Error("Failed to resolve dependencies for role '%s': %v", role.Name, err)
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	e.logger.Debug("Resolved role execution order for '%s': %v", role.Name, len(roleOrder))

	// Execute all dependencies first, then the main role
	for i, depRole := range roleOrder {
		isMainRole := (i == len(roleOrder)-1)
		roleType := "dependency"
		if isMainRole {
			roleType = "main"
		}

		e.logger.Debug("Executing %s role %d/%d: %s", roleType, i+1, len(roleOrder), depRole.Name)

		// Execute the role
		if err := e.executeRole(ctx, depRole, hosts, variables, playResult); err != nil {
			e.logger.Error("Role '%s' (%s) failed: %v", depRole.Name, roleType, err)
			return err
		}
	}

	e.logger.Debug("Completed role with dependencies: %s", role.Name)
	return nil
}

// mergeRoleVariables merges role variables with play variables using correct precedence:
// RoleVars > OverrideVars > PlayVars > Defaults
func (e *ExecutionEngine) mergeRoleVariables(role *types.Role, playVars map[string]interface{}) map[string]interface{} {
	if role == nil {
		return playVars
	}

	// Use the role loader's merge function for proper precedence handling
	// This ensures: RoleVars > OverrideVars > PlayVars > Defaults
	// Note: overrideVars are empty here; they should be passed from RoleReference if available
	overrideVars := make(map[string]interface{})
	return e.roleLoader.MergeVariables(role.Defaults, role.Vars, playVars, overrideVars)
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
		e.logger.Debug("Executing task %d/%d: %s (tags: %v)", i+1, len(tasks), task.Name, task.Tags)

		// Check if task should be skipped based on tags
		if e.tagFilter != nil && !e.tagFilter.ShouldRun(task.Tags) {
			e.logger.Debug("Skipping task '%s' due to tag filter (%s), task tags: %v", task.Name, e.tagFilter.String(), task.Tags)
			e.metricsManager.IncrementTasksSkipped()
			continue
		}

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

// collectTriggeredHandlers collects all handler names that were triggered by tasks
func (e *ExecutionEngine) collectTriggeredHandlers(playResult *types.PlayResult) []string {
	triggered := make(map[string]bool) // Use map to avoid duplicates
	var result []string

	// Iterate through all hosts and their tasks
	for _, hostResult := range playResult.Hosts {
		for _, taskResult := range hostResult.Tasks {
			// Check if task was successful and has notify directives
			if taskResult.Success && len(taskResult.Notify) > 0 {
				for _, notifyName := range taskResult.Notify {
					if !triggered[notifyName] {
						triggered[notifyName] = true
						result = append(result, notifyName)
					}
				}
			}
		}
	}

	return result
}

// executeHandlers executes all handlers that were triggered
func (e *ExecutionEngine) executeHandlers(ctx context.Context, handlers []types.Task,
	triggeredNames []string, hosts []types.Host, variables map[string]interface{},
	playResult *types.PlayResult) error {
	if len(handlers) == 0 || len(triggeredNames) == 0 {
		return nil
	}

	e.logger.Info("Executing %d handlers (triggered: %v)", len(handlers), len(triggeredNames))

	// Create set of triggered handler names for quick lookup
	triggeredSet := make(map[string]bool)
	for _, name := range triggeredNames {
		triggeredSet[name] = true
	}

	// Find and execute matching handlers
	for _, handler := range handlers {
		shouldExecute := false

		// Check if handler matches by name (explicit notify)
		if triggeredSet[handler.Name] {
			shouldExecute = true
		}

		// Check if handler matches by listen directive
		if handler.Listen != "" && triggeredSet[handler.Listen] {
			shouldExecute = true
		}

		if shouldExecute {
			e.logger.Debug("Executing handler: %s", handler.Name)
			if err := e.executeTask(ctx, &handler, hosts, variables, playResult); err != nil {
				e.logger.Error("Handler '%s' failed: %v", handler.Name, err)
				if !handler.IgnoreErrors {
					return fmt.Errorf("handler '%s' failed: %w", handler.Name, err)
				}
			}
		}
	}

	return nil
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
			"total_errors":     metrics.Errors.Total,
			"errors_by_module": metrics.Errors.ByModule,
			"errors_by_type":   metrics.Errors.ByType,
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
