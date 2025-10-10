package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/internal/progress"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// NewApplyCommand creates the apply command
func NewApplyCommand() *cobra.Command {
	var (
		// Command-specific flags
		check         bool
		diff          bool
		dryRun        bool
		pluginsConfig string
		logLevel      string
		logFormat     string
		outputFormat  string
		maxWorkers    int
		timeout       time.Duration
		interactive   bool
	)

	cmd := &cobra.Command{
		Use:   "apply [playbook]",
		Short: "Execute a playbook",
		Long: `Execute a playbook and apply configuration changes to target hosts.

This command runs the specified playbook, executing all tasks against the
inventory hosts. Changes are tracked in the state file for future reference.

Examples:
  # Execute a playbook
  onigirazu apply production.yml

  # Execute with custom inventory
  onigirazu apply production.yml --inventory hosts.yml

  # Dry-run mode (no changes)
  onigirazu apply production.yml --check

  # Show differences
  onigirazu apply production.yml --diff

  # Verbose output
  onigirazu apply production.yml --verbose`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			playbookPath := args[0]

			// Configure colors
			if noColor {
				utils.EnableColors(false)
			}

			// Create context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle signals for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
				cancel()
			}()

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Override config with command line flags
			if verbose {
				cfg.LogLevel = "debug"
				cfg.Verbose = true
			}
			if logLevel != "info" {
				cfg.LogLevel = logLevel
			}
			if logFormat != "text" {
				cfg.LogFormat = logFormat
			}
			if outputFormat != "text" {
				cfg.OutputFormat = outputFormat
			}
			if maxWorkers != 10 {
				cfg.MaxConcurrency = maxWorkers
			}
			if statePath != ".onigirazu-state" {
				cfg.StateFile = statePath
			}
			if check || dryRun {
				cfg.CheckMode = true
				cfg.DryRun = true
			}
			if diff {
				cfg.ShowDiff = true
			}
			if noColor {
				cfg.ColorOutput = false
			}
			if interactive {
				cfg.InteractiveMode = true
			}

			// Initialize logger
			log := logger.NewEnhanced(cfg.LogLevel, logger.LogFormat(cfg.LogFormat), os.Stdout)

			// Print startup banner
			if cfg.IsColorOutputEnabled() {
				fmt.Println(utils.Colors.Header("🍙 Starting Onigirazu Configuration Management Tool"))
			} else {
				fmt.Println("Starting Onigirazu Configuration Management Tool")
			}

			log.Info("Starting Onigirazu configuration management tool")
			log.Debug("Configuration loaded: max_concurrency=%d, log_level=%s", cfg.MaxConcurrency, cfg.LogLevel)

			// Initialize plugin system
			var pluginManager *plugins.Manager
			if pluginsConfig != "" {
				log.Info("Loading plugins from configuration: %s", pluginsConfig)
				pluginConfig, err := plugins.LoadConfig(pluginsConfig)
				if err != nil {
					log.Warn("Failed to load plugins configuration: %v", err)
					log.Info("Continuing without plugins")
				} else {
					// Create plugin manager with in-memory loader
					loader := plugins.NewInMemoryLoader()
					pluginManager = plugins.NewManager(loader)

					// Load plugins from configuration
					if err := plugins.LoadPluginsFromConfig(ctx, pluginManager, pluginConfig); err != nil {
						log.Warn("Failed to load plugins: %v", err)
						log.Info("Continuing without plugins")
						pluginManager = nil
					} else {
						log.Info("Plugins loaded successfully: %d plugins registered", len(pluginManager.List("")))
					}
				}
			} else {
				// Try to auto-detect plugins.yml in playbook directory
				if playbookPath != "" {
					playbookDir := filepath.Dir(playbookPath)
					autoPluginsPath := filepath.Join(playbookDir, "plugins.yml")
					if _, err := os.Stat(autoPluginsPath); err == nil {
						log.Info("Auto-detected plugins configuration: %s", autoPluginsPath)
						pluginConfig, err := plugins.LoadConfig(autoPluginsPath)
						if err == nil {
							loader := plugins.NewInMemoryLoader()
							pluginManager = plugins.NewManager(loader)
							if err := plugins.LoadPluginsFromConfig(ctx, pluginManager, pluginConfig); err != nil {
								log.Warn("Failed to load auto-detected plugins: %v", err)
								pluginManager = nil
							} else {
								log.Info("Auto-detected plugins loaded: %d plugins registered", len(pluginManager.List("")))
							}
						}
					}
				}
			}

			// Initialize components
			cacheManager := cache.NewManager(5 * time.Minute) // Default TTL of 5 minutes

			// Create template engine with plugin support
			var templateEngine *template.Engine
			if pluginManager != nil {
				templateEngine = template.NewEngineWithPlugins(pluginManager)
				log.Debug("Template engine initialized with plugin support")
			} else {
				templateEngine = template.NewEngine()
			}

			stateManager := state.NewEnhancedManager(cfg.StateFile, log)
			executionPool := execution.NewPool(cfg.MaxConcurrency, log)
			progressTracker := progress.NewTracker()
			moduleRegistry := modules.NewRegistry()

			// Initialize parser and inventory manager
			enhancedParser := parser.NewEnhancedParser(templateEngine, log)
			inventoryManager := inventory.NewManager(enhancedParser, log, cacheManager)

			// Initialize execution engine
			executionEngine := engine.NewExecutionEngine(
				cfg,
				log,
				stateManager,
				inventoryManager,
				moduleRegistry,
				templateEngine,
				progressTracker,
				executionPool,
				cacheManager,
			)

			// Set execution timeout
			if timeout > 0 {
				var timeoutCancel context.CancelFunc
				ctx, timeoutCancel = context.WithTimeout(ctx, timeout)
				defer timeoutCancel()
			}

			// Load inventory
			var finalInventoryPath string
			if inventoryPath != "" {
				// Use specified inventory path
				finalInventoryPath = inventoryPath
			} else {
				// Try to find inventory file in playbook directory
				playbookDir := filepath.Dir(playbookPath)
				foundPath, err := enhancedParser.FindInventoryFile(playbookDir)
				if err != nil {
					log.Warn("No inventory file found in playbook directory: %v", err)
					log.Info("Continuing without inventory (only 'localhost' will be available)")
				} else {
					finalInventoryPath = foundPath
					log.Info("Auto-detected inventory file: %s", finalInventoryPath)
				}
			}

			// Load inventory if we have a path
			if finalInventoryPath != "" {
				log.Info("Loading inventory from: %s", finalInventoryPath)
				if err := inventoryManager.LoadInventory(ctx, finalInventoryPath); err != nil {
					return fmt.Errorf("failed to load inventory: %w", err)
				}
			}

			// Parse playbook
			log.Info("Parsing playbook: %s", playbookPath)
			playbook, err := enhancedParser.ParsePlaybook(ctx, playbookPath)
			if err != nil {
				return fmt.Errorf("failed to parse playbook: %w", err)
			}

			log.Info("Playbook parsed successfully: %s (%d plays)", playbook.Name, len(playbook.Plays))

			// Set check mode if specified
			if cfg.IsCheckMode() {
				if cfg.IsColorOutputEnabled() {
					fmt.Println(utils.Colors.Warning("⚠️  Running in check mode (dry-run) - no changes will be made"))
				} else {
					fmt.Println("Running in check mode (dry-run) - no changes will be made")
				}
				log.Info("Running in check mode (dry-run)")
			}

			// Start progress tracking
			progressTracker.StartTracking()
			defer progressTracker.Stop()

			// Execute playbook
			log.Info("Starting playbook execution")
			startTime := time.Now()

			result, err := executionEngine.ExecutePlaybook(ctx, playbook)
			if err != nil {
				return fmt.Errorf("playbook execution failed: %w", err)
			}

			duration := time.Since(startTime)

			// Display results
			log.Info("Playbook execution completed in %v", duration)

			if result.Failed {
				log.Error("Playbook execution failed")
				displayExecutionSummary(log, result)
				return fmt.Errorf("playbook execution failed")
			}

			log.Info("Playbook execution successful")
			displayExecutionSummary(log, result)

			// Save final state
			if err := stateManager.SaveCurrentState(); err != nil {
				log.Warn("Failed to save final state: %v", err)
			}

			log.Info("Onigirazu execution completed successfully")
			return nil
		},
	}

	// Command-specific flags
	cmd.Flags().BoolVarP(&check, "check", "C", false, "Check mode (dry-run)")
	cmd.Flags().BoolVarP(&diff, "diff", "d", false, "Show differences when changing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run mode (alias for --check)")
	cmd.Flags().StringVar(&pluginsConfig, "plugins-config", "", "Path to plugins configuration file")
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&logFormat, "log-format", "text", "Log format (text, json)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().IntVarP(&maxWorkers, "max-workers", "w", 10, "Maximum number of worker threads")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 30*time.Minute, "Execution timeout")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive mode")

	return cmd
}

// displayExecutionSummary displays a summary of the execution results
func displayExecutionSummary(log *logger.EnhancedLogger, result *types.PlaybookResult) {
	log.Info("=== Execution Summary ===")
	log.Info("Playbook: %s", result.Name)
	log.Info("Duration: %v", result.Duration)
	log.Info("Plays executed: %d", len(result.Plays))

	if result.Stats != nil {
		if totalTasks, exists := result.Stats["total_tasks"]; exists {
			log.Info("Total tasks: %v", totalTasks)
		}
		if successfulTasks, exists := result.Stats["successful_tasks"]; exists {
			log.Info("Successful tasks: %v", successfulTasks)
		}
		if failedTasks, exists := result.Stats["failed_tasks"]; exists {
			log.Info("Failed tasks: %v", failedTasks)
		}
		if changedTasks, exists := result.Stats["changed_tasks"]; exists {
			log.Info("Changed tasks: %v", changedTasks)
		}
		if skippedTasks, exists := result.Stats["skipped_tasks"]; exists {
			log.Info("Skipped tasks: %v", skippedTasks)
		}
	}

	// Display per-host results
	if len(result.Plays) > 0 {
		for hostIndex, hostResult := range result.Plays[len(result.Plays)-1].Hosts {
			status := "OK"
			if hostResult.Failed {
				status = "FAILED"
			}
			log.Info("Host %d: %s (%d tasks)", hostIndex, status, len(hostResult.Tasks))
		}
	}
}
