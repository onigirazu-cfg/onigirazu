package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDriftMetricsText(t *testing.T) {
	r := &DriftReport{Playbook: "site.yml", CheckedAt: time.Unix(1700000000, 0), Hosts: 3, InSync: []string{"a"},
		Drift: map[string][]DriftItem{"b": {{Task: "x"}, {Task: "y"}}}, Errors: map[string][]DriftItem{"c": {{Task: "z"}}}}
	text := driftMetrics(r, map[string]string{"site": "red"})
	for _, want := range []string{
		`onigirazu_drift_check_timestamp_seconds{playbook="site.yml",site="red"} 1.7e+09`,
		`onigirazu_drift_hosts{playbook="site.yml",site="red"} 3`,
		`onigirazu_drift_host_in_sync{host="a",playbook="site.yml",site="red"} 1`,
		`onigirazu_drift_host_in_sync{host="b",playbook="site.yml",site="red"} 0`,
		`onigirazu_drift_tasks{host="b",playbook="site.yml",site="red"} 2`,
		`onigirazu_drift_errors{host="c",playbook="site.yml",site="red"} 1`,
		`onigirazu_drift_tasks{host="a",playbook="site.yml",site="red"} 0`,
	} {
		if !strings.Contains(text, want+"\n") {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if !strings.Contains(pullMetrics("git@x:r.git", "site.yml", "abc", 2, 0, false, nil), `onigirazu_pull_changed_tasks{playbook="site.yml",repo="git@x:r.git"} 2`) {
		t.Error("pull metrics")
	}
	if !strings.Contains(pullMetrics("r", "p", "abc", 1, 0, true, nil), "onigirazu_pull_drift_tasks") {
		t.Error("drift-only metric")
	}
	if _, err := parseLabels([]string{"novalue"}); err == nil {
		t.Error("expected an error for a label without =")
	}
}

func TestMetricsFileAndPush(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collector", "onigirazu.prom")
	if err := writeMetricsFile(path, "a 1\n"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "a 1\n" {
		t.Fatalf("file: %q", b)
	}
	var got string
	var ctype string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got, ctype = string(b), r.Header.Get("Content-Type")
		w.WriteHeader(204)
	}))
	defer srv.Close()
	if err := pushMetrics(srv.URL+"/api/v1/import/prometheus", "b 2\n"); err != nil {
		t.Fatal(err)
	}
	if got != "b 2\n" || !strings.HasPrefix(ctype, "text/plain") {
		t.Fatalf("pushed %q %q", got, ctype)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer bad.Close()
	if err := pushMetrics(bad.URL, "x 1\n"); err == nil {
		t.Error("expected an error on HTTP 500")
	}
}
