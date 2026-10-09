package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDriftMarkdown(t *testing.T) {
	r := &DriftReport{Playbook: "site.yml", Plan: true, CheckedAt: time.Unix(0, 0), Hosts: 2, InSync: []string{"db1"}, DriftTasks: 2,
		Drift: map[string][]DriftItem{"web1": {{Task: "nginx.conf", Module: "template", Diff: "--- a\n+++ b\n-x\n+y"}, {Task: "Restart nginx", Module: "service", Detail: "would be restarted"}}}}
	var b strings.Builder
	writeDriftMarkdown(&b, r)
	out := b.String()
	for _, want := range []string{"<!-- onigirazu-plan: site.yml -->", "### Plan: 2 task(s) would change on 1 of 2 host(s)", "**web1**", "- `nginx.conf` (template)", "```diff", "  +y", "- `Restart nginx` (service): would be restarted", "Unchanged: db1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	b.Reset()
	writeDriftMarkdown(&b, &DriftReport{Playbook: "site.yml", Plan: true, Hosts: 3})
	if !strings.Contains(b.String(), "no changes — 3 host(s) already match") {
		t.Errorf("no-change summary:\n%s", b.String())
	}
}

func TestGithubCommentReplacesOwn(t *testing.T) {
	var posted, patched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/o/r/issues/7/comments":
			_, _ = w.Write([]byte(`[{"id": 1, "body": "hello"}, {"id": 2, "body": "<!-- onigirazu-plan: site.yml -->\nold"}]`))
		case r.Method == "PATCH" && r.URL.Path == "/repos/o/r/issues/comments/2":
			patched = body["body"]
		case r.Method == "POST" && r.URL.Path == "/repos/o/r/issues/7/comments":
			posted = body["body"]
			w.WriteHeader(201)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	ev := filepath.Join(t.TempDir(), "event.json")
	_ = os.WriteFile(ev, []byte(`{"pull_request": {"number": 7}}`), 0o600)
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_REPOSITORY", "o/r")
	t.Setenv("GITHUB_API_URL", srv.URL)
	t.Setenv("GITHUB_EVENT_PATH", ev)
	if err := githubComment("<!-- onigirazu-plan: site.yml -->\nnew", "<!-- onigirazu-plan: site.yml -->"); err != nil {
		t.Fatal(err)
	}
	if patched != "<!-- onigirazu-plan: site.yml -->\nnew" || posted != "" {
		t.Fatalf("expected the old comment replaced: patched=%q posted=%q", patched, posted)
	}
	if err := githubComment("x", "<!-- onigirazu-plan: other.yml -->"); err != nil || posted != "x" {
		t.Fatalf("expected a new comment: %v posted=%q", err, posted)
	}
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	if err := githubComment("x", "m"); err == nil {
		t.Error("expected an error outside a pull request")
	}
}
