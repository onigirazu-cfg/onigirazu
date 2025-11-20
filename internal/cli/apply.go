package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/metrics"
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
	"github.com/onigirazu-cfg/onigirazu/internal/validator"
	"github.com/onigirazu-cfg/onigirazu/pkg/profiler"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// NewApplyCommand creates the apply command with improved TUI architecture
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
		lenient        bool
		// Profiling flags
		profileEnabled bool
		profileDir     string
		profileCPU     bool
		profileMem     bool
		profileTrace   bool
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
  onigirazu apply production.yml --verbose

  # Interactive mode with beautiful TUI
  onigirazu apply production.yml --interactive`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			playbookPath := args[0]

			// Configure colors
			if noColor {
				utils.EnableColors(false)
			}

			// Create context with cancellation
			ctx := context.Background()

			// Create signal handler with 10-second graceful shutdown timeout
			signalHandler := execution.NewSignalHandler(ctx, 10*time.Second)
			defer signalHandler.Close()

			// Use the signal handler's context for execution
			ctx = signalHandler.Context()

			// Register SSH pool cleanup (will be executed on graceful shutdown)
			signalHandler.RegisterCleanup(func() error {
				return sshpkg.GetGlobalPool().CloseAll()
			})

			// Determine playbook directory for discovery
			playbookDir := filepath.Dir(playbookPath)

			// Load configuration with priority-based discovery
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

			// Create TUI model early if interactive mode to capture all logs
			var tuiModel *execution.EnhancedTUIModel
			var logWriter io.Writer = os.Stdout

			if interactive {
				tuiModel = execution.NewEnhancedTUIModel()
				logWriter = tuiModel.GetLogWriter()
			}

			// Initialize logger - redirect to TUI if in interactive mode
			log := logger.NewEnhanced(cfg.LogLevel, logger.LogFormat(cfg.LogFormat), logWriter)

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

			// Create execution pool with signal handler's context for graceful shutdown support
			executionPool := execution.NewPoolWithContext(ctx, cfg.MaxConcurrency, log)
			progressTracker := progress.NewTracker()
			moduleRegistry := modules.NewRegistry()

			// Initialize parser and inventory manager
			enhancedParser := parser.NewEnhancedParser(templateEngine, log)

			// Enable lenient mode if requested
			if lenient {
				enhancedParser.SetLenient(true)
			}

			// Set up module syntax validator with available modules
			moduleSyntaxValidator := validator.NewModuleSyntaxValidator(moduleRegistry.ListModules())
			enhancedParser.SetModuleSyntaxValidator(moduleSyntaxValidator)

			inventoryManager := inventory.NewManager(enhancedParser, log, cacheManager)

			// Enable lenient mode in manager if requested
			if lenient {
				inventoryManager.SetLenient(true)
			}

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

			// Load inventory from multiple sources (files, directories, dynamic scripts)
			var finalInventoryPaths []string
			if len(inventoryPaths) > 0 {
				// Use specified inventory paths
				finalInventoryPaths = inventoryPaths
			} else {
				// Try to find inventory file in playbook directory (auto-discovery)
				playbookDir := filepath.Dir(playbookPath)
				foundPath, err := enhancedParser.FindInventoryFile(playbookDir)
				if err != nil {
					log.Warn("No inventory file found in playbook directory: %v", err)
					log.Info("Continuing without inventory (only 'localhost' will be available)")
				} else {
					finalInventoryPaths = []string{foundPath}
					log.Info("Auto-detected inventory source: %s", foundPath)
				}
			}

			// Load inventory from multiple sources if we have paths
			if len(finalInventoryPaths) > 0 {
				log.Info("Loading inventory from %d source(s) with last-occurrence-wins merge strategy",
					len(finalInventoryPaths))

				// Create multi-source loader
				multiSourceLoader := inventory.NewMultiSourceLoader(
					enhancedParser,
					log,
					cacheManager,
					10*time.Minute, // Dynamic inventory cache TTL
				)

				// Load from all sources
				mergedInventory, err := multiSourceLoader.LoadFromMultipleSources(ctx, finalInventoryPaths)
				if err != nil {
					return fmt.Errorf("failed to load inventory from multiple sources: %w", err)
				}

				// Update inventory manager with merged inventory
				if err := inventoryManager.SetInventory(mergedInventory); err != nil {
					return fmt.Errorf("failed to set merged inventory: %w", err)
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

				// Show merged inventory info
				sourcesStr := strings.Join(finalInventoryPaths, ", ")
				log.PrintInventoryLoaded(logger.InventoryInfo{
					Path:       sourcesStr,
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
				// Create a single inventory path string for audit recording
				inventoryPathForAudit := strings.Join(finalInventoryPaths, ", ")
				executionID, err = auditRecorder.StartExecution(playbookPath, inventoryPathForAudit, []string{})
				if err != nil {
					log.Warn("Failed to start audit recording: %v", err)
				} else {
					log.Info("Audit recording started: execution_id=%s", executionID)
				}
			}

			// Print execution start with formatter
			log.PrintExecutionStart()

			// Setup TUI if interactive mode is requested
			if interactive && tuiModel != nil {
				// Attach TUI as observer to execution engine to receive live events
				executionEngine.AttachObserver(tuiModel)
				// Start TUI in a goroutine
				go func() {
					if err := tuiModel.Start(); err != nil {
						log.Warn("Failed to start interactive TUI: %v", err)
					}
				}()
				// Wait for TUI to be ready before starting playbook
				if err := tuiModel.WaitForReady(5 * time.Second); err != nil {
					log.Warn("TUI startup timeout: %v", err)
				}
			}

			// Set up signal handler callbacks for graceful shutdown
			signalHandler.SetCancelCallbacks(
				// onConfirm callback - graceful shutdown with state saving
				func(saveState bool) error {
					// Stop all running tasks via execution pool
					executionPool.StopAll()

					// Record execution interruption in audit
					if auditRecorder != nil && executionID != "" {
						if saveState {
							_ = auditRecorder.SetMetadata("interrupt_status", "saved")
						} else {
							_ = auditRecorder.SetMetadata("interrupt_status", "discarded")
						}
					}

					// Save state if requested
					if saveState && stateManager != nil {
						fmt.Println("Saving state...")
						// Load current state first
						currentState, err := stateManager.LoadState(ctx)
						if err != nil {
							log.Warn("Failed to load state: %v", err)
							currentState = &types.State{
								Variables: make(map[string]interface{}),
								Checksums: make(map[string]string),
							}
						}
						// Save state with loaded data
						if err := stateManager.SaveState(ctx, currentState); err != nil {
							log.Warn("Failed to save state: %v", err)
						} else {
							fmt.Println("✓ State saved successfully")
						}
					} else {
						fmt.Println("State discarded as requested")
					}

					return nil
				},
				// onForce callback - immediate shutdown
				func() error {
					// Force stop all tasks immediately
					executionPool.KillAll()

					// Try to record force termination in audit
					if auditRecorder != nil && executionID != "" {
						_ = auditRecorder.SetMetadata("execution_force_terminated", map[string]interface{}{
							"timestamp": time.Now(),
							"reason":    "force_interrupt",
						})
					}

					return nil
				},
			)

			// === METRICS SERVER SETUP ===
			var metricsInstance *metrics.Metrics
			if cfg.IsMetricsEnabled() {
				log.Info("Metrics collection enabled")
				metricsInstance = metrics.NewMetricsWithPrometheus()

				// Build metrics server address
				metricsAddr := fmt.Sprintf("%s:%d", cfg.GetMetricsListenAddress(), cfg.GetMetricsPort())

				// Start metrics server in a goroutine
				go func() {
					log.Info("Starting metrics HTTP server on %s%s", metricsAddr, cfg.GetMetricsPath())
					if err := metricsInstance.StartMetricsServer(
						metricsAddr,
						cfg.GetMetricsAuthToken(),
						cfg.GetMetricsIPWhitelist(),
					); err != nil {
						log.Warn("Metrics server error: %v", err)
					}
				}()

				// Register cleanup for graceful shutdown
				signalHandler.RegisterCleanup(func() error {
					log.Info("Shutting down metrics server")
					// Note: metrics server shutdown would need to be implemented separately
					// The current implementation uses ListenAndServe which blocks
					return nil
				})

				log.Info("Metrics server configured - endpoints available at http://%s%s", metricsAddr, cfg.GetMetricsPath())
				if cfg.GetMetricsAuthToken() != "" {
					log.Info("  - Bearer token authentication enabled")
				}
				if len(cfg.GetMetricsIPWhitelist()) > 0 {
					log.Info("  - IP whitelist enabled (%d IPs)", len(cfg.GetMetricsIPWhitelist()))
				}
			}

			// === EXECUTION ===
			log.Info("Starting playbook execution")
			startTime := time.Now()

			// Initialize profiler if enabled
			var profileMgr *profiler.ProfileManager
			if profileEnabled {
				cfg := profiler.Config{
					Enabled:   true,
					OutputDir: profileDir,
					CPU:       profileCPU && !profileTrace, // Trace disables CPU profiling
					Memory:    profileMem,
					Trace:     profileTrace,
				}
				var err error
				profileMgr, err = profiler.NewProfileManager(cfg)
				if err != nil {
					log.Warn("Failed to initialize profiler: %v", err)
				} else {
					if err := profileMgr.StartProfiling(); err != nil {
						log.Warn("Failed to start profiling: %v", err)
					} else {
						log.Info("Profiling enabled - output: %s", profileDir)
					}
					// Ensure profiling stops even if execution fails
					defer func() {
						if err := profileMgr.StopProfiling(); err != nil {
							log.Warn("Failed to stop profiling: %v", err)
						}
						if report, err := profileMgr.GenerateReport(); err == nil && report != "" {
							log.Info("Profiling report:\n%s", report)
						}
					}()
				}
			}

			result, err := executionEngine.ExecutePlaybook(ctx, playbook)
			duration := time.Since(startTime)

			// Wait for TUI to finish if it's running (user presses Q to exit)
			if interactive && tuiModel != nil {
				tuiModel.WaitForExit()
			}

			if err != nil {
				if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
					fmt.Fprintf(os.Stderr, "\n⚠️  Playbook execution interrupted by user\n")
					os.Exit(130)
				}
				return fmt.Errorf("playbook execution failed: %w", err)
			}

			// Display results
			log.Info("Playbook execution completed in %v", duration)

			// Record plays and tasks in audit if recorder is available
			if auditRecorder != nil && executionID != "" {
				recordAuditResults(auditRecorder, result, log)
			}

			// === PHASE 2A: Cache Execution Results ===
			// Save results to cache for later retrieval without re-execution
			if homeDir != "" {
				cacheDir := filepath.Join(homeDir, ".onigirazu", "cache", "executions")
				cacheMgr, err := execution.NewCacheManagerWithPath(cacheDir)
				if err == nil {
					// Use pre-calculated task counts from PlaybookResult
					totalSuccess := result.SuccessTasks
					totalFailed := result.FailedTasks
					totalChanged := result.ChangedTasks
					totalSkipped := result.SkippedTasks

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

			// Display results if not in interactive mode (interactive mode handles its own display)
			if !interactive {
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
			}

			// Prepare execution summary
			// Use progress tracker stats if available (more accurate), otherwise use result fields
			summary := logger.ExecutionSummary{
				TotalDuration: duration,
				PlayCount:     len(result.Plays),
				Stats:         result.Stats,
				HostResults:   make(map[string]logger.HostResultSummary),
			}

			// Get task counts from progress tracker if available
			trackerStats := progressTracker.GetStats()
			if trackerStats != nil {
				if total, ok := trackerStats["total"].(int); ok {
					summary.TaskCount = total
				}
				if completed, ok := trackerStats["completed"].(int); ok {
					summary.SuccessCount = completed
				}
				if failed, ok := trackerStats["failed"].(int); ok {
					summary.FailedCount = failed
				}
				if skipped, ok := trackerStats["skipped"].(int); ok {
					summary.SkippedCount = skipped
				}
			} else {
				// Fallback to result fields
				summary.TaskCount = result.TotalTasks
				summary.SuccessCount = result.SuccessTasks
				summary.FailedCount = result.FailedTasks
				summary.SkippedCount = result.SkippedTasks
			}

			// Calculate changed count from result
			summary.ChangedCount = result.ChangedTasks

			// Populate per-host breakdown from plays
			if result.Plays != nil {
				for _, play := range result.Plays {
					if len(play.Hosts) > 0 {
						for _, hostResult := range play.Hosts {
							hostname := hostResult.Host
							existing, found := summary.HostResults[hostname]

							// Count task results for this host
							tasksOK := 0
							tasksChanged := 0
							tasksFailed := 0
							tasksSkipped := 0

							for _, task := range hostResult.Tasks {
								if task.Skipped {
									tasksSkipped++
								} else if task.Failed {
									tasksFailed++
								} else if task.Changed {
									tasksChanged++
								} else if task.Success {
									tasksOK++
								}
							}

							// Determine overall status
							hostStatus := "OK"
							if tasksFailed > 0 {
								hostStatus = "FAILED"
							} else if tasksChanged > 0 {
								hostStatus = "CHANGED"
							}

							// Accumulate counts for hosts that appear in multiple plays
							if found {
								existing.TasksOK += tasksOK
								existing.TasksChanged += tasksChanged
								existing.TasksFailed += tasksFailed
								existing.TasksSkipped += tasksSkipped
								// Update status if this play shows failure
								if hostStatus == "FAILED" {
									existing.Status = "FAILED"
								} else if hostStatus == "CHANGED" && existing.Status != "FAILED" {
									existing.Status = "CHANGED"
								}
								summary.HostResults[hostname] = existing
							} else {
								summary.HostResults[hostname] = logger.HostResultSummary{
									Hostname:     hostname,
									TasksOK:      tasksOK,
									TasksChanged: tasksChanged,
									TasksFailed:  tasksFailed,
									TasksSkipped: tasksSkipped,
									Status:       hostStatus,
								}
							}
						}
					}
				}
			}

			if result.Failed {
				log.Error("Playbook execution failed")
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
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive mode with beautiful TUI")
	cmd.Flags().StringVar(&tags, "tags", "", "Only run tasks with these tags (comma-separated). Use 'tagged' for tasks with any tag, 'untagged' for tasks without tags, 'all' for default behavior")
	cmd.Flags().StringVar(&skipTags, "skip-tags", "", "Skip tasks with these tags (comma-separated)")
	cmd.Flags().BoolVar(&listTags, "list-tags", false, "List all available tags in the playbook without executing")
	cmd.Flags().BoolVar(&listTasks, "list-tasks", false, "List tasks that would execute with current filters")
	cmd.Flags().BoolVar(&verboseOutput, "verbose-output", false, "Use verbose output formatting (more details)")
	cmd.Flags().BoolVar(&backgroundMode, "background", false, "Run in background mode (returns immediately, use show-execution to view results)")

	// Inventory flags
	cmd.Flags().BoolVar(&lenient, "lenient", false, "Lenient mode: skip inventory validation errors and process what is valid")

	// Profiling flags
	cmd.Flags().BoolVar(&profileEnabled, "profile", false, "Enable profiling (CPU, memory, trace)")
	cmd.Flags().StringVar(&profileDir, "profile-dir", ".onigirazu/profiles", "Output directory for profile files")
	cmd.Flags().BoolVar(&profileCPU, "profile-cpu", true, "Profile CPU usage (enabled by default with --profile)")
	cmd.Flags().BoolVar(&profileMem, "profile-mem", true, "Profile memory usage (enabled by default with --profile)")
	cmd.Flags().BoolVar(&profileTrace, "profile-trace", false, "Generate execution trace (disables CPU profiling)")

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

// recordAuditResults records playbook execution results in the audit log
func recordAuditResults(recorder *audit.Recorder, result *types.PlaybookResult, log interface{}) {
	if recorder == nil || result == nil {
		return
	}

	// Get a logger for error messages
	logger, ok := log.(interface {
		Warn(format string, args ...interface{})
	})
	if !ok {
		return
	}

	// Record each play
	for playIndex, play := range result.Plays {
		// Get unique hosts for this play
		hosts := make([]string, 0)
		hostMap := make(map[string]bool)

		for _, task := range play.Tasks {
			if _, seen := hostMap[task.Host]; !seen && task.Host != "" {
				hosts = append(hosts, task.Host)
				hostMap[task.Host] = true
			}
		}

		// Record play execution
		playRecorder := recorder.RecordPlay(play.Name, playIndex, hosts)
		if playRecorder == nil {
			logger.Warn("Failed to create play recorder for play %d: %s", playIndex, play.PlayName)
			continue
		}

		// Record each task in the play
		for _, task := range play.Tasks {
			// Convert task status
			var taskStatus audit.TaskStatus
			if task.Failed {
				taskStatus = audit.TaskStatusFailed
			} else if task.Skipped {
				taskStatus = audit.TaskStatusSkipped
			} else if task.Changed {
				taskStatus = audit.TaskStatusChanged
			} else {
				taskStatus = audit.TaskStatusOk
			}

			// Create audit task result
			auditTaskResult := audit.TaskResult{
				Name:       task.TaskName,
				Module:     task.Module,
				Status:     taskStatus,
				Host:       task.Host,
				Duration:   task.Duration.Seconds(),
				StartTime:  task.Timestamp,
				EndTime:    task.Timestamp.Add(task.Duration),
				Changed:    task.Changed,
				ReturnCode: 0,
				Error:      task.Error,
			}

			// Add output as JSON string if available
			if task.Output != nil {
				if jsonBytes, err := json.Marshal(task.Output); err == nil {
					auditTaskResult.Output = string(jsonBytes)
				}
			}

			// Record the task result
			if err := recorder.RecordTaskResult(auditTaskResult); err != nil {
				logger.Warn("Failed to record task result for %s on %s: %v", task.TaskName, task.Host, err)
			}
		}

		// Mark play as complete
		playStatus := audit.StatusSuccess
		if !play.Success {
			playStatus = audit.StatusFailure
		}
		playRecorder.Complete(playStatus)
	}
}
