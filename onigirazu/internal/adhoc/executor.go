package adhoc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Executor handles execution of ad-hoc commands
type Executor struct {
	moduleRegistry *modules.Registry
	inventoryMgr   *inventory.Manager
	logger         interfaces.Logger
}

// NewExecutor creates a new ad-hoc executor
func NewExecutor(
	moduleRegistry *modules.Registry,
	inventoryMgr *inventory.Manager,
	logger interfaces.Logger,
) *Executor {
	return &Executor{
		moduleRegistry: moduleRegistry,
		inventoryMgr:   inventoryMgr,
		logger:         logger,
	}
}

// Execute runs an ad-hoc command on specified hosts
func (e *Executor) Execute(
	ctx context.Context,
	cmd *Command,
	hostPattern string,
	opts Options,
) (*Summary, error) {
	startTime := time.Now()

	// Get target hosts
	hosts, err := e.getTargetHosts(hostPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get target hosts: %w", err)
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts match pattern: %s", hostPattern)
	}

	// Create task from command
	task := e.createTask(cmd, opts)

	// Execute on all hosts
	results := e.executeOnHosts(ctx, task, hosts, opts)

	// Generate summary
	summary := e.generateSummary(results, time.Since(startTime))

	return summary, nil
}

// getTargetHosts resolves host pattern to actual hosts
func (e *Executor) getTargetHosts(pattern string) ([]types.Host, error) {
	// Use inventory manager's GetHosts method
	hosts, err := e.inventoryMgr.GetHosts(pattern)
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts found for pattern: %s", pattern)
	}

	return hosts, nil
}

// createTask creates a task from a command
func (e *Executor) createTask(cmd *Command, opts Options) *types.Task {
	task := &types.Task{
		Name:   fmt.Sprintf("Ad-hoc: %s", cmd.Module),
		Module: cmd.Module,
		Args:   cmd.Args,
	}

	// Apply options
	if opts.Timeout > 0 {
		task.Timeout = opts.Timeout
	}

	return task
}

// executeOnHosts executes task on multiple hosts
func (e *Executor) executeOnHosts(
	ctx context.Context,
	task *types.Task,
	hosts []types.Host,
	opts Options,
) []*Result {
	results := make([]*Result, len(hosts))

	// Determine parallelism
	parallel := opts.Parallel
	if parallel <= 0 {
		parallel = 5 // Default parallelism
	}
	if parallel > len(hosts) {
		parallel = len(hosts)
	}

	// Create semaphore for parallel execution
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	// Execute on each host
	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h types.Host) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute task
			result := e.executeOnHost(ctx, task, h, opts)
			results[idx] = result
		}(i, host)
	}

	wg.Wait()
	return results
}

// executeOnHost executes task on a single host
func (e *Executor) executeOnHost(
	ctx context.Context,
	task *types.Task,
	host types.Host,
	opts Options,
) *Result {
	startTime := time.Now()
	result := &Result{
		Host: host,
		Task: task,
	}

	// Get module from registry
	module, err := e.moduleRegistry.GetModule(task.Module)
	if err != nil {
		result.Error = fmt.Errorf("module not found: %s", task.Module)
		result.Duration = time.Since(startTime)
		return result
	}

	// Prepare module arguments - add task name if not present
	moduleArgs := make(map[string]interface{})
	for k, v := range task.Args {
		moduleArgs[k] = v
	}
	if _, exists := moduleArgs["name"]; !exists {
		// Use task name or generate a default one
		if task.Name != "" {
			moduleArgs["name"] = task.Name
		} else {
			moduleArgs["name"] = fmt.Sprintf("ad-hoc %s", task.Module)
		}
	}

	// Execute module
	taskResult, err := module.Execute(ctx, host, moduleArgs)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result
	}

	result.Result = &taskResult
	result.Duration = time.Since(startTime)
	return result
}

// generateSummary creates execution summary
func (e *Executor) generateSummary(results []*Result, duration time.Duration) *Summary {
	summary := &Summary{
		Total:    len(results),
		Duration: duration,
		Results:  results,
	}

	for _, result := range results {
		if result.Error != nil {
			summary.Failed++
			continue
		}

		if result.Result == nil {
			summary.Skipped++
			continue
		}

		if result.Result.Failed {
			summary.Failed++
		} else {
			summary.Success++
			if result.Result.Changed {
				summary.Changed++
			}
		}
	}

	return summary
}
