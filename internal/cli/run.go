package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/adhoc"
	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// newRunCmd creates the run command for ad-hoc operations
func newRunCmd() *cobra.Command {
	var (
		// Module specification (Ansible-like)
		moduleName string
		moduleArgs []string

		// Execution options
		check    bool
		diff     bool
		timeout  time.Duration
		parallel int

		// Output options
		output      string
		verboseMode bool

		// Variables
		extraVars map[string]string

		// SSH connection options
		sshUser    string
		sshKeyFile string

		// Inventory options
		lenient bool
	)

	cmd := &cobra.Command{
		Use:   "run [host-pattern] [command]",
		Short: "Execute ad-hoc commands on target hosts",
		Long: `Execute ad-hoc commands on target hosts without creating a playbook.

This command supports multiple syntax formats:

1. Simple shell command (easiest):
   onigirazu run all "uptime"
   onigirazu run webservers "hostname"
   onigirazu run all "df -h"

2. Ansible-like syntax (explicit module):
   onigirazu run all -m ping
   onigirazu run webservers -m package name=nginx state=present
   onigirazu run all -m command command="uptime"

3. Natural language (auto-detected):
   onigirazu run all "install nginx package"
   onigirazu run webservers "start nginx service"
   onigirazu run all "create file /tmp/test.txt"

4. Module:args syntax:
   onigirazu run all "package:name=nginx,state=present"
   onigirazu run all "service:name=nginx,state=started"

5. JSON syntax:
   onigirazu run all '{"module":"package","args":{"name":"nginx","state":"present"}}'

Examples:
  # Simple shell commands (recommended for quick tasks)
  onigirazu run all "uptime" -i inventory.yml
  onigirazu run webservers "systemctl status nginx" -i inventory.yml

  # Ping all hosts
  onigirazu run all -m ping -i inventory.yml

  # Install nginx on webservers
  onigirazu run webservers -m package name=nginx state=present -i inventory.yml

  # Natural language
  onigirazu run all "install nginx package" -i inventory.yml

  # Check mode (dry-run)
  onigirazu run all -m package name=nginx state=present --check -i inventory.yml

  # Parallel execution
  onigirazu run all "uptime" --parallel 10 -i inventory.yml

  # JSON output
  onigirazu run all -m ping --output json -i inventory.yml

  # Override SSH user and key file
  onigirazu run all -m ping -u deploy -k ~/.ssh/id_rsa -i inventory.yml`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse arguments
			hostPattern := args[0]
			var commandStr string

			// If module name is provided (-m flag), additional args are module arguments
			if moduleName != "" {
				// All remaining args after host pattern are module arguments
				if len(args) > 1 {
					moduleArgs = append(moduleArgs, args[1:]...)
				}
			} else {
				// No module flag - second arg is command string
				if len(args) > 1 {
					commandStr = args[1]
				}
			}

			// Validate: either module name or command string must be provided
			if moduleName == "" && commandStr == "" {
				return fmt.Errorf("either -m/--module or command string must be provided")
			}

			return runAdHocCommand(
				hostPattern,
				moduleName,
				moduleArgs,
				commandStr,
				check,
				diff,
				timeout,
				parallel,
				output,
				verboseMode,
				extraVars,
				sshUser,
				sshKeyFile,
				lenient,
			)
		},
	}

	// Module specification flags
	cmd.Flags().StringVarP(&moduleName, "module", "m", "", "Module name (Ansible-like syntax)")
	cmd.Flags().StringArrayVarP(&moduleArgs, "args", "a", []string{}, "Module arguments (key=value)")

	// Execution option flags
	cmd.Flags().BoolVar(&check, "check", false, "Check mode (dry-run, don't make changes)")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show differences when changing files")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Execution timeout per host")
	cmd.Flags().IntVarP(&parallel, "parallel", "f", 10, "Number of parallel executions")

	// Output option flags
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format (text, json, yaml, table)")
	cmd.Flags().BoolVarP(&verboseMode, "verbose-mode", "V", false, "Verbose output (show detailed results)")

	// Variable flags
	cmd.Flags().StringToStringVarP(&extraVars, "extra-vars", "e", map[string]string{}, "Extra variables (key=value)")

	// SSH connection flags
	cmd.Flags().StringVarP(&sshUser, "user", "u", "", "SSH user (overrides inventory)")
	cmd.Flags().StringVarP(&sshKeyFile, "key-file", "k", "", "SSH private key file (overrides inventory)")

	// Inventory flags
	cmd.Flags().BoolVar(&lenient, "lenient", false, "Lenient mode: skip inventory validation errors and process what is valid")

	return cmd
}

// runAdHocCommand executes an ad-hoc command
func runAdHocCommand(
	hostPattern string,
	moduleName string,
	moduleArgs []string,
	commandStr string,
	check bool,
	diff bool,
	timeout time.Duration,
	parallel int,
	outputFormat string,
	verboseMode bool,
	extraVars map[string]string,
	sshUser string,
	sshKeyFile string,
	lenient bool,
) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger with combined verbose flags
	shouldBeVerbose := verbose || verboseMode || showDebug
	log := logger.New(shouldBeVerbose)

	// Initialize SSH connection pool with logger
	sshpkg.InitializeGlobalPoolWithLogger(cfg, log)

	// Load inventory
	if inventoryPath == "" {
		return fmt.Errorf("inventory file is required (use -i/--inventory)")
	}

	// Initialize required components for inventory manager
	templateEngine := template.NewEngine()
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)
	cacheManager := cache.NewManager(5 * time.Minute) // 5 minute cache TTL

	// Enable lenient mode if requested
	if lenient {
		enhancedParser.SetLenient(true)
	}

	// Create inventory manager
	inventoryMgr := inventory.NewManager(enhancedParser, log, cacheManager)

	// Enable lenient mode in manager if requested
	if lenient {
		inventoryMgr.SetLenient(true)
	}

	// Create context with graceful shutdown support
	ctx := context.Background()
	signalHandler := execution.NewSignalHandler(ctx, 10*time.Second)
	defer signalHandler.Close()

	// Use the signal handler's context for execution
	ctx = signalHandler.Context()

	// Register SSH pool cleanup
	sshPool := sshpkg.GetGlobalPool()
	signalHandler.RegisterCleanup(func() error {
		return sshPool.CloseAll()
	})

	// Load inventory file
	if err := inventoryMgr.LoadInventory(ctx, inventoryPath); err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}

	// Apply SSH overrides if provided
	if sshUser != "" || sshKeyFile != "" {
		inventoryMgr.ApplyHostOverrides(sshUser, sshKeyFile)
	}

	// Initialize module registry
	moduleRegistry := modules.NewRegistry()

	// Initialize ad-hoc components
	adhocParser := adhoc.NewParser()
	adhocExecutor := adhoc.NewExecutor(moduleRegistry, inventoryMgr, log)
	formatter := adhoc.NewFormatter(noColor)

	// Parse command
	var command *adhoc.Command
	if moduleName != "" {
		// Ansible-like syntax
		command, err = adhocParser.Parse("", moduleName, moduleArgs)
	} else {
		// Auto-detect format
		command, err = adhocParser.Parse(commandStr, "", nil)
	}
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	// Show what we're about to execute
	if !noColor {
		fmt.Fprintf(os.Stderr, "%s\n", utils.Colors.Info(fmt.Sprintf("🍙 Executing: %s on %s", command.Module, hostPattern)))
	} else {
		fmt.Fprintf(os.Stderr, "Executing: %s on %s\n", command.Module, hostPattern)
	}

	// Convert extra vars to interface map
	variables := make(map[string]interface{})
	for k, v := range extraVars {
		variables[k] = v
	}

	// Prepare options
	opts := adhoc.Options{
		Check:     check,
		Diff:      diff,
		Timeout:   timeout,
		Parallel:  parallel,
		Output:    outputFormat,
		Verbose:   verboseMode,
		NoColor:   noColor,
		Variables: variables,
	}

	// Execute command with signal handling
	summary, err := adhocExecutor.Execute(ctx, command, hostPattern, opts)
	if err != nil {
		// Check if error is due to context cancellation
		if err == context.Canceled {
			fmt.Fprintf(os.Stderr, "\n⚠️  Command execution interrupted by user\n")
			os.Exit(130) // Standard exit code for SIGINT
		}
		return fmt.Errorf("execution failed: %w", err)
	}

	// Format and print results
	output, err := formatter.Format(summary, outputFormat)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(output)

	// Exit with error if any host failed
	if summary.Failed > 0 {
		os.Exit(1)
	}

	return nil
}
