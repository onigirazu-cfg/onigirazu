// Package bridge runs modules onigirazu does not have through ansible-core:
// one ansible-playbook process per task and host, with the arguments, the
// connection and the secrets in 0600 files and the result read back from a
// callback plugin.
package bridge

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Config is ansible_bridge in onigirazu.yml
type Config struct {
	// Modules are the names allowed through the bridge: exact names or
	// patterns such as community.general.* and win_*
	Modules []string `yaml:"modules" json:"modules"`
	// AnsiblePlaybook is the ansible-playbook to run (default: from PATH)
	AnsiblePlaybook string `yaml:"ansible_playbook" json:"ansible_playbook"`
}

var (
	mu  sync.RWMutex
	cfg Config
)

// Configure sets the bridge configuration for this process
func Configure(c Config) {
	mu.Lock()
	defer mu.Unlock()
	cfg = c
}

// Allowed reports whether module may run through the bridge
func Allowed(module string) bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, p := range cfg.Modules {
		if ok, _ := path.Match(p, module); ok || p == module {
			return true
		}
	}
	return false
}

// Task is what the bridge runs on one host
type Task struct {
	Name         string
	Module       string
	Args         map[string]interface{}
	Check        bool
	Diff         bool
	Become       bool
	BecomeUser   string
	BecomeMethod string
	NoLog        bool
	Environment  map[string]interface{}
}

//go:embed onigirazu_bridge.py
var callbackPlugin []byte

// Run runs the task on host with ansible-playbook
func Run(ctx context.Context, t Task, host types.Host) types.TaskResult {
	start := time.Now()
	res := types.TaskResult{TaskName: t.Name, Host: host.Name, Module: t.Module, Timestamp: start,
		Output: map[string]interface{}{}}
	fail := func(format string, a ...interface{}) types.TaskResult {
		res.Failed, res.Success = true, false
		res.Error = fmt.Sprintf(format, a...)
		res.Duration = time.Since(start)
		return res
	}
	mu.RLock()
	bin := cfg.AnsiblePlaybook
	mu.RUnlock()
	if bin == "" {
		bin = "ansible-playbook"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fail("module %s runs through Ansible, but %s is not installed", t.Module, bin)
	}

	dir, err := os.MkdirTemp("", "onigirazu-bridge-")
	if err != nil {
		return fail("bridge: %v", err)
	}
	defer os.RemoveAll(dir)
	files := map[string][]byte{"callback_plugins/onigirazu_bridge.py": callbackPlugin}
	if files["inventory.yml"], err = yaml.Marshal(Inventory(host)); err != nil {
		return fail("bridge inventory: %v", err)
	}
	if files["playbook.yml"], err = Playbook(t); err != nil {
		return fail("bridge playbook: %v", err)
	}
	for name, data := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return fail("bridge: %v", err)
		}
		if err := os.WriteFile(p, data, 0o600); err != nil {
			return fail("bridge: %v", err)
		}
	}

	resultFile := filepath.Join(dir, "result.json")
	args := []string{"-i", filepath.Join(dir, "inventory.yml"), filepath.Join(dir, "playbook.yml")}
	if t.Check {
		args = append(args, "--check")
	}
	if t.Diff {
		args = append(args, "--diff")
	}
	cmd := exec.CommandContext(ctx, bin, args...) // #nosec G204 -- ansible-playbook with generated files
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"ANSIBLE_CALLBACK_PLUGINS="+filepath.Join(dir, "callback_plugins"),
		"ANSIBLE_STDOUT_CALLBACK=onigirazu_bridge",
		"ONIGIRAZU_BRIDGE_RESULT="+resultFile,
		"ANSIBLE_RETRY_FILES_ENABLED=False",
		"ANSIBLE_NOCOLOR=1",
		"ANSIBLE_LOCALHOST_WARNING=False",
		"ANSIBLE_INVENTORY_UNPARSED_WARNING=False",
		"ANSIBLE_DEPRECATION_WARNINGS=False",
	)
	if host.InsecureIgnoreHostKey {
		cmd.Env = append(cmd.Env, "ANSIBLE_HOST_KEY_CHECKING=False")
	}
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	runErr := cmd.Run()

	data, err := os.ReadFile(resultFile) // #nosec G304 -- our own temporary file
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" && runErr != nil {
			msg = runErr.Error()
		}
		return fail("ansible-playbook gave no result for %s: %s", t.Module, lastLines(msg, 20))
	}
	var out struct {
		Status string                 `json:"status"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return fail("bridge result: %v", err)
	}
	return Map(res, out.Status, out.Result, start)
}

// Map turns an Ansible task result into a TaskResult
func Map(res types.TaskResult, status string, result map[string]interface{}, start time.Time) types.TaskResult {
	for k, v := range result {
		if strings.HasPrefix(k, "_ansible") {
			continue
		}
		res.Output[k] = v
	}
	res.Output["via"] = "ansible"
	changed, _ := result["changed"].(bool)
	res.Changed = changed
	switch status {
	case "ok", "changed":
		res.Success = true
	case "skipped":
		res.Success, res.Skipped = true, true
	default: // failed, unreachable
		res.Failed = true
		msg := fmt.Sprint(result["msg"])
		if result["msg"] == nil {
			msg = fmt.Sprintf("%s through Ansible", status)
		}
		if stderr, ok := result["stderr"].(string); ok && stderr != "" && !strings.Contains(msg, stderr) {
			msg += ": " + strings.TrimSpace(stderr)
		}
		res.Error = msg
	}
	res.Duration = time.Since(start)
	return res
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// Inventory is a one-host Ansible inventory with the host's connection:
// its ansible_* variables, then address, port, user, key and passwords
func Inventory(host types.Host) map[string]interface{} {
	vars := map[string]interface{}{}
	for k, v := range host.Vars {
		if strings.HasPrefix(k, "ansible_") {
			vars[k] = v
		}
	}
	set := func(k string, v interface{}) {
		if _, ok := vars[k]; !ok {
			vars[k] = v
		}
	}
	if host.Address != "" {
		set("ansible_host", host.Address)
	}
	if host.Port > 0 {
		set("ansible_port", host.Port)
	}
	if host.User != "" {
		set("ansible_user", host.User)
	}
	if host.KeyFile != "" {
		set("ansible_ssh_private_key_file", host.KeyFile)
	}
	if host.Password != "" {
		set("ansible_password", host.Password)
	}
	if host.BecomePassword != "" {
		set("ansible_become_password", host.BecomePassword)
	}
	if host.SSHArgs != "" {
		set("ansible_ssh_common_args", host.SSHArgs)
	}
	if _, _, container := ssh.Container(host); !container && ssh.IsLocal(host) {
		set("ansible_connection", "local")
	}
	// values are data: Ansible must not template them again
	for k, v := range vars {
		vars[k] = unsafe(v)
	}
	return map[string]interface{}{"all": map[string]interface{}{"hosts": map[string]interface{}{host.Name: vars}}}
}

// Playbook is the one-task playbook for t
func Playbook(t Task) ([]byte, error) {
	task := &yaml.Node{Kind: yaml.MappingNode}
	add := func(k string, v *yaml.Node) {
		task.Content = append(task.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, v)
	}
	if t.Name != "" {
		add("name", unsafeNode(t.Name))
	}
	add(t.Module, unsafeNode(t.Args))
	if t.Become {
		add("become", boolNode(true))
		if t.BecomeUser != "" {
			add("become_user", unsafeNode(t.BecomeUser))
		}
		if t.BecomeMethod != "" {
			add("become_method", unsafeNode(t.BecomeMethod))
		}
	}
	if t.NoLog {
		add("no_log", boolNode(true))
	}
	if len(t.Environment) > 0 {
		add("environment", unsafeNode(t.Environment))
	}
	play := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "hosts"}, {Kind: yaml.ScalarNode, Value: "all"},
		{Kind: yaml.ScalarNode, Value: "gather_facts"}, boolNode(false),
		{Kind: yaml.ScalarNode, Value: "tasks"}, {Kind: yaml.SequenceNode, Content: []*yaml.Node{task}},
	}}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.SequenceNode, Content: []*yaml.Node{play}}}}
	return yaml.Marshal(doc)
}

func boolNode(b bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(b)}
}

// unsafeString marks a string !unsafe so that Ansible keeps {{ }} in it as text
type unsafeString string

func (s unsafeString) MarshalYAML() (interface{}, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!unsafe", Value: string(s), Style: yaml.DoubleQuotedStyle}, nil
}

func unsafe(v interface{}) interface{} {
	switch x := v.(type) {
	case string:
		return unsafeString(x)
	case map[string]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, e := range x {
			m[k] = unsafe(e)
		}
		return m
	case []interface{}:
		l := make([]interface{}, len(x))
		for i, e := range x {
			l[i] = unsafe(e)
		}
		return l
	}
	return v
}

// unsafeNode is v as a YAML node with every string !unsafe; map keys sorted
func unsafeNode(v interface{}) *yaml.Node {
	switch x := v.(type) {
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!unsafe", Value: x, Style: yaml.DoubleQuotedStyle}
	case map[string]interface{}:
		n := &yaml.Node{Kind: yaml.MappingNode}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, unsafeNode(x[k]))
		}
		return n
	case []interface{}:
		n := &yaml.Node{Kind: yaml.SequenceNode}
		for _, e := range x {
			n.Content = append(n.Content, unsafeNode(e))
		}
		return n
	case []string:
		n := &yaml.Node{Kind: yaml.SequenceNode}
		for _, e := range x {
			n.Content = append(n.Content, unsafeNode(e))
		}
		return n
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}
	n := &yaml.Node{}
	if err := n.Encode(v); err != nil {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!unsafe", Value: fmt.Sprint(v), Style: yaml.DoubleQuotedStyle}
	}
	return n
}
