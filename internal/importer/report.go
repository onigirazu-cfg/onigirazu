package importer

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Drift is a task of the imported playbook that would still change a host
type Drift struct {
	Host, Task, Detail string
}

// WriteReport writes IMPORT_REPORT.md
func WriteReport(path string, r *Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Import report\n\nHosts: %s\n\n", strings.Join(r.Hosts, ", "))
	b.WriteString("## Imported\n\n| What | Count |\n|---|---|\n")
	keys := make([]string, 0, len(r.Counts))
	for k := range r.Counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "| %s | %d |\n", k, r.Counts[k])
	}
	if len(r.Layers) > 0 {
		b.WriteString("\n## Roles\n\n| Role | Kind | Hosts | Tasks |\n|---|---|---|---|\n")
		for _, l := range r.Layers {
			fmt.Fprintf(&b, "| %s | %s | %s | %d |\n", l.Role, l.Kind, strings.Join(l.Hosts, ", "), l.Tasks)
		}
	}
	if r.Verified {
		if len(r.Drift) == 0 {
			b.WriteString("\n## Check\n\n`plan` of the new playbook against the hosts: nothing to change.\n")
		} else {
			fmt.Fprintf(&b, "\n## Check\n\n`plan` of the new playbook against the hosts would still change %d task(s):\n\n", len(r.Drift))
			for _, d := range r.Drift {
				fmt.Fprintf(&b, "- %s: %s", d.Host, d.Task)
				if d.Detail != "" {
					fmt.Fprintf(&b, " — %s", oneLine(d.Detail))
				}
				b.WriteString("\n")
			}
		}
	}
	if len(r.SecretVars) > 0 {
		b.WriteString("\n## Secret variables\n\nThe files use these variables instead of the values found on the hosts; " +
			"provide them (host_vars, group_vars, `-e @secrets.yml`, a vault lookup). secrets.example.yml lists them:\n\n")
		for _, v := range r.SecretVars {
			fmt.Fprintf(&b, "- `%s`: %s line %d (%s)\n", v.Var, v.Path, v.Line, strings.Join(v.Hosts, ", "))
		}
	}
	if len(r.Secrets) > 0 {
		b.WriteString("\n## Secrets (not written)\n\nProvide these yourself, e.g. from a vault:\n\n")
		for _, s := range r.Secrets {
			fmt.Fprintf(&b, "- `%s`: %s\n", s.Path, s.Reason)
		}
	}
	if len(r.Skipped) > 0 {
		b.WriteString("\n## Left out\n\n")
		for _, s := range r.Skipped {
			fmt.Fprintf(&b, "- `%s`: %s\n", s.Path, s.Reason)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// PrintSummary is the short result on the terminal
func PrintSummary(w io.Writer, r *Report, dir string) {
	var parts []string
	for _, k := range []string{"packages", "services", "users", "groups", "files", "directories", "links", "mounts"} {
		if n := r.Counts[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, k))
		}
	}
	fmt.Fprintf(w, "Imported %s: %s\n", strings.Join(r.Hosts, ", "), strings.Join(parts, ", "))
	var roles []string
	for _, l := range r.Layers {
		roles = append(roles, fmt.Sprintf("%s (%d)", l.Role, len(l.Hosts)))
	}
	if len(roles) > 0 {
		fmt.Fprintf(w, "Roles: %s\n", strings.Join(roles, ", "))
	}
	fmt.Fprintf(w, "Playbook: %s/site.yml, report: %s/IMPORT_REPORT.md\n", dir, dir)
	if len(r.SecretVars) > 0 {
		fmt.Fprintf(w, "Secret variables to provide: %d (secrets.example.yml)\n", len(r.SecretVars))
	}
	if len(r.Secrets) > 0 {
		fmt.Fprintf(w, "Secrets not written: %d (listed in the report)\n", len(r.Secrets))
	}
	if len(r.Skipped) > 0 {
		fmt.Fprintf(w, "Left out: %d (listed in the report)\n", len(r.Skipped))
	}
	switch {
	case !r.Verified:
	case len(r.Drift) == 0:
		fmt.Fprintln(w, "Check: plan of the new playbook against the hosts has nothing to change")
		if r.Adopted {
			fmt.Fprintln(w, "Adopted: the resources are in the new playbook's managed state (state resources)")
		}
	default:
		fmt.Fprintf(w, "Check: the new playbook would still change %d task(s):\n", len(r.Drift))
		for i, d := range r.Drift {
			if i == 10 {
				fmt.Fprintf(w, "  ... %d more in the report\n", len(r.Drift)-10)
				break
			}
			fmt.Fprintf(w, "  ~ %s: %s\n", d.Host, d.Task)
		}
	}
}
