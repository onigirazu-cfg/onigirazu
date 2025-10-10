package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/version"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the version, build date, and commit information for Onigirazu.",
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Println(version.GetVersion())
			} else {
				fmt.Println(version.GetFullVersion())
			}
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "Show only version number")

	return cmd
}
