package bridge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestAllowed(t *testing.T) {
	Configure(Config{Modules: []string{"community.general.*", "win_*", "tempfile"}})
	defer Configure(Config{})
	for name, want := range map[string]bool{
		"community.general.nmcli": true, "win_regedit": true, "tempfile": true,
		"community.docker.x": false, "tempfiles": false, "copy": false,
	} {
		if got := Allowed(name); got != want {
			t.Errorf("Allowed(%s) = %v", name, got)
		}
	}
}

func TestInventory(t *testing.T) {
	host := types.Host{Name: "web1", Address: "10.0.0.5", Port: 2222, User: "deploy", KeyFile: "/k",
		Password: "p{{w}}", BecomePassword: "b", SSHArgs: "-J bastion",
		Vars: map[string]interface{}{"ansible_user": "override", "app_port": 80, "ansible_winrm_transport": "ntlm"}}
	data, err := yaml.Marshal(Inventory(host))
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	for _, want := range []string{"web1:", `ansible_host: !unsafe "10.0.0.5"`, "ansible_port: 2222",
		`ansible_user: !unsafe "override"`, `ansible_password: !unsafe "p{{w}}"`, `ansible_ssh_common_args: !unsafe "-J bastion"`,
		`ansible_winrm_transport: !unsafe "ntlm"`, `ansible_become_password: !unsafe "b"`} {
		if !strings.Contains(out, want) {
			t.Errorf("inventory lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "app_port") {
		t.Errorf("only ansible_* variables go to the inventory:\n%s", out)
	}
	local, _ := yaml.Marshal(Inventory(types.Host{Name: "localhost", Address: "localhost"}))
	if !strings.Contains(string(local), "ansible_connection") {
		t.Errorf("local host: %s", local)
	}
}

func TestPlaybook(t *testing.T) {
	data, err := Playbook(Task{Name: "t", Module: "community.general.x",
		Args:   map[string]interface{}{"b": "{{ x }}", "a": 1, "l": []interface{}{"y", true}, "n": nil},
		Become: true, BecomeUser: "root", NoLog: true, Environment: map[string]interface{}{"E": "1"}})
	if err != nil {
		t.Fatal(err)
	}
	var plays []map[string]interface{}
	if err := yaml.Unmarshal(data, &plays); err != nil {
		t.Fatalf("%v\n%s", err, data)
	}
	out := string(data)
	for _, want := range []string{"hosts: all", "gather_facts: false", "community.general.x:", "a: 1",
		`b: !unsafe "{{ x }}"`, "- true", "n: null", "become: true", "no_log: true", `E: !unsafe "1"`} {
		if !strings.Contains(out, want) {
			t.Errorf("playbook lacks %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "a: 1") > strings.Index(out, "b: ") {
		t.Errorf("keys are sorted:\n%s", out)
	}
}

func TestMap(t *testing.T) {
	start := time.Now()
	base := func() types.TaskResult { return types.TaskResult{Output: map[string]interface{}{}} }
	r := Map(base(), "changed", map[string]interface{}{"changed": true, "path": "/x", "_ansible_no_log": false}, start)
	if !r.Success || !r.Changed || r.Output["path"] != "/x" || r.Output["via"] != "ansible" {
		t.Errorf("changed: %+v", r)
	}
	if _, ok := r.Output["_ansible_no_log"]; ok {
		t.Error("_ansible keys are dropped")
	}
	r = Map(base(), "failed", map[string]interface{}{"msg": "bad", "stderr": "boom"}, start)
	if !r.Failed || r.Error != "bad: boom" {
		t.Errorf("failed: %+v", r)
	}
	r = Map(base(), "unreachable", map[string]interface{}{}, start)
	if !r.Failed || !strings.Contains(r.Error, "unreachable") {
		t.Errorf("unreachable: %+v", r)
	}
	r = Map(base(), "skipped", map[string]interface{}{}, start)
	if !r.Skipped || r.Failed {
		t.Errorf("skipped: %+v", r)
	}
}

func TestRunWithoutAnsible(t *testing.T) {
	Configure(Config{AnsiblePlaybook: "/nonexistent/ansible-playbook"})
	defer Configure(Config{})
	r := Run(context.Background(), Task{Module: "tempfile"}, types.Host{Name: "h"})
	if !r.Failed || !strings.Contains(r.Error, "not installed") {
		t.Errorf("result: %+v", r)
	}
}

func TestRunReadsTheCallbackResult(t *testing.T) {
	dir := t.TempDir()
	fake := dir + "/ansible-playbook"
	script := "#!/bin/sh\n" +
		`inv=$2; pb=$3; [ "$4" = --check ] || exit 9` + "\n" +
		`[ "$(stat -c %a "$inv" 2>/dev/null || stat -f %Lp "$inv")" = 600 ] || exit 8` + "\n" +
		`grep -q "tempfile:" "$pb" || exit 7` + "\n" +
		`echo '{"status":"changed","result":{"changed":true,"path":"/tmp/x"}}' > "$ONIGIRAZU_BRIDGE_RESULT"` + "\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil { // #nosec G306 -- a test executable
		t.Fatal(err)
	}
	Configure(Config{AnsiblePlaybook: fake})
	defer Configure(Config{})
	r := Run(context.Background(), Task{Module: "tempfile", Check: true}, types.Host{Name: "h", Address: "10.0.0.1"})
	if !r.Success || !r.Changed || r.Output["path"] != "/tmp/x" {
		t.Errorf("result: %+v", r)
	}
	Configure(Config{AnsiblePlaybook: fake})
	r = Run(context.Background(), Task{Module: "tempfile"}, types.Host{Name: "h"})
	if !r.Failed || !strings.Contains(r.Error, "gave no result") {
		t.Errorf("no result: %+v", r)
	}
}

func TestRunRelativeBinaryAndCollections(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("ANSIBLE_COLLECTIONS_PATH", "/inherited")
	t.Setenv("PATH", "bin")
	if err := os.Mkdir("bin", 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		`printf '%s' "$ANSIBLE_COLLECTIONS_PATH" > "$BRIDGE_TEST_ENV"` + "\n" +
		`echo '{"status":"ok","result":{"changed":false}}' > "$ONIGIRAZU_BRIDGE_RESULT"` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "bin", "ansible-playbook"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	dump := filepath.Join(dir, "env")
	t.Setenv("BRIDGE_TEST_ENV", dump)
	// an explicit relative path works from the temporary directory
	cfg := Config{AnsiblePlaybook: "bin/ansible-playbook", CollectionsPath: []string{"/first", "/second", "/first"}}
	r := cfg.Run(context.Background(), Task{Module: "community.demo.test"}, types.Host{Name: "h"})
	if !r.Success {
		t.Fatalf("relative path: %+v", r)
	}
	data, err := os.ReadFile(dump)
	if err != nil || string(data) != "/first:/second" {
		t.Fatalf("collections = %q, %v", data, err)
	}
	// a bare name found only through a relative PATH entry stays refused
	cfg.AnsiblePlaybook = "ansible-playbook"
	if r := cfg.Run(context.Background(), Task{Module: "community.demo.test"}, types.Host{Name: "h"}); r.Success {
		t.Fatal("bare name through a relative PATH entry must be refused")
	}
}

func TestCloneCollectionsPath(t *testing.T) {
	cfg := Config{CollectionsPath: []string{"/first"}}
	clone := cfg.Clone()
	clone.CollectionsPath[0] = "/second"
	if cfg.CollectionsPath[0] != "/first" {
		t.Fatal("Clone shares collection paths")
	}
}
