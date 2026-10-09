package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// pull mode: the host converges itself from a git repository, on a timer.
// `onigirazu pull` clones or updates the repository, runs the playbook
// against this machine (a local connection unless an inventory is given)
// and, with --interval, repeats; `onigirazu pull install` writes a systemd
// timer that does the same. Nothing reaches out to the host: it pulls.

type pullOptions struct {
	repo, branch, playbook, inventory, dir, limit string
	extraVars                                     []string
	interval                                      time.Duration
	driftOnly, onlyOnChange, become               bool
}

func newPullCmd() *cobra.Command {
	o := &pullOptions{}
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Converge this host from a git repository (pull mode)",
		Long: `Clone or update a git repository and run a playbook from it against this
machine. Without an inventory the playbook runs on localhost with a local
connection. With --interval it keeps running; "pull install" writes a systemd
timer. Exit code 2 with --drift-only when something would change.`,
		Example: `  onigirazu pull --repo git@git.example.com:ops/site.git --playbook site.yml
  onigirazu pull --repo https://... --playbook site.yml --interval 30m --only-on-change
  sudo onigirazu pull install --repo ... --playbook site.yml --interval 30m`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.repo == "" {
				return fmt.Errorf("--repo is required")
			}
			return runPull(cmd.Context(), o, cmd.OutOrStdout())
		},
	}
	addPullFlags(cmd, o)
	install := &cobra.Command{
		Use:   "install",
		Short: "Write and start a systemd timer that runs pull with these options",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.repo == "" {
				return fmt.Errorf("--repo is required")
			}
			return installPullTimer(o, cmd.OutOrStdout())
		},
	}
	addPullFlags(install, o)
	cmd.AddCommand(install)
	return cmd
}

func addPullFlags(cmd *cobra.Command, o *pullOptions) {
	f := cmd.Flags()
	f.StringVar(&o.repo, "repo", "", "Git repository URL (required)")
	f.StringVar(&o.branch, "branch", "main", "Branch to follow")
	f.StringVar(&o.playbook, "playbook", "site.yml", "Playbook path inside the repository")
	f.StringVarP(&o.inventory, "inventory", "i", "", "Inventory path inside the repository (default: localhost, local connection)")
	f.StringVar(&o.dir, "dir", "", "Checkout directory (default: ~/.onigirazu/pull/<name of the repository>)")
	f.StringVar(&o.limit, "limit", "", "Run only on hosts matching this pattern (with --inventory)")
	f.StringArrayVarP(&o.extraVars, "extra-vars", "e", nil, "Extra variables, as for apply (repeatable)")
	f.DurationVar(&o.interval, "interval", 0, "Run again every interval (0: once)")
	f.BoolVar(&o.driftOnly, "drift-only", false, "Check mode: report what would change, change nothing (exit 2 on drift)")
	f.BoolVar(&o.onlyOnChange, "only-on-change", false, "Run the playbook only when the repository changed since the last run")
	f.BoolVarP(&o.become, "become", "b", false, "Use privilege escalation in every play")
}

// pullDir is the checkout directory: --dir, or ~/.onigirazu/pull/<repo name>
func pullDir(o *pullOptions) (string, error) {
	if o.dir != "" {
		return o.dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	name := strings.TrimSuffix(filepath.Base(strings.TrimRight(o.repo, "/")), ".git")
	if name == "" || name == "." {
		name = "repo"
	}
	return filepath.Join(home, ".onigirazu", "pull", name), nil
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// syncRepo clones the repository into dir or brings it to the branch's tip;
// it returns the commit and whether it differs from before
func syncRepo(ctx context.Context, o *pullOptions, dir string) (commit string, changed bool, err error) {
	before := ""
	if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr != nil {
		if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
			return "", false, err
		}
		if _, err := git(ctx, "", "clone", "--quiet", "--depth", "1", "--branch", o.branch, o.repo, dir); err != nil {
			return "", false, err
		}
	} else {
		before, _ = git(ctx, dir, "rev-parse", "HEAD")
		if _, err := git(ctx, dir, "fetch", "--quiet", "--depth", "1", "origin", o.branch); err != nil {
			return "", false, err
		}
		if _, err := git(ctx, dir, "reset", "--quiet", "--hard", "FETCH_HEAD"); err != nil {
			return "", false, err
		}
	}
	commit, err = git(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return "", false, err
	}
	return commit, commit != before, nil
}

// localInventory is the inventory of a pull without one: this machine
const localInventory = "all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"

// pullApplyArgs are the apply arguments of one pull run
func pullApplyArgs(o *pullOptions, dir string) []string {
	args := []string{filepath.Join(dir, o.playbook)}
	if o.driftOnly {
		args = append(args, "--check")
	}
	if o.limit != "" {
		args = append(args, "--limit", o.limit)
	}
	if o.become {
		args = append(args, "--become")
	}
	for _, v := range o.extraVars {
		args = append(args, "-e", v)
	}
	return args
}

func runPull(ctx context.Context, o *pullOptions, out interface{ Write([]byte) (int, error) }) error {
	dir, err := pullDir(o)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for {
		rc, err := pullOnce(ctx, o, dir, out)
		if err != nil {
			fmt.Fprintf(out, "pull: %v\n", err)
		}
		if o.interval <= 0 {
			if err != nil {
				return &ExitError{Code: 1}
			}
			if rc != 0 {
				return &ExitError{Code: rc}
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(o.interval):
		}
	}
}

// pullOnce syncs the repository and runs the playbook; rc is 2 when
// --drift-only found a change, 1 on a failed task, 0 otherwise
func pullOnce(ctx context.Context, o *pullOptions, dir string, out interface{ Write([]byte) (int, error) }) (int, error) {
	commit, changed, err := syncRepo(ctx, o, dir)
	if err != nil {
		return 1, err
	}
	stamp := filepath.Join(dir, ".onigirazu-pull-last")
	last, _ := os.ReadFile(stamp)
	if o.onlyOnChange && !changed && strings.TrimSpace(string(last)) == commit {
		fmt.Fprintf(out, "pull: %s unchanged at %s, nothing to do\n", o.branch, commit[:8])
		return 0, nil
	}
	inventory := o.inventory
	if inventory == "" {
		inventory = filepath.Join(dir, ".onigirazu-pull-inventory.yml")
		if err := os.WriteFile(inventory, []byte(localInventory), 0o600); err != nil {
			return 1, err
		}
	} else if !filepath.IsAbs(inventory) {
		inventory = filepath.Join(dir, inventory)
	}
	fmt.Fprintf(out, "pull: %s at %s, running %s\n", o.branch, commit[:8], o.playbook)
	// the inventory and the state file are global flags of the root command:
	// this run's are the checkout's
	inventoryPaths = []string{inventory}
	if statePath == "" || statePath == ".onigirazu-state" {
		statePath = filepath.Join(dir, ".onigirazu-state")
	}
	result, err := runPlaybook(pullApplyArgs(o, dir))
	if err != nil {
		return 1, err
	}
	_ = os.WriteFile(stamp, []byte(commit+"\n"), 0o600)
	changedTasks, failed := 0, 0
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			for _, t := range host.Tasks {
				if t.Failed && !t.Ignored {
					failed++
				} else if t.Changed && !t.Skipped {
					changedTasks++
				}
			}
		}
	}
	switch {
	case failed > 0:
		fmt.Fprintf(out, "pull: %d task(s) failed\n", failed)
		return 1, nil
	case o.driftOnly && changedTasks > 0:
		fmt.Fprintf(out, "pull: drift: %d task(s) would change\n", changedTasks)
		return 2, nil
	}
	fmt.Fprintf(out, "pull: ok, %d task(s) changed\n", changedTasks)
	return 0, nil
}

// pullUnitFiles are the systemd service and timer of `pull install`
func pullUnitFiles(o *pullOptions, self string) (service, timer string) {
	args := []string{"pull", "--repo", o.repo, "--branch", o.branch, "--playbook", o.playbook}
	if o.inventory != "" {
		args = append(args, "--inventory", o.inventory)
	}
	if o.dir != "" {
		args = append(args, "--dir", o.dir)
	}
	if o.limit != "" {
		args = append(args, "--limit", o.limit)
	}
	for _, v := range o.extraVars {
		args = append(args, "--extra-vars", v)
	}
	if o.driftOnly {
		args = append(args, "--drift-only")
	}
	if o.onlyOnChange {
		args = append(args, "--only-on-change")
	}
	if o.become {
		args = append(args, "--become")
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	every := o.interval
	if every <= 0 {
		every = 30 * time.Minute
	}
	service = fmt.Sprintf(`[Unit]
Description=onigirazu pull: converge this host from %s
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%s %s
`, o.repo, self, strings.Join(quoted, " "))
	timer = fmt.Sprintf(`[Unit]
Description=onigirazu pull every %s

[Timer]
OnBootSec=2min
OnUnitActiveSec=%s
RandomizedDelaySec=1min

[Install]
WantedBy=timers.target
`, every, every)
	return service, timer
}

func installPullTimer(o *pullOptions, out interface{ Write([]byte) (int, error) }) error {
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return fmt.Errorf("pull install needs systemd")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	service, timer := pullUnitFiles(o, self)
	for name, body := range map[string]string{"onigirazu-pull.service": service, "onigirazu-pull.timer": timer} {
		if err := os.WriteFile(filepath.Join("/etc/systemd/system", name), []byte(body), 0o644); err != nil { // #nosec G306 -- unit files are world-readable
			return err
		}
	}
	for _, c := range [][]string{{"systemctl", "daemon-reload"}, {"systemctl", "enable", "--now", "onigirazu-pull.timer"}} {
		if o, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil { // #nosec G204 -- fixed systemctl arguments
			return fmt.Errorf("%s: %v: %s", strings.Join(c, " "), err, strings.TrimSpace(string(o)))
		}
	}
	fmt.Fprintf(out, "onigirazu-pull.timer installed and started (every %s)\n", func() time.Duration {
		if o.interval > 0 {
			return o.interval
		}
		return 30 * time.Minute
	}())
	return nil
}
