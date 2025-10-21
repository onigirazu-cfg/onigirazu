package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/output"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/internal/progress"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/internal/tagdiscovery"
	"github.com/onigirazu-cfg/onigirazu/internal/tagfilter"
	"github.com/onigirazu-cfg/onigirazu/internal/taskpreview"
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
		parallel      int
		timeout       time.Duration
		interactive   bool
		tags          string
		skipTags      string
		listTags      bool
		listTasks     bool
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
			if parallel != 10 {
				cfg.MaxConcurrency = parallel
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

			// Initialize audit recorder
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Warn("Failed to get home directory for audit: %v", err)
			}
			var auditRecorder *audit.Recorder
			var executionID string
			if homeDir != "" {
				auditPath := filepath.Join(homeDir, ".onigirazu", "audit")
				auditConfig := audit.AuditConfig{
					Enabled:     true,
					StoragePath: auditPath,
				}
				auditRecorder, err = audit.NewRecorder(auditConfig, log)
				if err != nil {
					log.Warn("Failed to initialize audit recorder: %v", err)
				}
			} else {
				log.Warn("Audit recording disabled: cannot determine home directory")
			}

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

			// Initialize SSH connection pool with host key manager and logger
			sshpkg.InitializeGlobalPoolWithLogger(cfg, log)
			log.Debug("SSH connection pool initialized (strict_mode=%v)", cfg.IsSSHStrictHostKeyEnabled())

			// Create template engine with plugin support
			var templateEngine *template.Engine
			if pluginManager != nil {
				templateEngine = template.NewEngineWithPlugins(pluginManager)
				log.Debug("Template engine initialized with plugin support")
			} else {
				templateEngine = template.NewEngine()
			}

			// Initialize state backend based on configuration
			stateConfig := state.NewDefaultConfig()
			// You can customize this based on environment or config file
			// stateConfig.Backend = state.BackendTypeSQLite

			backendFactory := state.NewBackendFactory(stateConfig)
			stateBackend, err := backendFactory.CreateBackend(cfg.StateFile)
			if err != nil {
				log.Warn("Failed to create state backend: %v, falling back to file backend", err)
				// Fallback to file backend
				stateBackend, _ = backendFactory.CreateFileBackend(cfg.StateFile)
			}

			log.Info("Using state backend: %v", stateConfig.Backend)

			// Also keep the enhanced manager for compatibility with execution engine
			stateManager := state.NewEnhancedManager(cfg.StateFile, log)

			// Load existing state before execution
			if _, err := stateManager.LoadState(ctx); err != nil {
				log.Warn("Failed to load existing state: %v", err)
			}

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

			// Set up tag filtering
			if tags != "" || skipTags != "" {
				tagFilter, err := tagfilter.New(tags, skipTags)
				if err != nil {
					return fmt.Errorf("failed to create tag filter: %w", err)
				}
				executionEngine.SetTagFilter(tagFilter)
				log.Info("Tag filtering enabled: %s", tagFilter.String())
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

			// Handle --list-tags flag
			if listTags {
				tagsResult, err := tagdiscovery.DiscoverTags(playbook)
				if err != nil {
					return fmt.Errorf("failed to discover tags: %w", err)
				}

				// Format and display output
				var result string
				switch outputFormat {
				case "json":
					result = output.FormatTagsJSON(tagsResult)
				case "yaml":
					result = output.FormatTagsYAML(tagsResult)
				case "csv":
					result = output.FormatTagsCSV(tagsResult)
				default:
					result = output.FormatTagsText(tagsResult)
				}
				fmt.Print(result)
				return nil
			}

			// Handle --list-tasks flag
			if listTasks {
				tasksResult, err := taskpreview.PreviewTasks(playbook, tags, skipTags)
				if err != nil {
					return fmt.Errorf("failed to preview tasks: %w", err)
				}

				// Format and display output
				var result string
				switch outputFormat {
				case "json":
					result = output.FormatTasksJSON(tasksResult)
				case "yaml":
					result = output.FormatTasksYAML(tasksResult)
				case "csv":
					result = output.FormatTasksCSV(tasksResult)
				default:
					result = output.FormatTasksText(tasksResult)
				}
				fmt.Print(result)
				return nil
			}

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

			// Start audit recording
			if auditRecorder != nil {
				executionID, err = auditRecorder.StartExecution(playbookPath, finalInventoryPath, []string{})
				if err != nil {
					log.Warn("Failed to start audit recording: %v", err)
				} else {
					log.Info("Audit recording started: execution_id=%s", executionID)
				}
			}

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

			// Record plays and tasks in audit if recorder is available
			if auditRecorder != nil && executionID != "" {
				recordAuditResults(log, auditRecorder, result)
			}

			if result.Failed {
				log.Error("Playbook execution failed")
				displayExecutionSummary(log, result)

				// Still save state even on failure for audit trail
				currentState := &types.State{
					LastRun:   time.Now(),
					Playbook:  playbookPath,
					Variables: result.Variables,
					Checksums: make(map[string]string),
				}
				if currentState.Variables == nil {
					currentState.Variables = make(map[string]interface{})
				}
				if len(result.Plays) > 0 {
					currentState.Results = result.Plays
				}

				if err := stateManager.SaveState(ctx, currentState); err != nil {
					log.Warn("Failed to save state after failure (manager): %v", err)
				}

				// Also save to backend
				if err := stateBackend.SaveState(ctx, currentState); err != nil {
					log.Warn("Failed to save state to backend: %v", err)
				} else {
					log.Info("State file saved to backend (failure recorded)")
				}

				// Complete audit with failure status
				if auditRecorder != nil && executionID != "" {
					if _, err := auditRecorder.CompleteExecution(audit.StatusFailure, 1, result.Error); err != nil {
						log.Warn("Failed to complete audit recording: %v", err)
					}
				}

				return fmt.Errorf("playbook execution failed")
			}

			log.Info("Playbook execution successful")
			displayExecutionSummary(log, result)

			// Create snapshot for rollback capability
			if homeDir != "" {
				snapshotDir := filepath.Join(homeDir, ".onigirazu", "snapshots")
				snapshotMgr := rollback.NewSnapshotManager(snapshotDir)

				// Create snapshot
				snapshot, err := snapshotMgr.CreateSnapshot(playbook.Name, "Auto-created snapshot after playbook execution")
				if err == nil {
					// Add task results as resource snapshots
					for _, play := range result.Plays {
						for _, task := range play.Tasks {
							if task.Changed {
								// Add resource snapshot for each changed task
								resourceSnapshot := rollback.ResourceSnapshot{
									Type:       task.Module,
									TaskName:   task.TaskName,
									Host:       task.Host,
									State:      make(map[string]interface{}),
									Action:     "modified",
									Module:     task.Module,
									Reversible: isModuleReversible(task.Module),
								}

								// Try to extract resource identifier from output
								if task.Output != nil {
									// Copy output as state
									resourceSnapshot.State = task.Output

									// Extract resource identifier
									if name, ok := task.Output["name"]; ok {
										resourceSnapshot.Identifier = fmt.Sprintf("%v", name)
									} else if path, ok := task.Output["path"]; ok {
										resourceSnapshot.Identifier = fmt.Sprintf("%v", path)
									} else if pkg, ok := task.Output["package"]; ok {
										resourceSnapshot.Identifier = fmt.Sprintf("%v", pkg)
									} else if dest, ok := task.Output["dest"]; ok {
										resourceSnapshot.Identifier = fmt.Sprintf("%v", dest)
									}
								}

								snapshotMgr.AddResourceSnapshot(snapshot, resourceSnapshot)
							}
						}
					}

					// Save snapshot
					if err := snapshotMgr.SaveSnapshot(snapshot); err != nil {
						log.Warn("Failed to save snapshot: %v", err)
					} else {
						log.Info("Snapshot created successfully: %s", snapshot.ID)
					}
				} else {
					log.Warn("Failed to create snapshot: %v", err)
				}
			}

			// Save final state with playbook results
			currentState := &types.State{
				LastRun:   time.Now(),
				Playbook:  playbookPath,
				Variables: result.Variables,
				Checksums: make(map[string]string),
			}
			if currentState.Variables == nil {
				currentState.Variables = make(map[string]interface{})
			}

			// Add playbook results to state
			if len(result.Plays) > 0 {
				currentState.Results = result.Plays
			}

			log.Info("Saving state to: %s", cfg.StateFile)
			if err := stateManager.SaveState(ctx, currentState); err != nil {
				log.Warn("Failed to save final state (manager): %v", err)
			}

			// Also save to backend
			if err := stateBackend.SaveState(ctx, currentState); err != nil {
				log.Warn("Failed to save final state to backend: %v", err)
			} else {
				log.Info("State file successfully saved to backend with %d play results", len(currentState.Results))
				log.Debug("State backend path: %s", stateBackend.GetPath())
			}

			// Complete audit with success status
			if auditRecorder != nil && executionID != "" {
				if _, err := auditRecorder.CompleteExecution(audit.StatusSuccess, 0, ""); err != nil {
					log.Warn("Failed to complete audit recording: %v", err)
				}
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
	cmd.Flags().IntVarP(&parallel, "parallel", "f", 10, "Number of parallel executions")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 30*time.Minute, "Execution timeout")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive mode")
	cmd.Flags().StringVar(&tags, "tags", "", "Only run tasks with these tags (comma-separated). Use 'tagged' for tasks with any tag, 'untagged' for tasks without tags, 'all' for default behavior")
	cmd.Flags().StringVar(&skipTags, "skip-tags", "", "Skip tasks with these tags (comma-separated)")
	cmd.Flags().BoolVar(&listTags, "list-tags", false, "List all available tags in the playbook without executing")
	cmd.Flags().BoolVar(&listTasks, "list-tasks", false, "List tasks that would execute with current filters")

	return cmd
}

// isModuleReversible checks if a module's changes can be reversed
func isModuleReversible(module string) bool {
	reversibleModules := map[string]bool{
		"file":       true,
		"copy":       true,
		"template":   true,
		"lineinfile": true,
		"package":    true,
		"service":    true,
		"user":       true,
		"group":      true,
		"git":        true,
		"systemd":    true,
		"cron":       true,
		"command":    false, // shell commands can't be automatically reversed
		"shell":      false,
		"debug":      false,
	}

	return reversibleModules[module]
}

// recordAuditResults records the execution results in the audit system
func recordAuditResults(log *logger.EnhancedLogger, recorder *audit.Recorder, result *types.PlaybookResult) {
	if recorder == nil || result == nil {
		return
	}

	// Record plays from execution result
	for playIdx, play := range result.Plays {
		// Build list of hosts for this play
		playHosts := []string{}
		if len(play.Hosts) > 0 {
			for _, hostResult := range play.Hosts {
				playHosts = append(playHosts, hostResult.Host)
			}
		}

		// Record play in audit
		playRecorder := recorder.RecordPlay(play.Name, playIdx, playHosts)
		if playRecorder != nil {
			// Record all tasks in this play
			for _, task := range play.Tasks {
				taskStatus := audit.TaskStatusOk
				if task.Failed {
					taskStatus = audit.TaskStatusFailed
				} else if task.Changed {
					taskStatus = audit.TaskStatusChanged
				} else if task.Skipped {
					taskStatus = audit.TaskStatusSkipped
				}

				// Convert task output to JSON string for audit storage
				output := ""
				if len(task.Output) > 0 {
					if jsonBytes, err := json.Marshal(task.Output); err == nil {
						output = string(jsonBytes)
					}
				}

				taskErr := ""
				if task.Error != "" {
					taskErr = task.Error
				}

				taskResult := audit.TaskResult{
					Name:      task.TaskName,
					Module:    task.Module,
					Status:    taskStatus,
					Host:      task.Host,
					Duration:  task.Duration.Seconds(),
					StartTime: task.Timestamp,
					EndTime:   task.Timestamp.Add(task.Duration),
					Output:    output,
					Error:     taskErr,
					Changed:   task.Changed,
				}

				if err := playRecorder.RecordTask(taskResult); err != nil {
					log.Warn("Failed to record task in audit: %v", err)
				}
			}
			// Complete the play
			playStatus := audit.StatusSuccess
			if !play.Success {
				playStatus = audit.StatusFailure
			}
			playRecorder.Complete(playStatus)
		}
	}
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
