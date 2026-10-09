package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Metrics of a drift check or a pull run in the Prometheus text format:
// written to a file for node_exporter's textfile collector, or pushed to
// VictoriaMetrics (/api/v1/import/prometheus) or a Pushgateway.

// promLabels renders labels sorted by name, with the values escaped
func promLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(labels[k])
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func promLine(b *strings.Builder, name string, labels map[string]string, value float64) {
	fmt.Fprintf(b, "%s%s %g\n", name, promLabels(labels), value)
}

// withLabels is labels plus extra, extra winning
func withLabels(labels, extra map[string]string) map[string]string {
	out := make(map[string]string, len(labels)+len(extra))
	for k, v := range labels {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// parseLabels reads "name=value" pairs (--metrics-label)
func parseLabels(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("--metrics-label %q: want name=value", p)
		}
		out[k] = v
	}
	return out, nil
}

// driftMetrics renders a drift report: per host whether it is in sync and
// how many tasks drift or fail, and the check's time
func driftMetrics(r *DriftReport, extra map[string]string) string {
	var b strings.Builder
	pb := withLabels(map[string]string{"playbook": r.Playbook}, extra)
	b.WriteString("# HELP onigirazu_drift_check_timestamp_seconds When the playbook was last checked.\n# TYPE onigirazu_drift_check_timestamp_seconds gauge\n")
	promLine(&b, "onigirazu_drift_check_timestamp_seconds", pb, float64(r.CheckedAt.Unix()))
	b.WriteString("# HELP onigirazu_drift_hosts Hosts the check looked at.\n# TYPE onigirazu_drift_hosts gauge\n")
	promLine(&b, "onigirazu_drift_hosts", pb, float64(r.Hosts))
	b.WriteString("# HELP onigirazu_drift_orphans Managed resources no task claims any more.\n# TYPE onigirazu_drift_orphans gauge\n")
	promLine(&b, "onigirazu_drift_orphans", pb, float64(len(r.Orphans)))
	b.WriteString("# HELP onigirazu_drift_fixed 1 when the check applied the playbook (--fix).\n# TYPE onigirazu_drift_fixed gauge\n")
	promLine(&b, "onigirazu_drift_fixed", pb, map[bool]float64{true: 1}[r.Fixed])
	hosts := map[string]bool{}
	for _, h := range r.InSync {
		hosts[h] = true
	}
	for h := range r.Drift {
		hosts[h] = true
	}
	for h := range r.Errors {
		hosts[h] = true
	}
	names := make([]string, 0, len(hosts))
	for h := range hosts {
		names = append(names, h)
	}
	sort.Strings(names)
	b.WriteString("# HELP onigirazu_drift_host_in_sync 1 when nothing on the host would change and no task failed.\n# TYPE onigirazu_drift_host_in_sync gauge\n")
	for _, h := range names {
		inSync := len(r.Drift[h]) == 0 && len(r.Errors[h]) == 0
		promLine(&b, "onigirazu_drift_host_in_sync", withLabels(pb, map[string]string{"host": h}), map[bool]float64{true: 1}[inSync])
	}
	b.WriteString("# HELP onigirazu_drift_tasks Tasks that would change on the host.\n# TYPE onigirazu_drift_tasks gauge\n")
	for _, h := range names {
		promLine(&b, "onigirazu_drift_tasks", withLabels(pb, map[string]string{"host": h}), float64(len(r.Drift[h])))
	}
	b.WriteString("# HELP onigirazu_drift_errors Tasks that failed on the host during the check.\n# TYPE onigirazu_drift_errors gauge\n")
	for _, h := range names {
		promLine(&b, "onigirazu_drift_errors", withLabels(pb, map[string]string{"host": h}), float64(len(r.Errors[h])))
	}
	return b.String()
}

// pullMetrics renders a pull run on this host
func pullMetrics(repo, playbook, commit string, changed, failed int, driftOnly bool, extra map[string]string) string {
	var b strings.Builder
	l := withLabels(map[string]string{"repo": repo, "playbook": playbook}, extra)
	b.WriteString("# HELP onigirazu_pull_last_run_timestamp_seconds When pull last ran the playbook.\n# TYPE onigirazu_pull_last_run_timestamp_seconds gauge\n")
	promLine(&b, "onigirazu_pull_last_run_timestamp_seconds", l, float64(time.Now().Unix()))
	b.WriteString("# HELP onigirazu_pull_ok 1 when no task failed.\n# TYPE onigirazu_pull_ok gauge\n")
	promLine(&b, "onigirazu_pull_ok", l, map[bool]float64{true: 1}[failed == 0])
	b.WriteString("# HELP onigirazu_pull_failed_tasks Tasks that failed in the last run.\n# TYPE onigirazu_pull_failed_tasks gauge\n")
	promLine(&b, "onigirazu_pull_failed_tasks", l, float64(failed))
	if driftOnly {
		b.WriteString("# HELP onigirazu_pull_drift_tasks Tasks that would change (--drift-only).\n# TYPE onigirazu_pull_drift_tasks gauge\n")
		promLine(&b, "onigirazu_pull_drift_tasks", l, float64(changed))
	} else {
		b.WriteString("# HELP onigirazu_pull_changed_tasks Tasks the last run changed.\n# TYPE onigirazu_pull_changed_tasks gauge\n")
		promLine(&b, "onigirazu_pull_changed_tasks", l, float64(changed))
	}
	b.WriteString("# HELP onigirazu_pull_commit_info The commit the last run used.\n# TYPE onigirazu_pull_commit_info gauge\n")
	promLine(&b, "onigirazu_pull_commit_info", withLabels(l, map[string]string{"commit": commit}), 1)
	return b.String()
}

// writeMetricsFile replaces the file in one step (node_exporter reads it)
func writeMetricsFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { // #nosec G301 -- a textfile collector directory
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil { // #nosec G306 -- metrics are world-readable
		return err
	}
	return os.Rename(tmp, path)
}

// pushMetrics posts the text to VictoriaMetrics' /api/v1/import/prometheus
// or a Pushgateway's /metrics/job/<job>
func pushMetrics(url, text string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "text/plain; version=0.0.4", bytes.NewReader([]byte(text))) // #nosec G107 -- the user names the endpoint
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// emitMetrics writes and/or pushes; problems are reported, never fatal
func emitMetrics(text, file, push string) {
	if file != "" {
		if err := writeMetricsFile(file, text); err != nil {
			fmt.Fprintf(os.Stderr, "metrics file %s: %v\n", file, err)
		}
	}
	if push != "" {
		if err := pushMetrics(push, text); err != nil {
			fmt.Fprintf(os.Stderr, "metrics push %s: %v\n", redactURL(push), err)
		}
	}
}
