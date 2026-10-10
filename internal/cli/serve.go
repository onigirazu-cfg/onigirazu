package cli

import (
	"context"
	"crypto/subtle"
	_ "embed"
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

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
)

// serve: a small fleet server in the same binary. It runs jobs on a schedule
// (drift checks, applies, verify, compliance), keeps their results, serves
// the run history of the audit store and a web page over it, and a REST API
// with a bearer token or a reverse proxy's identity headers.

// ServeConfig is the serve.yml file
type ServeConfig struct {
	Listen  string      `yaml:"listen"` // default :8086
	Title   string      `yaml:"title"`
	DataDir string      `yaml:"data_dir"` // default ~/.onigirazu/serve
	Auth    ServeAuth   `yaml:"auth"`
	Jobs    []ServeJob  `yaml:"jobs"`
	Retain  int         `yaml:"retain"` // results kept per job, default 50
	Metrics ServeMetric `yaml:"metrics"`
}

// ServeAuth: a bearer token is an operator; a reverse proxy's user header is
// a viewer, an operator when one of its groups is in operators. Without any
// of them everything is open — put a proxy in front.
type ServeAuth struct {
	Token        string   `yaml:"token"` // ${VAR} expanded
	UserHeader   string   `yaml:"user_header"`
	GroupsHeader string   `yaml:"groups_header"`
	Operators    []string `yaml:"operators"`
}

// ServeMetric is an optional push of every job's metrics
type ServeMetric struct {
	Push   string            `yaml:"push"`
	Labels map[string]string `yaml:"labels"`
}

// ServeJob is a scheduled run
type ServeJob struct {
	Name      string            `yaml:"name"`
	Kind      string            `yaml:"kind"` // drift (default), apply, verify, comply
	Playbook  string            `yaml:"playbook"`
	Profile   string            `yaml:"profile"` // comply
	Inventory string            `yaml:"inventory"`
	Limit     string            `yaml:"limit"`
	ExtraVars map[string]string `yaml:"extra_vars"`
	Become    bool              `yaml:"become"`
	Every     time.Duration     `yaml:"every"` // 0: only on demand
	Notify    []string          `yaml:"notify"`
	NotifyOK  bool              `yaml:"notify_always"`
	FailOn    string            `yaml:"fail_on"` // comply
}

// JobResult is one run of a job, as stored
type JobResult struct {
	ID       string            `json:"id"`
	Job      string            `json:"job"`
	Kind     string            `json:"kind"`
	Started  time.Time         `json:"started"`
	Duration float64           `json:"duration_seconds"`
	Status   string            `json:"status"` // ok, drift, failed, error
	Summary  string            `json:"summary"`
	Trigger  string            `json:"trigger"`         // schedule, api:<user>
	Hosts    map[string]string `json:"hosts,omitempty"` // host -> ok|drift|failed|unreachable
	Report   json.RawMessage   `json:"report,omitempty"`
}

type serveOptions struct {
	config, addr string
}

func newServeCmd() *cobra.Command {
	o := &serveOptions{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Fleet server: scheduled drift/apply/verify/comply jobs, run history, web page and API",
		Long: `Serve a web page and a REST API over the fleet: jobs from serve.yml run on
their schedule or on demand (drift checks, applies, verify, compliance
profiles), their results are kept, the audit store's run history is shown per
run and per host. Access: a bearer token (operator) or the identity headers of
a reverse proxy such as Authentik's outpost (viewer; operator by group).`,
		Example: `  onigirazu serve --config-file serve.yml
  curl -H 'Authorization: Bearer ...' localhost:8086/api/jobs
  sudo onigirazu serve install -f /etc/onigirazu/serve.yml`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadServeConfig(o.config)
			if err != nil {
				return err
			}
			if o.addr != "" {
				cfg.Listen = o.addr
			}
			return runServe(cmd.Context(), cfg, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVarP(&o.config, "config-file", "f", "serve.yml", "The serve.yml with jobs and access settings")
	cmd.Flags().StringVar(&o.addr, "addr", "", "Address to serve on (default from the file, else :8086)")
	install := &cobra.Command{
		Use:   "install",
		Short: "Write and start a systemd service that runs serve with this file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installServeService(o, cmd.OutOrStdout())
		},
	}
	install.Flags().StringVarP(&o.config, "config-file", "f", "/etc/onigirazu/serve.yml", "The serve.yml the service reads")
	install.Flags().StringVar(&o.addr, "addr", "", "Address to serve on")
	cmd.AddCommand(install)
	return cmd
}

func loadServeConfig(path string) (*ServeConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the operator's own file
	if err != nil {
		return nil, err
	}
	cfg := &ServeConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8086"
	}
	if cfg.Title == "" {
		cfg.Title = "onigirazu"
	}
	if cfg.Retain <= 0 {
		cfg.Retain = 50
	}
	if cfg.DataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.DataDir = filepath.Join(home, ".onigirazu", "serve")
	}
	cfg.Auth.Token = os.ExpandEnv(cfg.Auth.Token)
	base := filepath.Dir(path)
	names := map[string]bool{}
	for i := range cfg.Jobs {
		j := &cfg.Jobs[i]
		if j.Name == "" {
			return nil, fmt.Errorf("%s: job %d has no name", path, i+1)
		}
		if names[j.Name] {
			return nil, fmt.Errorf("%s: job %s is defined twice", path, j.Name)
		}
		names[j.Name] = true
		switch j.Kind {
		case "":
			j.Kind = "drift"
		case "drift", "apply", "verify", "comply":
		default:
			return nil, fmt.Errorf("%s: job %s: kind %q (drift, apply, verify, comply)", path, j.Name, j.Kind)
		}
		if j.Kind == "comply" {
			if j.Profile == "" {
				return nil, fmt.Errorf("%s: job %s: comply needs a profile", path, j.Name)
			}
		} else if j.Playbook == "" {
			return nil, fmt.Errorf("%s: job %s has no playbook", path, j.Name)
		}
		if j.Playbook != "" && !filepath.IsAbs(j.Playbook) {
			j.Playbook = filepath.Join(base, j.Playbook)
		}
		if j.Inventory != "" && !filepath.IsAbs(j.Inventory) {
			j.Inventory = filepath.Join(base, j.Inventory)
		}
		if j.Profile != "" && strings.ContainsAny(j.Profile, "/.") && !filepath.IsAbs(j.Profile) {
			j.Profile = filepath.Join(base, j.Profile)
		}
	}
	return cfg, nil
}

// server holds the jobs, their results and the audit store
type server struct {
	cfg     *ServeConfig
	out     io.Writer
	mu      sync.Mutex
	results map[string][]*JobResult // job -> newest first
	running map[string]bool
	nextRun map[string]time.Time
	queue   chan queued
	// runJob runs a job; tests replace it
	runJob func(j *ServeJob) (*JobResult, error)
	audit  *audit.Storage
}

type queued struct {
	job     *ServeJob
	trigger string
}

func newServer(cfg *ServeConfig, out io.Writer) (*server, error) {
	s := &server{cfg: cfg, out: out, results: map[string][]*JobResult{}, running: map[string]bool{}, nextRun: map[string]time.Time{}, queue: make(chan queued, 64)}
	s.runJob = s.execute
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "results"), 0o700); err != nil {
		return nil, err
	}
	s.loadResults()
	lg, err := logger.NewEnhancedLogger("warn", "text", os.Stderr)
	if err == nil {
		if st, err := audit.NewStorage(getAuditPath(), lg); err == nil {
			s.audit = st
		}
	}
	return s, nil
}

func (s *server) loadResults() {
	dir := filepath.Join(s.cfg.DataDir, "results")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name())) // #nosec G304 -- our own data directory
		if err != nil {
			continue
		}
		var r JobResult
		if json.Unmarshal(data, &r) == nil && r.Job != "" {
			s.results[r.Job] = append(s.results[r.Job], &r)
		}
	}
	for job := range s.results {
		sort.Slice(s.results[job], func(i, k int) bool { return s.results[job][i].Started.After(s.results[job][k].Started) })
	}
}

func (s *server) store(r *JobResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := append([]*JobResult{r}, s.results[r.Job]...)
	dir := filepath.Join(s.cfg.DataDir, "results")
	for len(list) > s.cfg.Retain {
		old := list[len(list)-1]
		list = list[:len(list)-1]
		_ = os.Remove(filepath.Join(dir, old.ID+".json"))
	}
	s.results[r.Job] = list
	if data, err := json.Marshal(r); err == nil {
		_ = os.WriteFile(filepath.Join(dir, r.ID+".json"), data, 0o600)
	}
}

func (s *server) job(name string) *ServeJob {
	for i := range s.cfg.Jobs {
		if s.cfg.Jobs[i].Name == name {
			return &s.cfg.Jobs[i]
		}
	}
	return nil
}

// enqueue queues a job unless it is queued or running already
func (s *server) enqueue(j *ServeJob, trigger string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[j.Name] {
		return false
	}
	select {
	case s.queue <- queued{job: j, trigger: trigger}:
		s.running[j.Name] = true
		return true
	default:
		return false
	}
}

// worker runs queued jobs one at a time
func (s *server) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case q := <-s.queue:
			fmt.Fprintf(s.out, "serve: %s: running (%s)\n", q.job.Name, q.trigger)
			r, err := s.runJob(q.job)
			if err != nil {
				r = &JobResult{Job: q.job.Name, Kind: q.job.Kind, Started: time.Now().UTC(), Status: "error", Summary: err.Error()}
			}
			r.ID = fmt.Sprintf("%s-%s", q.job.Name, r.Started.Format("20060102T150405Z"))
			r.Trigger = q.trigger
			s.store(r)
			s.mu.Lock()
			delete(s.running, q.job.Name)
			s.mu.Unlock()
			fmt.Fprintf(s.out, "serve: %s: %s — %s (%.1fs)\n", q.job.Name, r.Status, r.Summary, r.Duration)
		}
	}
}

// scheduler queues the jobs with a period when it is due
func (s *server) scheduler(ctx context.Context) {
	s.mu.Lock()
	now := time.Now()
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		if j.Every > 0 {
			// the first run soon after start, staggered a little
			s.nextRun[j.Name] = now.Add(time.Duration(10+i*5) * time.Second)
		}
	}
	s.mu.Unlock()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.mu.Lock()
			var due []*ServeJob
			for i := range s.cfg.Jobs {
				j := &s.cfg.Jobs[i]
				if next, ok := s.nextRun[j.Name]; ok && !now.Before(next) {
					due = append(due, j)
					s.nextRun[j.Name] = now.Add(j.Every)
				}
			}
			s.mu.Unlock()
			for _, j := range due {
				s.enqueue(j, "schedule")
			}
		}
	}
}

// execute runs a job in this process and summarizes it
func (s *server) execute(j *ServeJob) (*JobResult, error) {
	started := time.Now().UTC()
	r := &JobResult{Job: j.Name, Kind: j.Kind, Started: started, Hosts: map[string]string{}}
	if j.Inventory != "" {
		inventoryPaths = []string{j.Inventory}
	}
	if statePath == "" || statePath == ".onigirazu-state" {
		statePath = filepath.Join(s.cfg.DataDir, "state", j.Name+".state")
		_ = os.MkdirAll(filepath.Dir(statePath), 0o700)
	}
	o := driftCheckOptions{limit: j.Limit, become: j.Become, notify: j.Notify, notifyOK: j.NotifyOK}
	keys := make([]string, 0, len(j.ExtraVars))
	for k := range j.ExtraVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		o.extraVars = append(o.extraVars, k+"="+j.ExtraVars[k])
	}
	switch j.Kind {
	case "comply":
		co := &complyOptions{driftCheckOptions: o, profile: j.Profile, failOn: j.FailOn}
		co.become = j.Become || !strings.Contains(j.Profile, "/")
		report, err := complyReport(co)
		if err != nil {
			return nil, err
		}
		for host, h := range report.Hosts {
			r.Hosts[host] = map[bool]string{true: "ok", false: "failed"}[h.Failed == 0]
		}
		for host := range report.Errors {
			r.Hosts[host] = "unreachable"
		}
		r.Status = "ok"
		if report.FailsAt(j.FailOn) || len(report.Errors) > 0 {
			r.Status = "failed"
		}
		r.Summary = fmt.Sprintf("%d/%d controls pass (%.0f%%) on %d host(s)", report.Passed, report.Passed+report.Failed, report.Score, len(report.Hosts))
		r.Report, _ = json.Marshal(report)
		s.pushMetrics(report.Metrics(s.cfg.Metrics.Labels))
	case "verify":
		result, err := runPlaybook(append(o.applyArgs(j.Playbook, false), "--verify-only"))
		if err != nil {
			return nil, err
		}
		vr := buildVerifyReport(j.Playbook, result)
		for host, h := range vr.Hosts {
			r.Hosts[host] = map[bool]string{true: "ok", false: "failed"}[h.Failed == 0]
		}
		for host := range vr.Errors {
			r.Hosts[host] = "unreachable"
		}
		r.Status = map[bool]string{true: "ok", false: "failed"}[vr.Failed == 0 && len(vr.Errors) == 0]
		r.Summary = fmt.Sprintf("%d passed, %d failed", vr.Passed, vr.Failed)
		r.Report, _ = json.Marshal(vr)
	default: // drift, apply
		check := j.Kind == "drift"
		result, err := runPlaybook(o.applyArgs(j.Playbook, check))
		if err != nil {
			return nil, err
		}
		report := buildDriftReport(j.Playbook, result)
		report.Plan = check
		for _, h := range report.InSync {
			r.Hosts[h] = "ok"
		}
		for h := range report.Drift {
			r.Hosts[h] = map[bool]string{true: "drift", false: "changed"}[check]
		}
		for h := range report.Errors {
			r.Hosts[h] = "failed"
		}
		switch {
		case len(report.Errors) > 0:
			r.Status = "failed"
		case check && len(report.Drift) > 0:
			r.Status = "drift"
		default:
			r.Status = "ok"
		}
		r.Summary = fmt.Sprintf("%d in sync, %d %s, %d failed", len(report.InSync), len(report.Drift), map[bool]string{true: "drifting", false: "changed"}[check], len(report.Errors))
		r.Report, _ = json.Marshal(report)
		if len(j.Notify) > 0 && (len(report.Errors) > 0 || (check && len(report.Drift) > 0) || j.NotifyOK) {
			for _, url := range j.Notify {
				if err := notifyWebhook(url, report); err != nil {
					fmt.Fprintf(os.Stderr, "notify %s: %v\n", redactURL(url), err)
				}
			}
		}
		if check {
			s.pushMetrics(driftMetrics(report, s.cfg.Metrics.Labels))
		}
	}
	r.Duration = time.Since(started).Seconds()
	return r, nil
}

func (s *server) pushMetrics(text string) {
	if s.cfg.Metrics.Push != "" {
		emitMetrics(text, "", s.cfg.Metrics.Push)
	}
}

// ---- HTTP

type serveIdentity struct {
	User     string
	Operator bool
}

var errServeAuth = errors.New("unauthorized")

// identify resolves who is calling: the bearer token (operator), the
// proxy's headers (viewer, operator by group), or nobody when no auth is
// configured (everything allowed)
func (s *server) identify(r *http.Request) (serveIdentity, error) {
	a := s.cfg.Auth
	if a.Token == "" && a.UserHeader == "" {
		return serveIdentity{User: "anonymous", Operator: true}, nil
	}
	if a.Token != "" {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(a.Token)) == 1 {
			return serveIdentity{User: "token", Operator: true}, nil
		}
	}
	if a.UserHeader != "" {
		if user := r.Header.Get(a.UserHeader); user != "" {
			id := serveIdentity{User: user}
			if a.GroupsHeader != "" {
				groups := strings.FieldsFunc(r.Header.Get(a.GroupsHeader), func(c rune) bool { return c == ',' || c == '|' || c == ' ' })
				for _, g := range groups {
					for _, op := range a.Operators {
						if strings.EqualFold(g, op) {
							id.Operator = true
						}
					}
				}
			}
			return id, nil
		}
	}
	return serveIdentity{}, errServeAuth
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

//go:embed serve_ui.html
var serveUI string

func (s *server) mux() *http.ServeMux {
	mux := http.NewServeMux()
	auth := func(operator bool, h func(http.ResponseWriter, *http.Request, serveIdentity)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id, err := s.identify(r)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			if operator && !id.Operator {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "operators only"})
				return
			}
			h(w, r, id)
		}
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok\n") })
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, strings.ReplaceAll(serveUI, "{{title}}", s.cfg.Title))
	})
	mux.HandleFunc("/api/me", auth(false, func(w http.ResponseWriter, _ *http.Request, id serveIdentity) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"user": id.User, "operator": id.Operator, "title": s.cfg.Title})
	}))
	mux.HandleFunc("/api/jobs", auth(false, func(w http.ResponseWriter, _ *http.Request, _ serveIdentity) {
		writeJSON(w, http.StatusOK, s.jobsView())
	}))
	mux.HandleFunc("/api/jobs/", auth(false, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
		name, action, _ := strings.Cut(rest, "/")
		j := s.job(name)
		if j == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such job"})
			return
		}
		switch {
		case action == "run" && r.Method == http.MethodPost:
			if !id.Operator {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "operators only"})
				return
			}
			if !s.enqueue(j, "api:"+id.User) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "already running or queued"})
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]string{"queued": j.Name})
		case action == "results" || action == "":
			s.mu.Lock()
			list := append([]*JobResult{}, s.results[j.Name]...)
			s.mu.Unlock()
			writeJSON(w, http.StatusOK, list)
		case strings.HasPrefix(action, "results/"):
			want := strings.TrimPrefix(action, "results/")
			s.mu.Lock()
			defer s.mu.Unlock()
			for _, res := range s.results[j.Name] {
				if res.ID == want {
					writeJSON(w, http.StatusOK, res)
					return
				}
			}
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such result"})
		default:
			http.NotFound(w, r)
		}
	}))
	mux.HandleFunc("/api/hosts", auth(false, func(w http.ResponseWriter, _ *http.Request, _ serveIdentity) {
		writeJSON(w, http.StatusOK, s.hostsView())
	}))
	mux.HandleFunc("/api/runs", auth(false, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
		if s.audit == nil {
			writeJSON(w, http.StatusOK, []interface{}{})
			return
		}
		records, err := s.audit.ListRecords(audit.FilterOptions{Limit: 100, HostFilter: r.URL.Query().Get("host"), PlaybookPath: r.URL.Query().Get("playbook"), SortBy: "time", SortOrder: "desc"})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		type row struct {
			ID       string    `json:"id"`
			Playbook string    `json:"playbook"`
			User     string    `json:"user"`
			Start    time.Time `json:"start"`
			Duration float64   `json:"duration"`
			Status   string    `json:"status"`
			Tasks    int       `json:"tasks"`
			Failed   int       `json:"failed"`
			Hosts    int       `json:"hosts"`
		}
		rows := make([]row, 0, len(records))
		for _, rec := range records {
			hosts := map[string]bool{}
			for _, p := range rec.Plays {
				for _, h := range p.Hosts {
					hosts[h] = true
				}
			}
			rows = append(rows, row{ID: rec.ID, Playbook: rec.PlaybookPath, User: rec.User, Start: rec.StartTime, Duration: rec.Duration, Status: string(rec.Status), Tasks: rec.TotalTasks, Failed: rec.FailedTasks, Hosts: len(hosts)})
		}
		writeJSON(w, http.StatusOK, rows)
	}))
	mux.HandleFunc("/api/runs/", auth(false, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
		if s.audit == nil {
			http.NotFound(w, r)
			return
		}
		rec, err := s.audit.LoadRecord(strings.TrimPrefix(r.URL.Path, "/api/runs/"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rec)
	}))
	return mux
}

type jobView struct {
	Name    string     `json:"name"`
	Kind    string     `json:"kind"`
	Target  string     `json:"target"`
	Every   string     `json:"every,omitempty"`
	Next    *time.Time `json:"next,omitempty"`
	Running bool       `json:"running"`
	Last    *JobResult `json:"last,omitempty"`
}

func (s *server) jobsView() []jobView {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]jobView, 0, len(s.cfg.Jobs))
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		v := jobView{Name: j.Name, Kind: j.Kind, Target: j.Playbook, Running: s.running[j.Name]}
		if j.Kind == "comply" {
			v.Target = j.Profile
		}
		if j.Every > 0 {
			v.Every = j.Every.String()
			if n, ok := s.nextRun[j.Name]; ok {
				next := n
				v.Next = &next
			}
		}
		if list := s.results[j.Name]; len(list) > 0 {
			last := *list[0]
			last.Report = nil
			v.Last = &last
		}
		out = append(out, v)
	}
	return out
}

type hostView struct {
	Host   string            `json:"host"`
	Status string            `json:"status"` // worst over the jobs' last results
	Jobs   map[string]string `json:"jobs"`
	Seen   time.Time         `json:"seen"`
}

func (s *server) hostsView() []hostView {
	s.mu.Lock()
	defer s.mu.Unlock()
	rank := map[string]int{"ok": 0, "changed": 1, "drift": 2, "failed": 3, "unreachable": 4}
	hosts := map[string]*hostView{}
	for job, list := range s.results {
		if len(list) == 0 {
			continue
		}
		for host, st := range list[0].Hosts {
			h := hosts[host]
			if h == nil {
				h = &hostView{Host: host, Status: "ok", Jobs: map[string]string{}}
				hosts[host] = h
			}
			h.Jobs[job] = st
			if rank[st] > rank[h.Status] {
				h.Status = st
			}
			if list[0].Started.After(h.Seen) {
				h.Seen = list[0].Started
			}
		}
	}
	out := make([]hostView, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, *h)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Host < out[k].Host })
	return out
}

func (s *server) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var b strings.Builder
	b.WriteString("# TYPE onigirazu_serve_job_last_status gauge\n# TYPE onigirazu_serve_job_last_run_timestamp_seconds gauge\n# TYPE onigirazu_serve_host_status gauge\n")
	codes := map[string]float64{"ok": 0, "drift": 1, "changed": 1, "failed": 2, "error": 3, "unreachable": 3}
	names := make([]string, 0, len(s.results))
	for n := range s.results {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if len(s.results[n]) == 0 {
			continue
		}
		last := s.results[n][0]
		promLine(&b, "onigirazu_serve_job_last_status", withLabels(map[string]string{"job": n, "kind": last.Kind}, s.cfg.Metrics.Labels), codes[last.Status])
		promLine(&b, "onigirazu_serve_job_last_run_timestamp_seconds", withLabels(map[string]string{"job": n}, s.cfg.Metrics.Labels), float64(last.Started.Unix()))
		hosts := make([]string, 0, len(last.Hosts))
		for h := range last.Hosts {
			hosts = append(hosts, h)
		}
		sort.Strings(hosts)
		for _, h := range hosts {
			promLine(&b, "onigirazu_serve_host_status", withLabels(map[string]string{"job": n, "host": h}, s.cfg.Metrics.Labels), codes[last.Hosts[h]])
		}
	}
	_, _ = io.WriteString(w, b.String())
}

func runServe(ctx context.Context, cfg *ServeConfig, out io.Writer) error {
	s, err := newServer(cfg, out)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go s.worker(ctx)
	go s.scheduler(ctx)
	srv := &http.Server{Addr: cfg.Listen, Handler: s.mux(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	open := ""
	if cfg.Auth.Token == "" && cfg.Auth.UserHeader == "" {
		open = " — no auth configured, everyone is an operator"
	}
	fmt.Fprintf(out, "serve: %s on %s: %d job(s), results in %s%s\n", cfg.Title, cfg.Listen, len(cfg.Jobs), cfg.DataDir, open)
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}

func installServeService(o *serveOptions, out io.Writer) error {
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return fmt.Errorf("serve install needs systemd")
	}
	if _, err := loadServeConfig(o.config); err != nil {
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
	args := self + " serve --config-file " + config
	if o.addr != "" {
		args += " --addr " + o.addr
	}
	unit := "[Unit]\nDescription=onigirazu serve: fleet server\nAfter=network-online.target\nWants=network-online.target\n\n[Service]\nExecStart=" + args + "\nRestart=on-failure\nRestartSec=5s\n\n[Install]\nWantedBy=multi-user.target\n"
	if err := os.WriteFile("/etc/systemd/system/onigirazu-serve.service", []byte(unit), 0o644); err != nil { // #nosec G306 -- unit files are world-readable
		return err
	}
	for _, c := range [][]string{{"systemctl", "daemon-reload"}, {"systemctl", "enable", "--now", "onigirazu-serve.service"}} {
		if o, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil { // #nosec G204 -- fixed systemctl arguments
			return fmt.Errorf("%s: %v: %s", strings.Join(c, " "), err, strings.TrimSpace(string(o)))
		}
	}
	fmt.Fprintf(out, "onigirazu-serve.service installed and started (%s)\n", config)
	return nil
}
