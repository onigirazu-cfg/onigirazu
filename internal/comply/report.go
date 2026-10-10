package comply

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
	"time"
)

// Report is the outcome of a profile on a set of hosts
type Report struct {
	Profile   string                 `json:"profile"`
	Title     string                 `json:"title"`
	Time      time.Time              `json:"time"`
	Hosts     map[string]*HostResult `json:"hosts"`
	Errors    map[string]string      `json:"errors,omitempty"` // hosts that could not be checked
	Passed    int                    `json:"passed"`           // controls passed, over all hosts
	Failed    int                    `json:"failed"`
	Score     float64                `json:"score"` // passed / (passed+failed) in percent
	Threshold string                 `json:"fail_on,omitempty"`
}

// HostResult is one host's controls
type HostResult struct {
	Passed   int             `json:"passed"`
	Failed   int             `json:"failed"`
	Score    float64         `json:"score"`
	Controls []ControlResult `json:"controls"`
}

// ControlResult is one control on one host
type ControlResult struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Severity    string   `json:"severity"`
	Tags        []string `json:"tags,omitempty"`
	OK          bool     `json:"ok"`
	Details     []string `json:"details,omitempty"` // the failed checks' detail
	Remediation string   `json:"remediation,omitempty"`
}

// Build maps the verify module's check results (in the order of
// Profile.Checks) back to controls, per host
func Build(p *Profile, hostChecks map[string][]map[string]interface{}, errors map[string]string) *Report {
	_, index := p.Checks()
	r := &Report{Profile: p.Name, Title: p.Title, Time: time.Now().UTC(), Hosts: map[string]*HostResult{}, Errors: errors}
	for host, checks := range hostChecks {
		h := &HostResult{}
		results := make([]ControlResult, len(p.Controls))
		for i, c := range p.Controls {
			results[i] = ControlResult{ID: c.ID, Title: c.Title, Severity: strings.ToLower(c.Severity), Tags: c.Tags, OK: true, Remediation: c.Remediation}
		}
		for i, check := range checks {
			if i >= len(index) {
				break
			}
			ok, _ := check["ok"].(bool)
			if !ok {
				cr := &results[index[i]]
				cr.OK = false
				cr.Details = append(cr.Details, fmt.Sprintf("%v: %v", check["check"], check["detail"]))
			}
		}
		for _, cr := range results {
			if cr.OK {
				h.Passed++
			} else {
				h.Failed++
			}
		}
		h.Score = percent(h.Passed, h.Failed)
		h.Controls = results
		r.Hosts[host] = h
		r.Passed += h.Passed
		r.Failed += h.Failed
	}
	r.Score = percent(r.Passed, r.Failed)
	return r
}

func percent(ok, bad int) float64 {
	if ok+bad == 0 {
		return 100
	}
	return float64(ok) * 100 / float64(ok+bad)
}

// FailsAt reports whether a failed control reaches the severity threshold
func (r *Report) FailsAt(minSeverity string) bool {
	if minSeverity == "" {
		return r.Failed > 0
	}
	for _, h := range r.Hosts {
		for _, c := range h.Controls {
			if !c.OK && SeverityRank(c.Severity) >= SeverityRank(minSeverity) {
				return true
			}
		}
	}
	return false
}

func (r *Report) hostNames() []string {
	names := make([]string, 0, len(r.Hosts))
	for h := range r.Hosts {
		names = append(names, h)
	}
	sort.Strings(names)
	return names
}

func (r *Report) errorHosts() []string {
	names := make([]string, 0, len(r.Errors))
	for h := range r.Errors {
		names = append(names, h)
	}
	sort.Strings(names)
	return names
}

// WriteText prints the report for a terminal
func (r *Report) WriteText(w io.Writer) {
	fmt.Fprintf(w, "%s — %s\n", r.Profile, r.Title)
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(w, "\n%s: %d/%d controls pass (%.0f%%)\n", name, h.Passed, h.Passed+h.Failed, h.Score)
		for _, c := range h.Controls {
			if c.OK {
				continue
			}
			fmt.Fprintf(w, "  ✗ [%s] %s — %s\n", c.Severity, c.ID, c.Title)
			for _, d := range c.Details {
				fmt.Fprintf(w, "      %s\n", d)
			}
			if c.Remediation != "" {
				fmt.Fprintf(w, "      fix: %s\n", c.Remediation)
			}
		}
	}
	for _, name := range r.errorHosts() {
		fmt.Fprintf(w, "\n%s: could not check: %s\n", name, r.Errors[name])
	}
	fmt.Fprintf(w, "\n%d host(s), %d/%d controls pass (%.0f%%)\n", len(r.Hosts), r.Passed, r.Passed+r.Failed, r.Score)
}

// WriteJSON prints the report as JSON
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteMarkdown prints the report as Markdown (a pull request comment, a
// wiki page)
func (r *Report) WriteMarkdown(w io.Writer) {
	fmt.Fprintf(w, "## Compliance: %s\n\n%s — %d host(s), **%d/%d controls pass (%.0f%%)**\n\n", r.Profile, r.Title, len(r.Hosts), r.Passed, r.Passed+r.Failed, r.Score)
	fmt.Fprintln(w, "| Host | Pass | Fail | Score |\n|---|---|---|---|")
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(w, "| %s | %d | %d | %.0f%% |\n", name, h.Passed, h.Failed, h.Score)
	}
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		if h.Failed == 0 {
			continue
		}
		fmt.Fprintf(w, "\n### %s\n\n| Control | Severity | Detail | Fix |\n|---|---|---|---|\n", name)
		for _, c := range h.Controls {
			if c.OK {
				continue
			}
			fmt.Fprintf(w, "| %s %s | %s | %s | %s |\n", c.ID, mdEscape(c.Title), c.Severity, mdEscape(strings.Join(c.Details, "; ")), mdEscape(c.Remediation))
		}
	}
	for _, name := range r.errorHosts() {
		fmt.Fprintf(w, "\n%s: could not check: %s\n", name, mdEscape(r.Errors[name]))
	}
}

func mdEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
}

// WriteHTML prints a self-contained HTML page
func (r *Report) WriteHTML(w io.Writer) {
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>Compliance: %s</title>
<style>body{font-family:system-ui,sans-serif;margin:2em;color:#222}table{border-collapse:collapse;margin:1em 0}td,th{border:1px solid #ccc;padding:.3em .6em;text-align:left;vertical-align:top}
.ok{color:#2a7}.bad{color:#c33}.critical,.high{font-weight:bold}.score{font-size:1.4em}</style></head><body>
<h1>Compliance: %s</h1><p>%s</p><p class="score">%d host(s), %d/%d controls pass (%.0f%%) — %s</p>
`, html.EscapeString(r.Profile), html.EscapeString(r.Profile), html.EscapeString(r.Title), len(r.Hosts), r.Passed, r.Passed+r.Failed, r.Score, r.Time.Format(time.RFC3339))
	fmt.Fprintln(w, "<table><tr><th>Host</th><th>Pass</th><th>Fail</th><th>Score</th></tr>")
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(w, "<tr><td>%s</td><td class=\"ok\">%d</td><td class=\"bad\">%d</td><td>%.0f%%</td></tr>\n", html.EscapeString(name), h.Passed, h.Failed, h.Score)
	}
	fmt.Fprintln(w, "</table>")
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(w, "<h2>%s</h2><table><tr><th>Control</th><th>Severity</th><th>Result</th><th>Detail</th><th>Fix</th></tr>\n", html.EscapeString(name))
		for _, c := range h.Controls {
			state, cls := "pass", "ok"
			if !c.OK {
				state, cls = "fail", "bad"
			}
			fmt.Fprintf(w, "<tr><td>%s %s</td><td class=\"%s\">%s</td><td class=\"%s\">%s</td><td>%s</td><td>%s</td></tr>\n",
				html.EscapeString(c.ID), html.EscapeString(c.Title), html.EscapeString(c.Severity), html.EscapeString(c.Severity), cls, state,
				html.EscapeString(strings.Join(c.Details, "; ")), html.EscapeString(c.Remediation))
		}
		fmt.Fprintln(w, "</table>")
	}
	for _, name := range r.errorHosts() {
		fmt.Fprintf(w, "<p class=\"bad\">%s: could not check: %s</p>\n", html.EscapeString(name), html.EscapeString(r.Errors[name]))
	}
	fmt.Fprintln(w, "</body></html>")
}

// Metrics is the report as Prometheus text: controls per host and result,
// and the score
func (r *Report) Metrics(extra map[string]string) string {
	var b strings.Builder
	labels := func(kv ...string) string {
		parts := make([]string, 0, len(kv)/2+len(extra))
		for i := 0; i+1 < len(kv); i += 2 {
			parts = append(parts, fmt.Sprintf("%s=%q", kv[i], kv[i+1]))
		}
		keys := make([]string, 0, len(extra))
		for k := range extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%q", k, extra[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	b.WriteString("# TYPE onigirazu_comply_controls gauge\n# TYPE onigirazu_comply_score gauge\n# TYPE onigirazu_comply_last_run_timestamp_seconds gauge\n")
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(&b, "onigirazu_comply_controls%s %d\n", labels("profile", r.Profile, "host", name, "result", "pass"), h.Passed)
		fmt.Fprintf(&b, "onigirazu_comply_controls%s %d\n", labels("profile", r.Profile, "host", name, "result", "fail"), h.Failed)
		fmt.Fprintf(&b, "onigirazu_comply_score%s %g\n", labels("profile", r.Profile, "host", name), h.Score)
	}
	fmt.Fprintf(&b, "onigirazu_comply_last_run_timestamp_seconds%s %d\n", labels("profile", r.Profile), r.Time.Unix())
	return b.String()
}
