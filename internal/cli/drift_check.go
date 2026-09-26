package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ExitError ends the program with Code (after printing Message, if any)
// instead of the generic exit code 1
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string { return fmt.Sprintf("exit %d: %s", e.Code, e.Message) }

// drift is the playbook in check mode: a task that would change a host is
// drift, a task that fails is an error. Exit code 0: in sync, 2: drift,
// 1: errors.

// DriftItem is one task that would change one host
type DriftItem struct {
	Play   string `json:"play,omitempty"`
	Task   string `json:"task"`
	Module string `json:"module,omitempty"`
	Detail string `json:"detail,omitempty"`
	// Diff is what the task would change, as a unified diff (file modules)
	Diff string `json:"diff,omitempty"`
}

// DriftReport is the outcome of a drift check
type DriftReport struct {
	Playbook   string                 `json:"playbook"`
	CheckedAt  time.Time              `json:"checked_at"`
	Hosts      int                    `json:"hosts"`
	InSync     []string               `json:"in_sync"`
	Drift      map[string][]DriftItem `json:"drift"`
	Errors     map[string][]DriftItem `json:"errors,omitempty"`
	Fixed      bool                   `json:"fixed,omitempty"`
	Plan       bool                   `json:"plan,omitempty"`
	DriftTasks int                    `json:"drift_tasks"`
}

type driftCheckOptions struct {
	plan       bool // plan: the same report, changes are not an error
	fix        bool
	format     string
	output     string
	extraVars  []string
	limit      string
	tags       string
	skipTags   string
	become     bool
	becomeUser string
	user       string
	privateKey string
	notify     []string // webhooks told about drift and errors
	notifyOK   bool     // tell them about a clean check too
}

// applyArgs are the apply arguments of a drift check (or of --fix)
func (o driftCheckOptions) applyArgs(playbook string, check bool) []string {
	args := []string{playbook}
	if check {
		args = append(args, "--check", "--diff")
	}
	for _, e := range o.extraVars {
		args = append(args, "-e", e)
	}
	for flag, value := range map[string]string{"--limit": o.limit, "--tags": o.tags, "--skip-tags": o.skipTags,
		"--become-user": o.becomeUser, "--user": o.user, "--private-key": o.privateKey} {
		if value != "" {
			args = append(args, flag, value)
		}
	}
	if o.become {
		args = append(args, "--become")
	}
	return args
}

// runPlaybook runs apply with args and returns its result
func runPlaybook(args []string) (*types.PlaybookResult, error) {
	var result *types.PlaybookResult
	apply := newApplyCommand(func(r *types.PlaybookResult) { result = r })
	apply.SetArgs(args)
	apply.SilenceUsage, apply.SilenceErrors = true, true
	err := apply.Execute()
	if result == nil && err != nil {
		return nil, err
	}
	return result, nil
}

func buildDriftReport(playbook string, result *types.PlaybookResult) *DriftReport {
	report := &DriftReport{
		Playbook: playbook, CheckedAt: time.Now(),
		Drift: map[string][]DriftItem{}, Errors: map[string][]DriftItem{}, InSync: []string{},
	}
	hosts := map[string]bool{}
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			hosts[host.Host] = true
			for _, t := range host.Tasks {
				item := DriftItem{Play: play.Name, Task: t.TaskName, Module: t.Module, Detail: taskDetail(t), Diff: taskDiffs(t)}
				switch {
				case t.Failed && !t.Ignored:
					report.Errors[host.Host] = append(report.Errors[host.Host], item)
				case t.Changed && !t.Skipped:
					report.Drift[host.Host] = append(report.Drift[host.Host], item)
					report.DriftTasks++
				}
			}
		}
	}
	report.Hosts = len(hosts)
	for h := range hosts {
		if len(report.Drift[h]) == 0 && len(report.Errors[h]) == 0 {
			report.InSync = append(report.InSync, h)
		}
	}
	sort.Strings(report.InSync)
	return report
}

// taskDetail is the short explanation a module gave for a result
func taskDetail(t types.TaskResult) string {
	if t.Failed {
		return t.Error
	}
	for _, key := range []string{"msg", "message"} {
		if s, ok := t.Output[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func writeDriftText(w io.Writer, r *DriftReport) {
	names := make([]string, 0, len(r.Drift))
	for h := range r.Drift {
		names = append(names, h)
	}
	sort.Strings(names)
	switch {
	case len(r.Drift) == 0 && len(r.Errors) == 0 && r.Plan:
		fmt.Fprintf(w, "No changes: %d host(s) already match %s\n", r.Hosts, r.Playbook)
	case len(r.Drift) == 0 && len(r.Errors) == 0:
		fmt.Fprintf(w, "In sync: %d host(s) match %s\n", r.Hosts, r.Playbook)
	case len(r.Drift) > 0 && r.Plan:
		fmt.Fprintf(w, "Plan: %d task(s) would change %d of %d host(s)\n", r.DriftTasks, len(r.Drift), r.Hosts)
	case len(r.Drift) > 0:
		fmt.Fprintf(w, "Drift: %d task(s) on %d of %d host(s) differ from %s\n", r.DriftTasks, len(r.Drift), r.Hosts, r.Playbook)
	}
	for _, h := range names {
		fmt.Fprintf(w, "\n%s\n", h)
		for _, it := range r.Drift[h] {
			line := fmt.Sprintf("  ~ %s", it.Task)
			if it.Module != "" {
				line += " (" + it.Module + ")"
			}
			if it.Detail != "" && it.Diff == "" { // the diff says more than the module message
				line += ": " + it.Detail
			}
			fmt.Fprintln(w, line)
			for _, l := range strings.Split(strings.TrimRight(it.Diff, "\n"), "\n") {
				if l != "" {
					fmt.Fprintln(w, "      "+l)
				}
			}
		}
	}
	if len(r.Errors) > 0 {
		fmt.Fprintf(w, "\nCould not check:\n")
		for h, items := range r.Errors {
			for _, it := range items {
				fmt.Fprintf(w, "  ! %s: %s: %s\n", h, it.Task, it.Detail)
			}
		}
	}
	if len(r.InSync) > 0 && len(r.Drift) > 0 {
		fmt.Fprintf(w, "\nIn sync: %s\n", strings.Join(r.InSync, ", "))
	}
	if r.Fixed {
		fmt.Fprintln(w, "\nFixed: the playbook was applied")
	}
}

func runDriftCheck(cmd *cobra.Command, playbook string, o driftCheckOptions) error {
	result, err := runPlaybook(o.applyArgs(playbook, true))
	if err != nil {
		return err
	}
	report := buildDriftReport(playbook, result)
	report.Plan = o.plan

	if o.fix && len(report.Drift) > 0 && len(report.Errors) == 0 {
		if _, err := runPlaybook(o.applyArgs(playbook, false)); err != nil {
			return fmt.Errorf("fix failed: %w", err)
		}
		report.Fixed = true
	}

	if len(report.Drift) > 0 || len(report.Errors) > 0 || o.notifyOK {
		for _, url := range o.notify {
			if err := notifyWebhook(url, report); err != nil {
				fmt.Fprintf(os.Stderr, "notify %s: %v\n", redactURL(url), err)
			}
		}
	}

	out := io.Writer(os.Stdout)
	if o.output != "" {
		f, err := os.Create(o.output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	switch o.format {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	case "text", "":
		writeDriftText(out, report)
	case "html":
		if err := writeDriftHTML(out, report); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown format %q (text, json, html)", o.format)
	}

	switch {
	case len(report.Errors) > 0:
		return &ExitError{Code: 1}
	case len(report.Drift) > 0 && !report.Fixed && !o.plan:
		return &ExitError{Code: 2}
	}
	return nil
}

// notifyWebhook posts the report summary as {"text": ...}, which Slack,
// Mattermost and most chat webhooks accept; the full report is under
// "report"
func notifyWebhook(url string, r *DriftReport) error {
	var text strings.Builder
	writeDriftText(&text, r)
	msg := text.String()
	if len(msg) > 3500 {
		msg = msg[:3500] + "\n…"
	}
	host, _ := os.Hostname()
	body, err := json.Marshal(map[string]interface{}{
		"text":   fmt.Sprintf("onigirazu on %s:\n```\n%s```", host, msg),
		"report": r,
	})
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body)) // #nosec G107 -- the user names the webhook
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// redactURL keeps a webhook's secret path out of messages
func redactURL(url string) string {
	if i := strings.Index(url, "://"); i >= 0 {
		if j := strings.Index(url[i+3:], "/"); j >= 0 {
			return url[:i+3+j] + "/…"
		}
	}
	return url
}
