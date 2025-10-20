package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/version"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	var short bool
	var showModules bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the version, build date, and commit information for Onigirazu.",
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Println(version.GetVersion())
			} else {
				fmt.Println(version.GetFullVersion())

				// Show modules if requested or by default (not in short mode)
				if !short || showModules {
					registry := modules.NewRegistry()
					moduleInfo := registry.GetModuleInfo()
					fmt.Print(version.FormatModulesList(moduleInfo))
				}
			}
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "Show only version number")
	cmd.Flags().BoolVar(&showModules, "modules", false, "Show available modules")

	return cmd
}
