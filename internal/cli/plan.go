package cli

import (
	"github.com/spf13/cobra"
)

// newPlanCmd creates the plan command
func newPlanCmd() *cobra.Command {
	var o driftCheckOptions
	cmd := &cobra.Command{
		Use:   "plan PLAYBOOK",
		Short: "Show what apply would change on the hosts",
		Long: `Run the playbook in check mode against the hosts and show, per host, every task
that would change something, with diffs of files. Nothing is changed.

plan is drift for a change you are about to make: the same report, exit code 0
when there are changes (1 when a task cannot be checked).`,
		Example: `  onigirazu plan site.yml -i hosts.yml
  onigirazu plan site.yml -i hosts.yml --limit web --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.plan = true
			return runDriftCheck(cmd, args[0], o)
		},
	}
	cmd.Flags().StringVar(&o.format, "format", "text", "Report format (text, json)")
	cmd.Flags().StringVar(&o.output, "output", "", "Write the report to this file")
	cmd.Flags().StringArrayVarP(&o.extraVars, "extra-vars", "e", nil, "Extra variables, as for apply (repeatable)")
	cmd.Flags().StringVar(&o.limit, "limit", "", "Plan only for hosts matching this pattern")
	cmd.Flags().StringVar(&o.tags, "tags", "", "Plan only tasks with these tags")
	cmd.Flags().StringVar(&o.skipTags, "skip-tags", "", "Skip tasks with these tags")
	cmd.Flags().BoolVarP(&o.become, "become", "b", false, "Use privilege escalation in every play")
	cmd.Flags().StringVar(&o.becomeUser, "become-user", "", "User to become")
	cmd.Flags().StringVarP(&o.user, "user", "u", "", "SSH user for every host")
	cmd.Flags().StringVar(&o.privateKey, "private-key", "", "SSH private key for every host")
	return cmd
}
