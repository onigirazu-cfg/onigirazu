package core

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CoreEngine represents the core execution engine for Onigirazu
type CoreEngine struct {
	logger    *logger.Logger
	parser    *parser.Parser
	inventory *inventory.Manager
	modules   *modules.Registry
	state     *state.Manager
}

// NewCoreEngine creates a new instance of the core engine
func NewCoreEngine(logger *logger.Logger) *CoreEngine {
	cacheManager := cache.NewManager(time.Hour) // 1 hour default TTL
	parserInstance := parser.New()

	return &CoreEngine{
		logger:    logger,
		parser:    parserInstance,
		inventory: inventory.NewManager(parserInstance, logger, cacheManager),
		modules:   modules.NewRegistry(),
	}
}

// Run executes playbook
func (e *CoreEngine) Run(playbookPath, inventoryPath string, checkMode bool, stateFile string) error {
	e.logger.Info("Starting Onigirazu Core Engine")
	e.logger.Debug("Playbook: %s", playbookPath)
	e.logger.Debug("Inventory: %s", inventoryPath)
	e.logger.Debug("Check mode: %v", checkMode)
	e.logger.Debug("State file: %s", stateFile)

	// Initialize state manager
	e.state = state.New(stateFile)

	// Load state
	currentState, err := e.state.LoadState()
	if err != nil {
		return fmt.Errorf("error loading state: %w", err)
	}

	// Parse playbook
	ctx := context.Background()
	playbook, err := e.parser.ParsePlaybook(ctx, playbookPath)
	if err != nil {
		return fmt.Errorf("error parsing playbook: %w", err)
	}

	// Load inventory
	err = e.inventory.LoadInventory(ctx, inventoryPath)
	if err != nil {
		return fmt.Errorf("error loading inventory: %w", err)
	}

	// Execute playbook
	results, err := e.executePlaybook(playbook, checkMode, currentState)
	if err != nil {
		return fmt.Errorf("error executing playbook: %w", err)
	}

	// Save state
	currentState.Playbook = playbookPath
	e.state.UpdateState(currentState, results)
	if err := e.state.SaveState(currentState); err != nil {
		e.logger.Warn("Error saving state: %v", err)
	}

	e.logger.Info("Execution completed successfully")
	return nil
}

// executePlaybook executes all plays in playbook
func (e *CoreEngine) executePlaybook(playbook *types.Playbook, checkMode bool, currentState *types.State) ([]types.PlayResult, error) {
	var allResults []types.PlayResult

	for _, play := range playbook.Plays {
		e.logger.Info("Executing play: %s", play.Name)

		// Get hosts for play
		hosts, err := e.inventory.GetHosts(play.Hosts)
		if err != nil {
			return nil, fmt.Errorf("error getting hosts for play '%s': %w", play.Name, err)
		}

		if len(hosts) == 0 {
			e.logger.Warn("No hosts for play '%s'", play.Name)
			continue
		}

		// Execute play on each host
		for _, host := range hosts {
			playResult, err := e.executePlay(play, host, checkMode, currentState)
			if err != nil {
				e.logger.Error("Error executing play '%s' on host '%s': %v", play.Name, host.Name, err)
				return allResults, fmt.Errorf("play '%s' failed on host '%s': %w", play.Name, host.Name, err)
			}

			allResults = append(allResults, playResult)
		}
	}

	return allResults, nil
}

// executePlay executes play on specific host
func (e *CoreEngine) executePlay(play types.Play, host types.Host, checkMode bool, currentState *types.State) (types.PlayResult, error) {
	startTime := time.Now()

	result := types.PlayResult{
		PlayName:  play.Name,
		Host:      host.Name,
		Tasks:     []types.TaskResult{},
		StartTime: startTime,
	}

	e.logger.PlayStart(play.Name, 0, 1)

	// Execute tasks
	for _, task := range play.Tasks {
		taskResult, err := e.executeTask(task, host, checkMode, currentState)
		if err != nil {
			e.logger.Error("Task '%s' on host '%s' failed: %v", task.Name, host.Name, err)
			result.Tasks = append(result.Tasks, taskResult)
			if !task.IgnoreErrors {
				result.Success = false
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				e.logger.PlayEnd(play.Name, host.Name, result.Success, result.Duration)
				return result, err
			}
		} else {
			if taskResult.Success {
				changeStatus := ""
				if taskResult.Changed {
					changeStatus = " (changed)"
				}
				e.logger.Info("Task '%s' on host '%s': SUCCESS%s", task.Name, host.Name, changeStatus)
			} else {
				e.logger.Error("Task '%s' on host '%s' failed: %s", task.Name, host.Name, taskResult.Error)
			}
			result.Tasks = append(result.Tasks, taskResult)
		}
	}

	result.Success = true
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	e.logger.PlayEnd(play.Name, host.Name, result.Success, result.Duration)

	return result, nil
}

// executeTask executes individual task
func (e *CoreEngine) executeTask(task types.Task, host types.Host, checkMode bool, currentState *types.State) (types.TaskResult, error) {
	e.logger.TaskStart(task.Name, host.Name)

	if checkMode {
		// In check mode only validate arguments
		module, err := e.modules.GetModule(task.Module)
		if err != nil {
			return types.TaskResult{}, err
		}

		args := make(map[string]interface{})
		args["name"] = task.Name
		for key, value := range task.Args {
			args[key] = value
		}

		if err := module.Validate(args); err != nil {
			return types.TaskResult{}, err
		}

		// Return dummy result for check mode
		return types.TaskResult{
			TaskName:  task.Name,
			Host:      host.Name,
			Module:    task.Module,
			Success:   true,
			Changed:   false,
			Output:    map[string]interface{}{"message": "Check mode - changes not applied"},
			Duration:  time.Since(time.Now()),
			Timestamp: time.Now(),
		}, nil
	}

	// Normal execution
	ctx := context.Background()
	variables := make(map[string]interface{})
	return e.modules.ExecuteTask(ctx, &task, host, variables)
}
