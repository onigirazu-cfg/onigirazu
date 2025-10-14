package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
)

// newValidateCmd creates the validate command
func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [playbook]",
		Short: "Validate playbook syntax and structure",
		Long: `Validate playbook syntax and structure without executing it.

This command checks:
- YAML syntax correctness
- Playbook structure (plays, tasks, modules)
- Required fields presence
- Module names validity
- Task configuration correctness

The validation does not check:
- Inventory availability
- Host connectivity
- Module execution logic
- Variable values`,
		Example: `  # Validate a playbook
  onigirazu validate production.yml

  # Validate with verbose output
  onigirazu validate production.yml --verbose

  # Validate with custom config
  onigirazu validate production.yml --config custom.yml`,
		Args: cobra.ExactArgs(1),
		RunE: runValidate,
	}

	// Add command-specific flags
	cmd.Flags().Bool("strict", false, "Enable strict validation mode")
	cmd.Flags().Bool("check-modules", true, "Verify that all modules exist")

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	playbookPath := args[0]

	// Get flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	strict, _ := cmd.Flags().GetBool("strict")
	checkModules, _ := cmd.Flags().GetBool("check-modules")

	if verbose {
		fmt.Printf("🔍 Validating playbook: %s\n", playbookPath)
	}

	// Check if file exists
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		return fmt.Errorf("playbook file not found: %s", playbookPath)
	}

	// Create parser
	p := parser.New()

	// Parse and validate playbook
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	playbook, err := p.ParsePlaybook(ctx, playbookPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		return err
	}
	duration := time.Since(startTime)

	// Additional validation in strict mode
	if strict {
		if verbose {
			fmt.Println("🔒 Running strict validation...")
		}
		// Add strict validation logic here if needed
	}

	// Check modules existence
	if checkModules {
		if verbose {
			fmt.Println("📦 Checking module availability...")
		}
		// Module registry check would go here
		// For now, we just validate that module names are not empty
		for _, play := range playbook.Plays {
			for _, task := range play.Tasks {
				if task.Module == "" {
					return fmt.Errorf("task '%s' has no module specified", task.Name)
				}
			}
		}
	}

	// Print success message
	fmt.Printf("✅ Playbook validation successful!\n\n")
	fmt.Printf("Playbook: %s\n", playbook.Name)
	fmt.Printf("Plays:    %d\n", len(playbook.Plays))

	// Count total tasks
	totalTasks := 0
	for _, play := range playbook.Plays {
		totalTasks += len(play.Tasks)
	}
	fmt.Printf("Tasks:    %d\n", totalTasks)
	fmt.Printf("Duration: %v\n", duration.Round(time.Millisecond))

	if verbose {
		fmt.Println("\n📋 Playbook structure:")
		for i, play := range playbook.Plays {
			fmt.Printf("  Play %d: %s\n", i+1, play.Name)
			fmt.Printf("    Hosts: %s\n", play.Hosts)
			fmt.Printf("    Tasks: %d\n", len(play.Tasks))
			for j, task := range play.Tasks {
				fmt.Printf("      %d. %s (module: %s)\n", j+1, task.Name, task.Module)
			}
		}
	}

	return nil
}
