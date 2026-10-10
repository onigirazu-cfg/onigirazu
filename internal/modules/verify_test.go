package modules

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func checks(t *testing.T, list ...map[string]interface{}) []verifyCheck {
	t.Helper()
	raw := make([]interface{}, 0, len(list))
	for _, c := range list {
		raw = append(raw, c)
	}
	cs, err := verifyChecks(raw)
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

// the script runs on this machine (sh, stat, grep): file, command, user and
// kernel checks need nothing else
func TestVerifyScriptRuns(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	f := t.TempDir() + "/app.conf"
	if err := os.WriteFile(f, []byte("listen 80\nworker_processes 4\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	cs := checks(t,
		map[string]interface{}{"file": map[string]interface{}{"path": f, "mode": "0640", "contains": []interface{}{"listen 80", "/^worker_processes [0-9]+$/"}}},
		map[string]interface{}{"file": map[string]interface{}{"path": f, "contains": "nothing like this"}},
		map[string]interface{}{"file": map[string]interface{}{"path": f + ".missing", "exists": false}},
		map[string]interface{}{"command": map[string]interface{}{"cmd": "echo hello; echo oops >&2", "stdout": []interface{}{"hello"}, "stderr": "oops"}},
		map[string]interface{}{"command": map[string]interface{}{"cmd": "exit 3", "exit_status": 2}},
		map[string]interface{}{"user": map[string]interface{}{"name": "no-such-user-xyz", "exists": false}},
	) // dns needs getent (not on macOS): covered by the e2e case
	script, err := verifyScript(cs)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command("sh", "-c", script).CombinedOutput()
	report, passed, failed := parseVerifyOutput(cs, string(out))
	if passed != 4 || failed != 2 {
		t.Fatalf("passed=%d failed=%d:\n%s\n%s", passed, failed, out, script)
	}
	if ok, _ := report[1]["ok"].(bool); ok || !strings.Contains(report[1]["detail"].(string), "contains lacks") {
		t.Errorf("check 2: %v", report[1])
	}
	if ok, _ := report[4]["ok"].(bool); ok || !strings.Contains(report[4]["detail"].(string), "exit 3") {
		t.Errorf("check 5: %v", report[4])
	}
	if report[0]["check"] != "file "+f {
		t.Errorf("label: %v", report[0]["check"])
	}
}

func TestVerifyChecksValidation(t *testing.T) {
	if _, err := verifyChecks([]interface{}{map[string]interface{}{"bogus": map[string]interface{}{}}}); err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("unknown kind: %v", err)
	}
	if _, err := verifyChecks([]interface{}{map[string]interface{}{"file": "x"}}); err == nil {
		t.Error("arguments must be a map")
	}
	cs := checks(t, map[string]interface{}{"port": map[string]interface{}{"port": 443, "ip": "127.0.0.1"}})
	s, err := verifyScript(cs)
	if err != nil || !strings.Contains(s, `'^127\.0\.0\.1:443$'`) {
		t.Errorf("port script: %v\n%s", err, s)
	}
	// a result the script never printed is a failure, not a pass
	_, passed, failed := parseVerifyOutput(cs, "")
	if passed != 0 || failed != 1 {
		t.Errorf("missing result: passed=%d failed=%d", passed, failed)
	}
}
