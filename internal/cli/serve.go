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
	Listen          string        `yaml:"listen"` // default :8086
	Title           string        `yaml:"title"`
	DataDir         string        `yaml:"data_dir"` // default ~/.onigirazu/serve
	Auth            ServeAuth     `yaml:"auth"`
	Jobs            []ServeJob    `yaml:"jobs"`
	Retain          int           `yaml:"retain"` // results kept per job, default 50
	Metrics         ServeMetric   `yaml:"metrics"`
	ApprovalTimeout time.Duration `yaml:"approval_timeout"` // how long a pending run waits, default 1h
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
	Name            string            `yaml:"name"`
	Kind            string            `yaml:"kind"` // drift (default), apply, verify, comply
	Playbook        string            `yaml:"playbook"`
	Profile         string            `yaml:"profile"` // comply
	Inventory       string            `yaml:"inventory"`
	Limit           string            `yaml:"limit"`
	ExtraVars       map[string]string `yaml:"extra_vars"`
	Become          bool              `yaml:"become"`
	Every           time.Duration     `yaml:"every"`            // 0: only on demand
	Approval        string            `yaml:"approval"`         // required: a run waits for an operator
	ApprovalTimeout time.Duration     `yaml:"approval_timeout"` // 0: the server's
	Notify          []string          `yaml:"notify"`
	NotifyOK        bool              `yaml:"notify_always"`
	FailOn          string            `yaml:"fail_on"` // comply
}

// JobResult is one run of a job, as stored
type JobResult struct {
	ID        string            `json:"id"`
	Job       string            `json:"job"`
	Kind      string            `json:"kind"`
	Requested time.Time         `json:"requested"` // when it was asked for
	Started   time.Time         `json:"started"`   // the request time until the run starts
	Duration  float64           `json:"duration_seconds"`
	Status    string            `json:"status"` // pending, queued, running, ok, drift, failed, error, rejected, expired
	Summary   string            `json:"summary"`
	Trigger   string            `json:"trigger"`            // schedule, api:<user>
	Hosts     map[string]string `json:"hosts,omitempty"`    // host -> ok|drift|failed|unreachable
	Approval  *RunApproval      `json:"approval,omitempty"` // jobs with approval: required
	Report    json.RawMessage   `json:"report,omitempty"`
}

// RunApproval is the approval state of a guarded run
type RunApproval struct {
	Required bool       `json:"required"`
	Expires  time.Time  `json:"expires"`
	Decision string     `json:"decision,omitempty"`
	By       string     `json:"by,omitempty"`
	At       *time.Time `json:"at,omitempty"`
	Comment  string     `json:"comment,omitempty"`
}

// executed: the run ran (as opposed to waiting, rejected or expired)
func (r *JobResult) executed() bool {
	switch r.Status {
	case "ok", "drift", "changed", "failed", "error":
		return true
	}
	return false
}

// resultCopy is a copy safe to hand out of the lock
func resultCopy(r *JobResult) *JobResult {
	copy := *r
	if r.Approval != nil {
		approval := *r.Approval
		copy.Approval = &approval
	}
	return &copy
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
	if cfg.ApprovalTimeout == 0 {
		cfg.ApprovalTimeout = time.Hour
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
		if j.Approval != "" && j.Approval != "required" {
			return nil, fmt.Errorf("job %s: approval %q (required or empty)", j.Name, j.Approval)
		}
		if j.ApprovalTimeout == 0 {
			j.ApprovalTimeout = cfg.ApprovalTimeout
		}
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
	running map[string]bool         // job is pending, queued or running
	pending map[string]*JobResult
	lastID  map[string]string // job -> the base of the last id given (ids stay unique when records are pruned)
	lastSeq map[string]int
	nextRun map[string]time.Time
	queue   chan queued
	// runJob runs a job; tests replace it
	runJob func(j *ServeJob) (*JobResult, error)
	audit  *audit.Storage
}

type queued struct {
	job *ServeJob
	run *JobResult
}

func newServer(cfg *ServeConfig, out io.Writer) (*server, error) {
	s := &server{cfg: cfg, out: out, results: map[string][]*JobResult{}, running: map[string]bool{}, pending: map[string]*JobResult{}, lastID: map[string]string{}, lastSeq: map[string]int{}, nextRun: map[string]time.Time{}, queue: make(chan queued, 64)}
	s.runJob = s.execute
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "results"), 0o700); err != nil {
		return nil, err
	}
	s.loadResults()
	s.expirePending(time.Now())
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
			if r.Requested.IsZero() {
				r.Requested = r.Started
			}
			s.results[r.Job] = append(s.results[r.Job], &r)
		}
	}
	for job := range s.results {
		sort.Slice(s.results[job], func(i, k int) bool { return s.results[job][i].Requested.After(s.results[job][k].Requested) })
	}
	for _, list := range s.results {
		for _, r := range append([]*JobResult(nil), list...) {
			switch r.Status {
			case "pending":
				if s.job(r.Job) == nil { // removed from serve.yml meanwhile
					r.Status, r.Summary = "error", "interrupted: job removed from serve.yml"
					s.store(r)
					continue
				}
				s.pending[r.Job] = r
				s.running[r.Job] = true
			case "queued", "running":
				r.Status = "error"
				r.Summary = "interrupted: serve restarted"
				s.store(r)
			}
		}
	}
}

// store upserts a result; the caller holds mu.
func (s *server) store(r *JobResult) {
	list := s.results[r.Job]
	found := false
	for i, old := range list {
		if old.ID == r.ID {
			list[i] = r
			found = true
			break
		}
	}
	if !found {
		list = append([]*JobResult{r}, list...)
	}
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

var (
	errJobBusy      = errors.New("already running, queued or pending")
	errNoPendingRun = errors.New("no such pending run")
	errRunExpired   = errors.New("the run expired before the decision")
	errQueueFull    = errors.New("queue full")
)

// resultByID finds a record of a job; the caller holds mu
func (s *server) resultByID(job, id string) *JobResult {
	for _, r := range s.results[job] {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// nextID: <job>-<second>, with a sequence number for further requests in
// the same second — counted in memory, so a pruned record cannot free an id
// (an approval of an old id must never reach a new run); the caller holds mu
func (s *server) nextID(job string, now time.Time) string {
	base := fmt.Sprintf("%s-%s", job, now.Format("20060102T150405Z"))
	if s.lastID[job] == base {
		s.lastSeq[job]++
	} else {
		s.lastID[job], s.lastSeq[job] = base, 1
	}
	id := base
	for n := s.lastSeq[job]; n > 1 || s.resultByID(job, id) != nil; n++ { // records of an earlier process too
		id = fmt.Sprintf("%s-%d", base, n)
		s.lastSeq[job] = n
		if s.resultByID(job, id) == nil {
			break
		}
	}
	return id
}

// request creates a run: pending when the job wants approval, else queued
func (s *server) request(j *ServeJob, trigger string) (*JobResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[j.Name] {
		return nil, errJobBusy
	}
	now := time.Now().UTC()
	id := s.nextID(j.Name, now)
	r := &JobResult{ID: id, Job: j.Name, Kind: j.Kind, Requested: now, Started: now, Trigger: trigger, Status: "queued"}
	if j.Approval == "required" {
		r.Status = "pending"
		r.Approval = &RunApproval{Required: true, Expires: now.Add(j.ApprovalTimeout)}
		s.pending[j.Name] = r
	} else {
		select {
		case s.queue <- queued{job: j, run: r}:
		default:
			return nil, errQueueFull
		}
	}
	s.running[j.Name] = true
	s.store(r)
	return resultCopy(r), nil
}

func (s *server) approve(id, by, comment string) (*JobResult, error) {
	return s.decide(id, by, comment, "approved")
}

func (s *server) reject(id, by, comment string) (*JobResult, error) {
	return s.decide(id, by, comment, "rejected")
}

// decide settles a pending run: approved runs queue, rejected ones free the job
func (s *server) decide(id, by, comment, decision string) (*JobResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for job, r := range s.pending {
		if r.ID != id {
			continue
		}
		now := time.Now().UTC()
		if r.Approval.Expires.Before(now) { // the scheduler has not noticed yet
			s.expire(job, r, now)
			return nil, errRunExpired
		}
		if decision == "approved" {
			select {
			case s.queue <- queued{job: s.job(job), run: r}:
			default:
				return nil, errQueueFull
			}
			r.Status = "queued"
		} else {
			r.Status = "rejected"
			s.running[job] = false
		}
		r.Approval.Decision, r.Approval.By, r.Approval.At, r.Approval.Comment = decision, by, &now, comment
		delete(s.pending, job)
		s.store(r)
		return resultCopy(r), nil
	}
	return nil, errNoPendingRun
}

// expire ends one pending run; the caller holds mu
func (s *server) expire(job string, r *JobResult, now time.Time) {
	r.Approval.Decision, r.Approval.At = "expired", &now
	r.Status = "expired"
	delete(s.pending, job)
	s.running[job] = false
	s.store(r)
}

// expirePending ends the pending runs whose wait is over
func (s *server) expirePending(now time.Time) []*JobResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	var expired []*JobResult
	now = now.UTC()
	for job, r := range s.pending {
		if r.Approval == nil || !r.Approval.Expires.Before(now) {
			continue
		}
		s.expire(job, r, now)
		expired = append(expired, resultCopy(r))
	}
	return expired
}

// worker runs queued jobs one at a time
func (s *server) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case q := <-s.queue:
			s.mu.Lock()
			r := q.run
			r.Status, r.Started = "running", time.Now().UTC()
			s.store(r)
			s.mu.Unlock()
			fmt.Fprintf(s.out, "serve: %s: running (%s)\n", q.job.Name, r.Trigger)
			result, err := s.runJob(q.job)
			s.mu.Lock()
			if err != nil {
				r.Status, r.Summary = "error", err.Error()
			} else {
				r.Started, r.Duration, r.Status, r.Summary, r.Hosts, r.Report = result.Started, result.Duration, result.Status, result.Summary, result.Hosts, result.Report
			}
			s.store(r)
			s.running[q.job.Name] = false
			fmt.Fprintf(s.out, "serve: %s: %s — %s (%.1fs)\n", q.job.Name, r.Status, r.Summary, r.Duration)
			s.mu.Unlock()
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
			s.expirePending(now)
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
				_, _ = s.request(j, "schedule")
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
			run, err := s.request(j, "api:"+id.User)
			if err != nil {
				code := http.StatusServiceUnavailable
				if errors.Is(err, errJobBusy) {
					code = http.StatusConflict
				}
				writeJSON(w, code, map[string]string{"error": err.Error()})
				return
			}
			if run.Status == "pending" {
				writeJSON(w, http.StatusAccepted, map[string]string{"pending": run.ID, "run": run.ID})
			} else {
				writeJSON(w, http.StatusAccepted, map[string]string{"queued": j.Name, "run": run.ID})
			}
		case action == "results" || action == "":
			s.mu.Lock()
			list := make([]*JobResult, 0, len(s.results[j.Name]))
			for _, r := range s.results[j.Name] {
				list = append(list, resultCopy(r))
			}
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
	mux.HandleFunc("/api/results/", auth(false, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
		s.mu.Lock()
		defer s.mu.Unlock()
		id := strings.TrimPrefix(r.URL.Path, "/api/results/")
		for _, list := range s.results {
			for _, result := range list {
				if result.ID == id {
					writeJSON(w, http.StatusOK, result)
					return
				}
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such result"})
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
	mux.HandleFunc("/api/runs/", auth(false, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
		runID, action, _ := strings.Cut(path, "/")
		if r.Method == http.MethodPost && (action == "approve" || action == "reject") {
			auth(true, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
				var body struct {
					Comment string `json:"comment"`
				}
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					body.Comment = ""
				}
				decide := s.approve
				if action == "reject" {
					decide = s.reject
				}
				result, err := decide(runID, id.User, body.Comment)
				if err != nil {
					code := http.StatusServiceUnavailable
					switch {
					case errors.Is(err, errNoPendingRun):
						code = http.StatusNotFound
					case errors.Is(err, errRunExpired):
						code = http.StatusConflict
					}
					writeJSON(w, code, map[string]string{"error": err.Error()})
					return
				}
				result.Report = nil
				writeJSON(w, http.StatusOK, result)
			})(w, r)
			return
		}
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
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	Target   string     `json:"target"`
	Approval string     `json:"approval,omitempty"` // required
	Every    string     `json:"every,omitempty"`
	Next     *time.Time `json:"next,omitempty"`
	Running  bool       `json:"running"`           // queued or running
	Pending  *JobResult `json:"pending,omitempty"` // the run that waits for approval
	Last     *JobResult `json:"last,omitempty"`    // the last executed run
}

func (s *server) jobsView() []jobView {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]jobView, 0, len(s.cfg.Jobs))
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		v := jobView{Name: j.Name, Kind: j.Kind, Target: j.Playbook, Running: s.running[j.Name] && s.pending[j.Name] == nil, Approval: j.Approval}
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
		if pending := s.pending[j.Name]; pending != nil {
			v.Pending = resultCopy(pending)
			v.Pending.Report = nil
		}
		if result := s.lastExecuted(j.Name); result != nil {
			last := resultCopy(result)
			last.Report = nil
			v.Last = last
		}
		out = append(out, v)
	}
	return out
}

// lastExecuted returns the newest executed result; the caller holds mu.
func (s *server) lastExecuted(job string) *JobResult {
	for _, r := range s.results[job] {
		if r.executed() {
			return r
		}
	}
	return nil
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
	for job := range s.results {
		last := s.lastExecuted(job)
		if last == nil {
			continue
		}
		for host, st := range last.Hosts {
			h := hosts[host]
			if h == nil {
				h = &hostView{Host: host, Status: "ok", Jobs: map[string]string{}}
				hosts[host] = h
			}
			h.Jobs[job] = st
			if rank[st] > rank[h.Status] {
				h.Status = st
			}
			if last.Started.After(h.Seen) {
				h.Seen = last.Started
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
	b.WriteString("# TYPE onigirazu_serve_job_pending gauge\n")
	for _, j := range s.cfg.Jobs {
		if j.Approval == "required" {
			value := float64(0)
			if s.pending[j.Name] != nil {
				value = 1
			}
			promLine(&b, "onigirazu_serve_job_pending", withLabels(map[string]string{"job": j.Name}, s.cfg.Metrics.Labels), value)
		}
	}
	codes := map[string]float64{"ok": 0, "drift": 1, "changed": 1, "failed": 2, "error": 3, "unreachable": 3}
	names := make([]string, 0, len(s.results))
	for n := range s.results {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		last := s.lastExecuted(n)
		if last == nil {
			continue
		}
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
