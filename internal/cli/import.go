package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/internal/importer"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
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
		baseline   string
		adopt      bool
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
			hosts, groups, err := importHosts(cmd.Context(), args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			collect := hosts
			if baseline != "" {
				base, _, err := importHosts(cmd.Context(), []string{baseline})
				if err != nil {
					return fmt.Errorf("--baseline: %w", err)
				}
				collect = append(append([]types.Host(nil), hosts...), base[0])
			}
			// known_hosts file, strict mode and timeout from the configuration, as
			// apply uses them (the default pool would write ~/.ssh/known_hosts)
			if cfg, err := config.LoadConfigWithDiscovery(configPath, "."); err == nil {
				sshpkg.InitializeGlobalPoolWithLogger(cfg, logger.New(false))
			} else {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			fmt.Fprintf(out, "Collecting %d host(s)...\n", len(collect))
			snaps, err := collectHosts(cmd.Context(), collect, !noBecome, becomeUser)
			if err != nil {
				return err
			}
			opts := importer.Options{Groups: groups}
			if baseline != "" {
				opts.Baseline, snaps = snaps[len(snaps)-1], snaps[:len(snaps)-1]
			}
			rep, err := importer.GenerateWith(snaps, outDir, opts)
			if err != nil {
				return err
			}
			if !noVerify || adopt {
				// --adopt records, in the same check run, what exists as
				// adopted in the new playbook's managed state
				drift, err := verifyImport(outDir, hosts, importer.SecretValues(snaps), adopt)
				if err != nil {
					return fmt.Errorf("verify: %w", err)
				}
				rep.Verified, rep.Drift, rep.Adopted = true, drift, adopt
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
	cmd.Flags().StringVar(&baseline, "baseline", "", "A fresh host of the same system: what it has too is not imported")
	cmd.Flags().BoolVar(&adopt, "adopt", false, "Record the imported resources as adopted in the new playbook's managed state")
	return cmd
}

// importHosts resolves the host patterns against the inventory
func importHosts(ctx context.Context, patterns []string) ([]types.Host, map[string][]string, error) {
	if len(inventoryPaths) == 0 {
		return nil, nil, fmt.Errorf("inventory source is required (use -i/--inventory)")
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
		return nil, nil, fmt.Errorf("inventory: %w", err)
	}
	if err := mgr.SetInventory(merged); err != nil {
		return nil, nil, fmt.Errorf("inventory: %w", err)
	}
	seen := map[string]bool{}
	var hosts []types.Host
	for _, pattern := range patterns {
		found, err := mgr.GetHosts(pattern)
		if err != nil || len(found) == 0 {
			return nil, nil, fmt.Errorf("no host matches %q in the inventory", pattern)
		}
		for _, h := range found {
			if !seen[h.Name] {
				seen[h.Name] = true
				hosts = append(hosts, h)
			}
		}
	}
	groups := map[string][]string{}
	for _, h := range hosts {
		groups[h.Name] = mgr.GetHostGroups(h.Name)
	}
	return hosts, groups, nil
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
func verifyImport(dir string, hosts []types.Host, secrets map[string]map[string]string, adopt bool) ([]importer.Drift, error) {
	names := make([]string, len(hosts))
	for i, h := range hosts {
		names[i] = h.Name
	}
	// secret values differ per host and exist only in memory: one run per
	// host that has them, with its values as extra vars
	runs := [][]string{}
	var plain []string
	for _, h := range names {
		if len(secrets[h]) == 0 {
			plain = append(plain, h)
			continue
		}
		vals, err := json.Marshal(secrets[h])
		if err != nil {
			return nil, err
		}
		runs = append(runs, []string{"--limit", h, "-e", string(vals)})
	}
	if len(plain) > 0 {
		runs = append(runs, []string{"--limit", strings.Join(plain, ",")})
	}
	var drift []importer.Drift
	for _, extra := range runs {
		if adopt {
			extra = append(extra, "--adopt")
		}
		d, err := planDrift(dir, extra)
		if err != nil {
			return nil, err
		}
		drift = append(drift, d...)
	}
	sort.Slice(drift, func(i, j int) bool {
		if drift[i].Host != drift[j].Host {
			return drift[i].Host < drift[j].Host
		}
		return drift[i].Task < drift[j].Task
	})
	return drift, nil
}

// planDrift plans the imported playbook with extra apply arguments
func planDrift(dir string, extra []string) ([]importer.Drift, error) {
	result, err := runPlaybook(append([]string{filepath.Join(dir, "site.yml"), "--check", "--diff", "--lock=false"}, extra...))
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
	return drift, nil
}
