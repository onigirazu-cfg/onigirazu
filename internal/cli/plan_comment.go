package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// A plan as a pull request comment: `plan --format markdown` renders the
// report for GitHub, and `--github-comment` posts it on the pull request the
// workflow runs for, replacing the comment of the previous run (a marker
// line names the playbook), the way Atlantis comments a terraform plan.

// writeDriftMarkdown renders the report as Markdown: a summary line, per
// host the tasks that would change, diffs folded under <details>
func writeDriftMarkdown(w io.Writer, r *DriftReport) {
	fmt.Fprintf(w, "<!-- onigirazu-plan: %s -->\n", r.Playbook)
	verb := "would change"
	title := "Plan"
	if !r.Plan {
		title, verb = "Drift", "differ"
	}
	switch {
	case len(r.Drift) == 0 && len(r.Errors) == 0:
		fmt.Fprintf(w, "### %s: no changes — %d host(s) already match `%s`\n", title, r.Hosts, r.Playbook)
	default:
		fmt.Fprintf(w, "### %s: %d task(s) %s on %d of %d host(s) — `%s`\n", title, r.DriftTasks, verb, len(r.Drift), r.Hosts, r.Playbook)
	}
	names := make([]string, 0, len(r.Drift))
	for h := range r.Drift {
		names = append(names, h)
	}
	sort.Strings(names)
	for _, h := range names {
		fmt.Fprintf(w, "\n**%s**\n\n", h)
		for _, it := range r.Drift[h] {
			line := "- `" + it.Task + "`"
			if it.Module != "" {
				line += " (" + it.Module + ")"
			}
			if it.Detail != "" && it.Diff == "" {
				line += ": " + it.Detail
			}
			fmt.Fprintln(w, line)
			if d := strings.TrimRight(it.Diff, "\n"); d != "" {
				fmt.Fprintf(w, "  <details><summary>diff</summary>\n\n  ```diff\n")
				for _, l := range strings.Split(d, "\n") {
					fmt.Fprintln(w, "  "+l)
				}
				fmt.Fprintf(w, "  ```\n  </details>\n")
			}
		}
	}
	if len(r.Orphans) > 0 {
		fmt.Fprintf(w, "\n**Orphans** (managed resources no task claims any more)\n\n")
		for _, o := range r.Orphans {
			fmt.Fprintf(w, "- %s `%s` on %s\n", o.Type, o.ID, o.Host)
		}
	}
	if len(r.Errors) > 0 {
		fmt.Fprintf(w, "\n**Could not check**\n\n")
		hosts := make([]string, 0, len(r.Errors))
		for h := range r.Errors {
			hosts = append(hosts, h)
		}
		sort.Strings(hosts)
		for _, h := range hosts {
			for _, it := range r.Errors[h] {
				fmt.Fprintf(w, "- %s: `%s`: %s\n", h, it.Task, it.Detail)
			}
		}
	}
	if len(r.InSync) > 0 && len(r.Drift) > 0 {
		fmt.Fprintf(w, "\nUnchanged: %s\n", strings.Join(r.InSync, ", "))
	}
	fmt.Fprintf(w, "\n<sub>onigirazu %s · %s</sub>\n", title, r.CheckedAt.UTC().Format(time.RFC3339))
}

// githubPRNumber is the pull request of this workflow run, from the event
// payload or the ref (refs/pull/N/merge)
func githubPRNumber() (int, error) {
	if p := os.Getenv("GITHUB_EVENT_PATH"); p != "" {
		if data, err := os.ReadFile(p); err == nil { // #nosec G304 -- set by the runner
			var ev struct {
				PullRequest struct{ Number int } `json:"pull_request"`
				Issue       struct{ Number int } `json:"issue"`
			}
			if json.Unmarshal(data, &ev) == nil {
				if ev.PullRequest.Number > 0 {
					return ev.PullRequest.Number, nil
				}
				if ev.Issue.Number > 0 {
					return ev.Issue.Number, nil
				}
			}
		}
	}
	if m := regexp.MustCompile(`^refs/pull/(\d+)/`).FindStringSubmatch(os.Getenv("GITHUB_REF")); m != nil {
		n := 0
		_, _ = fmt.Sscanf(m[1], "%d", &n)
		return n, nil
	}
	return 0, fmt.Errorf("not a pull request run (no pull request in GITHUB_EVENT_PATH, GITHUB_REF is %q)", os.Getenv("GITHUB_REF"))
}

// githubComment posts body on the pull request, replacing an earlier
// comment that carries the same marker line
func githubComment(body, marker string) error {
	token, repo := os.Getenv("GITHUB_TOKEN"), os.Getenv("GITHUB_REPOSITORY")
	if token == "" || repo == "" {
		return fmt.Errorf("GITHUB_TOKEN and GITHUB_REPOSITORY are needed to comment")
	}
	pr, err := githubPRNumber()
	if err != nil {
		return err
	}
	api := strings.TrimRight(os.Getenv("GITHUB_API_URL"), "/")
	if api == "" {
		api = "https://api.github.com"
	}
	client := &http.Client{Timeout: 30 * time.Second}
	do := func(method, url string, payload interface{}, out interface{}) error {
		var rd io.Reader
		if payload != nil {
			b, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			rd = bytes.NewReader(b)
		}
		req, err := http.NewRequest(method, url, rd) // #nosec G704 -- the API URL and repository come from the Actions runner's environment
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req) // #nosec G704 -- see above
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
			return fmt.Errorf("%s %s: HTTP %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(msg)))
		}
		if out != nil {
			return json.NewDecoder(resp.Body).Decode(out)
		}
		return nil
	}
	// the previous run's comment, by its marker
	var comments []struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	}
	if err := do(http.MethodGet, fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=100", api, repo, pr), nil, &comments); err != nil {
		return err
	}
	for _, c := range comments {
		if strings.Contains(c.Body, marker) {
			return do(http.MethodPatch, fmt.Sprintf("%s/repos/%s/issues/comments/%d", api, repo, c.ID), map[string]string{"body": body}, nil)
		}
	}
	return do(http.MethodPost, fmt.Sprintf("%s/repos/%s/issues/%d/comments", api, repo, pr), map[string]string{"body": body}, nil)
}
