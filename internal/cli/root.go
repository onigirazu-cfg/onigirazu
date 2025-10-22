package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/version"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

var (
	// Global flags
	configPath    string
	inventoryPath string
	statePath     string
	verbose       bool
	noColor       bool
	showDebug     bool

	// Legacy flags for backward compatibility
	playbookPath string
)

// NewRootCommand creates the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "onigirazu",
		Short: "Onigirazu Configuration Management Tool",
		Long: `🍙 Onigirazu - A modern configuration management tool inspired by Ansible.

Onigirazu provides a simple, powerful way to automate configuration management
across your infrastructure with a focus on simplicity and reliability.`,
		Version: version.GetVersion(),
		Run: func(cmd *cobra.Command, args []string) {
			// Handle legacy mode: if --playbook is specified without subcommand
			if playbookPath != "" {
				handleLegacyMode(cmd, args)
				return
			}

			// Show help if no subcommand is provided
			_ = cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&inventoryPath, "inventory", "i", "", "Path to inventory file")
	rootCmd.PersistentFlags().StringVarP(&statePath, "state", "s", ".onigirazu-state", "Path to state file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&showDebug, "show-debug", false, "Show debug and info messages")

	// Legacy flags for backward compatibility (hidden from help)
	rootCmd.PersistentFlags().StringVarP(&playbookPath, "playbook", "p", "", "Path to playbook file (legacy)")
	_ = rootCmd.PersistentFlags().MarkHidden("playbook")

	// Additional legacy flags
	var legacyCheck, legacyDiff, legacyDryRun bool
	rootCmd.PersistentFlags().BoolVarP(&legacyCheck, "check", "C", false, "Check mode (legacy)")
	rootCmd.PersistentFlags().BoolVarP(&legacyDiff, "diff", "d", false, "Show differences (legacy)")
	rootCmd.PersistentFlags().BoolVar(&legacyDryRun, "dry-run", false, "Dry run mode (legacy)")
	_ = rootCmd.PersistentFlags().MarkHidden("check")
	_ = rootCmd.PersistentFlags().MarkHidden("diff")
	_ = rootCmd.PersistentFlags().MarkHidden("dry-run")

	// Add subcommands
	rootCmd.AddCommand(NewApplyCommand())
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newPlanCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newStateCmd())
	rootCmd.AddCommand(newFmtCmd())
	rootCmd.AddCommand(newLintCmd())
	rootCmd.AddCommand(newGraphCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(newDriftCmd())
	rootCmd.AddCommand(inventoryCmd)
	rootCmd.AddCommand(healthcheckCmd)
	rootCmd.AddCommand(auditCmd)

	// Three-mode execution system commands
	rootCmd.AddCommand(NewShowExecutionCommand())
	rootCmd.AddCommand(NewShowLastExecutionCommand())
	rootCmd.AddCommand(NewListExecutionsCommand())

	return rootCmd
}

// handleLegacyMode handles the legacy --playbook flag for backward compatibility
func handleLegacyMode(cmd *cobra.Command, args []string) {
	// Show warning about legacy syntax
	if !noColor {
		fmt.Fprintf(os.Stderr, "%s\n", utils.Colors.Warning("⚠️  Using legacy syntax. Consider using: onigirazu apply "+playbookPath))
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Using legacy syntax. Consider using: onigirazu apply %s\n", playbookPath)
	}

	// Execute apply command with legacy flags
	applyCmd := NewApplyCommand()

	// Copy global flags to apply command
	_ = applyCmd.Flags().Set("config", configPath)
	_ = applyCmd.Flags().Set("inventory", inventoryPath)
	_ = applyCmd.Flags().Set("state", statePath)
	_ = applyCmd.Flags().Set("verbose", fmt.Sprintf("%t", verbose))
	_ = applyCmd.Flags().Set("no-color", fmt.Sprintf("%t", noColor))

	// Copy other legacy flags if they exist
	if check, _ := cmd.Flags().GetBool("check"); check {
		_ = applyCmd.Flags().Set("check", "true")
	}
	if diff, _ := cmd.Flags().GetBool("diff"); diff {
		_ = applyCmd.Flags().Set("diff", "true")
	}
	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		_ = applyCmd.Flags().Set("check", "true")
	}

	// Set playbook path as argument
	applyCmd.SetArgs([]string{playbookPath})

	// Execute apply command
	if err := applyCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Execute runs the root command
func Execute() error {
	rootCmd := NewRootCommand()
	return rootCmd.Execute()
}
