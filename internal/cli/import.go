package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/internal/importer"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// newImportCmd turns running hosts into a playbook
func newImportCmd() *cobra.Command {
	var (
		outDir     string
		noVerify   bool
		noBecome   bool
		becomeUser string
		force      bool
	)
	cmd := &cobra.Command{
		Use:   "import HOST_PATTERN... -i INVENTORY -o DIR",
		Short: "Write a playbook that recreates what running hosts have",
		Long: `Connect to the hosts (read-only) and write a playbook that recreates what makes each of
them differ from a fresh install: packages installed by hand, services enabled or disabled
against the vendor preset, accounts (uid/gid 1000-59999), files no package owns under /etc,
/opt, /srv, /usr/local and user crontabs, changed package configuration files, the timezone
and fstab mounts.

Private keys and credential files are not written; IMPORT_REPORT.md lists them with
everything else that was left out. Then the new playbook is planned against the same hosts:
a faithful import has nothing to change.`,
		Example: `  onigirazu import web1 -i hosts.yml -o imported/
  onigirazu import web1 db1 -i hosts.yml -o imported/ --no-verify`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if outDir == "" {
				return fmt.Errorf("-o/--output DIR is required")
			}
			if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 && !force {
				return fmt.Errorf("%s is not empty (--force writes into it)", outDir)
			}
			hosts, err := importHosts(cmd.Context(), args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Collecting %d host(s)...\n", len(hosts))
			snaps, err := collectHosts(cmd.Context(), hosts, !noBecome, becomeUser)
			if err != nil {
				return err
			}
			rep, err := importer.Generate(snaps, outDir)
			if err != nil {
				return err
			}
			if !noVerify {
				rep.Verified = true
				rep.Drift, err = verifyImport(outDir, hosts)
				if err != nil {
					return fmt.Errorf("verify: %w", err)
				}
			}
			if err := importer.WriteReport(filepath.Join(outDir, "IMPORT_REPORT.md"), rep); err != nil {
				return err
			}
			importer.PrintSummary(out, rep, outDir)
			if len(rep.Drift) > 0 {
				return &ExitError{Code: 2, Message: "the imported playbook would still change something (see IMPORT_REPORT.md)"}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "output", "o", "", "Directory to write the playbook into")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Do not plan the new playbook against the hosts")
	cmd.Flags().BoolVar(&noBecome, "no-become", false, "Collect as the login user (root-only files are missed)")
	cmd.Flags().StringVar(&becomeUser, "become-user", "root", "User to collect as")
	cmd.Flags().BoolVar(&force, "force", false, "Write into a directory that is not empty")
	return cmd
}

// importHosts resolves the host patterns against the inventory
func importHosts(ctx context.Context, patterns []string) ([]types.Host, error) {
	if len(inventoryPaths) == 0 {
		return nil, fmt.Errorf("inventory source is required (use -i/--inventory)")
	}
	log := logger.NewWithWriter(false, os.Stderr)
	tmpl := template.NewEngine()
	p := parser.NewEnhancedParser(tmpl, log)
	cacheMgr := cache.NewManager(5 * time.Minute)
	mgr := inventory.NewManager(p, log, cacheMgr)
	if ctx == nil {
		ctx = context.Background()
	}
	merged, err := inventory.NewMultiSourceLoader(p, log, cacheMgr, 10*time.Minute).LoadFromMultipleSources(ctx, inventoryPaths)
	if err != nil {
		return nil, fmt.Errorf("inventory: %w", err)
	}
	if err := mgr.SetInventory(merged); err != nil {
		return nil, fmt.Errorf("inventory: %w", err)
	}
	seen := map[string]bool{}
	var hosts []types.Host
	for _, pattern := range patterns {
		found, err := mgr.GetHosts(pattern)
		if err != nil || len(found) == 0 {
			return nil, fmt.Errorf("no host matches %q in the inventory", pattern)
		}
		for _, h := range found {
			if !seen[h.Name] {
				seen[h.Name] = true
				hosts = append(hosts, h)
			}
		}
	}
	return hosts, nil
}

// collectHosts runs the collector on every host, a few at a time
func collectHosts(ctx context.Context, hosts []types.Host, become bool, becomeUser string) ([]*importer.Snapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	snaps := make([]*importer.Snapshot, len(hosts))
	errs := make([]error, len(hosts))
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Add(1)
		go func(i int, h types.Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			exec, err := executor.NewCommandExecutor(h)
			if err != nil {
				errs[i] = fmt.Errorf("%s: %w", h.Name, err)
				return
			}
			defer exec.Close()
			if become {
				exec.SetBecome(true, becomeUser, "sudo")
			}
			snaps[i], errs[i] = importer.Collect(ctx, h.Name, exec)
		}(i, h)
	}
	wg.Wait()
	var failed []string
	for _, err := range errs {
		if err != nil {
			failed = append(failed, err.Error())
		}
	}
	if len(failed) > 0 {
		return nil, fmt.Errorf("collect failed:\n  %s", strings.Join(failed, "\n  "))
	}
	return snaps, nil
}

// verifyImport plans the new playbook against the hosts; what it would
// change is what the import got wrong
func verifyImport(dir string, hosts []types.Host) ([]importer.Drift, error) {
	names := make([]string, len(hosts))
	for i, h := range hosts {
		names[i] = h.Name
	}
	result, err := runPlaybook([]string{filepath.Join(dir, "site.yml"), "--check", "--diff",
		"--limit", strings.Join(names, ","), "--lock=false"})
	if err != nil {
		return nil, err
	}
	var drift []importer.Drift
	for _, play := range result.Plays {
		for _, h := range play.Hosts {
			for _, t := range h.Tasks {
				switch {
				case t.Failed && !t.Ignored:
					drift = append(drift, importer.Drift{Host: h.Host, Task: t.TaskName, Detail: "failed: " + t.Error})
				case t.Changed && !t.Skipped:
					drift = append(drift, importer.Drift{Host: h.Host, Task: t.TaskName, Detail: taskDetail(t)})
				}
			}
		}
	}
	sort.Slice(drift, func(i, j int) bool {
		if drift[i].Host != drift[j].Host {
			return drift[i].Host < drift[j].Host
		}
		return drift[i].Task < drift[j].Task
	})
	return drift, nil
}
