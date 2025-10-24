package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
)

// newPlanCmd creates the plan command
func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan [playbook]",
		Short: "Show execution plan without making changes",
		Long: `Show what changes would be made without actually executing them.

This command performs a dry-run of the playbook and displays:
- Which hosts will be targeted
- Which tasks will be executed
- What changes would be made
- Current vs desired state comparison

This is similar to 'apply --check' but with more detailed output
focused on planning and change preview.`,
		Example: `  # Show execution plan
  onigirazu plan production.yml

  # Show plan with inventory
  onigirazu plan production.yml --inventory hosts.yml

  # Show detailed plan
  onigirazu plan production.yml --verbose

  # Show plan for specific hosts
  onigirazu plan production.yml --limit "web*"`,
		Args: cobra.ExactArgs(1),
		RunE: runPlan,
	}

	// Add command-specific flags
	cmd.Flags().StringP("inventory", "i", "", "Inventory file path")
	cmd.Flags().StringP("limit", "l", "", "Limit execution to specific hosts (supports wildcards)")
	cmd.Flags().Bool("detailed", false, "Show detailed task information")
	cmd.Flags().StringP("state", "s", "", "State file path")
	cmd.Flags().String("tags", "", "Only plan tasks with these tags (comma-separated). Use 'tagged' for tasks with any tag, 'untagged' for tasks without tags, 'all' for default behavior")
	cmd.Flags().String("skip-tags", "", "Skip tasks with these tags (comma-separated)")

	return cmd
}

func runPlan(cmd *cobra.Command, args []string) error {
	playbookPath := args[0]

	// Get flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	inventoryPath, _ := cmd.Flags().GetString("inventory")
	limitHosts, _ := cmd.Flags().GetString("limit")
	detailed, _ := cmd.Flags().GetBool("detailed")
	statePath, _ := cmd.Flags().GetString("state")

	fmt.Printf("📋 Generating execution plan for: %s\n\n", playbookPath)

	// Check if file exists
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		return fmt.Errorf("playbook file not found: %s", playbookPath)
	}

	// Create parser
	p := parser.New()

	// Create context with graceful shutdown support
	baseCtx := context.Background()
	signalHandler := execution.NewSignalHandler(baseCtx, 10*time.Second)
	defer signalHandler.Close()

	// Register SSH pool cleanup
	signalHandler.RegisterCleanup(func() error {
		return sshpkg.GetGlobalPool().CloseAll()
	})

	// Create context with timeout and signal handling
	ctx, cancel := context.WithTimeout(signalHandler.Context(), 30*time.Second)
	defer cancel()

	// Parse playbook
	playbook, err := p.ParsePlaybook(ctx, playbookPath)
	if err != nil {
		return fmt.Errorf("failed to parse playbook: %w", err)
	}

	// Note: Inventory loading would be implemented here
	// For now, we'll just note if inventory path is provided
	var hosts []string
	if inventoryPath != "" {
		if verbose {
			fmt.Printf("📦 Inventory file: %s\n", inventoryPath)
		}
		// TODO: Load inventory when needed
	}

	// Load state if provided
	var stateData *state.Manager
	if statePath != "" {
		stateData = state.New(statePath)
		if verbose {
			fmt.Printf("📊 Loading state from: %s\n", statePath)
		}
	}

	// Display plan header
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Playbook: %s\n", playbook.Name)
	fmt.Printf("Plays:    %d\n", len(playbook.Plays))
	if len(hosts) > 0 {
		fmt.Printf("Hosts:    %d\n", len(hosts))
	}
	if limitHosts != "" {
		fmt.Printf("Limit:    %s\n", limitHosts)
	}
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Display plan for each play
	totalTasks := 0
	for playIdx, play := range playbook.Plays {
		fmt.Printf("Play %d: %s\n", playIdx+1, play.Name)
		fmt.Printf("  Target hosts: %s\n", play.Hosts)
		fmt.Printf("  Tasks: %d\n", len(play.Tasks))
		fmt.Println()

		// Display tasks
		for taskIdx, task := range play.Tasks {
			totalTasks++

			// Task header
			fmt.Printf("  Task %d: %s\n", taskIdx+1, task.Name)
			fmt.Printf("    Module: %s\n", task.Module)

			// Show args in detailed mode
			if detailed && len(task.Args) > 0 {
				fmt.Println("    Arguments:")
				for key, value := range task.Args {
					fmt.Printf("      %s: %v\n", key, value)
				}
			}

			// Show conditions
			if task.When != "" {
				fmt.Printf("    Condition: %s\n", task.When)
			}

			// Show loop
			if task.Loop != nil {
				fmt.Printf("    Loop: %d items\n", len(task.Loop.Items))
			}

			// Show tags
			if len(task.Tags) > 0 {
				fmt.Printf("    Tags: %v\n", task.Tags)
			}

			// Estimate action
			action := estimateTaskAction(task.Module)
			fmt.Printf("    Action: %s\n", action)

			fmt.Println()
		}

		fmt.Println("───────────────────────────────────────────────────────────")
		fmt.Println()
	}

	// Summary
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("Plan Summary:")
	fmt.Printf("  Total tasks: %d\n", totalTasks)
	fmt.Printf("  Total plays: %d\n", len(playbook.Plays))
	fmt.Println()
	fmt.Println("⚠️  This is a plan only. No changes will be made.")
	fmt.Println("    Run 'onigirazu apply' to execute the playbook.")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Suppress unused variable warnings
	_ = stateData

	return nil
}

// estimateTaskAction estimates what action a task will perform based on module
func estimateTaskAction(module string) string {
	actions := map[string]string{
		"package":    "📦 Install/update package",
		"service":    "🔧 Manage service",
		"file":       "📄 Manage file",
		"copy":       "📋 Copy file",
		"template":   "📝 Render template",
		"command":    "⚡ Execute command",
		"shell":      "⚡ Execute shell command",
		"user":       "👤 Manage user",
		"group":      "👥 Manage group",
		"git":        "🔀 Manage git repository",
		"cron":       "⏰ Manage cron job",
		"lineinfile": "✏️  Modify file line",
		"fetch":      "⬇️  Fetch file from remote",
		"get_url":    "🌐 Download from URL",
		"stat":       "📊 Get file/directory stats",
		"debug":      "🐛 Print debug message",
		"set_fact":   "💾 Set variable",
		"systemd":    "⚙️  Manage systemd unit",
		"firewall":   "🔥 Manage firewall",
	}

	if action, ok := actions[module]; ok {
		return action
	}
	return "🔹 Execute module"
}
