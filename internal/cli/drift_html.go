package cli

import (
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

// writeDriftHTML writes the report as one self-contained HTML page (a CI
// artifact or a file to share): hosts with their drifting tasks and diffs
func writeDriftHTML(w io.Writer, r *DriftReport) error {
	type line struct{ Class, Text string }
	type item struct {
		DriftItem
		Lines []line
	}
	type orphan struct {
		Mark, Class, Text string
	}
	type host struct {
		Name    string
		Items   []item
		Errors  []DriftItem
		Orphans []orphan
	}
	var hosts []host
	names := map[string]bool{}
	for h := range r.Drift {
		names[h] = true
	}
	for h := range r.Errors {
		names[h] = true
	}
	orphans := map[string][]orphan{}
	for _, o := range r.Orphans {
		names[o.Host] = true
		text := o.Type + " " + o.ID + ": " + orphanText[o.Action]
		if o.Task != "" {
			text += " (was: " + o.Task + ")"
		}
		orphans[o.Host] = append(orphans[o.Host], orphan{orphanMarks[o.Action], map[string]string{
			"destroy": "del", "restore": "hunk"}[o.Action], text})
	}
	for h := range names {
		entry := host{Name: h, Errors: r.Errors[h], Orphans: orphans[h]}
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

var driftHTML = template.Must(template.New("drift").Funcs(template.FuncMap{"tone": brand.Tone}).Parse(`<!doctype html>
<html lang="en"><head>` + brand.ReportHead + `<title>{{.Title}}: {{.R.Playbook}}</title></head><body><main class="onz-page">
` + brand.HeaderOpen + `{{.Title}}: {{.R.Playbook}}` + brand.HeaderClose + `
<h1 class="onz-title">{{.Title}}: {{.R.Playbook}}</h1>
<p class="onz-sub">checked {{.R.CheckedAt.Format "2006-01-02 15:04:05 MST"}}</p>
<div class="onz-stats">
<div class="onz-stat"><b>{{.R.Hosts}}</b><span>hosts</span></div>
<div class="onz-stat {{if .R.Drift}}warn{{else}}ok{{end}}"><b>{{len .R.Drift}}</b><span>with changes</span></div>
<div class="onz-stat"><b>{{.R.DriftTasks}}</b><span>tasks</span></div>
<div class="onz-stat ok"><b>{{len .R.InSync}}</b><span>in sync</span></div>
{{if .R.Orphans}}<div class="onz-stat warn"><b>{{len .R.Orphans}}</b><span>left the playbook</span></div>{{end}}
{{if .R.Errors}}<div class="onz-stat bad"><b>{{len .R.Errors}}</b><span>could not be checked</span></div>{{end}}
</div>
{{if .R.Fixed}}<p><span class="onz-badge {{tone "fixed"}}">fixed</span>: the playbook was applied</p>{{end}}
{{range .Hosts}}<section class="onz-card"><h2>{{.Name}}</h2>
{{range .Items}}<div class="onz-body"><strong>{{.Task}}</strong>{{if .Module}} <span class="muted">({{.Module}})</span>{{end}}{{if .Since}} <span class="muted">· since {{.Since.Local.Format "2006-01-02 15:04"}}</span>{{end}}{{if and .Detail (not .Lines)}} — {{.Detail}}{{end}}
{{if .Lines}}<pre class="onz-diff">{{range .Lines}}<span class="{{.Class}}">{{.Text}}</span>{{end}}</pre>{{end}}</div>{{end}}
{{if .Orphans}}<div class="onz-body"><strong>No longer in the playbook</strong><pre class="onz-diff">{{range .Orphans}}<span class="{{.Class}}">{{.Mark}} {{.Text}}</span>{{end}}</pre></div>{{end}}
{{range .Errors}}<div class="onz-body"><span class="onz-badge {{tone "error"}}">error</span> <strong>{{.Task}}</strong>: {{.Detail}}</div>{{end}}
</section>{{end}}
{{if .R.InSync}}<p class="muted">In sync: {{range $i, $h := .R.InSync}}{{if $i}}, {{end}}{{$h}}{{end}}</p>{{end}}
<footer class="onz-foot">Generated {{.R.CheckedAt.Format "2006-01-02 15:04:05 MST"}}</footer></main></body></html>
`))
