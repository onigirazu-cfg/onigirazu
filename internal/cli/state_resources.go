package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/managed"
)

// newStateResourcesCmd lists a playbook's managed resources
func newStateResourcesCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "resources PLAYBOOK",
		Short: "List the resources a playbook manages on its hosts",
		Long: `List the managed state of a playbook (.onigirazu/<playbook>.state.json next to it):
every file, package, service, user and group its tasks keep, the tasks that claim it,
and whether onigirazu created it or took it over. Resources no task claims any more
are marked "orphan".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := managed.Load(managed.Path(args[0]))
			if err != nil {
				return err
			}
			if asJSON {
				// the previous content of files stays out of the listing
				type row struct {
					*managed.Record
					Before interface{} `json:"before,omitempty"`
				}
				rows := make([]row, len(st.Resources))
				for i, r := range st.Resources {
					rows[i] = row{Record: r}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}
			if len(st.Resources) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No managed resources for %s\n", args[0])
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "HOST\tTYPE\tID\tORIGIN\tTASK")
			for _, r := range st.Resources {
				task := r.TaskName
				if len(r.Tasks) == 0 {
					task = "orphan: " + r.Action()
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Host, r.Type, r.ID, r.Origin, task)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print JSON")
	return cmd
}

// newStateRmCmd forgets a managed resource
func newStateRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm PLAYBOOK HOST TYPE ID",
		Short: "Forget a managed resource; the host is not touched",
		Example: `  onigirazu state rm site.yml web1 file /etc/app.conf
  onigirazu state rm site.yml web1 package nginx`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := managed.Path(args[0])
			lock, err := managed.AcquireLock(path, "state rm", 0)
			if err != nil {
				return err
			}
			defer func() { _ = lock.Release() }()
			st, err := managed.Load(path)
			if err != nil {
				return err
			}
			host, typ, id := args[1], strings.ToLower(args[2]), args[3]
			if !st.Remove(host, typ, id) {
				return fmt.Errorf("no managed %s %s on %s in %s", typ, id, host, path)
			}
			if err := st.Save(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Forgot %s %s on %s\n", typ, id, host)
			return nil
		},
	}
}

// newStateUnlockCmd removes a lock left by a run that died
func newStateUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock PLAYBOOK LOCK_ID",
		Short: "Remove the managed state lock of a run that is gone",
		Long: `Remove the lock of a playbook's managed state that a crashed or killed apply left
behind. The lock ID is in the error of the apply that found the state locked. Make sure no
apply of the playbook is still running: two at once can lose records.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := managed.ForceUnlock(managed.Path(args[0]), args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unlocked the managed state of %s\n", args[0])
			return nil
		},
	}
}
