package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestBuildDriftReport(t *testing.T) {
	result := &types.PlaybookResult{Plays: []types.PlayResult{{Name: "p", Hosts: []types.HostResult{
		{Host: "web1", Tasks: []types.TaskResult{
			{TaskName: "conf", Module: "copy", Changed: true, Output: map[string]interface{}{"msg": "file would be updated"}},
			{TaskName: "pkg", Module: "apt"},
		}},
		{Host: "web2", Tasks: []types.TaskResult{{TaskName: "conf", Module: "copy"}}},
		{Host: "db", Tasks: []types.TaskResult{{TaskName: "probe", Failed: true, Error: "unreachable"}}},
	}}}}
	r := buildDriftReport("site.yml", result)
	assert.Equal(t, 3, r.Hosts)
	assert.Equal(t, []string{"web2"}, r.InSync)
	assert.Equal(t, []DriftItem{{Play: "p", Task: "conf", Module: "copy", Detail: "file would be updated"}}, r.Drift["web1"])
	assert.Equal(t, "unreachable", r.Errors["db"][0].Detail)

	var out bytes.Buffer
	writeDriftText(&out, r)
	assert.Contains(t, out.String(), "Drift: 1 task(s) on 1 of 3 host(s)")
	assert.Contains(t, out.String(), "~ conf (copy): file would be updated")
	assert.Contains(t, out.String(), "! db: probe: unreachable")
}

func TestDriftApplyArgs(t *testing.T) {
	o := driftCheckOptions{limit: "web", extraVars: []string{"a=1"}, become: true}
	args := o.applyArgs("site.yml", true)
	assert.Equal(t, []string{"site.yml", "--check", "--diff", "-e", "a=1"}, args[:5])
	assert.Contains(t, args, "--limit")
	assert.Contains(t, args, "--become")
	assert.NotContains(t, o.applyArgs("site.yml", false), "--check")
}

func TestPlanWording(t *testing.T) {
	r := &DriftReport{Playbook: "site.yml", Hosts: 2, Plan: true, Drift: map[string][]DriftItem{"web1": {{Task: "conf"}}}, DriftTasks: 1}
	var out bytes.Buffer
	writeDriftText(&out, r)
	assert.Contains(t, out.String(), "Plan: 1 task(s) would change 1 of 2 host(s)")
	out.Reset()
	writeDriftText(&out, &DriftReport{Playbook: "site.yml", Hosts: 2, Plan: true})
	assert.Contains(t, out.String(), "No changes: 2 host(s) already match site.yml")
}

func TestNotifyWebhook(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
	}))
	defer srv.Close()
	r := &DriftReport{Playbook: "site.yml", Hosts: 1, Drift: map[string][]DriftItem{"web1": {{Task: "conf"}}}, DriftTasks: 1}
	assert.NoError(t, notifyWebhook(srv.URL+"/hooks/secret", r))
	assert.Contains(t, got["text"], "Drift: 1 task(s) on 1 of 1 host(s)")
	assert.NotNil(t, got["report"])
	assert.Equal(t, "https://chat.example/…", redactURL("https://chat.example/hooks/abc123"))
}

func TestWriteDriftHTML(t *testing.T) {
	r := &DriftReport{Playbook: "site.yml", Hosts: 2, InSync: []string{"db1"}, DriftTasks: 1,
		Drift:  map[string][]DriftItem{"web1": {{Task: "conf <x>", Module: "copy", Diff: "--- before: /a\n+++ after: /a\n-old\n+new\n"}}},
		Errors: map[string][]DriftItem{"web2": {{Task: "probe", Detail: "unreachable"}}}}
	var out bytes.Buffer
	assert.NoError(t, writeDriftHTML(&out, r))
	html := out.String()
	assert.Contains(t, html, "<title>Drift: site.yml</title>")
	assert.Contains(t, html, `<span class="add">&#43;new</span>`)
	assert.Contains(t, html, `<span class="del">-old</span>`)
	assert.Contains(t, html, "conf &lt;x&gt;", "task names are escaped")
	assert.Contains(t, html, "unreachable")
	assert.Contains(t, html, "In sync: db1")
}

func TestAnnotateSince(t *testing.T) {
	t0 := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	drift := func(at time.Time, hosts ...string) historyEntry {
		r := DriftReport{CheckedAt: at, Drift: map[string][]DriftItem{}}
		for _, h := range hosts {
			r.Drift[h] = []DriftItem{{Task: "conf"}}
		}
		return historyEntry{DriftReport: r}
	}
	inSync := func(at time.Time, host string) historyEntry {
		return historyEntry{DriftReport: DriftReport{CheckedAt: at, InSync: []string{host}}}
	}
	history := []historyEntry{
		drift(t0, "web1"),
		inSync(t0.Add(time.Hour), "web1"),  // fixed: breaks the run
		drift(t0.Add(2*time.Hour), "web1"), // drifts again
		drift(t0.Add(3*time.Hour), "db1"),  // did not look at web1
		drift(t0.Add(4*time.Hour), "web1", "db1"),
	}
	now := DriftReport{CheckedAt: t0.Add(5 * time.Hour), Drift: map[string][]DriftItem{
		"web1": {{Task: "conf"}}, "web2": {{Task: "conf"}}}}
	annotateSince(&now, history)
	assert.Equal(t, t0.Add(2*time.Hour), *now.Drift["web1"][0].Since)
	assert.Equal(t, now.CheckedAt, *now.Drift["web2"][0].Since, "first seen now")
	assert.Equal(t, "since 3h", sinceText(*now.Drift["web1"][0].Since, now.CheckedAt))
}

func TestDriftHistoryStore(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		r := &DriftReport{CheckedAt: time.Now().Add(time.Duration(i) * time.Second)}
		assert.NoError(t, saveDriftHistory(dir, "/a/site.yml", r))
	}
	assert.NoError(t, saveDriftHistory(dir, "/b/site.yml", &DriftReport{CheckedAt: time.Now()}))
	h, err := loadDriftHistory(dir, "/a/site.yml")
	assert.NoError(t, err)
	assert.Len(t, h, 3)
	assert.True(t, h[0].CheckedAt.Before(h[2].CheckedAt))
	var out bytes.Buffer
	writeDriftHistory(&out, "site.yml", h)
	assert.Equal(t, 4, strings.Count(out.String(), "\n"))
}
