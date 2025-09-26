package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/progress"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

func main() {
	var (
		playbookPath  = flag.String("playbook", "", "Path to playbook file")
		inventoryPath = flag.String("inventory", "", "Path to inventory file")
		configPath    = flag.String("config", "", "Path to configuration file")
		verbose       = flag.Bool("verbose", false, "Verbose output")
		check         = flag.Bool("check", false, "Check mode (dry-run)")
		diff          = flag.Bool("diff", false, "Show differences when changing files")
		dryRun        = flag.Bool("dry-run", false, "Dry run mode (alias for --check)")
		stateFile     = flag.String("state", ".onigirazu-state", "State file for saving state")
		logLevel      = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		logFormat     = flag.String("log-format", "text", "Log format (text, json)")
		outputFormat  = flag.String("output", "text", "Output format (text, json, yaml)")
		maxWorkers    = flag.Int("max-workers", 10, "Maximum number of worker threads")
		timeout       = flag.Duration("timeout", 30*time.Minute, "Execution timeout")
		noColor       = flag.Bool("no-color", false, "Disable colored output")
		interactive   = flag.Bool("interactive", false, "Interactive mode")
		listModules   = flag.Bool("list-modules", false, "List available modules and exit")
		version       = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Println(version.GetFullVersion())
		fmt.Println("Configuration Management Tool inspired by Ansible")
		os.Exit(0)
	}

	// Handle list-modules flag
	if *listModules {
		moduleRegistry := modules.NewRegistry()
		fmt.Println("Available modules:")
		for _, name := range moduleRegistry.ListModules() {
			module, _ := moduleRegistry.GetModule(name)
			fmt.Printf("  %-12s - %s\n", name, module.GetDescription())
		}
		os.Exit(0)
	}

	if *playbookPath == "" {
		fmt.Println(utils.Colors.Header("Onigirazu Configuration Management Tool"))
		fmt.Println()
		fmt.Println("Usage: onigirazu -playbook <path> [-inventory <path>] [-config <path>] [options]")
		fmt.Println()
		fmt.Println(utils.Colors.Bold("Options:"))
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Configure colors
	if *noColor {
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
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags
	if *verbose {
		cfg.LogLevel = "debug"
		cfg.Verbose = true
	}
	if *logLevel != "info" {
		cfg.LogLevel = *logLevel
	}
	if *logFormat != "text" {
		cfg.LogFormat = *logFormat
	}
	if *outputFormat != "text" {
		cfg.OutputFormat = *outputFormat
	}
	if *maxWorkers != 10 {
		cfg.MaxConcurrency = *maxWorkers
	}
	if *stateFile != ".onigirazu-state" {
		cfg.StateFile = *stateFile
	}
	if *check || *dryRun {
		cfg.CheckMode = true
		cfg.DryRun = true
	}
	if *diff {
		cfg.ShowDiff = true
	}
	if *noColor {
		cfg.ColorOutput = false
	}
	if *interactive {
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

	// Initialize components
	cacheManager := cache.NewManager(5 * time.Minute) // Default TTL of 5 minutes
	templateEngine := template.NewEngine()
	stateManager := state.NewEnhancedManager(cfg.StateFile, log)
	executionPool := execution.NewPool(cfg.MaxConcurrency, log)
	progressTracker := progress.NewTracker()
	moduleRegistry := modules.NewRegistry()

	// Initialize parser and inventory manager
	playbookParser := parser.NewEnhancedParser(templateEngine, log)
	inventoryManager := inventory.NewManager(playbookParser, log, cacheManager)

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
	if *timeout > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, *timeout)
		defer timeoutCancel()
	}

	// Load inventory if specified
	if *inventoryPath != "" {
		log.Info("Loading inventory from: %s", *inventoryPath)
		if err := inventoryManager.LoadInventory(ctx, *inventoryPath); err != nil {
			log.Error("Failed to load inventory: %v", err)
			os.Exit(1)
		}
	}

	// Parse playbook
	log.Info("Parsing playbook: %s", *playbookPath)
	playbook, err := playbookParser.ParsePlaybook(ctx, *playbookPath)
	if err != nil {
		log.Error("Failed to parse playbook: %v", err)
		os.Exit(1)
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
		log.Error("Playbook execution failed: %v", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)

	// Display results
	log.Info("Playbook execution completed in %v", duration)

	if result.Failed {
		log.Error("Playbook execution failed")
		displayExecutionSummary(log, result)
		os.Exit(1)
	}

	log.Info("Playbook execution successful")
	displayExecutionSummary(log, result)

	// Save final state
	if err := stateManager.SaveCurrentState(); err != nil {
		log.Warn("Failed to save final state: %v", err)
	}

	log.Info("Onigirazu execution completed successfully")
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
