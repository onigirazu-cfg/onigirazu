package comply

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundledProfilesParse(t *testing.T) {
	profiles, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) < 2 {
		t.Fatalf("bundled profiles: %d", len(profiles))
	}
	for _, p := range profiles {
		if p.Title == "" || len(p.Controls) == 0 {
			t.Errorf("%s: no title or controls", p.Name)
		}
		for _, c := range p.Controls {
			if c.Remediation == "" || len(c.Checks) == 0 || c.Severity == "" {
				t.Errorf("%s/%s: every control has checks, a severity and a fix", p.Name, c.ID)
			}
		}
	}
	if _, err := Load("ssh"); err != nil {
		t.Error(err)
	}
	if _, err := Load("no-such-profile"); err == nil || !strings.Contains(err.Error(), "linux-baseline") {
		t.Errorf("an unknown profile names the bundled ones: %v", err)
	}
}

func TestProfileFileFilterAndBuild(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "own.yml")
	if err := os.WriteFile(p, []byte(`
title: own
controls:
  - id: a
    title: passwd mode
    severity: high
    tags: [fs]
    check: {file: {path: /etc/passwd, mode: "0644"}}
    remediation: chmod 644
  - id: b
    title: two checks
    tags: [net]
    checks:
      - {kernel_param: {name: net.ipv4.ip_forward, value: "0"}}
      - {kernel_param: {name: net.ipv4.tcp_syncookies, value: "1"}}
  - id: c
    title: low one
    severity: low
    tags: [fs]
    check: {file: {path: /tmp, type: directory}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	prof, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if prof.Name != "own" || prof.Controls[1].Severity != "medium" || len(prof.Controls[1].Checks) != 2 {
		t.Errorf("defaults: %+v", prof)
	}
	checks, index := prof.Checks()
	if len(checks) != 4 || index[1] != 1 || index[2] != 1 || index[3] != 2 {
		t.Errorf("flattened checks: %d %v", len(checks), index)
	}
	if f := prof.Filter([]string{"fs"}, ""); len(f.Controls) != 2 {
		t.Errorf("tag filter: %d", len(f.Controls))
	}
	if f := prof.Filter(nil, "medium"); len(f.Controls) != 2 || f.Controls[1].ID != "b" {
		t.Errorf("severity filter: %+v", f.Controls)
	}
	// results: check 2 (second of control b) fails on web1; web2 passes all
	r := Build(prof, map[string][]map[string]interface{}{
		"web1": {{"ok": true}, {"ok": true}, {"ok": false, "check": "kernel_param net.ipv4.tcp_syncookies", "detail": "value 0"}, {"ok": true}},
		"web2": {{"ok": true}, {"ok": true}, {"ok": true}, {"ok": true}},
	}, map[string]string{"db1": "unreachable"})
	if r.Passed != 5 || r.Failed != 1 || r.Hosts["web1"].Failed != 1 || r.Hosts["web2"].Score != 100 {
		t.Errorf("report: passed=%d failed=%d %+v", r.Passed, r.Failed, r.Hosts["web1"])
	}
	bad := r.Hosts["web1"].Controls[1]
	if bad.OK || bad.ID != "b" || len(bad.Details) != 1 || !strings.Contains(bad.Details[0], "value 0") {
		t.Errorf("control b on web1: %+v", bad)
	}
	if !r.FailsAt("") || !r.FailsAt("medium") || r.FailsAt("high") {
		t.Errorf("thresholds: any=%v medium=%v high=%v", r.FailsAt(""), r.FailsAt("medium"), r.FailsAt("high"))
	}
	var text, md, h bytes.Buffer
	r.WriteText(&text)
	r.WriteMarkdown(&md)
	r.WriteHTML(&h)
	if !strings.Contains(text.String(), "✗ [medium] b — two checks") || !strings.Contains(text.String(), "db1: could not check: unreachable") {
		t.Errorf("text:\n%s", text.String())
	}
	if !strings.Contains(md.String(), "| web1 | 2 | 1 | 67% |") || !strings.Contains(h.String(), "<td class=\"bad\">fail</td>") {
		t.Errorf("markdown/html:\n%s\n%s", md.String(), h.String())
	}
	m := r.Metrics(map[string]string{"env": "prod"})
	if !strings.Contains(m, `onigirazu_comply_controls{profile="own",host="web1",result="fail",env="prod"} 1`) || !strings.Contains(m, `onigirazu_comply_score{profile="own",host="web2",env="prod"} 100`) {
		t.Errorf("metrics:\n%s", m)
	}
}
