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
	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CoreEngine represents the core execution engine for Onigirazu
type CoreEngine struct {
	logger          *logger.Logger
	parser          *parser.Parser
	inventory       *inventory.Manager
	modules         *modules.Registry
	state           *state.Manager
	pluginManager   *plugins.Manager
	callbackManager *plugins.CallbackManager
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

// NewCoreEngineWithPlugins creates a new instance of the core engine with plugin support
func NewCoreEngineWithPlugins(logger *logger.Logger, pluginManager *plugins.Manager) *CoreEngine {
	engine := NewCoreEngine(logger)
	engine.pluginManager = pluginManager
	engine.callbackManager = plugins.NewCallbackManager()

	// Load callback plugins
	engine.loadCallbackPlugins()

	return engine
}

// loadCallbackPlugins loads all registered callback plugins
func (e *CoreEngine) loadCallbackPlugins() {
	if e.pluginManager == nil {
		return
	}

	callbackPlugins := e.pluginManager.List(plugins.PluginTypeCallback)

	for _, plugin := range callbackPlugins {
		callbackPlugin, ok := plugin.(plugins.CallbackPlugin)
		if !ok {
			e.logger.Warn("Failed to cast plugin '%s' to CallbackPlugin", plugin.GetName())
			continue
		}

		e.callbackManager.AddPlugin(callbackPlugin)
		e.logger.Debug("Loaded callback plugin: %s", callbackPlugin.GetName())
	}
}

// SetPluginManager sets the plugin manager for the engine
func (e *CoreEngine) SetPluginManager(pluginManager *plugins.Manager) {
	e.pluginManager = pluginManager
	e.callbackManager = plugins.NewCallbackManager()
	e.loadCallbackPlugins()
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
	ctx := context.Background()
	playbookStartTime := time.Now()

	// Trigger OnPlaybookStart callback
	if e.callbackManager != nil {
		if err := e.callbackManager.OnPlaybookStart(ctx, playbook); err != nil {
			e.logger.Warn("Callback OnPlaybookStart failed: %v", err)
		}
	}

	var playbookErr error
	for _, play := range playbook.Plays {
		e.logger.Info("Executing play: %s", play.Name)

		// Get hosts for play
		hosts, err := e.inventory.GetHosts(play.Hosts)
		if err != nil {
			playbookErr = fmt.Errorf("error getting hosts for play '%s': %w", play.Name, err)
			break
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
				playbookErr = fmt.Errorf("play '%s' failed on host '%s': %w", play.Name, host.Name, err)
				allResults = append(allResults, playResult)
				break
			}

			allResults = append(allResults, playResult)
		}

		if playbookErr != nil {
			break
		}
	}

	// Trigger OnPlaybookEnd callback
	if e.callbackManager != nil {
		playbookDuration := time.Since(playbookStartTime)
		success := playbookErr == nil
		if err := e.callbackManager.OnPlaybookEnd(ctx, playbook, success, playbookDuration); err != nil {
			e.logger.Warn("Callback OnPlaybookEnd failed: %v", err)
		}
	}

	return allResults, playbookErr
}

// executePlay executes play on specific host
func (e *CoreEngine) executePlay(play types.Play, host types.Host, checkMode bool, currentState *types.State) (types.PlayResult, error) {
	startTime := time.Now()
	ctx := context.Background()

	result := types.PlayResult{
		PlayName:  play.Name,
		Host:      host.Name,
		Tasks:     []types.TaskResult{},
		StartTime: startTime,
	}

	e.logger.PlayStart(play.Name, 0, 1)

	// Trigger OnPlayStart callback
	if e.callbackManager != nil {
		if err := e.callbackManager.OnPlayStart(ctx, &play); err != nil {
			e.logger.Warn("Callback OnPlayStart failed: %v", err)
		}
	}

	// Execute tasks
	var playErr error
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

				// Trigger OnPlayEnd callback
				if e.callbackManager != nil {
					if err := e.callbackManager.OnPlayEnd(ctx, &play, false, result.Duration); err != nil {
						e.logger.Warn("Callback OnPlayEnd failed: %v", err)
					}
				}

				playErr = err
				break
			}
		} else {
			if taskResult.Success {
				changeStatus := ""
				if taskResult.Changed {
					changeStatus = " (changed)"
				}
				e.logger.Info("Task '%s' on host '%s': SUCCESS%s", task.Name, host.Name, changeStatus)

				// Print debug output if this is a debug module
				if task.Module == "debug" {
					if msg, ok := taskResult.Output["msg"]; ok {
						e.logger.Info("  %v", msg)
					}
				}
			} else {
				e.logger.Error("Task '%s' on host '%s' failed: %s", task.Name, host.Name, taskResult.Error)
			}
			result.Tasks = append(result.Tasks, taskResult)
		}
	}

	if playErr == nil {
		result.Success = true
	}
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	e.logger.PlayEnd(play.Name, host.Name, result.Success, result.Duration)

	// Trigger OnPlayEnd callback
	if e.callbackManager != nil {
		if err := e.callbackManager.OnPlayEnd(ctx, &play, result.Success, result.Duration); err != nil {
			e.logger.Warn("Callback OnPlayEnd failed: %v", err)
		}
	}

	return result, playErr
}

// executeTask executes individual task
func (e *CoreEngine) executeTask(task types.Task, host types.Host, checkMode bool, currentState *types.State) (types.TaskResult, error) {
	ctx := context.Background()
	e.logger.TaskStart(task.Name, host.Name)

	// Trigger OnTaskStart callback
	if e.callbackManager != nil {
		if err := e.callbackManager.OnTaskStart(ctx, &task, host); err != nil {
			e.logger.Warn("Callback OnTaskStart failed: %v", err)
		}
	}

	var result types.TaskResult
	var taskErr error

	if checkMode {
		// In check mode only validate arguments
		module, err := e.modules.GetModule(task.Module)
		if err != nil {
			taskErr = err
			result = types.TaskResult{
				TaskName:  task.Name,
				Host:      host.Name,
				Module:    task.Module,
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			args := make(map[string]interface{})
			for key, value := range task.Args {
				args[key] = value
			}
			// Add task name only if not already specified in args
			if _, exists := args["name"]; !exists {
				args["name"] = task.Name
			}

			if err := module.Validate(args); err != nil {
				taskErr = err
				result = types.TaskResult{
					TaskName:  task.Name,
					Host:      host.Name,
					Module:    task.Module,
					Success:   false,
					Error:     err.Error(),
					Timestamp: time.Now(),
				}
			} else {
				// Return dummy result for check mode
				result = types.TaskResult{
					TaskName:  task.Name,
					Host:      host.Name,
					Module:    task.Module,
					Success:   true,
					Changed:   false,
					Output:    map[string]interface{}{"message": "Check mode - changes not applied"},
					Duration:  time.Since(time.Now()),
					Timestamp: time.Now(),
				}
			}
		}
	} else {
		// Normal execution
		variables := make(map[string]interface{})
		result, taskErr = e.modules.ExecuteTask(ctx, &task, host, variables)
	}

	// Trigger OnTaskEnd callback
	if e.callbackManager != nil {
		if err := e.callbackManager.OnTaskEnd(ctx, &task, host, result); err != nil {
			e.logger.Warn("Callback OnTaskEnd failed: %v", err)
		}
	}

	return result, taskErr
}
