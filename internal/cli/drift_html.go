package cli

import (
	"html/template"
	"io"
	"sort"
	"strings"
)

// writeDriftHTML writes the report as one self-contained HTML page (a CI
// artifact or a file to share): hosts with their drifting tasks and diffs
func writeDriftHTML(w io.Writer, r *DriftReport) error {
	type line struct{ Class, Text string }
	type item struct {
		DriftItem
		Lines []line
	}
	type host struct {
		Name   string
		Items  []item
		Errors []DriftItem
	}
	var hosts []host
	names := map[string]bool{}
	for h := range r.Drift {
		names[h] = true
	}
	for h := range r.Errors {
		names[h] = true
	}
	for h := range names {
		entry := host{Name: h, Errors: r.Errors[h]}
		for _, it := range r.Drift[h] {
			var lines []line
			for _, l := range strings.Split(strings.TrimRight(it.Diff, "\n"), "\n") {
				class := ""
				switch {
				case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"):
					class = "hdr"
				case strings.HasPrefix(l, "+"):
					class = "add"
				case strings.HasPrefix(l, "-"):
					class = "del"
				case strings.HasPrefix(l, "@@"):
					class = "hunk"
				}
				if l != "" {
					lines = append(lines, line{class, l})
				}
			}
			entry.Items = append(entry.Items, item{it, lines})
		}
		hosts = append(hosts, entry)
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	title := "Drift"
	if r.Plan {
		title = "Plan"
	}
	return driftHTML.Execute(w, map[string]interface{}{"R": r, "Hosts": hosts, "Title": title})
}

var driftHTML = template.Must(template.New("drift").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}: {{.R.Playbook}}</title>
<style>
:root{--bg:#fff;--fg:#1f2328;--muted:#656d76;--line:#d0d7de;--add:#dafbe1;--del:#ffebe9;--hunk:#ddf4ff;--ok:#1a7f37;--warn:#9a6700;--bad:#cf222e}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3;--muted:#8d96a0;--line:#30363d;--add:#12361f;--del:#4c1a1d;--hunk:#0c2d6b;--ok:#3fb950;--warn:#d29922;--bad:#f85149}}
body{background:var(--bg);color:var(--fg);font:14px/1.5 system-ui,sans-serif;margin:0 auto;max-width:1100px;padding:16px}
h1{font-size:20px;margin:0 0 4px}.muted{color:var(--muted)}
.summary{display:flex;gap:16px;flex-wrap:wrap;margin:12px 0 20px}.summary div{border:1px solid var(--line);border-radius:6px;padding:8px 12px}
.ok{color:var(--ok)}.warn{color:var(--warn)}.bad{color:var(--bad)}
section{border:1px solid var(--line);border-radius:6px;margin:0 0 16px;overflow:hidden}
section h2{font-size:15px;margin:0;padding:8px 12px;border-bottom:1px solid var(--line)}
.task{padding:8px 12px;border-top:1px solid var(--line)}.task:first-of-type{border-top:0}
pre{margin:6px 0 0;overflow-x:auto;font:12px/1.45 ui-monospace,monospace}
pre span{display:block;padding:0 6px}.add{background:var(--add)}.del{background:var(--del)}.hunk{background:var(--hunk)}.hdr{color:var(--muted)}
</style></head><body>
<h1>{{.Title}}: {{.R.Playbook}}</h1>
<div class="muted">checked {{.R.CheckedAt.Format "2006-01-02 15:04:05 MST"}}</div>
<div class="summary">
<div>{{.R.Hosts}} host(s)</div>
<div class="{{if .R.Drift}}warn{{else}}ok{{end}}">{{len .R.Drift}} with changes, {{.R.DriftTasks}} task(s)</div>
<div class="ok">{{len .R.InSync}} in sync</div>
{{if .R.Errors}}<div class="bad">{{len .R.Errors}} could not be checked</div>{{end}}
{{if .R.Fixed}}<div class="ok">fixed: the playbook was applied</div>{{end}}
</div>
{{range .Hosts}}<section><h2>{{.Name}}</h2>
{{range .Items}}<div class="task"><strong>{{.Task}}</strong>{{if .Module}} <span class="muted">({{.Module}})</span>{{end}}{{if and .Detail (not .Lines)}} — {{.Detail}}{{end}}
{{if .Lines}}<pre>{{range .Lines}}<span class="{{.Class}}">{{.Text}}</span>{{end}}</pre>{{end}}</div>{{end}}
{{range .Errors}}<div class="task bad"><strong>{{.Task}}</strong>: {{.Detail}}</div>{{end}}
</section>{{end}}
{{if .R.InSync}}<p class="muted">In sync: {{range $i, $h := .R.InSync}}{{if $i}}, {{end}}{{$h}}{{end}}</p>{{end}}
</body></html>
`))
