package comply

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
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
	return build(p, hostChecks, errors, false)
}

// BuildPartial scores only controls whose checks finished before the run stopped.
func BuildPartial(p *Profile, hostChecks map[string][]map[string]interface{}, errors map[string]string) *Report {
	return build(p, hostChecks, errors, true)
}

func build(p *Profile, hostChecks map[string][]map[string]interface{}, errors map[string]string, partial bool) *Report {
	_, index := p.Checks()
	r := &Report{Profile: p.Name, Title: p.Title, Time: time.Now().UTC(), Hosts: map[string]*HostResult{}, Errors: errors}
	for host, checks := range hostChecks {
		h := &HostResult{}
		results := make([]ControlResult, len(p.Controls))
		completed := make([]int, len(p.Controls))
		expected := make([]int, len(p.Controls))
		for _, control := range index {
			expected[control]++
		}
		for i, c := range p.Controls {
			results[i] = ControlResult{ID: c.ID, Title: c.Title, Severity: strings.ToLower(c.Severity), Tags: c.Tags, OK: true, Remediation: c.Remediation}
		}
		for i, check := range checks {
			if i >= len(index) {
				break
			}
			if partial && check["detail"] == "no result" {
				continue
			}
			completed[index[i]]++
			ok, _ := check["ok"].(bool)
			if !ok {
				cr := &results[index[i]]
				cr.OK = false
				cr.Details = append(cr.Details, fmt.Sprintf("%v: %v", check["check"], check["detail"]))
			}
		}
		for i, cr := range results {
			if partial && completed[i] != expected[i] {
				continue
			}
			h.Controls = append(h.Controls, cr)
			if cr.OK {
				h.Passed++
			} else {
				h.Failed++
			}
		}
		if partial && len(h.Controls) == 0 {
			continue
		}
		h.Score = percent(h.Passed, h.Failed)
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
	fmt.Fprintf(w, "\n%d %s, %d/%d controls pass (%.0f%%)\n", len(r.Hosts), brand.Plural(len(r.Hosts), "host", "hosts"), r.Passed, r.Passed+r.Failed, r.Score)
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
	fmt.Fprintf(w, "## Compliance: %s\n\n%s — %d %s, **%d/%d controls pass (%.0f%%)**\n\n", r.Profile, r.Title, len(r.Hosts), brand.Plural(len(r.Hosts), "host", "hosts"), r.Passed, r.Passed+r.Failed, r.Score)
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
	scoreTone := brand.Tone("pass")
	if r.Failed > 0 {
		scoreTone = brand.Tone("fail")
	}
	fmt.Fprintf(w, `<!doctype html><html lang="en"><head>%s<title>Compliance: %s</title></head><body><main class="onz-page">
%s<h1 class="onz-title">Compliance: %s</h1><p class="onz-sub">%s · Checked %s</p>
<div class="onz-stats"><div class="onz-stat %s"><b>%.0f%%</b><span>Score</span></div><div class="onz-stat"><b>%d</b><span>Hosts</span></div><div class="onz-stat %s"><b>%d</b><span>Controls passed</span></div><div class="onz-stat %s"><b>%d</b><span>Controls failed</span></div></div>
`, brand.ReportHead, html.EscapeString(r.Profile), brand.Header("Compliance: "+r.Profile), html.EscapeString(r.Profile), html.EscapeString(r.Title), html.EscapeString(r.Time.Local().Format(brand.ReportTime)), scoreTone, r.Score, len(r.Hosts), brand.StatTone(r.Passed, "ok"), r.Passed, brand.StatTone(r.Failed, "bad"), r.Failed)
	r.writeControlMatrix(w)
	fmt.Fprintln(w, `<div class="onz-scroll onz-summary"><table class="onz-table"><tr><th>Host</th><th class="num">Pass</th><th class="num">Fail</th><th class="num">Score</th></tr>`)
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(w, `<tr><td>%s</td><td class="num">%d</td><td class="num">%d</td><td class="num">%.0f%%</td></tr>`, html.EscapeString(name), h.Passed, h.Failed, h.Score)
	}
	fmt.Fprintln(w, "</table></div>")
	for _, name := range r.hostNames() {
		h := r.Hosts[name]
		fmt.Fprintf(w, `<section class="onz-card"><h2>%s</h2><div class="onz-scroll"><table class="onz-table"><tr><th>Control</th><th>Severity</th><th>Result</th><th>Details</th><th>Remediation</th></tr>`, html.EscapeString(name))
		for _, c := range h.Controls {
			result := `<span class="muted">pass</span>`
			if !c.OK {
				result = `<span class="onz-badge bad">fail</span>`
			}
			fmt.Fprintf(w, `<tr><td>%s %s</td><td><span class="onz-badge %s">%s</span></td><td>%s</td><td>%s</td><td>%s</td></tr>`,
				html.EscapeString(c.ID), html.EscapeString(c.Title), brand.SeverityTone(c.Severity), html.EscapeString(c.Severity), result,
				html.EscapeString(strings.Join(c.Details, "; ")), html.EscapeString(c.Remediation))
		}
		fmt.Fprintln(w, "</table></div></section>")
	}
	for _, name := range r.errorHosts() {
		fmt.Fprintf(w, `<section class="onz-card"><h2>%s <span class="onz-badge bad">could not be checked</span></h2><div class="onz-body">%s</div></section>`, html.EscapeString(name), html.EscapeString(r.Errors[name]))
	}
	fmt.Fprintf(w, `<footer class="onz-foot">%s</footer></main></body></html>`, html.EscapeString(brand.ReportFooter()))
}

// writeControlMatrix uses the union of observed controls, so partial reports
// distinguish missing results from passes. Error hosts always remain unchecked.
func (r *Report) writeControlMatrix(w io.Writer) {
	controls := map[string]string{}
	names := map[string]bool{}
	results := map[string]map[string]bool{}
	for name, host := range r.Hosts {
		names[name] = true
		results[name] = map[string]bool{}
		if host == nil {
			continue
		}
		for _, c := range host.Controls {
			// Choose deterministically even if host metadata differs.
			if title, ok := controls[c.ID]; !ok || c.Title < title {
				controls[c.ID] = c.Title
			}
			results[name][c.ID] = c.OK
		}
	}
	for name := range r.Errors {
		names[name] = true
	}
	hosts := make([]string, 0, len(names))
	for name := range names {
		hosts = append(hosts, name)
	}
	sort.Strings(hosts)
	// Rows follow the profile. Every host lists its controls in profile order,
	// but a partial host may skip some, so merge: a control missing so far goes
	// right after the control that precedes it on that host.
	ids := make([]string, 0, len(controls))
	for _, name := range hosts {
		host := r.Hosts[name]
		if host == nil {
			continue
		}
		at := 0
		for _, c := range host.Controls {
			if i := slices.Index(ids, c.ID); i >= 0 {
				at = i + 1
				continue
			}
			ids = slices.Insert(ids, at, c.ID)
			at++
		}
	}
	if len(hosts) == 0 || len(ids) == 0 {
		return
	}
	fmt.Fprint(w, `<section class="onz-card"><h2>Controls across hosts</h2><div class="onz-body onz-legend"><span><span class="onz-square"></span>Passed</span><span><span class="onz-square fail"></span>Failed</span><span><span class="onz-square unchecked"></span>Not checked or unreachable</span></div><div class="onz-scroll"><table class="onz-table onz-matrix"><thead><tr><th scope="col">Control</th>`)
	for _, name := range hosts {
		fmt.Fprintf(w, `<th scope="col">%s</th>`, html.EscapeString(name))
	}
	fmt.Fprint(w, `</tr></thead><tbody>`)
	for _, id := range ids {
		label := id + " " + controls[id]
		fmt.Fprintf(w, `<tr><th scope="row">%s</th>`, html.EscapeString(label))
		for _, name := range hosts {
			class, result := "unchecked", "not checked"
			if _, unreachable := r.Errors[name]; unreachable {
				result = "unreachable"
			} else if ok, checked := results[name][id]; checked {
				if ok {
					class, result = "", "pass"
				} else {
					class, result = "fail", "fail"
				}
			}
			title := html.EscapeString(label + ": " + result)
			fmt.Fprintf(w, `<td><span class="onz-square %s" role="img" title="%s" aria-label="%s"></span></td>`, class, title, html.EscapeString(name+": "+label+": "+result))
		}
		fmt.Fprint(w, `</tr>`)
	}
	fmt.Fprint(w, `</tbody></table></div></section>`)
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
