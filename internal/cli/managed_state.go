package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/managed"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// managedRun is the managed state of a run, between the update after the
// plays and the destroy step
type managedRun struct {
	store           managed.Store
	st              *managed.State
	check, complete bool
	// untargeted: hosts no play matches any more
	untargeted map[string]bool
}

// updateManagedState records what the run's tasks manage in the playbook's
// managed state and puts the orphans into the result. In check mode the
// state file is left as it is.
func updateManagedState(ctx context.Context, store managed.Store, result *types.PlaybookResult, scopes engine.ManagedScopes,
	complete, allHosts, check bool, log interfaces.Logger) *managedRun {
	if store == nil {
		return nil
	}
	st, err := store.Load(ctx)
	if err != nil {
		log.Warn("Managed state not updated: %v", err)
		return nil
	}
	if check {
		st = st.Clone()
	}
	run := managed.Run{Result: result, Complete: complete, AllHosts: allHosts,
		Scopes: managed.Scopes{Keys: scopes.Keys, Kept: scopes.Kept, Hosts: scopes.Hosts, AllPlays: scopes.AllPlays}}
	untargeted := st.Untargeted(run)
	orphans := st.Update(run)
	result.Orphans = nil
	for _, r := range orphans {
		result.Orphans = append(result.Orphans, types.ManagedOrphan{Host: r.Host, Type: r.Type, ID: r.ID,
			Task: r.TaskName, Action: r.Action()})
	}
	m := &managedRun{store: store, st: st, check: check, complete: complete, untargeted: untargeted}
	if !check {
		m.save(ctx, log)
	}
	return m
}

func (m *managedRun) save(ctx context.Context, log interfaces.Logger) {
	if err := m.store.Save(ctx, m.st); err != nil {
		log.Warn("Failed to save managed state %s: %v", m.store, err)
	}
}

// managedStore is the managed state of a playbook, where the config puts it
func managedStore(cfg *config.Config, playbook string) (managed.Store, error) {
	switch ms := cfg.ManagedState; ms.Backend {
	case "", "file":
		return managed.NewFileStore(playbook), nil
	case "s3":
		return managed.NewS3Store(managed.S3Config{Bucket: ms.Bucket, Prefix: ms.Prefix, Endpoint: ms.Endpoint,
			Region: ms.Region, Insecure: ms.Insecure, PathStyle: ms.PathStyle}, playbook)
	default:
		return nil, fmt.Errorf("managed_state: unknown backend %q (file, s3)", ms.Backend)
	}
}

// playbookStore is the store of a playbook for the state commands: the
// config is looked up next to the playbook, as apply does
func playbookStore(playbook string) (managed.Store, error) {
	cfg, err := config.LoadConfigWithDiscovery("", filepath.Dir(playbook))
	if err != nil {
		return nil, err
	}
	return managedStore(cfg, playbook)
}

// undoer puts one resource back from its before capture
type undoer interface {
	Undo(ctx context.Context, t types.TaskResult) (bool, error)
}

// destroyOrphans removes the orphans onigirazu created, puts back the ones
// it adopted and forgets the rest, on the hosts of the run that had no
// failure. confirm is asked once, with the number of resources the host
// changes touch.
func (m *managedRun) destroyOrphans(ctx context.Context, u undoer, result *types.PlaybookResult,
	known func(host string) bool, confirm func(n int) bool, w io.Writer, log interfaces.Logger) error {
	if m == nil || m.check {
		return nil
	}
	var act, forget []*managed.Record
	gone := map[string]int{}
	for _, r := range m.st.Orphans() {
		switch {
		case m.untargeted[r.Host] && !known(r.Host):
			// not in the inventory: nothing to connect to
			gone[r.Host]++
			continue
		case m.untargeted[r.Host]:
		case !managed.HostRan(result, r.Host) || managed.HostFailed(result, r.Host):
			continue
		}
		if r.Action() == managed.ActionForget {
			forget = append(forget, r)
		} else {
			act = append(act, r)
		}
	}
	for host, n := range gone {
		fmt.Fprintf(w, "\nManaged state: %s is in no play and not in the inventory; its %d resource(s) stay recorded (state rm to forget them)\n", host, n)
	}
	if len(act)+len(forget) == 0 {
		return nil
	}
	if !m.complete {
		fmt.Fprintf(w, "\nManaged state: %d resource(s) left the playbook; a partial run (--tags, --skip-tags, --start-at-task) leaves them, a full apply cleans up\n", len(act)+len(forget))
		return nil
	}
	if len(act) > 0 && !confirm(len(act)) {
		fmt.Fprintf(w, "\nManaged state: %d resource(s) kept for later (plan lists them)\n", len(act))
		act = nil
	}
	managed.SortForDestroy(act)
	var done, kept, failed []string
	for _, r := range act {
		ok, err := u.Undo(ctx, managed.UndoResult(r))
		var keptErr *rollback.KeptError
		switch {
		case errors.As(err, &keptErr):
			kept = append(kept, fmt.Sprintf("%s %s on %s: %s", r.Type, r.ID, r.Host, keptErr.Reason))
		case err != nil:
			failed = append(failed, fmt.Sprintf("%s %s on %s: %v", r.Type, r.ID, r.Host, err))
			continue
		case !ok:
			kept = append(kept, fmt.Sprintf("%s %s on %s: no previous state", r.Type, r.ID, r.Host))
		default:
			verb := "removed"
			if r.Action() == managed.ActionRestore {
				verb = "put back"
			}
			done = append(done, fmt.Sprintf("%s %s %s on %s", verb, r.Type, r.ID, r.Host))
		}
		m.st.Remove(r.Host, r.Type, r.ID)
	}
	for _, r := range forget {
		kept = append(kept, fmt.Sprintf("%s %s on %s: forgotten, left as it is", r.Type, r.ID, r.Host))
		m.st.Remove(r.Host, r.Type, r.ID)
	}
	m.save(ctx, log)
	if len(done)+len(kept)+len(failed) > 0 {
		fmt.Fprintf(w, "\nManaged state: resources that left the playbook\n")
		for _, l := range done {
			fmt.Fprintf(w, "  %s\n", l)
		}
		for _, l := range kept {
			fmt.Fprintf(w, "  kept %s\n", l)
		}
		for _, l := range failed {
			fmt.Fprintf(w, "  failed %s\n", l)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("managed state: %d resource(s) could not be removed or put back", len(failed))
	}
	return nil
}

// confirmDestroy asks once in a terminal; without one (CI, cron) or with
// --auto-approve apply goes ahead, as the plan showed
func confirmDestroy(autoApprove bool) func(int) bool {
	return func(n int) bool {
		if autoApprove || !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
			return true
		}
		fmt.Fprintf(os.Stderr, "\nRemove or put back %d resource(s) that left the playbook? [y/N] ", n)
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		return answer == "y" || answer == "yes"
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
