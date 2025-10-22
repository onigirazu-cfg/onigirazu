package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
		check          bool
		diff           bool
		dryRun         bool
		pluginsConfig  string
		logLevel       string
		logFormat      string
		outputFormat   string
		parallel       int
		timeout        time.Duration
		interactive    bool
		tags           string
		skipTags       string
		listTags       bool
		listTasks      bool
		verboseOutput  bool
		backgroundMode bool
		tmuxMode       bool
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

			// Determine playbook directory for discovery
			playbookDir := filepath.Dir(playbookPath)

			// Load configuration with priority-based discovery
			// Priority 1: Explicitly specified path
			// Priority 2: Config file in playbook directory
			// Priority 3: Config file in /etc/onigirazu/
			cfg, err := config.LoadConfigWithDiscovery(configPath, playbookDir)
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

			// Set display mode based on verbosity flag or log level
			if verboseOutput || cfg.LogLevel == "debug" {
				if cfg.LogLevel == "debug" {
					log.SetMode(logger.LogModeDebug)
				} else {
					log.SetMode(logger.LogModeVerbose)
				}
			}

			// Print startup banner
			if cfg.IsColorOutputEnabled() {
				fmt.Println(utils.Colors.Header("🍙 Starting Onigirazu Configuration Management Tool"))
			} else {
				fmt.Println("Starting Onigirazu Configuration Management Tool")
			}

			log.Info("Starting Onigirazu configuration management tool")
			log.Debug("Configuration loaded: max_concurrency=%d, log_level=%s", cfg.MaxConcurrency, cfg.LogLevel)

			// Print initialization phase with formatter
			log.PrintInitialization(logger.InitConfig{
				StateBackend:   cfg.StateFile,
				MaxConcurrency: cfg.MaxConcurrency,
				SSHStrictMode:  cfg.IsSSHStrictHostKeyEnabled(),
				ConfigPath:     configPath,
				LogLevel:       cfg.LogLevel,
				ColorOutput:    cfg.IsColorOutputEnabled(),
			})

			// Initialize audit recorder
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Warn("Failed to get home directory for audit: %v", err)
			}
			var auditRecorder *audit.Recorder
			var executionID string
			var auditEnabled bool
			if homeDir != "" {
				auditPath := filepath.Join(homeDir, ".onigirazu", "audit")
				auditConfig := audit.AuditConfig{
					Enabled:     true,
					StoragePath: auditPath,
				}
				auditRecorder, err = audit.NewRecorder(auditConfig, log)
				if err != nil {
					log.Warn("Failed to initialize audit recorder: %v", err)
					auditEnabled = false
				} else {
					auditEnabled = true
				}
			} else {
				log.Warn("Audit recording disabled: cannot determine home directory")
				auditEnabled = false
			}

			// Update initialization config with audit info if available
			initConfig := logger.InitConfig{
				StateBackend:   cfg.StateFile,
				MaxConcurrency: cfg.MaxConcurrency,
				SSHStrictMode:  cfg.IsSSHStrictHostKeyEnabled(),
				ConfigPath:     configPath,
				LogLevel:       cfg.LogLevel,
				ColorOutput:    cfg.IsColorOutputEnabled(),
				AuditEnabled:   auditEnabled,
			}

			// Initialize plugin system
			var pluginManager *plugins.Manager
			var pluginInfos []logger.PluginInfo
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
						allPlugins := pluginManager.ListAll()
						totalPlugins := 0
						for _, pluginList := range allPlugins {
							totalPlugins += len(pluginList)
							for _, plugin := range pluginList {
								pluginInfos = append(pluginInfos, logger.PluginInfo{
									Name:    plugin.GetName(),
									Version: plugin.GetVersion(),
								})
							}
						}
						log.Info("Plugins loaded successfully: %d plugins registered", totalPlugins)
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
								allPlugins := pluginManager.ListAll()
								totalPlugins := 0
								for _, pluginList := range allPlugins {
									totalPlugins += len(pluginList)
									for _, plugin := range pluginList {
										pluginInfos = append(pluginInfos, logger.PluginInfo{
											Name:    plugin.GetName(),
											Version: plugin.GetVersion(),
										})
									}
								}
								log.Info("Auto-detected plugins loaded: %d plugins registered", totalPlugins)
							}
						}
					}
				}
			}

			// Add plugins to init config
			initConfig.Plugins = pluginInfos

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

				// Print inventory info with formatter using stats
				stats := inventoryManager.GetInventoryStats()
				hostCount := 0
				groupCount := 0
				if groupsCnt, ok := stats["groups"].(int); ok {
					groupCount = groupsCnt
				}
				if hostsCnt, ok := stats["total_hosts"].(int); ok {
					hostCount = hostsCnt
				}
				log.PrintInventoryLoaded(logger.InventoryInfo{
					Path:       finalInventoryPath,
					GroupCount: groupCount,
					HostCount:  hostCount,
				})
			}

			// Parse playbook
			log.Info("Parsing playbook: %s", playbookPath)
			playbook, err := enhancedParser.ParsePlaybook(ctx, playbookPath)
			if err != nil {
				return fmt.Errorf("failed to parse playbook: %w", err)
			}

			log.Info("Playbook parsed successfully: %s (%d plays)", playbook.Name, len(playbook.Plays))

			// Print playbook info with formatter
			log.PrintPlaybookLoaded(logger.PlaybookInfo{
				Path:      playbookPath,
				PlayCount: len(playbook.Plays),
				Name:      playbook.Name,
			})

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
					initConfig.ExecutionID = executionID
				}
			}

			// Print execution start with formatter
			log.PrintExecutionStart()

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

			// === PHASE 2A: Cache Execution Results ===
			// Save results to cache for later retrieval without re-execution
			if homeDir != "" {
				cacheDir := filepath.Join(homeDir, ".onigirazu", "cache", "executions")
				cacheMgr, err := execution.NewCacheManagerWithPath(cacheDir)
				if err == nil {
					// Count successes and failures from results
					totalSuccess := 0
					totalFailed := 0
					totalChanged := 0
					totalSkipped := 0

					for _, play := range result.Plays {
						for _, task := range play.Tasks {
							if task.Failed {
								totalFailed++
							} else if task.Changed {
								totalChanged++
							} else if task.Skipped {
								totalSkipped++
							} else {
								totalSuccess++
							}
						}
					}

					// Build execution result for caching
					execResult := &execution.ExecutionResult{
						ExecutionID:    fmt.Sprintf("exec-%d", time.Now().Unix()),
						PlaybookName:   filepath.Base(playbookPath),
						PlaybookPath:   playbookPath,
						Status:         "completed",
						StartTime:      startTime,
						EndTime:        time.Now(),
						Duration:       duration,
						TotalSuccess:   totalSuccess,
						TotalFailed:    totalFailed,
						TotalChanged:   totalChanged,
						TotalSkipped:   totalSkipped,
						TotalHosts:     len(result.Stats),
						PlaybookResult: result,
					}

					// Save to cache
					if err := cacheMgr.Save(execResult); err != nil {
						log.Warn("Failed to cache execution results: %v", err)
					} else {
						log.Debug("Execution results cached: %s", execResult.ExecutionID)
					}
				}
			}

			// === PHASE 2B: Handle Execution Modes ===
			// Background mode: Return execution ID immediately
			if backgroundMode {
				execID := fmt.Sprintf("exec-%d", time.Now().Unix())
				fmt.Printf("\n✓ Playbook started in background\n")
				fmt.Printf("Execution ID: %s\n", execID)
				fmt.Printf("\nView results with:\n")
				fmt.Printf("  onigirazu show-execution %s\n", execID)
				fmt.Printf("  onigirazu show-execution %s --verbose\n", execID)
				fmt.Printf("  onigirazu show-execution %s --debug\n", execID)
				return nil
			}

			// Tmux mode: Start tmux session if available
			if tmuxMode {
				available, instructions := execution.CheckTmuxInstallation()
				if available {
					tm := execution.NewTmuxManager()
					_, err := tm.Start()
					if err != nil {
						log.Warn("Failed to start tmux session: %v, falling back to normal mode", err)
					} else {
						// Tmux session started, execution is displayed there
						return nil
					}
				} else {
					// Tmux not available, show instructions and fall back to interactive
					fmt.Printf("\n%s\n", execution.GetFallbackInstructions())
					fmt.Printf("Installation instructions: %s\n", instructions)
					interactive = true // Fall back to interactive mode
				}
			}

			// Interactive mode: Display with keyboard controls
			if interactive || (backgroundMode == false && tmuxMode == false) {
				// Initialize display formatter for interactive mode
				displayMode := execution.DisplayNormal
				if verboseOutput || cfg.LogLevel == "debug" {
					if cfg.LogLevel == "debug" {
						displayMode = execution.DisplayDebug
					} else {
						displayMode = execution.DisplayVerbose
					}
				}

				useColors := cfg.IsColorOutputEnabled() && utils.IsColorTerminal()
				displayer := execution.NewDisplayer(displayMode, useColors)

				// Create and display execution result
				if homeDir != "" {
					cacheDir := filepath.Join(homeDir, ".onigirazu", "cache", "executions")
					cacheMgr, err := execution.NewCacheManagerWithPath(cacheDir)
					if err == nil {
						latestResult, err := cacheMgr.LoadLatest()
						if err == nil && latestResult != nil {
							displayer.DisplayExecution(latestResult)
						}
					}
				}

				// Set up interactive mode if requested
				if interactive {
					interactiveMode := execution.NewInteractiveMode(useColors)
					interactiveMode.Start()
				}
			}

			// Prepare execution summary
			summary := logger.ExecutionSummary{
				TotalDuration: duration,
				PlayCount:     len(result.Plays),
				Stats:         result.Stats,
			}

			// Calculate task counts from plays
			for _, play := range result.Plays {
				for _, task := range play.Tasks {
					summary.TaskCount++
					if task.Failed {
						summary.FailedCount++
					} else if task.Changed {
						summary.ChangedCount++
					} else if task.Skipped {
						summary.SkippedCount++
					} else {
						summary.SuccessCount++
					}
				}
			}

			if result.Failed {
				log.Error("Playbook execution failed")
				displayExecutionSummary(log, result)
				// Print formatted execution end
				log.PrintExecutionEnd(summary)

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
			// Print formatted execution end
			log.PrintExecutionEnd(summary)

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
	cmd.Flags().BoolVar(&verboseOutput, "verbose-output", false, "Use verbose output formatting (more details)")
	cmd.Flags().BoolVar(&backgroundMode, "background", false, "Run in background mode (returns immediately, use show-execution to view results)")
	cmd.Flags().BoolVar(&tmuxMode, "tmux", false, "Run in tmux session with interactive controls (requires tmux installed)")

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

// displayExecutionSummary displays a summary of the execution results using beautiful formatting
func displayExecutionSummary(log *logger.EnhancedLogger, result *types.PlaybookResult) {
	// Build aggregated results from the last play
	if len(result.Plays) == 0 {
		// Fallback for when plays aren't available (e.g., during early errors)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("\n📋 Playbook: %s\n", result.Name))
		sb.WriteString(fmt.Sprintf("⏱️  Total Duration: %v\n", result.Duration))
		if result.Stats != nil && result.Stats["total_tasks"] != nil {
			sb.WriteString(fmt.Sprintf("📦 Tasks: %v total", result.Stats["total_tasks"]))
			if result.Stats["successful_tasks"] != nil {
				sb.WriteString(fmt.Sprintf(" | ✓ %v successful", result.Stats["successful_tasks"]))
			}
			if result.Stats["failed_tasks"] != nil && result.Stats["failed_tasks"] != 0 {
				sb.WriteString(fmt.Sprintf(" | ✗ %v failed", result.Stats["failed_tasks"]))
			}
			if result.Stats["changed_tasks"] != nil && result.Stats["changed_tasks"] != 0 {
				sb.WriteString(fmt.Sprintf(" | ⚡ %v changed", result.Stats["changed_tasks"]))
			}
			sb.WriteString("\n")
		}
		log.Info("%s", sb.String())

		// Show error if available
		if result.Error != "" && result.Failed {
			log.Error("\n🔍 ERROR ANALYSIS")
			log.Error("=================\n")
			errorAnalyzer := output.NewErrorAnalyzer()
			errType, suggestions := errorAnalyzer.AnalyzeError(result.Error)
			log.Error("❌ Error Type: %s", errType)
			log.Error("   Message: %s", result.Error)
			if len(suggestions) > 0 {
				log.Error("   💡 Suggestions:")
				for _, suggestion := range suggestions {
					log.Error("      • %s", suggestion)
				}
			}
		}
		return
	}

	lastPlay := result.Plays[len(result.Plays)-1]

	// Aggregate results
	aggregator := output.NewResultAggregator()
	errorAnalyzer := output.NewErrorAnalyzer()
	var failedErrors []string
	var failedHosts []string

	for _, hostResult := range lastPlay.Hosts {
		host := output.AggregatedHost{
			Name:    hostResult.Host,
			Details: make(map[string]interface{}),
		}

		// Aggregate task statistics
		var totalDuration time.Duration
		var changedTasks int
		var errorMsg string

		for _, task := range hostResult.Tasks {
			totalDuration += task.Duration
			if task.Changed {
				changedTasks++
			}
			if task.Failed && errorMsg == "" {
				errorMsg = task.Error
			}
		}

		host.Duration = totalDuration

		// Determine status
		if hostResult.Failed {
			host.Status = output.StatusFailed
			host.ErrorMessage = errorMsg
			if errorMsg != "" {
				failedErrors = append(failedErrors, errorMsg)
				failedHosts = append(failedHosts, hostResult.Host)
				// Analyze the error
				errType, suggestions := errorAnalyzer.AnalyzeError(errorMsg)
				host.Suggestions = suggestions
				host.Details["error_type"] = errType
			}
		} else if changedTasks > 0 {
			host.Status = output.StatusChanged
			host.Changed = true
		} else {
			host.Status = output.StatusSuccess
		}

		host.Details["tasks_count"] = len(hostResult.Tasks)
		host.Details["changed_tasks"] = changedTasks

		aggregator.Add(host)
	}

	// Format results
	formatter := output.NewConsoleFormatter(os.Getenv("NO_COLOR") != "")
	aggregated := aggregator.Aggregate()
	metrics := aggregator.GetMetrics()

	// Create header with playbook info
	var headerBuilder strings.Builder
	headerBuilder.WriteString(fmt.Sprintf("\n📋 Playbook: %s\n", result.Name))
	headerBuilder.WriteString(fmt.Sprintf("⏱️  Total Duration: %v\n", result.Duration))
	if result.Stats != nil && result.Stats["total_tasks"] != nil {
		headerBuilder.WriteString(fmt.Sprintf("📦 Tasks: %v total", result.Stats["total_tasks"]))
		if result.Stats["successful_tasks"] != nil {
			headerBuilder.WriteString(fmt.Sprintf(" | ✓ %v successful", result.Stats["successful_tasks"]))
		}
		if result.Stats["failed_tasks"] != nil && result.Stats["failed_tasks"] != 0 {
			headerBuilder.WriteString(fmt.Sprintf(" | ✗ %v failed", result.Stats["failed_tasks"]))
		}
		if result.Stats["changed_tasks"] != nil && result.Stats["changed_tasks"] != 0 {
			headerBuilder.WriteString(fmt.Sprintf(" | ⚡ %v changed", result.Stats["changed_tasks"]))
		}
		headerBuilder.WriteString("\n")
	}

	// Output formatted results
	log.Info("%s", headerBuilder.String())
	log.Info("%s", formatter.FormatAggregatedResults(aggregated, metrics))

	// If there are errors, show detailed error analysis
	if len(failedErrors) > 0 {
		log.Error("\n🔍 ERROR ANALYSIS")
		log.Error("=================\n")

		// Analyze all errors together for summary
		errorSummary := errorAnalyzer.SummarizeErrors(failedErrors)

		for i, host := range failedHosts {
			if i < len(failedErrors) {
				analyzed := errorAnalyzer.AnalyzeHostError(host, failedErrors[i])

				log.Error("❌ Host: %s", analyzed.Host)
				log.Error("   Error Type: %s", analyzed.Type)
				log.Error("   Message: %s", analyzed.Message)

				if len(analyzed.Suggestions) > 0 {
					log.Error("   💡 Suggestions:")
					for _, suggestion := range analyzed.Suggestions {
						log.Error("      • %s", suggestion)
					}
				}

				if analyzed.RetryAdvice != "" {
					log.Error("   🔄 Retry Advice: %s", analyzed.RetryAdvice)
				}
				log.Error("")
			}
		}

		if errorSummary.Total > 1 {
			log.Error("📊 Error Summary:")
			log.Error("   Total Errors: %d", errorSummary.Total)
			if len(errorSummary.ByType) > 0 {
				log.Error("   By Type:")
				for errType, count := range errorSummary.ByType {
					log.Error("      • %s: %d", errType, count)
				}
			}
		}
	}
}
