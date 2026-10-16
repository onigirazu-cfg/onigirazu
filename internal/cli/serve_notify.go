package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	eventPending  = "pending"
	eventStart    = "start"
	eventSuccess  = "success"
	eventDrift    = "drift"
	eventFailure  = "failure"
	eventRejected = "rejected"
	eventExpired  = "expired"
)

// NotifyTarget is a webhook and the run events it receives.
type NotifyTarget struct {
	URL    string   `yaml:"url"`
	Format string   `yaml:"format"`
	On     []string `yaml:"on"`
}

// UnmarshalYAML accepts a URL or a target mapping.
func (t *NotifyTarget) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		*t = NotifyTarget{URL: node.Value}
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("notify target must be a URL or mapping")
	}
	type target NotifyTarget
	return node.Decode((*target)(t))
}

// expandEnvStrict is os.ExpandEnv that refuses a variable that is not set
// (a webhook URL with a typo'd ${SECRET} would otherwise pass validation and
// fail at every delivery)
func expandEnvStrict(s string) (string, error) {
	var missing []string
	out := os.Expand(s, func(name string) string {
		v, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
		}
		return v
	})
	if len(missing) > 0 {
		return out, fmt.Errorf("environment variable %s is not set", strings.Join(missing, ", "))
	}
	return out, nil
}

func validateNotifyTargets(targets []NotifyTarget) error {
	for i := range targets {
		t := &targets[i]
		url, err := expandEnvStrict(t.URL)
		if err != nil {
			return fmt.Errorf("target %d: %w", i+1, err)
		}
		t.URL = url
		if strings.TrimSpace(t.URL) == "" {
			return fmt.Errorf("target %d: URL is empty", i+1)
		}
		if t.Format == "" {
			t.Format = "mattermost"
		}
		if t.Format != "mattermost" && t.Format != "json" {
			return fmt.Errorf("target %d: format %q (mattermost or json)", i+1, t.Format)
		}
		for _, event := range t.On {
			switch event {
			case eventPending, eventStart, eventSuccess, eventDrift, eventFailure, eventRejected, eventExpired:
			default:
				return fmt.Errorf("target %d: unknown event %q", i+1, event)
			}
		}
	}
	return nil
}

// notify receives a result copy taken under mu; delivery never holds the lock.
func (s *server) notify(event string, j *ServeJob, r *JobResult) {
	if j == nil {
		return
	}
	targets := append(append([]NotifyTarget(nil), s.cfg.Notify...), j.Notify...)
	client := s.notifyClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	for _, target := range targets {
		on := target.On
		if on == nil {
			on = []string{eventPending, eventDrift, eventFailure}
			if j.NotifyOK {
				on = append(on, eventSuccess)
			}
		}
		if !slices.Contains(on, event) {
			continue
		}
		s.notifyWait.Add(1)
		go func() {
			defer s.notifyWait.Done()
			body, err := s.notifyBody(event, j, r, target.Format)
			if err == nil {
				err = postServeNotification(client, target.URL, body)
			}
			if err != nil {
				// net/http errors include the secret URL; log only their cause.
				var requestError *url.Error
				if errors.As(err, &requestError) {
					err = requestError.Err
				}
				fmt.Fprintf(os.Stderr, "serve: notify %s %s %s: %v\n", j.Name, event, redactURL(target.URL), err)
			}
		}()
	}
}

func postServeNotification(client *http.Client, target string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *server) notifyBody(event string, j *ServeJob, r *JobResult, format string) ([]byte, error) {
	link := ""
	if s.cfg.URL != "" {
		link = strings.TrimRight(s.cfg.URL, "/") + "/#run/" + r.ID
	}
	target := j.Playbook
	if j.Kind == "comply" {
		target = j.Profile
	}
	plain, emoji, color := notifyHeadline(event, j.Name, r)
	if format == "json" {
		run := *r
		run.Report = nil
		return json.Marshal(map[string]interface{}{
			"event": event,
			"job":   map[string]string{"name": j.Name, "kind": j.Kind, "target": target, "limit": j.Limit},
			"run":   &run,
			"url":   link,
			"text":  plain,
		})
	}
	details := fmt.Sprintf("kind: %s · target: %s", j.Kind, target)
	hosts := notifyHosts(event, j, r)
	if hosts != "" {
		details += " · hosts: " + hosts
	}
	details += "\ntrigger: " + r.Trigger
	if event == eventPending && r.Approval != nil {
		details += "\nexpires: " + r.Approval.Expires.Format(time.RFC3339)
	} else if event != eventStart {
		details += fmt.Sprintf("\nduration: %.1fs", r.Duration)
	}
	if link != "" {
		details += "\nrun: " + link
	}
	attachment := map[string]string{"color": color, "title": j.Name + " — " + event, "text": details}
	if link != "" {
		attachment["title_link"] = link
	}
	headline := emoji + " " + "**" + j.Name + "**" + strings.TrimPrefix(plain, j.Name)
	return json.Marshal(map[string]interface{}{"username": "onigirazu", "text": headline, "attachments": []map[string]string{attachment}})
}

func notifyHeadline(event, name string, r *JobResult) (text, emoji, color string) {
	switch event {
	case eventPending:
		wait := ""
		if r.Approval != nil {
			d := r.Approval.Expires.Sub(r.Requested).Round(time.Minute).String()
			if strings.HasSuffix(d, "m0s") {
				d = strings.TrimSuffix(d, "0s")
			}
			if strings.HasSuffix(d, "h0m") {
				d = strings.TrimSuffix(d, "0m")
			}
			wait = ", expires in " + d
		}
		return fmt.Sprintf("%s waits for approval (requested by %s%s)", name, r.Trigger, wait), ":hourglass:", "#d98f1a"
	case eventStart:
		return fmt.Sprintf("%s started (%s)", name, r.Trigger), ":arrow_forward:", "#3b5bdb"
	case eventSuccess:
		return name + " ok — " + r.Summary, ":white_check_mark:", "#2a9d6f"
	case eventDrift:
		return name + " drift — " + r.Summary, ":warning:", "#d98f1a"
	case eventFailure:
		return name + " failed — " + r.Summary, ":x:", "#d14343"
	case eventRejected:
		text = name + " rejected"
		if r.Approval != nil {
			text += " by " + r.Approval.By
			if r.Approval.Comment != "" {
				text += ": " + r.Approval.Comment
			}
		}
		return text, ":no_entry_sign:", "#6b7280"
	default: // expired
		return name + " expired without approval", ":hourglass_flowing_sand:", "#6b7280"
	}
}

func notifyHosts(event string, j *ServeJob, r *JobResult) string {
	if event == eventPending || event == eventStart {
		if j.Limit == "" {
			return "all hosts"
		}
		return j.Limit
	}
	counts := map[string]int{}
	for _, status := range r.Hosts {
		counts[status]++
	}
	order := []string{"ok", "changed", "drift", "failed", "unreachable"}
	var extra []string
	for status := range counts {
		if !slices.Contains(order, status) {
			extra = append(extra, status)
		}
	}
	sort.Strings(extra)
	var groups []string
	for _, status := range append(order, extra...) {
		if counts[status] > 0 {
			groups = append(groups, fmt.Sprintf("%d %s", counts[status], status))
		}
	}
	return strings.Join(groups, ", ")
}
