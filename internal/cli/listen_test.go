package cli

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeListenConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "listen.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const testListenYML = `
sources:
  - name: alerts
    type: alertmanager
    token: secret
  - name: gh
    type: github
    path: /github
    token: ghsecret
  - name: chat
    type: mattermost
    token: mmtoken
rules:
  - name: disk
    source: alerts
    when: "event.alerts | selectattr('labels.alertname', 'equalto', 'DiskFull') | list | length > 0"
    playbook: fix-disk.yml
    inventory: hosts.yml
    limit: "{{ event.alerts | map(attribute='labels.instance') | map('regex_replace', ':.*$', '') | join(',') }}"
    extra_vars: {severity: "{{ event.labels.severity }}"}
    become: true
    throttle: 10m
  - name: deploy
    source: gh
    when: "event.type == 'push' and event.ref == 'refs/heads/main'"
    playbook: deploy.yml
    extra_vars: {sha: "{{ event.after }}"}
  - name: chat-restart
    source: chat
    when: "event.args[0] == 'restart'"
    playbook: restart.yml
    limit: "{{ event.args[1] }}"
`

func TestListenConfigAndMatch(t *testing.T) {
	p := writeListenConfig(t, testListenYML)
	cfg, err := loadListenConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8085" || cfg.Sources[0].Path != "/alerts" || cfg.Sources[1].Path != "/github" {
		t.Errorf("defaults: %+v", cfg)
	}
	if !filepath.IsAbs(cfg.Rules[0].Playbook) || filepath.Base(cfg.Rules[0].Inventory) != "hosts.yml" {
		t.Errorf("paths are made absolute from the file's directory: %+v", cfg.Rules[0])
	}
	d := newDispatcher(cfg, func(*listenJob) (int, int, error) { return 0, 0, nil })
	event := map[string]interface{}{
		"status": "firing",
		"alerts": []interface{}{
			map[string]interface{}{"labels": map[string]interface{}{"alertname": "DiskFull", "instance": "web1:9100", "severity": "critical"}},
			map[string]interface{}{"labels": map[string]interface{}{"alertname": "DiskFull", "instance": "web2:9100", "severity": "critical"}},
		},
		"labels": map[string]interface{}{"alertname": "DiskFull", "severity": "critical"},
	}
	jobs, err := d.match("alerts", event, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].limit != "web1,web2" {
		t.Fatalf("jobs: %+v", jobs)
	}
	args := strings.Join(jobs[0].args, " ")
	if !strings.Contains(args, "--limit web1,web2") || !strings.Contains(args, "--become") || !strings.Contains(args, "-e severity=critical") {
		t.Errorf("args: %s", args)
	}
	// another alert does not match
	other := map[string]interface{}{"alerts": []interface{}{map[string]interface{}{"labels": map[string]interface{}{"alertname": "HighLoad"}}}}
	if none, _ := d.match("alerts", other, nil); len(none) != 0 {
		t.Errorf("HighLoad must not match: %+v", none)
	}
	// throttle: the same rule and limit once per period
	jobs, _ = d.match("alerts", event, nil)
	q, th := d.enqueue(jobs)
	if len(q) != 1 || len(th) != 0 {
		t.Fatalf("first enqueue: %v %v", q, th)
	}
	q, th = d.enqueue(jobs)
	if len(q) != 0 || len(th) != 1 {
		t.Errorf("second enqueue within the throttle: %v %v", q, th)
	}
	d.lastRun["disk\x00web1,web2"] = time.Now().Add(-11 * time.Minute)
	if q, _ := d.enqueue(jobs); len(q) != 1 {
		t.Errorf("after the period: %v", q)
	}
}

func TestListenHTTP(t *testing.T) {
	p := writeListenConfig(t, testListenYML)
	cfg, err := loadListenConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	var ran []string
	d := newDispatcher(cfg, func(j *listenJob) (int, int, error) {
		ran = append(ran, j.rule.Name+" "+strings.Join(j.args, " "))
		return 1, 0, nil
	})
	d.out = io.Discard
	srv := httptest.NewServer(d.mux())
	defer srv.Close()

	post := func(path, ct, body string, headers map[string]string) (int, map[string]interface{}) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", ct)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]interface{}
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	alert := `{"status":"firing","alerts":[{"labels":{"alertname":"DiskFull","instance":"db1:9100","severity":"warning"}}]}`
	if code, _ := post("/alerts", "application/json", alert, nil); code != http.StatusUnauthorized {
		t.Errorf("no token: %d", code)
	}
	if code, out := post("/alerts", "application/json", alert, map[string]string{"Authorization": "Bearer secret"}); code != http.StatusAccepted || out["matched"] != float64(1) {
		t.Errorf("alert: %d %v", code, out)
	}
	// github: HMAC checked, event type from the header
	push := `{"ref":"refs/heads/main","after":"abc123"}`
	mac := hmac.New(sha256.New, []byte("ghsecret"))
	mac.Write([]byte(push))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if code, _ := post("/github", "application/json", push, map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": "sha256=00"}); code != http.StatusUnauthorized {
		t.Errorf("bad signature: %d", code)
	}
	if code, out := post("/github", "application/json", push, map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": sig}); code != http.StatusAccepted || out["matched"] != float64(1) {
		t.Errorf("push: %d %v", code, out)
	}
	if _, out := post("/github", "application/json", push, map[string]string{"X-GitHub-Event": "ping", "X-Hub-Signature-256": sig}); out["matched"] != float64(0) {
		t.Errorf("ping must not match: %v", out)
	}
	// mattermost slash command as a form
	form := "token=mmtoken&trigger_word=/oni&text=/oni+restart+web3&user_name=denys"
	if code, out := post("/chat", "application/x-www-form-urlencoded", form, nil); code != http.StatusAccepted || out["matched"] != float64(1) {
		t.Errorf("chat: %d %v", code, out)
	}
	if code, _ := post("/chat", "application/x-www-form-urlencoded", strings.Replace(form, "mmtoken", "bad", 1), nil); code != http.StatusUnauthorized {
		t.Errorf("bad mattermost token: %d", code)
	}
	// the worker runs what was queued, in order
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go d.worker(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for len(ran) < 3 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if len(ran) != 3 || !strings.HasPrefix(ran[0], "disk ") || !strings.Contains(ran[1], "-e sha=abc123") || !strings.Contains(ran[2], "--limit web3") {
		t.Errorf("ran: %v", ran)
	}
	res, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	body := string(raw)
	if !strings.Contains(body, `onigirazu_listen_events_total{source="alerts"} 1`) || !strings.Contains(body, `onigirazu_listen_runs_total{result="ok",rule="disk"} 1`) {
		t.Errorf("metrics:\n%s", body)
	}
}
