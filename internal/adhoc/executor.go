package adhoc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Executor handles execution of ad-hoc commands
type Executor struct {
	moduleRegistry *modules.Registry
	inventoryMgr   *inventory.Manager
	logger         interfaces.Logger
	templates      *template.Engine
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
		templates:      template.NewEngine(),
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

	module, err := e.moduleRegistry.GetModule(task.Module)
	if err != nil {
		result.Error = fmt.Errorf("module not found: %s", task.Module)
		result.Duration = time.Since(startTime)
		return result
	}

	// through the registry, as a playbook task: --check runs only modules
	// that support check mode (the others are skipped, not run), --diff asks
	// for before/after, arguments are normalized like in apply
	vars := map[string]interface{}{}
	for k, v := range host.Vars {
		vars[k] = v
	}
	for k, v := range opts.Variables {
		vars[k] = v
	}
	vars["inventory_hostname"] = host.Name

	// {{ }} in the arguments, as in a playbook task
	args, err := e.templates.RenderTaskArgs(ctx, copyArgs(task.Args), vars)
	if err != nil {
		result.Error = fmt.Errorf("failed to render arguments: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}
	if err := module.Validate(copyArgs(args)); err != nil {
		result.Error = fmt.Errorf("invalid arguments for %s: %w", task.Module, err)
		result.Duration = time.Since(startTime)
		return result
	}

	check := opts.Check
	run := &types.Task{Name: task.Name, Module: task.Module, Args: args, CheckMode: &check, Diff: opts.Diff,
		Become: opts.Become, BecomeUser: opts.BecomeUser}
	if run.Name == "" {
		run.Name = fmt.Sprintf("ad-hoc %s", task.Module)
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	taskResult, err := e.moduleRegistry.ExecuteTask(ctx, run, host, vars)
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

func copyArgs(args map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(args))
	for k, v := range args {
		out[k] = v
	}
	return out
}
