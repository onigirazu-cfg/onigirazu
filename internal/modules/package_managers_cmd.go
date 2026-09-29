package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
)

// cmdPackageManager drives pacman (Arch) and zypper (openSUSE/SLES). It runs
// the tool directly: installing needs become, as in Ansible.
type cmdPackageManager struct {
	tool     string // "pacman" or "zypper"
	executor *executor.CommandExecutor
	cache    *PackageStateCache
}

func newCmdPackageManager(tool string, exec *executor.CommandExecutor) *cmdPackageManager {
	return &cmdPackageManager{tool: tool, executor: exec, cache: NewPackageStateCache(10 * time.Minute)}
}

func (p *cmdPackageManager) run(ctx context.Context, args ...string) (string, error) {
	return p.executor.ExecuteWithContext(ctx, args[0], args[1:]...)
}

// op runs a command that changes packages and wraps it as a PackageOperation
func (p *cmdPackageManager) op(ctx context.Context, name, operation string, args ...string) (*PackageOperation, error) {
	start := time.Now()
	out, err := p.run(ctx, args...)
	p.cache.Delete(name)
	result := &PackageOperation{Package: name, Operation: operation, Output: out, Duration: time.Since(start)}
	if err != nil {
		result.Error = fmt.Sprintf("%s %s failed: %v", p.tool, operation, err)
		return result, err
	}
	result.Success = true
	result.Changed = !strings.Contains(out, "there is nothing to do") && !strings.Contains(out, "Nothing to do")
	if state, err := p.IsInstalled(ctx, name); err == nil && state.Installed {
		result.NewVersion = state.Version
	}
	return result, nil
}

func (p *cmdPackageManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	if p.tool == "pacman" {
		if version != "" {
			err := fmt.Errorf("pacman cannot install a given version of %s", name)
			return &PackageOperation{Package: name, Operation: "install", Error: err.Error()}, err
		}
		return p.op(ctx, name, "install", "pacman", "-S", "--noconfirm", "--needed", name)
	}
	spec := name
	args := []string{"zypper", "--non-interactive", "install"}
	if version != "" {
		spec = name + "=" + version
		args = append(args, "--oldpackage")
	}
	return p.op(ctx, name, "install", append(args, spec)...)
}

func (p *cmdPackageManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	if p.tool == "pacman" {
		return p.op(ctx, name, "remove", "pacman", "-R", "--noconfirm", name)
	}
	return p.op(ctx, name, "remove", "zypper", "--non-interactive", "remove", name)
}

func (p *cmdPackageManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	if p.tool == "pacman" {
		return p.op(ctx, name, "update", "pacman", "-S", "--noconfirm", name)
	}
	return p.op(ctx, name, "update", "zypper", "--non-interactive", "update", name)
}

func (p *cmdPackageManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	if p.tool == "pacman" {
		return p.op(ctx, "all", "update_all", "pacman", "-Syu", "--noconfirm")
	}
	return p.op(ctx, "all", "update_all", "zypper", "--non-interactive", "update")
}

// IsInstalled reports the installed version and the version the repositories offer
func (p *cmdPackageManager) IsInstalled(ctx context.Context, name string) (*PackageState, error) {
	if cached, found := p.cache.Get(name); found {
		return cached, nil
	}
	state := &PackageState{Name: name, LastChecked: time.Now()}
	if p.tool == "pacman" {
		if out, err := p.run(ctx, "pacman", "-Q", name); err == nil {
			if f := strings.Fields(out); len(f) >= 2 {
				state.Installed, state.Version = true, f[1]
			}
		}
		if out, err := p.run(ctx, "pacman", "-Sp", "--print-format", "%v", name); err == nil {
			state.AvailableVersion = lastLine(out)
		}
	} else {
		if out, err := p.run(ctx, "rpm", "-q", "--qf", `%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\n`, name); err == nil {
			state.Installed, state.Version = true, firstLine(out)
		}
		if out, err := p.run(ctx, "zypper", "--non-interactive", "--quiet", "info", name); err == nil {
			state.AvailableVersion = zypperField(out, "Version")
		}
	}
	p.cache.Set(name, state)
	return state, nil
}

func (p *cmdPackageManager) GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error) {
	state, err := p.IsInstalled(ctx, name)
	if err != nil {
		return nil, err
	}
	return &PackageInfo{Name: name, Version: state.Version, Installed: state.Installed,
		NewVersion: state.AvailableVersion, Upgradable: state.Installed && state.AvailableVersion != "" && state.AvailableVersion != state.Version}, nil
}

func (p *cmdPackageManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	return p.batch(packages, func(s PackageSpec) (*PackageOperation, error) { return p.Install(ctx, s.Name, s.Version) })
}

func (p *cmdPackageManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	specs := make([]PackageSpec, 0, len(packages))
	for _, name := range packages {
		specs = append(specs, PackageSpec{Name: name})
	}
	return p.batch(specs, func(s PackageSpec) (*PackageOperation, error) { return p.Remove(ctx, s.Name) })
}

func (p *cmdPackageManager) batch(specs []PackageSpec, do func(PackageSpec) (*PackageOperation, error)) (*BatchOperation, error) {
	start := time.Now()
	b := &BatchOperation{Success: true, TotalCount: len(specs)}
	var firstErr error
	for _, s := range specs {
		result, err := do(s)
		b.Operations = append(b.Operations, *result)
		if err != nil {
			b.Success = false
			b.FailedCount++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		b.SuccessCount++
		if result.Changed {
			b.Changed = true
			b.ChangedCount++
		}
	}
	b.Duration = time.Since(start)
	b.Summary = fmt.Sprintf("%d succeeded, %d failed, %d changed", b.SuccessCount, b.FailedCount, b.ChangedCount)
	return b, firstErr
}

func (p *cmdPackageManager) RefreshCache(ctx context.Context) error {
	if p.tool == "pacman" {
		_, err := p.run(ctx, "pacman", "-Sy")
		return err
	}
	_, err := p.run(ctx, "zypper", "--non-interactive", "refresh")
	return err
}

func (p *cmdPackageManager) ValidateState(ctx context.Context) error {
	_, err := p.run(ctx, p.tool, "--version")
	return err
}

func (p *cmdPackageManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	return &OperationPreview{WillChange: true, Actions: []string{fmt.Sprintf("%s %v", operation, args)}}, nil
}

func (p *cmdPackageManager) GetDependencies(ctx context.Context, name string) ([]string, error) {
	if p.tool == "pacman" {
		out, err := p.run(ctx, "pacman", "-Qi", name)
		if err != nil {
			return nil, err
		}
		deps := strings.Fields(zypperField(out, "Depends On"))
		if len(deps) == 1 && deps[0] == "None" {
			return nil, nil
		}
		return deps, nil
	}
	out, err := p.run(ctx, "rpm", "-q", "--requires", name)
	if err != nil {
		return nil, err
	}
	var deps []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			deps = append(deps, l)
		}
	}
	return deps, nil
}

func (p *cmdPackageManager) VerifyChecksum(ctx context.Context, name, version string) (bool, error) {
	return true, nil
}

func (p *cmdPackageManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	var found []PackageInfo
	if p.tool == "pacman" {
		out, err := p.run(ctx, "pacman", "-Ss", query)
		if err != nil {
			return nil, nil // no match
		}
		for _, l := range strings.Split(out, "\n") {
			if l == "" || strings.HasPrefix(l, " ") {
				continue
			}
			f := strings.Fields(l)
			if len(f) < 2 {
				continue
			}
			repoName := strings.SplitN(f[0], "/", 2)
			info := PackageInfo{Name: repoName[len(repoName)-1], Version: f[1], Installed: strings.Contains(l, "[installed")}
			if len(repoName) == 2 {
				info.Repository = repoName[0]
			}
			found = append(found, info)
		}
		return found, nil
	}
	out, err := p.run(ctx, "zypper", "--non-interactive", "--quiet", "search", query)
	if err != nil {
		return nil, nil // zypper exits 104 when nothing matches
	}
	for _, row := range zypperTable(out) {
		if len(row) >= 3 {
			found = append(found, PackageInfo{Name: row[1], Description: row[2], Installed: strings.HasPrefix(row[0], "i")})
		}
	}
	return found, nil
}

func (p *cmdPackageManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	args := []string{"pacman", "-Q"}
	if p.tool == "zypper" {
		args = []string{"rpm", "-qa", "--qf", `%{NAME} %|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\n`}
	}
	out, err := p.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var list []PackageInfo
	for _, l := range strings.Split(out, "\n") {
		if f := strings.Fields(l); len(f) == 2 {
			list = append(list, PackageInfo{Name: f[0], Version: f[1], Installed: true})
		}
	}
	return list, nil
}

func (p *cmdPackageManager) ListUpgradable(ctx context.Context) ([]PackageInfo, error) {
	var list []PackageInfo
	if p.tool == "pacman" {
		// "name old -> new"; exits 1 when there is nothing to upgrade
		out, _ := p.run(ctx, "pacman", "-Qu")
		for _, l := range strings.Split(out, "\n") {
			if f := strings.Fields(l); len(f) >= 4 && f[2] == "->" {
				list = append(list, PackageInfo{Name: f[0], Version: f[1], NewVersion: f[3], Installed: true, Upgradable: true})
			}
		}
		return list, nil
	}
	out, err := p.run(ctx, "zypper", "--non-interactive", "--quiet", "list-updates")
	if err != nil {
		return nil, err
	}
	// S | Repository | Name | Current Version | Available Version | Arch
	for _, row := range zypperTable(out) {
		if len(row) >= 5 {
			list = append(list, PackageInfo{Name: row[2], Repository: row[1], Version: row[3], NewVersion: row[4], Installed: true, Upgradable: true})
		}
	}
	return list, nil
}

func (p *cmdPackageManager) Clean(ctx context.Context) error {
	if p.tool == "pacman" {
		_, err := p.run(ctx, "pacman", "-Sc", "--noconfirm")
		return err
	}
	_, err := p.run(ctx, "zypper", "--non-interactive", "clean", "--all")
	return err
}

func (p *cmdPackageManager) AutoRemove(ctx context.Context) ([]string, error) {
	if p.tool == "zypper" {
		return nil, fmt.Errorf("autoremove is not supported with zypper")
	}
	// orphans: installed as dependencies, needed by nothing; exits 1 when none
	out, _ := p.run(ctx, "pacman", "-Qdtq")
	orphans := strings.Fields(out)
	if len(orphans) == 0 {
		return nil, nil
	}
	if _, err := p.run(ctx, append([]string{"pacman", "-Rns", "--noconfirm"}, orphans...)...); err != nil {
		return nil, err
	}
	return orphans, nil
}

func (p *cmdPackageManager) VerifyIntegrity(ctx context.Context) error {
	if p.tool == "pacman" {
		_, err := p.run(ctx, "pacman", "-Dk")
		return err
	}
	_, err := p.run(ctx, "zypper", "--non-interactive", "verify", "--dry-run")
	return err
}

func firstLine(s string) string {
	return strings.TrimSpace(strings.SplitN(strings.TrimSpace(s), "\n", 2)[0])
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// zypperField reads "Key : value" from zypper info / pacman -Qi output
func zypperField(out, key string) string {
	for _, l := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(l, ":")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// zypperTable splits the rows of a zypper table, skipping the header and rules
func zypperTable(out string) [][]string {
	var rows [][]string
	header := true
	for _, l := range strings.Split(out, "\n") {
		if !strings.Contains(l, "|") {
			continue
		}
		if header {
			header = false
			continue
		}
		cells := strings.Split(l, "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, cells)
	}
	return rows
}
