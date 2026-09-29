package moltest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Runner runs the steps of one scenario
type Runner struct {
	S       *Scenario
	Bin     string    // onigirazu
	Out     io.Writer // progress and playbook output
	NoDeps  bool      // skip the dependency step
	Destroy string    // always (default) or never, for test
	work    string
}

// Steps are the steps a Runner knows
var Steps = []string{"dependency", "cleanup", "destroy", "syntax", "create", "prepare", "converge",
	"idempotence", "side_effect", "verify"}

// WorkDir is the scenario's ephemeral directory: the inventory and the
// roles and collections of the dependency step
func (r *Runner) WorkDir() string {
	if r.work == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			base = os.TempDir()
		}
		sum := sha256.Sum256([]byte(r.S.Dir))
		r.work = filepath.Join(base, "onigirazu-test", hex.EncodeToString(sum[:6]))
	}
	return r.work
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// Container is the container name of a platform; the project and scenario
// are part of it, so scenarios of different roles can run side by side
func (r *Runner) Container(p Platform) string {
	return unsafeName.ReplaceAllString(fmt.Sprintf("molecule-%s-%s-%s",
		filepath.Base(r.S.ProjectDir), r.S.Name, p.Name), "-")
}

func (r *Runner) logf(format string, args ...interface{}) {
	fmt.Fprintf(r.Out, "--> "+format+"\n", args...)
}

// Test runs the scenario's test sequence; with Destroy "always" the
// containers go even when a step fails
func (r *Runner) Test(ctx context.Context) error {
	seq := r.S.Sequence()
	var failed error
	for i, step := range seq {
		// the last destroy runs below, after a failure too
		if step == "destroy" && i == len(seq)-1 {
			break
		}
		if err := r.Run(ctx, step); err != nil {
			failed = fmt.Errorf("%s: %w", step, err)
			break
		}
	}
	if r.Destroy != "never" {
		if err := r.Run(ctx, "destroy"); err != nil && failed == nil {
			failed = fmt.Errorf("destroy: %w", err)
		}
	}
	return failed
}

// Run runs one step
func (r *Runner) Run(ctx context.Context, step string) error {
	r.logf("%s: %s", r.S.Name, step)
	switch step {
	case "dependency":
		return r.dependency(ctx)
	case "create":
		return r.create(ctx)
	case "destroy":
		return r.destroy(ctx)
	case "syntax":
		pb := r.S.playbook("converge")
		if pb == "" {
			return fmt.Errorf("no converge playbook in %s", r.S.Dir)
		}
		return r.onigirazu(ctx, r.Out, "apply", pb, "-i", "localhost,", "--syntax-check")
	case "cleanup":
		if _, err := os.Stat(r.inventoryPath()); err != nil {
			return nil // nothing to clean up without instances
		}
		return r.playbookStep(ctx, step, false)
	case "prepare", "side_effect":
		return r.playbookStep(ctx, step, false)
	case "converge":
		return r.playbookStep(ctx, step, true)
	case "verify":
		if (r.S.Verifier.Enabled != nil && !*r.S.Verifier.Enabled) ||
			(r.S.Verifier.Name != "" && r.S.Verifier.Name != "ansible") {
			r.logf("verifier %q is skipped (only ansible is supported)", r.S.Verifier.Name)
			return nil
		}
		return r.playbookStep(ctx, step, false)
	case "idempotence":
		return r.idempotence(ctx)
	case "lint":
		r.logf("lint is skipped: run onigirazu lint")
		return nil
	}
	return fmt.Errorf("unknown step %q", step)
}

func (r *Runner) engine() string { return r.S.Driver.Name }

func (r *Runner) containerCmd(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, r.engine(), args...) // #nosec G204 -- docker/podman with the scenario's settings
}

func (r *Runner) running(ctx context.Context, name string) bool {
	out, err := r.containerCmd(ctx, "inspect", "-f", "{{.State.Running}}", name).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// runArgs are the docker/podman run arguments of a platform
func (r *Runner) runArgs(p Platform) []string {
	args := []string{"run", "-d", "--name", r.Container(p), "--label", "onigirazu-test=" + r.S.Dir}
	hostname := p.Hostname
	if hostname == "" {
		hostname = p.Name
	}
	args = append(args, "--hostname", hostname)
	if p.Privileged {
		args = append(args, "--privileged")
	}
	if p.CgroupnsMode != "" {
		args = append(args, "--cgroupns="+p.CgroupnsMode)
	}
	for _, v := range p.Volumes {
		args = append(args, "-v", v)
	}
	for _, t := range p.Tmpfs {
		args = append(args, "--tmpfs", t)
	}
	for _, c := range p.CapAdd {
		args = append(args, "--cap-add", c)
	}
	for _, port := range p.PublishedPorts {
		args = append(args, "-p", port)
	}
	for _, n := range p.Networks {
		args = append(args, "--network", n.Name)
	}
	keys := make([]string, 0, len(p.Env))
	for k := range p.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", k+"="+p.Env[k])
	}
	args = append(args, p.Image)
	// Molecule keeps the container up with a sleep loop unless
	// override_command is false (an image that runs systemd)
	if p.OverrideCommand == nil || *p.OverrideCommand {
		cmd := p.Command
		if cmd == "" {
			cmd = "while true; do sleep 10000; done"
		}
		args = append(args, "sh", "-c", cmd)
	}
	return args
}

func (r *Runner) create(ctx context.Context) error {
	for _, p := range r.S.Platforms {
		name := r.Container(p)
		if r.running(ctx, name) {
			r.logf("%s is running", name)
			continue
		}
		_ = r.containerCmd(ctx, "rm", "-f", name).Run()
		cmd := r.containerCmd(ctx, r.runArgs(p)...)
		cmd.Stdout, cmd.Stderr = io.Discard, r.Out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("start %s: %w", name, err)
		}
		if err := r.waitReady(ctx, name); err != nil {
			return err
		}
	}
	return r.writeInventory()
}

// waitReady waits until the container runs commands
func (r *Runner) waitReady(ctx context.Context, name string) error {
	deadline := time.Now().Add(60 * time.Second)
	for {
		if r.containerCmd(ctx, "exec", name, "true").Run() == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s does not run commands after 60s", name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (r *Runner) destroy(ctx context.Context) error {
	for _, p := range r.S.Platforms {
		_ = r.containerCmd(ctx, "rm", "-f", "-v", r.Container(p)).Run()
	}
	// the dependency step's roles and collections stay for the next create
	return os.RemoveAll(r.runDir())
}

// Inventory is the scenario's inventory: every platform with the container
// connection, its groups, and the provisioner's group_vars and host_vars
func (r *Runner) Inventory() map[string]interface{} {
	hosts := map[string]interface{}{}
	children := map[string]interface{}{}
	group := func(name string) map[string]interface{} {
		g, ok := children[name].(map[string]interface{})
		if !ok {
			g = map[string]interface{}{}
			children[name] = g
		}
		return g
	}
	for _, p := range r.S.Platforms {
		vars := map[string]interface{}{"ansible_connection": r.engine(), "ansible_host": r.Container(p)}
		for k, v := range r.S.Provisioner.Inventory.HostVars[p.Name] {
			vars[k] = v
		}
		hosts[p.Name] = vars
		for _, g := range p.Groups {
			gh, _ := group(g)["hosts"].(map[string]interface{})
			if gh == nil {
				gh = map[string]interface{}{}
				group(g)["hosts"] = gh
			}
			gh[p.Name] = nil
		}
	}
	all := map[string]interface{}{"hosts": hosts}
	for g, vars := range r.S.Provisioner.Inventory.GroupVars {
		if g == "all" {
			all["vars"] = vars
			continue
		}
		group(g)["vars"] = vars
	}
	if len(children) > 0 {
		all["children"] = children
	}
	return map[string]interface{}{"all": all}
}

// runDir holds the inventory and the state of the runs until destroy
func (r *Runner) runDir() string { return filepath.Join(r.WorkDir(), "run") }

func (r *Runner) inventoryPath() string { return filepath.Join(r.runDir(), "inventory.yml") }

func (r *Runner) writeInventory() error {
	if err := os.MkdirAll(r.runDir(), 0o750); err != nil {
		return err
	}
	data, err := yaml.Marshal(r.Inventory())
	if err != nil {
		return err
	}
	return os.WriteFile(r.inventoryPath(), data, 0o600)
}

// env is the environment of the playbook runs: the provisioner's env
// (relative paths start at the scenario directory), the MOLECULE_*
// variables, and role and collection paths that find the role under test
// and what the dependency step installed
func (r *Runner) env() []string {
	env := os.Environ()
	extra := map[string]string{}
	for k, v := range r.S.Provisioner.Env {
		if strings.HasSuffix(k, "_PATH") || strings.HasSuffix(k, "_PATHS") || k == "ANSIBLE_CONFIG" {
			parts := filepath.SplitList(v)
			for i, p := range parts {
				if p != "" && !filepath.IsAbs(p) && !strings.HasPrefix(p, "~") {
					parts[i] = filepath.Join(r.S.Dir, p)
				}
			}
			v = strings.Join(parts, string(os.PathListSeparator))
		}
		extra[k] = v
	}
	join := func(key string, more ...string) {
		list := []string{}
		if v := extra[key]; v != "" {
			list = append(list, v)
		} else if v := os.Getenv(key); v != "" {
			list = append(list, v)
		}
		extra[key] = strings.Join(append(list, more...), string(os.PathListSeparator))
	}
	join("ANSIBLE_ROLES_PATH", filepath.Join(r.WorkDir(), "roles"), filepath.Dir(r.S.ProjectDir))
	join("ANSIBLE_COLLECTIONS_PATH", filepath.Join(r.WorkDir(), "collections"))
	extra["MOLECULE_SCENARIO_NAME"] = r.S.Name
	extra["MOLECULE_SCENARIO_DIRECTORY"] = r.S.Dir
	extra["MOLECULE_PROJECT_DIRECTORY"] = r.S.ProjectDir
	extra["MOLECULE_EPHEMERAL_DIRECTORY"] = r.WorkDir()
	extra["MOLECULE_INVENTORY_FILE"] = r.inventoryPath()
	extra["MOLECULE_DRIVER_NAME"] = r.engine()
	extra["ONIGIRAZU_MANAGED_STATE_DIR"] = r.runDir()
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		env = append(env, k+"="+extra[k])
	}
	return env
}

func (r *Runner) onigirazu(ctx context.Context, stdout io.Writer, args ...string) error {
	// runs in the ephemeral directory: the state and snapshots of the runs
	// stay out of the role
	if err := os.MkdirAll(r.runDir(), 0o750); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, r.Bin, args...) // #nosec G204 -- onigirazu with the scenario's playbooks
	cmd.Dir = r.runDir()
	cmd.Env = r.env()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, stdout, r.Out
	return cmd.Run()
}

// playbookStep runs the step's playbook; a missing optional one is skipped
func (r *Runner) playbookStep(ctx context.Context, step string, required bool) error {
	pb := r.S.playbook(step)
	if pb == "" {
		if required {
			return fmt.Errorf("no %s playbook in %s", step, r.S.Dir)
		}
		r.logf("no %s playbook, skipped", step)
		return nil
	}
	if _, err := os.Stat(r.inventoryPath()); err != nil {
		return errors.New("no instances: run create first")
	}
	return r.onigirazu(ctx, r.Out, "apply", pb, "-i", r.inventoryPath())
}

// changedTasks lists "task (hosts)" for every task that changed in an
// apply -o json result
func changedTasks(result []byte) ([]string, error) {
	var run struct {
		Tasks []struct {
			Name        string `json:"name"`
			Changed     int    `json:"changed"`
			HostResults map[string]struct {
				Status string `json:"status"`
			} `json:"host_results"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(result, &run); err != nil {
		return nil, fmt.Errorf("reading the run's result: %w", err)
	}
	var out []string
	for _, t := range run.Tasks {
		if t.Changed == 0 {
			continue
		}
		var hosts []string
		for h, res := range t.HostResults {
			if res.Status == "changed" {
				hosts = append(hosts, h)
			}
		}
		sort.Strings(hosts)
		out = append(out, fmt.Sprintf("%s (%s)", t.Name, strings.Join(hosts, ", ")))
	}
	return out, nil
}

// idempotence runs converge again: no task may change
func (r *Runner) idempotence(ctx context.Context) error {
	pb := r.S.playbook("converge")
	if pb == "" {
		return fmt.Errorf("no converge playbook in %s", r.S.Dir)
	}
	var out bytes.Buffer
	if err := r.onigirazu(ctx, &out, "apply", pb, "-i", r.inventoryPath(), "-o", "json"); err != nil {
		return fmt.Errorf("the second converge failed: %w", err)
	}
	changed, err := changedTasks(out.Bytes())
	if err != nil {
		return err
	}
	if len(changed) > 0 {
		return fmt.Errorf("not idempotent, changed on the second run:\n  %s", strings.Join(changed, "\n  "))
	}
	r.logf("idempotence: no changes")
	return nil
}

// dependency installs the requirements file's roles and collections with
// onigirazu galaxy into the ephemeral directory
func (r *Runner) dependency(ctx context.Context) error {
	d := r.S.Dependency
	if r.NoDeps || (d.Enabled != nil && !*d.Enabled) || (d.Name != "" && d.Name != "galaxy") {
		r.logf("dependency skipped")
		return nil
	}
	file := d.Options.RequirementsFile
	if file == "" {
		file = d.Options.RoleFile
	}
	if file == "" {
		file = "requirements.yml"
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(r.S.Dir, file)
	}
	if _, err := os.Stat(file); err != nil {
		r.logf("no %s, dependency skipped", file)
		return nil
	}
	return r.onigirazu(ctx, r.Out, "galaxy", "install", "-r", file,
		"--roles-path", filepath.Join(r.WorkDir(), "roles"),
		"--collections-path", filepath.Join(r.WorkDir(), "collections"))
}
