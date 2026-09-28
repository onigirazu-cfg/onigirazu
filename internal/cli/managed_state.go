package cli

import (
	"fmt"
	"io"

	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/managed"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// updateManagedState records what the run's tasks manage in the playbook's
// managed state and puts the orphans into the result. In check mode the
// state file is left as it is.
func updateManagedState(playbookPath string, result *types.PlaybookResult, scopes engine.ManagedScopes,
	complete, check bool, log interfaces.Logger) {
	path := managed.Path(playbookPath)
	st, err := managed.Load(path)
	if err != nil {
		log.Warn("Managed state not updated: %v", err)
		return
	}
	if check {
		st = st.Clone()
	}
	orphans := st.Update(managed.Run{Result: result, Scopes: managed.Scopes{Keys: scopes.Keys, Kept: scopes.Kept},
		Complete: complete})
	result.Orphans = nil
	for _, r := range orphans {
		result.Orphans = append(result.Orphans, types.ManagedOrphan{Host: r.Host, Type: r.Type, ID: r.ID,
			Task: r.TaskName, Action: r.Action()})
	}
	if check {
		return
	}
	if err := st.Save(path); err != nil {
		log.Warn("Failed to save managed state %s: %v", path, err)
	}
}

// orphanMarks are the plan signs of the orphan actions
var orphanMarks = map[string]string{
	managed.ActionDestroy: "-", managed.ActionRestore: "<", managed.ActionForget: "?",
}

// printOrphans lists the resources that left the playbook
func printOrphans(w io.Writer, orphans []types.ManagedOrphan) {
	if len(orphans) == 0 {
		return
	}
	fmt.Fprintf(w, "\nNo longer in the playbook: %d resource(s)\n", len(orphans))
	for _, o := range orphans {
		what := map[string]string{
			managed.ActionDestroy: "remove (onigirazu created it)",
			managed.ActionRestore: "put back as it was",
			managed.ActionForget:  "forget (left as it is)",
		}[o.Action]
		fmt.Fprintf(w, "  %s %s %s on %s: %s", orphanMarks[o.Action], o.Type, o.ID, o.Host, what)
		if o.Task != "" {
			fmt.Fprintf(w, " [was: %s]", o.Task)
		}
		fmt.Fprintln(w)
	}
}
