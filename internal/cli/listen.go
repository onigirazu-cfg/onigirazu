package cli

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
)

// listen mode: an HTTP server takes events (Alertmanager, GitHub, Mattermost
// or any JSON webhook), matches them against rules and runs playbooks for
// the ones that match — remediation on an alert, a deploy on a push, a
// chat command. Runs are serial, one at a time, and throttled per rule.

// ListenConfig is the listen.yml file
type ListenConfig struct {
	Listen  string         `yaml:"listen"` // address, default :8085
	Sources []ListenSource `yaml:"sources"`
	Rules   []ListenRule   `yaml:"rules"`
}

// ListenSource is one URL path events arrive on
type ListenSource struct {
	Name string `yaml:"name"`
	// Type: webhook (any JSON body), alertmanager (the Alertmanager webhook
	// receiver), github (X-Hub-Signature-256 checked, X-GitHub-Event as
	// event.type), mattermost (outgoing webhook or slash command)
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	// Token guards the path: "Authorization: Bearer <token>", header
	// X-Token or ?token=; for github it is the HMAC secret, for mattermost
	// the token Mattermost sends. ${VAR} is expanded.
	Token string `yaml:"token"`
}

// ListenRule runs a playbook for the events of a source that match `when`
type ListenRule struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source"`
	// When is a Jinja expression over event, headers and source; empty
	// matches every event
	When      string            `yaml:"when"`
	Playbook  string            `yaml:"playbook"`
	Inventory string            `yaml:"inventory"`
	Limit     string            `yaml:"limit"` // a template; empty = the playbook's hosts
	ExtraVars map[string]string `yaml:"extra_vars"`
	Become    bool              `yaml:"become"`
	DriftOnly bool              `yaml:"drift_only"`
	// Throttle: the rule runs at most once per period for the same
	// rendered limit (e.g. the same host)
	Throttle time.Duration `yaml:"throttle"`
	Notify   []string      `yaml:"notify"`
	NotifyOK bool          `yaml:"notify_always"`
}

type listenOptions struct {
	config, addr string
	dryRun       bool
}

func newListenCmd() *cobra.Command {
	o := &listenOptions{}
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Run playbooks on events: Alertmanager, GitHub, Mattermost or any webhook",
		Long: `Serve HTTP and run playbooks for the events that match the rules of a
listen.yml: alerts from Alertmanager (or vmalert), pushes and other GitHub
events, Mattermost outgoing webhooks and slash commands, or any JSON POST.
Rules are Jinja expressions over the event; a matching rule queues a playbook
run (serial, throttled per rule and host). "listen test" shows what an event
file would run; "listen install" writes a systemd service.`,
		Example: `  onigirazu listen --config listen.yml
  onigirazu listen test --config listen.yml --source alerts --file alert.json
  sudo onigirazu listen install --config /etc/onigirazu/listen.yml`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadListenConfig(o.config)
			if err != nil {
				return err
			}
			if o.addr != "" {
				cfg.Listen = o.addr
			}
			return runListen(cmd.Context(), cfg, o.dryRun, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVarP(&o.config, "config-file", "f", "listen.yml", "The listen.yml with sources and rules")
	cmd.Flags().StringVar(&o.addr, "addr", "", "Address to serve on (default from the file, else :8085)")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "Match and log, run nothing")

	var source, file string
	test := &cobra.Command{
		Use:   "test",
		Short: "Show which rules an event file matches and what they would run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadListenConfig(o.config)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(file) // #nosec G304 -- the operator's own event file
			if err != nil {
				return err
			}
			var event map[string]interface{}
			if err := json.Unmarshal(data, &event); err != nil {
				return fmt.Errorf("%s: %w", file, err)
			}
			d := newDispatcher(cfg, nil)
			jobs, err := d.match(source, event, map[string]interface{}{})
			if err != nil {
				return err
			}
			if len(jobs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no rule matches")
				return nil
			}
			for _, j := range jobs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: onigirazu apply %s\n", j.rule.Name, strings.Join(j.args, " "))
			}
			return nil
		},
	}
	test.Flags().StringVarP(&o.config, "config-file", "f", "listen.yml", "The listen.yml with sources and rules")
	test.Flags().StringVar(&source, "source", "", "Source name the event arrives on (default: the first source)")
	test.Flags().StringVar(&file, "file", "", "JSON event file (required)")
	_ = test.MarkFlagRequired("file")
	cmd.AddCommand(test)

	install := &cobra.Command{
		Use:   "install",
		Short: "Write and start a systemd service that runs listen with this file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installListenService(o, cmd.OutOrStdout())
		},
	}
	install.Flags().StringVarP(&o.config, "config-file", "f", "/etc/onigirazu/listen.yml", "The listen.yml the service reads")
	install.Flags().StringVar(&o.addr, "addr", "", "Address to serve on")
	cmd.AddCommand(install)
	return cmd
}

func loadListenConfig(path string) (*ListenConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the operator's own file
	if err != nil {
		return nil, err
	}
	cfg := &ListenConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8085"
	}
	names := map[string]*ListenSource{}
	for i := range cfg.Sources {
		s := &cfg.Sources[i]
		if s.Name == "" {
			return nil, fmt.Errorf("%s: source %d has no name", path, i+1)
		}
		switch s.Type {
		case "", "webhook":
			s.Type = "webhook"
		case "alertmanager", "github", "mattermost":
		default:
			return nil, fmt.Errorf("%s: source %s: unknown type %q (webhook, alertmanager, github, mattermost)", path, s.Name, s.Type)
		}
		if s.Path == "" {
			s.Path = "/" + s.Name
		}
		if !strings.HasPrefix(s.Path, "/") {
			s.Path = "/" + s.Path
		}
		s.Token = os.ExpandEnv(s.Token)
		if names[s.Name] != nil {
			return nil, fmt.Errorf("%s: source %s is defined twice", path, s.Name)
		}
		names[s.Name] = s
	}
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("%s: no sources", path)
	}
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if r.Name == "" {
			r.Name = fmt.Sprintf("rule %d", i+1)
		}
		if r.Playbook == "" {
			return nil, fmt.Errorf("%s: rule %s has no playbook", path, r.Name)
		}
		if r.Source == "" {
			r.Source = cfg.Sources[0].Name
		}
		if names[r.Source] == nil {
			return nil, fmt.Errorf("%s: rule %s: unknown source %q", path, r.Name, r.Source)
		}
		// relative paths start at the file's directory
		base := filepath.Dir(path)
		if !filepath.IsAbs(r.Playbook) {
			r.Playbook = filepath.Join(base, r.Playbook)
		}
		if r.Inventory != "" && !filepath.IsAbs(r.Inventory) {
			r.Inventory = filepath.Join(base, r.Inventory)
		}
	}
	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("%s: no rules", path)
	}
	return cfg, nil
}

// listenJob is one matched rule with its rendered apply arguments
type listenJob struct {
	rule      *ListenRule
	args      []string
	inventory string
	limit     string
	event     map[string]interface{}
}

// listenRunner runs a job; the dispatcher's default is runPlaybook
type listenRunner func(job *listenJob) (changed, failed int, err error)

type dispatcher struct {
	cfg     *ListenConfig
	engine  *template.Engine
	run     listenRunner
	jobs    chan *listenJob
	mu      sync.Mutex
	lastRun map[string]time.Time // rule + limit -> last start, for throttle
	events  map[string]int       // counters for /metrics
	runs    map[string]int
	out     io.Writer
	dryRun  bool
}

func newDispatcher(cfg *ListenConfig, run listenRunner) *dispatcher {
	d := &dispatcher{cfg: cfg, engine: template.NewEngine(), run: run, jobs: make(chan *listenJob, 256),
		lastRun: map[string]time.Time{}, events: map[string]int{}, runs: map[string]int{}, out: os.Stdout}
	if d.run == nil {
		d.run = runListenJob
	}
	return d
}

// match evaluates the rules of a source against an event and renders the
// apply arguments of those that hold
func (d *dispatcher) match(source string, event, headers map[string]interface{}) ([]*listenJob, error) {
	if source == "" {
		source = d.cfg.Sources[0].Name
	}
	vars := map[string]interface{}{"event": event, "headers": headers, "source": source}
	var jobs []*listenJob
	for i := range d.cfg.Rules {
		r := &d.cfg.Rules[i]
		if r.Source != source {
			continue
		}
		if r.When != "" {
			v, err := expression.Eval(r.When, vars)
			if err != nil {
				return nil, fmt.Errorf("rule %s: when: %w", r.Name, err)
			}
			if !expression.Truthy(v) {
				continue
			}
		}
		job := &listenJob{rule: r, event: event, inventory: r.Inventory}
		var err error
		if job.limit, err = d.render(r.Limit, vars); err != nil {
			return nil, fmt.Errorf("rule %s: limit: %w", r.Name, err)
		}
		job.args = []string{r.Playbook}
		if r.DriftOnly {
			job.args = append(job.args, "--check")
		}
		if job.limit != "" {
			job.args = append(job.args, "--limit", job.limit)
		}
		if r.Become {
			job.args = append(job.args, "--become")
		}
		keys := make([]string, 0, len(r.ExtraVars))
		for k := range r.ExtraVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v, err := d.render(r.ExtraVars[k], vars)
			if err != nil {
				return nil, fmt.Errorf("rule %s: extra_vars %s: %w", r.Name, k, err)
			}
			job.args = append(job.args, "-e", k+"="+v)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (d *dispatcher) render(text string, vars map[string]interface{}) (string, error) {
	if !strings.Contains(text, "{{") && !strings.Contains(text, "{%") {
		return text, nil
	}
	return d.engine.Render(context.Background(), text, vars)
}

// enqueue applies the throttle and queues the jobs; it returns the names of
// the rules queued and of those throttled
func (d *dispatcher) enqueue(jobs []*listenJob) (queued, throttled []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	for _, j := range jobs {
		key := j.rule.Name + "\x00" + j.limit
		if j.rule.Throttle > 0 {
			if last, ok := d.lastRun[key]; ok && now.Sub(last) < j.rule.Throttle {
				throttled = append(throttled, j.rule.Name)
				continue
			}
		}
		d.lastRun[key] = now
		select {
		case d.jobs <- j:
			queued = append(queued, j.rule.Name)
		default:
			throttled = append(throttled, j.rule.Name+" (queue full)")
		}
	}
	return queued, throttled
}

// worker runs the queued jobs one at a time
func (d *dispatcher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-d.jobs:
			started := time.Now()
			fmt.Fprintf(d.out, "listen: %s: running %s\n", j.rule.Name, strings.Join(j.args, " "))
			result := "ok"
			if d.dryRun {
				result = "dry-run"
			} else {
				changed, failed, err := d.run(j)
				switch {
				case err != nil:
					result = "error"
					fmt.Fprintf(os.Stderr, "listen: %s: %v\n", j.rule.Name, err)
				case failed > 0:
					result = "failed"
				case j.rule.DriftOnly && changed > 0:
					result = "drift"
				}
				fmt.Fprintf(d.out, "listen: %s: %s, %d changed, %d failed, %s\n", j.rule.Name, result, changed, failed, time.Since(started).Round(time.Millisecond))
			}
			d.mu.Lock()
			d.runs[j.rule.Name+"\x00"+result]++
			d.mu.Unlock()
		}
	}
}

// runListenJob runs the playbook in this process, as pull does; the
// inventory and state file of the job replace the global ones for the run
func runListenJob(j *listenJob) (changed, failed int, err error) {
	if j.inventory != "" {
		inventoryPaths = []string{j.inventory}
	}
	if statePath == "" || statePath == ".onigirazu-state" {
		statePath = filepath.Join(filepath.Dir(j.rule.Playbook), ".onigirazu-state")
	}
	result, err := runPlaybook(j.args)
	if err != nil {
		return 0, 0, err
	}
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			for _, t := range host.Tasks {
				if t.Failed && !t.Ignored {
					failed++
				} else if t.Changed && !t.Skipped {
					changed++
				}
			}
		}
	}
	if len(j.rule.Notify) > 0 && (failed > 0 || (j.rule.DriftOnly && changed > 0) || j.rule.NotifyOK) {
		report := buildDriftReport(j.rule.Playbook, result)
		report.Plan = j.rule.DriftOnly
		for _, url := range j.rule.Notify {
			if err := notifyWebhook(url, report); err != nil {
				fmt.Fprintf(os.Stderr, "notify %s: %v\n", redactURL(url), err)
			}
		}
	}
	return changed, failed, nil
}

const listenMaxBody = 1 << 20

// handler serves one source
func (d *dispatcher) handler(src *ListenSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, listenMaxBody+1))
		if err != nil || len(body) > listenMaxBody {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		event, err := parseListenEvent(src, r, body)
		if err != nil {
			if errors.Is(err, errListenAuth) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		headers := map[string]interface{}{}
		for k, v := range r.Header {
			headers[strings.ToLower(k)] = strings.Join(v, ",")
		}
		d.mu.Lock()
		d.events[src.Name]++
		d.mu.Unlock()
		jobs, err := d.match(src.Name, event, headers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "listen: %s: %v\n", src.Name, err)
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		queued, throttled := d.enqueue(jobs)
		fmt.Fprintf(d.out, "listen: %s: event, %d rule(s) matched, queued %v, throttled %v\n", src.Name, len(jobs), queued, throttled)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"matched": len(jobs), "queued": orEmpty(queued), "throttled": orEmpty(throttled)})
	}
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

var errListenAuth = errors.New("unauthorized")

// parseListenEvent checks the source's credential and turns the request
// into the event map the rules see
func parseListenEvent(src *ListenSource, r *http.Request, body []byte) (map[string]interface{}, error) {
	event := map[string]interface{}{}
	switch src.Type {
	case "github":
		if src.Token != "" {
			sig := strings.TrimPrefix(r.Header.Get("X-Hub-Signature-256"), "sha256=")
			mac := hmac.New(sha256.New, []byte(src.Token))
			mac.Write(body)
			if sig == "" || !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(mac.Sum(nil)))) {
				return nil, errListenAuth
			}
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &event); err != nil {
				return nil, fmt.Errorf("github: %w", err)
			}
		}
		event["type"] = r.Header.Get("X-GitHub-Event")
		event["delivery"] = r.Header.Get("X-GitHub-Delivery")
	case "mattermost":
		// an outgoing webhook or slash command posts a form (or JSON); the
		// token is a field
		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "application/json") {
			if err := json.Unmarshal(body, &event); err != nil {
				return nil, fmt.Errorf("mattermost: %w", err)
			}
		} else {
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			if err := r.ParseForm(); err != nil {
				return nil, fmt.Errorf("mattermost: %w", err)
			}
			for k, v := range r.PostForm {
				event[k] = strings.Join(v, " ")
			}
		}
		if src.Token != "" && subtle.ConstantTimeCompare([]byte(fmt.Sprint(event["token"])), []byte(src.Token)) != 1 {
			return nil, errListenAuth
		}
		delete(event, "token")
		if text, ok := event["text"].(string); ok {
			event["args"] = strings.Fields(strings.TrimSpace(strings.TrimPrefix(text, fmt.Sprint(event["trigger_word"]))))
		}
	default: // webhook, alertmanager: a bearer token and a JSON body
		if src.Token != "" && !bearerOK(r, src.Token) {
			return nil, errListenAuth
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &event); err != nil {
				return nil, fmt.Errorf("%s: %w", src.Type, err)
			}
		}
		if src.Type == "alertmanager" {
			// the alerts' labels under event.labels too, as a convenience:
			// the first alert's, as Alertmanager groups by them
			if alerts, ok := event["alerts"].([]interface{}); ok && len(alerts) > 0 {
				if first, ok := alerts[0].(map[string]interface{}); ok {
					if _, set := event["labels"]; !set {
						event["labels"] = first["labels"]
					}
				}
			}
		}
	}
	return event, nil
}

func bearerOK(r *http.Request, token string) bool {
	for _, got := range []string{strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), r.Header.Get("X-Token"), r.URL.Query().Get("token")} {
		if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

// metrics is a small Prometheus text endpoint
func (d *dispatcher) metrics(w http.ResponseWriter, _ *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var b strings.Builder
	b.WriteString("# TYPE onigirazu_listen_events_total counter\n")
	for _, name := range sortedKeys(d.events) {
		promLine(&b, "onigirazu_listen_events_total", map[string]string{"source": name}, float64(d.events[name]))
	}
	b.WriteString("# TYPE onigirazu_listen_runs_total counter\n")
	for _, key := range sortedKeys(d.runs) {
		rule, result, _ := strings.Cut(key, "\x00")
		promLine(&b, "onigirazu_listen_runs_total", map[string]string{"rule": rule, "result": result}, float64(d.runs[key]))
	}
	_, _ = io.WriteString(w, b.String())
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (d *dispatcher) mux() *http.ServeMux {
	mux := http.NewServeMux()
	for i := range d.cfg.Sources {
		src := &d.cfg.Sources[i]
		mux.HandleFunc(src.Path, d.handler(src))
	}
	mux.HandleFunc("/metrics", d.metrics)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok\n") })
	return mux
}

func runListen(ctx context.Context, cfg *ListenConfig, dryRun bool, out io.Writer) error {
	d := newDispatcher(cfg, nil)
	d.out, d.dryRun = out, dryRun
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go d.worker(ctx)
	srv := &http.Server{Addr: cfg.Listen, Handler: d.mux(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	fmt.Fprintf(out, "listen: serving on %s: %d source(s), %d rule(s)%s\n", cfg.Listen, len(cfg.Sources), len(cfg.Rules), map[bool]string{true: " (dry run)", false: ""}[dryRun])
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}

// listenUnitFile is the systemd service of `listen install`
func listenUnitFile(self, config, addr string) string {
	args := self + " listen --config-file " + config
	if addr != "" {
		args += " --addr " + addr
	}
	return `[Unit]
Description=onigirazu listen: run playbooks on events
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=` + args + `
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`
}

func installListenService(o *listenOptions, out io.Writer) error {
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return fmt.Errorf("listen install needs systemd")
	}
	if _, err := loadListenConfig(o.config); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	config, err := filepath.Abs(o.config)
	if err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/onigirazu-listen.service", []byte(listenUnitFile(self, config, o.addr)), 0o644); err != nil { // #nosec G306 -- unit files are world-readable
		return err
	}
	for _, c := range [][]string{{"systemctl", "daemon-reload"}, {"systemctl", "enable", "--now", "onigirazu-listen.service"}} {
		if o, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil { // #nosec G204 -- fixed systemctl arguments
			return fmt.Errorf("%s: %v: %s", strings.Join(c, " "), err, strings.TrimSpace(string(o)))
		}
	}
	fmt.Fprintf(out, "onigirazu-listen.service installed and started (%s)\n", config)
	return nil
}
