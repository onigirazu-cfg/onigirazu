package cli

import (
	"cmp"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// serve: a small fleet server in the same binary. It runs jobs on a schedule
// (drift checks, applies, verify, compliance), keeps their results, serves
// the run history of the audit store and a web page over it, and a REST API
// with a bearer token or a reverse proxy's identity headers.

// ServeConfig is the serve.yml file
type ServeConfig struct {
	Concurrency     int            `yaml:"concurrency"` // default 1
	Listen          string         `yaml:"listen"`      // default :8086
	Title           string         `yaml:"title"`
	URL             string         `yaml:"url"`      // where the page is reached, for links in notifications
	DataDir         string         `yaml:"data_dir"` // default ~/.onigirazu/serve
	Auth            ServeAuth      `yaml:"auth"`
	Jobs            []ServeJob     `yaml:"jobs"`
	Retain          int            `yaml:"retain"` // results kept per job, default 50
	RetainAge       time.Duration  `yaml:"retain_age"`
	Metrics         ServeMetric    `yaml:"metrics"`
	Notify          []NotifyTarget `yaml:"notify"`           // targets for every job
	ApprovalTimeout time.Duration  `yaml:"approval_timeout"` // how long a pending run waits, default 1h
	RunTimeout      time.Duration  `yaml:"run_timeout"`      // 0: no run limit
}

// ServeAuth: a bearer token is an operator (the plain one is the user
// "token", named ones carry a name and groups a job's operators list can
// match); a reverse proxy's user header is a viewer, an operator when one of
// its groups is in operators. Without any of them everything is open — put a
// proxy in front.
type ServeAuth struct {
	Token        string       `yaml:"token"` // ${VAR} expanded
	Tokens       []ServeToken `yaml:"tokens"`
	UserHeader   string       `yaml:"user_header"`
	GroupsHeader string       `yaml:"groups_header"`
	Operators    []string     `yaml:"operators"`
}

// ServeToken is a named bearer identity
type ServeToken struct {
	Name   string   `yaml:"name"`
	Token  string   `yaml:"token"`
	Groups []string `yaml:"groups"`
}

// ServeMetric is an optional push of every job's metrics
type ServeMetric struct {
	Push   string            `yaml:"push"`
	Labels map[string]string `yaml:"labels"`
}

// ServeAdhoc is what a caller of POST /api/jobs/{name}/run may set
type ServeAdhoc struct {
	Limit      bool       `yaml:"limit" json:"limit"`
	ExtraVars  []string   `yaml:"-" json:"extra_vars,omitempty"`
	SecretVars []string   `yaml:"-" json:"secret_vars,omitempty"`
	Vars       []adhocVar `yaml:"-" json:"vars,omitempty"`
}

// adhocVar accepts a name or a typed declaration.
type adhocVar struct {
	Name        string   `yaml:"name" json:"name"`
	Secret      bool     `yaml:"secret" json:"secret"`
	Type        string   `yaml:"type" json:"type"`
	Choices     []string `yaml:"choices" json:"choices,omitempty"`
	Default     *string  `yaml:"default" json:"default,omitempty"`
	Pattern     string   `yaml:"pattern" json:"pattern,omitempty"`
	Min         *int64   `yaml:"min" json:"min,omitempty"`
	Max         *int64   `yaml:"max" json:"max,omitempty"`
	Description string   `yaml:"description" json:"description,omitempty"`
	Required    bool     `yaml:"required" json:"required"`
}

func (v adhocVar) MarshalJSON() ([]byte, error) {
	type plain adhocVar
	if v.Secret {
		v.Default = nil
	}
	return json.Marshal(plain(v))
}

func (v adhocVar) validate(value string) error {
	rule := ""
	switch v.Type {
	case "int":
		n, err := strconv.ParseInt(value, 10, 64)
		switch {
		case err != nil:
			rule = "must be an integer"
		case v.Min != nil && n < *v.Min:
			rule = fmt.Sprintf("must be >= %d", *v.Min)
		case v.Max != nil && n > *v.Max:
			rule = fmt.Sprintf("must be <= %d", *v.Max)
		}
	case "bool":
		if value != "true" && value != "false" {
			rule = "must be true or false"
		}
	case "choice":
		if !slices.Contains(v.Choices, value) {
			rule = "must be one of the declared choices"
		}
	}
	if rule == "" && v.Pattern != "" {
		re, err := regexp.Compile(`\A(?:` + v.Pattern + `)\z`)
		if err != nil || !re.MatchString(value) {
			rule = "must match the declared pattern"
		}
	}
	if rule != "" {
		return fmt.Errorf("extra var %q %s", v.Name, rule)
	}
	return nil
}

func (v *adhocVar) validateDeclaration() error {
	if v.Type == "" {
		v.Type = "string"
	}
	switch v.Type {
	case "string", "int", "bool", "choice":
	default:
		return fmt.Errorf("extra var %q: type must be string, int, bool or choice", v.Name)
	}
	if (v.Type == "choice" && len(v.Choices) == 0) || (v.Type != "choice" && v.Choices != nil) {
		return fmt.Errorf("extra var %q: choices must be non-empty and only used with type choice", v.Name)
	}
	if v.Secret && (v.Type == "bool" || v.Type == "choice") {
		return fmt.Errorf("extra var %q: secret cannot use type bool or choice", v.Name)
	}
	if v.Min != nil && v.Max != nil && *v.Min > *v.Max {
		return fmt.Errorf("extra var %q: min must be <= max", v.Name)
	}
	if (v.Min != nil || v.Max != nil) && v.Type != "int" {
		return fmt.Errorf("extra var %q: min and max require type int", v.Name)
	}
	if v.Pattern != "" {
		if _, err := regexp.Compile(`\A(?:` + v.Pattern + `)\z`); err != nil {
			return fmt.Errorf("extra var %q: pattern must be valid RE2", v.Name)
		}
	}
	if v.Default != nil {
		if err := v.validate(*v.Default); err != nil {
			return fmt.Errorf("invalid default: %w", err)
		}
	}
	return nil
}

func (v *adhocVar) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		v.Name = node.Value
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return errors.New("adhoc.extra_vars entries must be names or mappings")
	}
	type plain adhocVar
	return node.Decode((*plain)(v))
}

// UnmarshalYAML keeps the public list of names compatible with existing clients.
func (a *ServeAdhoc) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		Limit     bool       `yaml:"limit"`
		ExtraVars []adhocVar `yaml:"extra_vars"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}
	*a = ServeAdhoc{Limit: raw.Limit}
	for _, v := range raw.ExtraVars {
		if strings.TrimSpace(v.Name) == "" {
			return errors.New("adhoc.extra_vars: name must not be empty")
		}
		if slices.Contains(a.ExtraVars, v.Name) {
			return fmt.Errorf("adhoc.extra_vars: duplicate name %q", v.Name)
		}
		if v.Type == "" {
			v.Type = "string"
		}
		a.Vars = append(a.Vars, v)
		a.ExtraVars = append(a.ExtraVars, v.Name)
		if v.Secret {
			a.SecretVars = append(a.SecretVars, v.Name)
		}
	}
	return nil
}

// RunParams is what the caller set on a run, with the limit resolved
type RunParams struct {
	NotBefore  *time.Time        `json:"-"`
	Mode       string            `json:"-"`
	SecretVars []string          `json:"secret_vars,omitempty"`
	Limit      string            `json:"limit,omitempty"`
	ExtraVars  map[string]string `json:"extra_vars,omitempty"`
	Hosts      []string          `json:"hosts,omitempty"`
}

// values is the extra vars as k=v, sorted
func (p *RunParams) values() []string {
	keys := slices.Sorted(maps.Keys(p.ExtraVars))
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, k+"="+p.ExtraVars[k])
	}
	return values
}

// ServeJob is a scheduled run
type ServeJob struct {
	Windows         []string `yaml:"windows"`
	WindowsTZ       string   `yaml:"windows_tz"`
	windows         []serveWindow
	windowsLoc      *time.Location
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
	CheckApproval   string            `yaml:"check_approval"`   // required: check runs wait for an operator
	ApprovalTimeout time.Duration     `yaml:"approval_timeout"` // 0: the server's
	Timeout         time.Duration     `yaml:"timeout"`          // 0: the server's
	Operators       []string          `yaml:"operators"`        // users or groups that may run, approve, reject; empty: every operator
	Adhoc           ServeAdhoc        `yaml:"adhoc"`            // what a caller may set on a run
	Notify          []NotifyTarget    `yaml:"notify"`
	NotifyOK        bool              `yaml:"notify_always"`
	FailOn          string            `yaml:"fail_on"` // comply
}

const (
	statusCancelling = "cancelling" //nolint:misspell // Public API spelling.
	statusCancelled  = "cancelled"  //nolint:misspell // Public API spelling.
	statusTimedOut   = "timed_out"
)

// JobResult is one run of a job, as stored
type JobResult struct {
	AuditExecutionID string            `json:"audit_execution_id,omitempty"`
	PlaybookHash     string            `json:"playbook_hash,omitempty"`
	ID               string            `json:"id"`
	Parent           string            `json:"parent,omitempty"`
	Job              string            `json:"job"`
	Kind             string            `json:"kind"`
	Mode             string            `json:"mode,omitempty"`
	NotBefore        *time.Time        `json:"not_before,omitempty"`
	Requested        time.Time         `json:"requested"` // when it was asked for
	Started          time.Time         `json:"started"`   // the request time until the run starts
	Duration         float64           `json:"duration_seconds"`
	Status           string            `json:"status"` // pending, scheduled, queued, running, statusCancelling, statusCancelled, timed_out, ok, drift, changed, failed, error, rejected, expired
	Summary          string            `json:"summary"`
	Trigger          string            `json:"trigger"`          // schedule, api:<user>
	Params           *RunParams        `json:"params,omitempty"` // what the caller set
	Hosts            map[string]string `json:"hosts,omitempty"`  // host -> ok|drift|failed|unreachable
	Cancel           *RunCancel        `json:"cancel,omitempty"`
	Approval         *RunApproval      `json:"approval,omitempty"` // jobs with approval: required
	Report           json.RawMessage   `json:"report,omitempty"`
}

// RunCancel records who canceled a run.
type RunCancel struct {
	By      string    `json:"by"`
	At      time.Time `json:"at"`
	Comment string    `json:"comment,omitempty"`
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

// executed reports terminal results, including cancellation.
func (r *JobResult) executed() bool {
	switch r.Status {
	case "ok", "drift", "changed", "failed", "error", statusCancelled, statusTimedOut:
		return true
	}
	return false
}

// resultCopy is a copy safe to hand out of the lock
func resultCopy(r *JobResult) *JobResult {
	copy := *r
	if r.NotBefore != nil {
		at := *r.NotBefore
		copy.NotBefore = &at
	}
	copy.Hosts = maps.Clone(r.Hosts)
	if r.Params != nil {
		params := *r.Params
		params.ExtraVars = maps.Clone(r.Params.ExtraVars)
		params.Hosts = slices.Clone(r.Params.Hosts)
		params.SecretVars = slices.Clone(r.Params.SecretVars)
		copy.Params = &params
	}
	if r.Cancel != nil {
		cancellation := *r.Cancel
		copy.Cancel = &cancellation
	}
	copy.Report = append(json.RawMessage(nil), r.Report...)
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
	if cfg.Concurrency < 0 {
		return nil, fmt.Errorf("concurrency must be >= 1")
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 1
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
	if cfg.RunTimeout < 0 {
		return nil, fmt.Errorf("run_timeout must be >= 0")
	}
	if cfg.RetainAge < 0 {
		return nil, fmt.Errorf("retain_age must be >= 0")
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
	tokenNames := map[string]bool{}
	for i := range cfg.Auth.Tokens {
		t := &cfg.Auth.Tokens[i]
		t.Token = os.ExpandEnv(t.Token)
		if strings.TrimSpace(t.Name) == "" || t.Token == "" {
			return nil, fmt.Errorf("auth.tokens: entry %d needs a name and a value", i+1)
		}
		name := strings.ToLower(t.Name)
		if tokenNames[name] {
			return nil, fmt.Errorf("auth.tokens: duplicate name %q", t.Name)
		}
		tokenNames[name] = true
	}
	if err := validateNotifyTargets(cfg.Notify); err != nil {
		return nil, fmt.Errorf("notify: %w", err)
	}
	base := filepath.Dir(path)
	names := map[string]bool{}
	for i := range cfg.Jobs {
		j := &cfg.Jobs[i]
		for n := range j.Adhoc.Vars {
			if err := j.Adhoc.Vars[n].validateDeclaration(); err != nil {
				return nil, fmt.Errorf("job %s: %w", j.Name, err)
			}
		}
		if j.WindowsTZ == "" {
			j.WindowsTZ = "UTC"
		}
		j.windows, j.windowsLoc, err = parseServeWindows(j.Windows, j.WindowsTZ)
		if err != nil {
			return nil, fmt.Errorf("job %s: %w", j.Name, err)
		}
		if err := validateNotifyTargets(j.Notify); err != nil {
			return nil, fmt.Errorf("job %s: notify: %w", j.Name, err)
		}
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
		if j.CheckApproval != "" && j.CheckApproval != "required" {
			return nil, fmt.Errorf("job %s: check_approval %q (required or empty)", j.Name, j.CheckApproval)
		}
		if j.Timeout < 0 {
			return nil, fmt.Errorf("job %s: timeout must be >= 0", j.Name)
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
	inventoryMu        sync.Mutex
	inventorySnapshots map[string]*serveInventorySnapshot
	// inventoryLoad replaces the loader in tests.
	inventoryLoad    func([]string) (*types.Inventory, error)
	cfg              *ServeConfig
	out              io.Writer
	auditMu          sync.Mutex
	pickupMu         sync.Mutex // dequeue through the running transition, in FIFO order
	mu               sync.Mutex
	storeErr         string
	results          map[string][]*JobResult      // job -> newest first
	running          map[string]bool              // job is pending, queued or running
	secrets          map[string]map[string]string // run -> secret parameters, never stored
	progress         map[string]*runProgress
	finishedProgress []string
	cancels          map[string]context.CancelFunc
	pending          map[string]*JobResult
	lastID           map[string]string // job -> the base of the last id given (ids stay unique when records are pruned)
	lastSeq          map[string]int
	schedulerTick    time.Duration
	nextRun          map[string]time.Time
	pause            servePause
	resumed          chan struct{}
	queue            chan queued
	// runJob runs a job; tests replace it
	runJob func(ctx context.Context, j *ServeJob, mode string) (*JobResult, error)
	audit  *audit.Storage
	// notifications go out in goroutines; tests set the client and wait
	notifyClient *http.Client
	notifyWait   sync.WaitGroup
	inventory    []string // the -i sources the server started with, for jobs without their own
}

type servePause struct {
	Paused bool      `json:"paused"`
	Reason string    `json:"reason,omitempty"`
	By     string    `json:"by,omitempty"`
	At     time.Time `json:"at,omitempty"`
}

type queued struct {
	job *ServeJob
	run *JobResult
}

func newServer(cfg *ServeConfig, out io.Writer) (*server, error) {
	if cfg.Retain <= 0 {
		cfg.Retain = 50
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 1
	}
	for i := range cfg.Jobs {
		j := &cfg.Jobs[i]
		if j.WindowsTZ == "" {
			j.WindowsTZ = "UTC"
		}
		var err error
		j.windows, j.windowsLoc, err = parseServeWindows(j.Windows, j.WindowsTZ)
		if err != nil {
			return nil, fmt.Errorf("job %s: %w", j.Name, err)
		}
	}
	s := &server{cancels: map[string]context.CancelFunc{}, cfg: cfg, out: out, results: map[string][]*JobResult{}, running: map[string]bool{}, pending: map[string]*JobResult{}, lastID: map[string]string{}, lastSeq: map[string]int{}, nextRun: map[string]time.Time{}, queue: make(chan queued, 64), resumed: make(chan struct{})}
	s.secrets = map[string]map[string]string{}
	s.progress = make(map[string]*runProgress)
	s.runJob = s.execute
	s.inventory = slices.Clone(inventoryPaths)
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "results"), 0o700); err != nil {
		return nil, err
	}
	s.loadPause()
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

// loadPause reads data_dir/pause.json; a file that is there but cannot be
// read pauses the server (fail closed: an operator resumes it explicitly)
func (s *server) loadPause() {
	path := filepath.Join(s.cfg.DataDir, "pause.json")
	data, err := os.ReadFile(path) // #nosec G304 -- our own data directory
	if err == nil {
		err = json.Unmarshal(data, &s.pause)
	}
	if err != nil && !os.IsNotExist(err) {
		s.pause = servePause{Paused: true, Reason: fmt.Sprintf("%s unreadable (%v): resume explicitly", path, err), By: "serve", At: time.Now().UTC()}
		fmt.Fprintf(os.Stderr, "serve: %s\n", s.pause.Reason)
	}
}

// setPause pauses or resumes and writes the state; when the file cannot be
// written the new state still holds in this process and the error says so
func (s *server) setPause(paused bool, by, reason string) (servePause, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !paused && !s.pause.Paused {
		return s.pause, nil
	}
	if !paused {
		reason = ""
		close(s.resumed)
		s.resumed = make(chan struct{})
	}
	s.pause = servePause{Paused: paused, Reason: reason, By: by, At: time.Now().UTC()}
	if paused {
		fmt.Fprintf(s.out, "serve: paused by %s: %s\n", by, reason)
	} else {
		fmt.Fprintf(s.out, "serve: resumed by %s\n", by)
	}
	path := filepath.Join(s.cfg.DataDir, "pause.json")
	data, err := json.Marshal(s.pause)
	if err == nil { // whole or not at all: a crash mid-write must not leave half a file
		if err = os.WriteFile(path+".tmp", data, 0o600); err == nil {
			err = os.Rename(path+".tmp", path)
		}
	}
	if err != nil {
		err = fmt.Errorf("pause state not saved, it holds until a restart: %w", err)
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
	}
	return s.pause, err
}

func (s *server) paused() (servePause, <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pause, s.resumed
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
	for job := range s.results {
		if err := s.pruneResults(job, time.Now()); err != nil {
			s.storeErr = err.Error()
		}
	}
	for _, list := range s.results {
		for _, r := range append([]*JobResult(nil), list...) {
			if (r.Status == "pending" || r.Status == "scheduled" || r.Status == "queued") && s.hasSecretParams(r) {
				r.Status = "error"
				r.Summary = "interrupted: serve restarted (secret parameters are not kept)"
				s.saveResult(r)
				continue
			}
			switch r.Status {
			case "scheduled":
				if s.job(r.Job) == nil {
					r.Status, r.Summary = "error", "interrupted: job removed from serve.yml"
					s.saveResult(r)
				}
			case "pending":
				if s.job(r.Job) == nil { // removed from serve.yml meanwhile
					r.Status, r.Summary = "error", "interrupted: job removed from serve.yml"
					s.saveResult(r)
					continue
				}
				s.pending[r.Job] = r
				s.running[r.Job] = true
			case statusCancelling:
				r.Status, r.Summary = statusCancelled, "interrupted: serve restarted"
				s.saveResult(r)
			case "queued", "running":
				r.Status = "error"
				r.Summary = "interrupted: serve restarted"
				s.saveResult(r)
			}
		}
	}
}

func (s *server) hasSecretParams(r *JobResult) bool {
	return r.Params != nil && len(r.Params.SecretVars) > 0
}

// store upserts a result; the caller holds mu.
func (s *server) store(r *JobResult) error {
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
	s.results[r.Job] = list
	err := errors.Join(s.writeResult(r), s.pruneResults(r.Job, time.Now()))
	s.storageError(err)
	return err
}

func (s *server) storageError(err error) {
	s.storeErr = ""
	if err != nil {
		s.storeErr = err.Error()
	}
}

// saveResult persists a recovered result without pruning its former active state.
func (s *server) saveResult(r *JobResult) {
	err := s.writeResult(r)
	s.storageError(err)
	if err != nil {
		fmt.Fprintf(s.out, "serve: %s: result not saved: %v\n", r.Job, err)
	}
}

func (s *server) writeResult(r *JobResult) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	path := filepath.Join(s.cfg.DataDir, "results", r.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return errors.Join(err, removeResultFile(tmp))
	}
	if err := os.Rename(tmp, path); err != nil {
		return errors.Join(err, removeResultFile(tmp))
	}
	return nil
}

func removeResultFile(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// pruneResults retries failed removals on the next store or restart.
func (s *server) pruneResults(job string, now time.Time) error {
	list := s.results[job]
	kept := make([]*JobResult, 0, len(list))
	var errs []error
	finished := 0 // active runs do not use up the count limit
	for _, r := range list {
		active := r.Status == "pending" || r.Status == "scheduled" || r.Status == "queued" || r.Status == "running" || r.Status == statusCancelling
		old := s.cfg.RetainAge > 0 && !active && r.Requested.Before(now.Add(-s.cfg.RetainAge))
		if active {
			kept = append(kept, r)
			continue
		}
		finished++
		if finished <= s.cfg.Retain && !old {
			kept = append(kept, r)
			continue
		}
		if err := removeResultFile(filepath.Join(s.cfg.DataDir, "results", r.ID+".json")); err != nil {
			errs = append(errs, err)
			kept = append(kept, r)
		}
	}
	s.results[job] = kept
	return errors.Join(errs...)
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
	errAlreadyScheduled  = errors.New("a run is already scheduled for this job")
	errNoSuchRun         = errors.New("no such run")
	errRunNotCancellable = errors.New("run is not scheduled, queued or running; for a pending run, reject it instead")
	errJobBusy           = errors.New("already running, queued or pending")
	errNoPendingRun      = errors.New("no such pending run")
	errRunExpired        = errors.New("the run expired before the decision")
	errQueueFull         = errors.New("queue full")
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

// request creates a run, awaiting approval, its start time or a worker.
func (s *server) request(j *ServeJob, trigger string, params *RunParams) (*JobResult, error) {
	return s.requestParent(j, trigger, params, "")
}

func (s *server) requestParent(j *ServeJob, trigger string, params *RunParams, parent string) (*JobResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var notBefore *time.Time
	mode := ""
	if params != nil {
		notBefore, mode = params.NotBefore, params.Mode
	}
	future := notBefore != nil && notBefore.After(now)
	gated := j.gated(mode) && !j.windowOpen(now)
	// one explicit request for later per job; the schedule keeps one deferral, a
	// caller's run outside the window is deferred next to it
	if (notBefore != nil || (gated && trigger == "schedule")) && s.scheduledRun(j.Name) != nil {
		return nil, errAlreadyScheduled
	}
	needsApproval := (mode == "check" && j.CheckApproval == "required") || (mode != "check" && j.Approval == "required")
	if s.running[j.Name] && (needsApproval || (!future && !gated)) {
		return nil, errJobBusy
	}
	id := s.nextID(j.Name, now)
	r := &JobResult{ID: id, Parent: parent, Job: j.Name, Kind: j.Kind, Requested: now, Started: now, Trigger: trigger, Status: "queued", Params: params, NotBefore: notBefore}
	if params != nil {
		r.Mode = params.Mode
		r.Params = resultCopy(r).Params
		r.Params.SecretVars = nil
		for _, name := range j.Adhoc.SecretVars {
			if value, ok := r.Params.ExtraVars[name]; ok {
				if s.secrets[id] == nil {
					s.secrets[id] = map[string]string{}
				}
				s.secrets[id][name] = value
				r.Params.ExtraVars[name] = "***"
				r.Params.SecretVars = append(r.Params.SecretVars, name)
			}
		}
		if r.Params.Limit == "" && len(r.Params.ExtraVars) == 0 { // only the mode was set
			r.Params = nil
		}
	}
	if (r.Mode == "check" && j.CheckApproval == "required") || (r.Mode != "check" && j.Approval == "required") {
		r.Status = "pending"
		r.Approval = &RunApproval{Required: true, Expires: now.Add(j.ApprovalTimeout)}
		s.pending[j.Name] = r
		s.running[j.Name] = true
	} else if at := j.deferUntil(r, now); at != nil {
		s.schedule(r, *at)
	} else {
		select {
		case s.queue <- queued{job: j, run: r}:
			s.running[j.Name] = true
		default:
			delete(s.secrets, id)
			return nil, errQueueFull
		}
	}
	err := s.store(r)
	if r.Status == "pending" {
		s.notify(eventPending, j, resultCopy(r))
	} else if r.Status == "scheduled" {
		s.notify(eventScheduled, j, resultCopy(r))
	}
	return resultCopy(r), err
}

// cancel stops a scheduled, queued or running run.
func (s *server) cancel(id, by, comment string) (*JobResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for job, list := range s.results {
		for _, r := range list {
			if r.ID != id {
				continue
			}
			if r.Status != "scheduled" && r.Status != "queued" && r.Status != "running" {
				return nil, errRunNotCancellable
			}
			// the comment is printed to the log and sent in messages: one line
			comment = strings.Map(func(c rune) rune {
				if unicode.IsControl(c) {
					return ' '
				}
				return c
			}, comment)
			r.Cancel = &RunCancel{By: by, At: time.Now().UTC(), Comment: comment}
			r.Summary = statusCancelled + " by " + by
			if comment != "" {
				r.Summary += ": " + comment
			}
			var err error
			if r.Status == "queued" || r.Status == "scheduled" {
				if r.Status == "queued" {
					s.running[job] = false
				}
				r.Status = statusCancelled
				delete(s.secrets, r.ID)
				err = s.store(r)
				s.notify(eventCancelled, s.job(job), resultCopy(r))
				fmt.Fprintf(s.out, "serve: %s: %s (queued)\n", job, r.Summary)
			} else {
				stop := s.cancels[id]
				if stop == nil {
					return nil, errRunNotCancellable
				}
				r.Status = statusCancelling
				stop()
				err = s.store(r)
			}
			return resultCopy(r), err
		}
	}
	return nil, errNoSuchRun
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
			if err := s.expire(job, r, now); err != nil {
				return resultCopy(r), err
			}
			return nil, errRunExpired
		}
		if decision == "approved" {
			if at := s.job(job).deferUntil(r, now); at != nil {
				s.schedule(r, *at)
				s.running[job] = false
			} else {
				select {
				case s.queue <- queued{job: s.job(job), run: r}:
				default:
					return nil, errQueueFull
				}
				r.Status = "queued"
			}
		} else {
			r.Status = "rejected"
			delete(s.secrets, r.ID)
			s.running[job] = false
		}
		r.Approval.Decision, r.Approval.By, r.Approval.At, r.Approval.Comment = decision, by, &now, comment
		delete(s.pending, job)
		err := s.store(r)
		if decision == "rejected" {
			s.notify(eventRejected, s.job(job), resultCopy(r))
		} else if r.Status == "scheduled" {
			s.notify(eventScheduled, s.job(job), resultCopy(r))
		}
		return resultCopy(r), err
	}
	return nil, errNoPendingRun
}

// expire ends one pending run; the caller holds mu
func (s *server) expire(job string, r *JobResult, now time.Time) error {
	r.Approval.Decision, r.Approval.At = "expired", &now
	r.Status = "expired"
	delete(s.secrets, r.ID)
	delete(s.pending, job)
	s.running[job] = false
	err := s.store(r)
	s.notify(eventExpired, s.job(job), resultCopy(r))
	return err
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
		if err := s.expire(job, r, now); err != nil {
			fmt.Fprintf(s.out, "serve: %s: result not saved: %v\n", job, err)
		}
		expired = append(expired, resultCopy(r))
	}
	return expired
}

// worker runs queued jobs; there are cfg.Concurrency of them, picking up
// in FIFO order
func (s *server) worker(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		for _, list := range s.results {
			for _, r := range list {
				if r.Status != "running" && r.Status != statusCancelling {
					delete(s.secrets, r.ID)
				}
			}
		}
		s.mu.Unlock()
	}()
next:
	for {
		s.pickupMu.Lock()
		select {
		case <-ctx.Done():
			s.pickupMu.Unlock()
			return
		case q := <-s.queue:
			waiting := false
			for {
				s.mu.Lock()
				skip := q.run.Status == statusCancelled
				s.mu.Unlock()
				if skip {
					s.pickupMu.Unlock()
					continue next
				}
				p, ch := s.paused()
				if !p.Paused {
					s.mu.Lock()
					if ctx.Err() != nil {
						s.mu.Unlock()
						s.pickupMu.Unlock()
						return
					}
					if !s.pause.Paused {
						// Hold mu until the run is marked running.
						break
					}
					s.mu.Unlock()
					continue
				}
				if !waiting {
					s.mu.Lock()
					fmt.Fprintf(s.out, "serve: %s: waiting, paused\n", q.job.Name)
					s.mu.Unlock()
					waiting = true
				}
				select {
				case <-ctx.Done():
					s.pickupMu.Unlock()
					return
				case <-ch:
				}
			}
			if q.run.Status == statusCancelled {
				s.mu.Unlock()
				s.pickupMu.Unlock()
				continue next
			}
			r := q.run
			if at := q.job.deferUntil(r, time.Now().UTC()); at != nil {
				s.schedule(r, *at)
				s.running[q.job.Name] = false
				s.saveResult(r)
				s.notify(eventScheduled, q.job, resultCopy(r))
				s.mu.Unlock()
				s.pickupMu.Unlock()
				continue next
			}
			timeout := q.job.Timeout
			if timeout == 0 {
				timeout = s.cfg.RunTimeout
			}
			var runCtx context.Context
			var cancel context.CancelFunc
			if timeout > 0 {
				runCtx, cancel = context.WithTimeout(ctx, timeout)
			} else {
				runCtx, cancel = context.WithCancel(ctx)
			}
			progress := newRunProgress(s.runSecretMask(r.ID))
			s.progress[r.ID] = progress
			runCtx = withRunObserver(runCtx, progress)
			s.cancels[r.ID] = cancel
			r.Status, r.Started = "running", time.Now().UTC()
			if err := s.store(r); err != nil {
				fmt.Fprintf(s.out, "serve: %s: result not saved: %v\n", r.Job, err)
			}
			s.notify(eventStart, q.job, resultCopy(r))
			fmt.Fprintf(s.out, "serve: %s: running (%s)\n", q.job.Name, r.Trigger)
			started := r.Started
			s.mu.Unlock()
			s.pickupMu.Unlock()
			j, err := s.effectiveJob(q.job, r.Params, r.ID)
			var result *JobResult
			if err == nil {
				result, err = s.runJob(runCtx, j, r.Mode)
			}
			s.mu.Lock()
			delete(s.cancels, r.ID)
			if result != nil {
				r.PlaybookHash = result.PlaybookHash
			}
			// a run that finished keeps its result, even when a cancel,
			// deadline or shutdown arrived between its end and here
			switch {
			case err == nil:
				r.Started, r.Duration, r.Status, r.Summary, r.Hosts, r.Report = result.Started, result.Duration, result.Status, result.Summary, result.Hosts, result.Report
				r.AuditExecutionID = result.AuditExecutionID
			case ctx.Err() != nil, runCtx.Err() != nil:
				// stopped: what the partial result knows about the hosts stays
				r.Status = statusCancelled
				deadline, _ := runCtx.Deadline()
				parentDeadline, parentHasDeadline := ctx.Deadline()
				if errors.Is(runCtx.Err(), context.DeadlineExceeded) && timeout > 0 &&
					(!parentHasDeadline || deadline.Before(parentDeadline)) &&
					(r.Cancel == nil || !r.Cancel.At.Before(deadline)) {
					r.Status, r.Summary = statusTimedOut, "timed out after "+timeout.String()
				} else if ctx.Err() != nil && r.Cancel == nil {
					r.Status, r.Summary = "error", "interrupted: serve stopping"
				}
				r.Duration = time.Since(started).Seconds()
				if result != nil {
					r.Hosts, r.Report = result.Hosts, result.Report
				}
			default:
				r.Status, r.Summary = "error", err.Error()
			}
			cancel()
			s.retainProgress(r.ID, r.Status)
			s.maskResult(r)
			delete(s.secrets, r.ID)
			if err := s.store(r); err != nil {
				fmt.Fprintf(s.out, "serve: %s: result not saved: %v\n", r.Job, err)
			}
			switch r.Status {
			case statusCancelled:
				s.notify(eventCancelled, q.job, resultCopy(r))
			case "ok":
				s.notify(eventSuccess, q.job, resultCopy(r))
			case "drift":
				s.notify(eventDrift, q.job, resultCopy(r))
			case "failed", "error", statusTimedOut:
				s.notify(eventFailure, q.job, resultCopy(r))
			}
			s.running[q.job.Name] = false
			fmt.Fprintf(s.out, "serve: %s: %s — %s (%.1fs)\n", q.job.Name, r.Status, r.Summary, r.Duration)
			s.mu.Unlock()
		}
	}
}

// runSecretMask uses the same secret values for reports and progress; mu is held.
func (s *server) runSecretMask(id string) *strings.Replacer {
	values := slices.Collect(maps.Values(s.secrets[id]))
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	var pairs []string
	for _, value := range values {
		if value != "" {
			pairs = append(pairs, value, "***")
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	return strings.NewReplacer(pairs...)
}

// maskResult also hides parameters echoed in execution errors and reports; mu is held.
func (s *server) maskResult(r *JobResult) {
	mask := s.runSecretMask(r.ID)
	if mask == nil {
		return
	}
	r.Summary = mask.Replace(r.Summary)
	if len(r.Report) > 0 {
		var report interface{}
		decoder := json.NewDecoder(strings.NewReader(string(r.Report)))
		decoder.UseNumber()
		if err := decoder.Decode(&report); err != nil {
			r.Report = json.RawMessage(`"***"`)
		} else {
			r.Report, _ = json.Marshal(maskRunValue(report, mask))
		}
	}
}

func maskRunValue(value interface{}, mask *strings.Replacer) interface{} {
	switch v := value.(type) {
	case string:
		return mask.Replace(v)
	case json.Number:
		// a secret made of digits comes back as a number
		if masked := mask.Replace(string(v)); masked != string(v) {
			return masked
		}
	case []interface{}:
		for i := range v {
			v[i] = maskRunValue(v[i], mask)
		}
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, item := range v {
			out[mask.Replace(key)] = maskRunValue(item, mask)
		}
		return out
	}
	return value
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
	tick := s.schedulerTick
	if tick <= 0 {
		tick = 5 * time.Second
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.expirePending(now)
			s.releaseScheduled(now)
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
				_, _ = s.request(j, "schedule", nil)
			}
		}
	}
}

// effectiveJob is the job as a run's parameters shape it: the hosts validated
// at request time become the limit (by name, and checked again now: an
// inventory that changed while the run waited must not widen it), the
// caller's extra vars go on top of the job's
func (s *server) effectiveJob(j *ServeJob, p *RunParams, runID string) (*ServeJob, error) {
	if p == nil {
		return j, nil
	}
	jj := *j
	if len(p.Hosts) > 0 {
		jj.Limit = strings.Join(p.Hosts, ",")
		names, err := s.resolveLimit(j, jj.Limit)
		if err == nil && !slices.Equal(names, p.Hosts) { // a group now named like a host, a host gone
			err = fmt.Errorf("%q resolves to %s now", jj.Limit, strings.Join(names, ", "))
		}
		if err != nil {
			return nil, fmt.Errorf("the hosts validated for this run no longer fit: %w", err)
		}
	}
	jj.ExtraVars = make(map[string]string, len(j.ExtraVars)+len(p.ExtraVars))
	maps.Copy(jj.ExtraVars, j.ExtraVars)
	maps.Copy(jj.ExtraVars, p.ExtraVars)
	s.mu.Lock()
	maps.Copy(jj.ExtraVars, s.secrets[runID])
	s.mu.Unlock()
	return &jj, nil
}

type serveInventoryError struct{ error }

func (s *server) jobInventory(j *ServeJob) (*inventory.Manager, error) {
	log, err := logger.NewEnhancedLogger("error", "text", os.Stderr)
	if err != nil {
		return nil, serveInventoryError{err}
	}
	paths := s.inventory
	if j.Inventory != "" {
		paths = []string{j.Inventory}
	}
	manager, err := loadInventoryFrom(context.Background(), log, inventorySources(log, j.Playbook, paths))
	if err != nil {
		return nil, serveInventoryError{err}
	}
	return manager, nil
}

// resolveLimit names the hosts of an ad-hoc limit and checks they stay
// inside the job's own limit
func (s *server) resolveLimit(j *ServeJob, limit string) ([]string, error) {
	manager, err := s.jobInventory(j)
	if err != nil {
		return nil, err
	}
	hosts, err := manager.GetHosts(limit)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("limit %q matches no host", limit)
	}
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, h.Name)
	}
	sort.Strings(names)
	if j.Limit != "" {
		allowed, err := manager.GetHosts(j.Limit)
		if err != nil {
			return nil, err
		}
		set := make(map[string]bool, len(allowed))
		for _, h := range allowed {
			set[h.Name] = true
		}
		var outside []string
		for _, name := range names {
			if !set[name] {
				outside = append(outside, name)
			}
		}
		if len(outside) > 0 {
			return nil, fmt.Errorf("limit %q reaches hosts outside the job's limit %q: %s", limit, j.Limit, strings.Join(outside, ", "))
		}
	}
	return names, nil
}

func validateServeLimit(limit string) error {
	if strings.ContainsFunc(limit, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._-*:,!&~[]@", r))
	}) {
		return errors.New("limit has characters a host pattern cannot have")
	}
	return nil
}

type serveInventoryGroup struct {
	Name  string   `json:"name"`
	Hosts []string `json:"hosts"`
}

func (s *server) jobInventoryView(w http.ResponseWriter, r *http.Request, j *ServeJob) {
	if !j.Adhoc.Limit {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "this job does not accept a limit"})
		return
	}
	if r.URL.Query().Has("limit") {
		limit := r.URL.Query().Get("limit")
		if err := validateServeLimit(limit); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		// Keep line breaks out of inventory logs, also for the static analyser.
		limit = strings.ReplaceAll(strings.ReplaceAll(limit, "\n", ""), "\r", "")
		pattern := limit
		if pattern == "" {
			pattern = j.Limit
			if pattern == "" {
				pattern = "all"
			}
		}
		hosts, err := s.resolveLimit(j, pattern)
		if err != nil {
			code := http.StatusBadRequest
			var loadErr serveInventoryError
			if errors.As(err, &loadErr) {
				code = http.StatusInternalServerError
			}
			writeJSON(w, code, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Limit string   `json:"limit"`
			Hosts []string `json:"hosts"`
		}{limit, hosts})
		return
	}
	manager, err := s.jobInventory(j)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	pattern := j.Limit
	if pattern == "" {
		pattern = "all"
	}
	hosts, err := manager.GetHosts(pattern)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	names := make([]string, 0, len(hosts))
	reachable := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		names = append(names, h.Name)
		reachable[h.Name] = true
	}
	sort.Strings(names)
	groups := make([]serveInventoryGroup, 0)
	for _, name := range manager.ListGroups() {
		if name == "all" {
			continue
		}
		members, err := manager.GetAllHostsInGroup(name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// a group reaching past the job's limit would be refused as a limit
		if len(members) > 0 && !slices.ContainsFunc(members, func(host string) bool { return !reachable[host] }) {
			sort.Strings(members)
			groups = append(groups, serveInventoryGroup{name, members})
		}
	}
	writeJSON(w, http.StatusOK, struct {
		Hosts  []string              `json:"hosts"`
		Groups []serveInventoryGroup `json:"groups"`
	}{names, groups})
}

// runParams reads the body of a run request: nothing, or the limit and
// extra vars the job allows
func (s *server) runParams(r *http.Request, j *ServeJob) (*RunParams, error) {
	var body struct {
		NotBefore *time.Time        `json:"not_before"`
		Mode      string            `json:"mode"`
		Limit     string            `json:"limit"`
		ExtraVars map[string]string `json:"extra_vars"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		if !errors.Is(err, io.EOF) {
			if strings.HasPrefix(err.Error(), "json: unknown field ") {
				return nil, errors.New("invalid run parameters JSON: unknown field")
			}
			return nil, errors.New("invalid run parameters JSON")
		}
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("expected one JSON body")
	}
	if body.NotBefore != nil {
		now := time.Now()
		if !body.NotBefore.After(now) || body.NotBefore.After(now.Add(30*24*time.Hour)) {
			return nil, errors.New("not_before must be in the future and at most 30 days ahead")
		}
	}
	switch body.Mode {
	case "", "apply":
		body.Mode = ""
	case "check":
		if j.Kind != "apply" {
			return nil, errors.New("only apply jobs have a check mode")
		}
	default:
		return nil, errors.New("mode must be check, apply or empty")
	}
	if body.Limit != "" && !j.Adhoc.Limit {
		return nil, errors.New("this job does not accept a limit")
	}
	if err := validateServeLimit(body.Limit); err != nil {
		return nil, err
	}
	// the pattern is logged by the inventory: no line breaks can reach a log
	// (the check above already refuses them; this is for the static analyser)
	body.Limit = strings.ReplaceAll(strings.ReplaceAll(body.Limit, "\n", ""), "\r", "")
	vars, err := checkExtraVars(j, body.ExtraVars)
	if err != nil {
		return nil, err
	}
	body.ExtraVars = vars
	if body.NotBefore == nil && body.Mode == "" && body.Limit == "" && len(body.ExtraVars) == 0 {
		return nil, nil
	}
	params := &RunParams{NotBefore: body.NotBefore, Mode: body.Mode, Limit: body.Limit, ExtraVars: body.ExtraVars}
	if body.Limit != "" {
		hosts, err := s.resolveLimit(j, body.Limit)
		if err != nil {
			return nil, err
		}
		params.Hosts = hosts
	}
	return params, nil
}

func (s *server) rerunFailed(w http.ResponseWriter, r *http.Request, id serveIdentity) {
	parentID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/runs/"), "/rerun-failed")
	a := s.requestAction(r, id)
	a.Action, a.Run = "rerun_failed", parentID
	defer func() { s.logAction(a) }()
	fail := func(code int, reason string) {
		a.Reason = "parent=" + parentID + " " + reason
		writeJSON(w, code, map[string]string{"error": reason})
	}
	var parent *JobResult
	s.mu.Lock()
	for job := range s.results {
		if found := s.resultByID(job, parentID); found != nil {
			parent = resultCopy(found)
			break
		}
	}
	s.mu.Unlock()
	if parent == nil {
		fail(http.StatusNotFound, "no such run")
		return
	}
	a.Job = parent.Job
	j := s.job(parent.Job)
	if j == nil || !s.allowed(id, j) {
		fail(http.StatusForbidden, "not an operator of this job")
		return
	}
	if !parent.executed() {
		fail(http.StatusBadRequest, "the run has not executed")
		return
	}
	if s.hasSecretParams(parent) {
		fail(http.StatusBadRequest, "the run had secret parameters; start it again from the form")
		return
	}
	var hosts []string
	for host, status := range parent.Hosts {
		if status == "failed" || status == "unreachable" {
			hosts = append(hosts, host)
		}
	}
	if len(hosts) == 0 {
		fail(http.StatusBadRequest, "the run has no failed hosts")
		return
	}
	sort.Strings(hosts)
	limit := strings.Join(hosts, ",")
	if err := validateServeLimit(limit); err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	resolved, err := s.resolveLimit(j, limit)
	if err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	if !slices.Equal(resolved, hosts) {
		fail(http.StatusBadRequest, "failed hosts no longer resolve to the same hosts")
		return
	}
	params := &RunParams{Mode: parent.Mode, Limit: limit, Hosts: hosts}
	if parent.Params != nil {
		params.ExtraVars = maps.Clone(parent.Params.ExtraVars)
	}
	// the job's declarations as they are now, not as they were for the parent
	if params.ExtraVars, err = checkExtraVars(j, params.ExtraVars); err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	// A variable newly marked secret must also be entered through the form.
	for _, name := range j.Adhoc.SecretVars {
		if _, ok := params.ExtraVars[name]; ok {
			fail(http.StatusBadRequest, "the run had secret parameters; start it again from the form")
			return
		}
	}
	result, err := s.requestParent(j, "rerun:"+id.User, params, parent.ID)
	if err != nil {
		if result != nil {
			a.Reason = "parent=" + parent.ID + " " + err.Error()
			writeResultError(w, result, err)
			return
		}
		code := http.StatusServiceUnavailable
		if errors.Is(err, errJobBusy) || errors.Is(err, errAlreadyScheduled) {
			code = http.StatusConflict
		}
		fail(code, err.Error())
		return
	}
	a.OK, a.Run, a.Reason = true, result.ID, "parent="+parent.ID
	writeJSON(w, http.StatusAccepted, map[string]string{"run": result.ID})
}

// checkExtraVars holds caller variables to the job's declarations: only
// declared names, each value valid, required ones present, defaults filled in
func checkExtraVars(j *ServeJob, vars map[string]string) (map[string]string, error) {
	for _, k := range slices.Sorted(maps.Keys(vars)) {
		if !slices.Contains(j.Adhoc.ExtraVars, k) {
			return nil, fmt.Errorf("extra var %q is not allowed for this job", k)
		}
	}
	for _, v := range j.Adhoc.Vars {
		value, provided := vars[v.Name]
		if !provided && v.Default != nil {
			value, provided = *v.Default, true
			if vars == nil {
				vars = make(map[string]string, len(j.Adhoc.Vars))
			}
			vars[v.Name] = value
		}
		if !provided {
			if v.Required {
				return nil, fmt.Errorf("extra var %q is required", v.Name)
			}
			continue
		}
		if err := v.validate(value); err != nil {
			return nil, err
		}
	}
	return vars, nil
}

// extraVarsArg is the -e value for a job's variables: one JSON object, so a
// value with spaces or = stays one variable (a k=v list is split on spaces)
func extraVarsArg(vars map[string]string) string {
	data, _ := json.Marshal(vars)
	return string(data)
}

// execute runs a job in this process and summarizes it
func (s *server) execute(ctx context.Context, j *ServeJob, mode string) (*JobResult, error) {
	started := time.Now().UTC()
	r := &JobResult{Job: j.Name, Kind: j.Kind, Mode: mode, Started: started, Hosts: map[string]string{}}
	if j.Kind == "drift" || j.Kind == "apply" || j.Kind == "verify" {
		if content, err := os.ReadFile(j.Playbook); err == nil {
			sum := sha256.Sum256(content)
			r.PlaybookHash = fmt.Sprintf("%x", sum)[:16]
		}
	}
	var runErr error
	o := driftCheckOptions{limit: j.Limit, become: j.Become}
	if j.Inventory != "" {
		o.inventory = []string{j.Inventory}
	} else {
		o.inventory = s.inventory
	}
	o.state = filepath.Join(s.cfg.DataDir, "state", j.Name+".state")
	if err := os.MkdirAll(filepath.Dir(o.state), 0o700); err != nil {
		return nil, err
	}
	if len(j.ExtraVars) > 0 {
		o.extraVars = []string{extraVarsArg(j.ExtraVars)}
	}
	switch j.Kind {
	case "comply":
		co := &complyOptions{driftCheckOptions: o, profile: j.Profile, failOn: j.FailOn}
		co.become = j.Become || !strings.Contains(j.Profile, "/")
		report, executionID, err := complyReport(ctx, co)
		if report == nil {
			return nil, cmp.Or(err, errors.New("the playbook returned no result"))
		}
		runErr = err
		r.AuditExecutionID = executionID
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
		if runErr == nil {
			s.pushMetrics(report.Metrics(s.cfg.Metrics.Labels))
		}
	case "verify":
		result, err := runPlaybookContext(ctx, append(o.applyArgs(j.Playbook, false), "--verify-only"))
		if result == nil {
			return nil, cmp.Or(err, errors.New("the playbook returned no result"))
		}
		runErr = err // stopped: the partial result goes back with the error
		r.AuditExecutionID = result.ExecutionID
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
		check := j.Kind == "drift" || mode == "check"
		args := o.applyArgs(j.Playbook, check)
		if !check {
			args = append(args, "--diff")
		}
		if mode == "check" { // the verify: checks would test the state before the changes
			args = append(args, "--skip-verify")
		}
		result, err := runPlaybookContext(ctx, args)
		if result == nil {
			return nil, cmp.Or(err, errors.New("the playbook returned no result"))
		}
		runErr = err
		r.AuditExecutionID = result.ExecutionID
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
		if j.Kind == "drift" && runErr == nil {
			s.pushMetrics(driftMetrics(report, s.cfg.Metrics.Labels))
		}
	}
	r.Duration = time.Since(started).Seconds()
	return r, runErr
}

func (s *server) pushMetrics(text string) {
	if s.cfg.Metrics.Push != "" {
		emitMetrics(text, "", s.cfg.Metrics.Push)
	}
}

// ---- HTTP

type serveIdentity struct {
	User     string
	Groups   []string
	Operator bool
}

var errServeAuth = errors.New("unauthorized")

// identify resolves who is calling: the bearer token (operator), the
// proxy's headers (viewer, operator by group), or nobody when no auth is
// configured (everything allowed)
func (s *server) identify(r *http.Request) (serveIdentity, error) {
	a := s.cfg.Auth
	if a.Token == "" && len(a.Tokens) == 0 && a.UserHeader == "" {
		return serveIdentity{User: "anonymous", Operator: true}, nil
	}
	if a.Token != "" || len(a.Tokens) > 0 {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(a.Token)) == 1 {
			return serveIdentity{User: "token", Operator: true}, nil
		}
		for _, t := range a.Tokens {
			if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(t.Token)) == 1 {
				return serveIdentity{User: t.Name, Groups: t.Groups, Operator: true}, nil
			}
		}
	}
	if a.UserHeader != "" {
		if user := r.Header.Get(a.UserHeader); user != "" {
			id := serveIdentity{User: user}
			if a.GroupsHeader != "" {
				id.Groups = strings.FieldsFunc(r.Header.Get(a.GroupsHeader), func(c rune) bool { return c == ',' || c == '|' || c == ' ' })
				for _, g := range id.Groups {
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

// matches: the user name or one of the groups is in names
func (id serveIdentity) matches(names []string) bool {
	for _, name := range names {
		if strings.EqualFold(id.User, name) {
			return true
		}
		for _, group := range id.Groups {
			if strings.EqualFold(group, name) {
				return true
			}
		}
	}
	return false
}

// allowed: an operator, and one the job's operators list admits
func (s *server) allowed(id serveIdentity, j *ServeJob) bool {
	return id.Operator && (len(j.Operators) == 0 || id.matches(j.Operators))
}

// writePause answers a pause/resume: the state, with 500 and the error
// when it could not be written to disk
func writePause(w http.ResponseWriter, p servePause, err error) {
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error(), "paused": p.Paused, "reason": p.Reason, "by": p.By, "at": p.At})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func writeResultError(w http.ResponseWriter, result *JobResult, err error) {
	writeJSON(w, http.StatusInternalServerError, struct {
		*JobResult
		Error string `json:"error"`
	}{result, err.Error()})
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
				if r.Method == http.MethodPost {
					a := s.requestAction(r, id)
					a.Reason = "operators only"
					s.logAction(a)
				}
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
		s.mu.Lock()
		running, queued := s.activity()
		status := map[string]interface{}{"user": id.User, "operator": id.Operator, "title": s.cfg.Title, "paused": s.pause.Paused, "running": running, "queued": queued, "scheduled": s.scheduledCount(), "concurrency": s.cfg.Concurrency, "storage_error": s.storeErr}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, status)
	}))
	mux.HandleFunc("/api/pause", auth(false, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		switch r.Method {
		case http.MethodGet:
			p, _ := s.paused()
			writeJSON(w, http.StatusOK, p)
		case http.MethodPost:
			auth(true, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
				var body struct {
					Reason string `json:"reason"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
					a := s.requestAction(r, id)
					a.Reason = "invalid JSON"
					s.logAction(a)
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
					return
				}
				p, err := s.setPause(true, id.User, body.Reason)
				a := s.requestAction(r, id)
				a.OK, a.Reason = err == nil, body.Reason
				if err != nil { // the pause holds in memory only: the log says so
					a.Reason = err.Error()
				}
				s.logAction(a)
				writePause(w, p, err)
			})(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/resume", auth(true, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		p, err := s.setPause(false, id.User, "")
		a := s.requestAction(r, id)
		a.OK = err == nil
		if err != nil {
			a.Reason = err.Error()
		}
		s.logAction(a)
		writePause(w, p, err)
	}))
	mux.HandleFunc("/api/jobs", auth(false, func(w http.ResponseWriter, _ *http.Request, id serveIdentity) {
		writeJSON(w, http.StatusOK, s.jobsView(id))
	}))
	mux.HandleFunc("/api/jobs/", auth(false, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
		name, action, _ := strings.Cut(rest, "/")
		j := s.job(name)
		if j == nil {
			if action == "run" && r.Method == http.MethodPost {
				a := s.requestAction(r, id)
				a.Reason = "no such job"
				s.logAction(a)
			}
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such job"})
			return
		}
		switch {
		case action == "inventory" && r.Method == http.MethodGet:
			if !s.allowed(id, j) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "not an operator of this job"})
				return
			}
			s.jobInventoryView(w, r, j)
		case action == "run" && r.Method == http.MethodPost:
			auth(true, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
				a := s.requestAction(r, id)
				if !s.allowed(id, j) {
					a.Reason = "not an operator of this job"
					s.logAction(a)
					writeJSON(w, http.StatusForbidden, map[string]string{"error": a.Reason})
					return
				}
				params, err := s.runParams(r, j)
				if err != nil {
					a.Reason = err.Error()
					s.logAction(a)
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": a.Reason})
					return
				}
				run, err := s.request(j, "api:"+id.User, params)
				if err != nil {
					code := http.StatusServiceUnavailable
					a.Reason = err.Error()
					s.logAction(a)
					if errors.Is(err, errJobBusy) || errors.Is(err, errAlreadyScheduled) {
						code = http.StatusConflict
					}
					if run != nil {
						writeResultError(w, run, err)
					} else {
						writeJSON(w, code, map[string]string{"error": err.Error()})
					}
					return
				}
				a.OK, a.Run = true, run.ID
				if params != nil {
					var details []string
					if run.Params != nil {
						details = run.Params.values()
					}
					if params.Limit != "" {
						details = append([]string{"limit=" + params.Limit}, details...)
					}
					if run.Mode == "check" {
						details = append([]string{"mode=check"}, details...)
					}
					a.Reason = strings.Join(details, " ")
				}
				if run.NotBefore != nil {
					a.Reason = strings.TrimSpace(a.Reason + " not_before=" + run.NotBefore.Format(time.RFC3339Nano))
				}
				s.logAction(a)
				if run.Status == "scheduled" {
					writeJSON(w, http.StatusAccepted, map[string]string{"scheduled": run.ID, "run": run.ID})
				} else if run.Status == "pending" {
					writeJSON(w, http.StatusAccepted, map[string]string{"pending": run.ID, "run": run.ID})
				} else {
					writeJSON(w, http.StatusAccepted, map[string]string{"queued": j.Name, "run": run.ID})
				}
			})(w, r)
		case action == "history" && r.Method == http.MethodGet:
			limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
			if err != nil || limit <= 0 {
				limit = 50
			}
			if limit > 500 {
				limit = 500
			}
			writeJSON(w, http.StatusOK, s.jobHistory(j.Name, limit))
		case action == "compare" && r.Method == http.MethodGet:
			s.compareRuns(w, r, j.Name)
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
		if id, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/api/results/"), "/events"); ok {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.progressHandler(w, r, id)
			return
		}
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
	mux.HandleFunc("GET /api/inventory", auth(false, func(w http.ResponseWriter, _ *http.Request, id serveIdentity) {
		writeJSON(w, http.StatusOK, s.inventoryView(id.Operator))
	}))
	mux.HandleFunc("POST /api/inventory/refresh", auth(true, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		s.inventoryMu.Lock()
		s.inventorySnapshots = nil
		s.inventoryMu.Unlock()
		a := s.requestAction(r, id)
		a.OK = true
		s.logAction(a)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("/api/hosts", auth(false, func(w http.ResponseWriter, _ *http.Request, _ serveIdentity) {
		writeJSON(w, http.StatusOK, s.hostsView())
	}))
	mux.HandleFunc("GET /api/hosts/{name}/results", auth(false, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
		writeJSON(w, http.StatusOK, s.hostResults(r.PathValue("name")))
	}))
	mux.HandleFunc("/api/runs", auth(false, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
		limit, from, to, err := servePageOptions(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		q := r.URL.Query()
		offset := 0
		if q.Has("offset") {
			offset, err = strconv.Atoi(q.Get("offset"))
		}
		if err != nil || offset < 0 || offset > int(^uint(0)>>1)-1001 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offset"})
			return
		}
		if s.audit == nil {
			writeJSON(w, http.StatusOK, []interface{}{})
			return
		}
		filter := audit.FilterOptions{Limit: limit + 1, Offset: offset, HostFilter: q.Get("host"), PlaybookPath: q.Get("playbook"), Status: audit.ExecutionStatus(q.Get("status")), DateFrom: from, DateTo: to, SortBy: "time", SortOrder: "desc"}
		if job := q.Get("job"); job != "" { // the store pages over this job's executions only
			filter.IDs = make(map[string]bool)
			s.mu.Lock()
			for _, result := range s.results[job] {
				if result.AuditExecutionID != "" {
					filter.IDs[result.AuditExecutionID] = true
				}
			}
			s.mu.Unlock()
		}
		records, err := s.audit.ListRecords(filter)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		serveRuns := s.auditServeRuns()
		if len(records) > limit {
			w.Header().Set("X-Next-Offset", strconv.Itoa(offset+limit))
			records = records[:limit]
		}
		type row struct {
			ServeRun string    `json:"serve_run,omitempty"`
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
			rows = append(rows, row{ServeRun: serveRuns[rec.ID], ID: rec.ID, Playbook: rec.PlaybookPath, User: rec.User, Start: rec.StartTime, Duration: rec.Duration, Status: string(rec.Status), Tasks: rec.TotalTasks, Failed: rec.FailedTasks, Hosts: len(hosts)})
		}
		writeJSON(w, http.StatusOK, rows)
	}))
	mux.HandleFunc("/api/runs/", auth(false, func(w http.ResponseWriter, r *http.Request, id serveIdentity) {
		path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
		runID, action, _ := strings.Cut(path, "/")
		if action == "rerun-failed" {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			auth(true, s.rerunFailed)(w, r)
			return
		}
		if r.Method == http.MethodPost && (action == "approve" || action == "reject" || action == "cancel") {
			auth(true, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
				a := s.requestAction(r, id)
				if a.Job == "" {
					a.Reason = errNoPendingRun.Error()
					if action == "cancel" {
						a.Reason = errNoSuchRun.Error()
					}
					s.logAction(a)
					writeJSON(w, http.StatusNotFound, map[string]string{"error": a.Reason})
					return
				}
				job := s.job(a.Job)
				if job == nil || !s.allowed(id, job) {
					a.Reason = "not an operator of this job"
					s.logAction(a)
					writeJSON(w, http.StatusForbidden, map[string]string{"error": a.Reason})
					return
				}
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
				if action == "cancel" {
					decide = s.cancel
				}
				result, err := decide(runID, id.User, body.Comment)
				a.OK, a.Reason = err == nil, body.Comment
				if result != nil && result.NotBefore != nil {
					a.Reason = strings.TrimSpace(a.Reason + " not_before=" + result.NotBefore.Format(time.RFC3339Nano))
				}
				if err != nil {
					a.Reason = err.Error()
				}
				s.logAction(a)
				if err != nil {
					code := http.StatusServiceUnavailable
					switch {
					case errors.Is(err, errNoPendingRun), errors.Is(err, errNoSuchRun):
						code = http.StatusNotFound
					case errors.Is(err, errRunExpired), errors.Is(err, errRunNotCancellable):
						code = http.StatusConflict
					}
					if result != nil {
						writeResultError(w, result, err)
					} else {
						writeJSON(w, code, map[string]string{"error": err.Error()})
					}
					return
				}
				result.Report = nil
				code := http.StatusOK
				if result.Status == statusCancelling {
					code = http.StatusAccepted
				}
				writeJSON(w, code, result)
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
		data, err := json.Marshal(rec)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		var detail map[string]json.RawMessage
		if err := json.Unmarshal(data, &detail); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if serveRun := s.auditServeRuns()[rec.ID]; serveRun != "" {
			detail["serve_run"], _ = json.Marshal(serveRun)
		}
		writeJSON(w, http.StatusOK, detail)
	}))
	mux.HandleFunc("GET /api/audit", auth(false, func(w http.ResponseWriter, r *http.Request, _ serveIdentity) {
		limit, from, to, err := servePageOptions(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		q := r.URL.Query()
		filter := serveActionFilter{Limit: limit, From: from, To: to, User: q.Get("user"), Action: q.Get("action"), Job: q.Get("job"), Run: q.Get("run")}
		if q.Has("ok") {
			if q.Get("ok") != "true" && q.Get("ok") != "false" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ok: expected true or false"})
				return
			}
			ok := q.Get("ok") == "true"
			filter.OK = &ok
		}
		if q.Has("before") {
			before, err := s.actionCursor(q.Get("before"))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			filter.Before = &before
		}
		entries, next := s.actions(filter)
		if next != "" {
			w.Header().Set("X-Next-Cursor", next)
		}
		writeJSON(w, http.StatusOK, entries)
	}))
	return mux
}

// auditServeRuns links the execution history to retained serve results.
func (s *server) auditServeRuns() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	runs := make(map[string]string)
	for _, results := range s.results {
		for _, result := range results {
			if result.AuditExecutionID != "" {
				runs[result.AuditExecutionID] = result.ID
			}
		}
	}
	return runs
}

type jobRecent struct {
	Status  string    `json:"status"`
	Mode    string    `json:"mode"`
	Started time.Time `json:"started"`
}

type jobHistoryRow struct {
	ID          string       `json:"id"`
	Status      string       `json:"status"`
	Mode        string       `json:"mode"`
	Trigger     string       `json:"trigger"`
	Requested   time.Time    `json:"requested"`
	Started     time.Time    `json:"started"`
	Duration    float64      `json:"duration_seconds"`
	Summary     string       `json:"summary"`
	HostsCount  int          `json:"hosts_count"`
	FailedHosts int          `json:"failed_hosts"`
	Params      *RunParams   `json:"params,omitempty"`   // as stored: secret values are already masked
	Approval    *RunApproval `json:"approval,omitempty"` // who approved or rejected it
}

func (s *server) jobHistory(job string, limit int) []jobHistoryRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]jobHistoryRow, 0) // no capacity from the caller's limit
	for _, r := range s.results[job] {
		if len(rows) == limit {
			break
		}
		failed := 0
		for _, status := range r.Hosts {
			if status == "failed" || status == "unreachable" {
				failed++
			}
		}
		row := jobHistoryRow{ID: r.ID, Status: r.Status, Mode: r.Mode, Trigger: r.Trigger,
			Requested: r.Requested, Started: r.Started, Duration: r.Duration, Summary: r.Summary,
			HostsCount: len(r.Hosts), FailedHosts: failed}
		if r.Params != nil || r.Approval != nil { // copies: the rows are encoded after mu is released
			light := *r
			light.Report = nil
			c := resultCopy(&light)
			row.Params, row.Approval = c.Params, c.Approval
		}
		rows = append(rows, row)
	}
	return rows
}

type compareRunView struct {
	ID           string    `json:"id"`
	Started      time.Time `json:"started"`
	Status       string    `json:"status"`
	PlaybookHash string    `json:"playbook_hash"`
}

type compareHost struct {
	Host string `json:"host"`
	A    string `json:"a"`
	B    string `json:"b"`
}

type compareItem struct {
	Host   string `json:"host"`
	Key    string `json:"key"`
	Play   string `json:"play"`
	Task   string `json:"task"`
	Module string `json:"module"`
}

type runComparison struct {
	A            compareRunView `json:"a"`
	B            compareRunView `json:"b"`
	SamePlaybook bool           `json:"same_playbook"`
	Hosts        []compareHost  `json:"hosts"`
	Appeared     []compareItem  `json:"appeared"`
	Resolved     []compareItem  `json:"resolved"`
}

func (s *server) compareRuns(w http.ResponseWriter, req *http.Request, job string) {
	ids := []string{req.URL.Query().Get("a"), req.URL.Query().Get("b")}
	if ids[0] == "" || ids[1] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "both a and b run IDs are required"})
		return
	}
	if ids[0] == ids[1] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a and b must be different runs"})
		return
	}
	runs := make([]*JobResult, 2)
	s.mu.Lock()
	for _, list := range s.results {
		for _, r := range list {
			for i, id := range ids {
				if r.ID == id {
					runs[i] = resultCopy(r)
				}
			}
		}
	}
	s.mu.Unlock()
	for _, r := range runs {
		if r == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such result"})
			return
		}
	}
	reports := make([]*DriftReport, 2)
	for i, r := range runs {
		if r.Job != job || (r.Kind != "drift" && r.Kind != "apply") || !r.executed() ||
			len(r.Report) == 0 || json.Unmarshal(r.Report, &reports[i]) != nil || reports[i] == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "comparison requires executed drift or apply runs of this job with reports"})
			return
		}
	}
	a, b := runs[0], runs[1]
	view := func(r *JobResult) compareRunView {
		return compareRunView{ID: r.ID, Started: r.Started, Status: r.Status, PlaybookHash: r.PlaybookHash}
	}
	out := runComparison{A: view(a), B: view(b), SamePlaybook: a.PlaybookHash != "" && a.PlaybookHash == b.PlaybookHash,
		Hosts: []compareHost{}, Appeared: []compareItem{}, Resolved: []compareItem{}}
	hosts := maps.Clone(a.Hosts)
	if hosts == nil {
		hosts = make(map[string]string, len(b.Hosts))
	}
	for host, status := range b.Hosts {
		hosts[host] = status
	}
	for host := range hosts {
		av, aok := a.Hosts[host]
		bv, bok := b.Hosts[host]
		if av != bv || aok != bok {
			out.Hosts = append(out.Hosts, compareHost{Host: host, A: av, B: bv})
		}
	}
	sort.Slice(out.Hosts, func(i, j int) bool { return out.Hosts[i].Host < out.Hosts[j].Host })
	out.Appeared = driftDifference(reports[1], reports[0])
	out.Resolved = driftDifference(reports[0], reports[1])
	writeJSON(w, http.StatusOK, out)
}

// driftDifference returns drift items in one report that are absent from the other.
func driftDifference(from, other *DriftReport) []compareItem {
	type taskName struct{ host, play, task string }
	type taskKey struct{ host, key string }
	keys := make(map[taskKey]bool)
	names := make(map[taskName]bool)
	unkeyed := make(map[taskName]bool)
	for host, items := range other.Drift {
		for _, item := range items {
			name := taskName{host, item.Play, item.Task}
			names[name] = true
			if item.Key == "" {
				unkeyed[name] = true
			} else {
				keys[taskKey{host, item.Key}] = true
			}
		}
	}
	out := make([]compareItem, 0)
	for host, items := range from.Drift {
		for _, item := range items {
			name := taskName{host, item.Play, item.Task}
			matched := names[name]
			if item.Key != "" {
				matched = keys[taskKey{host, item.Key}] || unkeyed[name]
			}
			if !matched {
				out = append(out, compareItem{Host: host, Key: item.Key, Play: item.Play, Task: item.Task, Module: item.Module})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		return cmp.Or(strings.Compare(a.Host, b.Host), strings.Compare(a.Play, b.Play), strings.Compare(a.Task, b.Task), strings.Compare(a.Key, b.Key), strings.Compare(a.Module, b.Module)) < 0
	})
	return out
}

type jobView struct {
	Windows       []string     `json:"windows,omitempty"`
	WindowsTZ     string       `json:"windows_tz,omitempty"`
	NextWindow    *time.Time   `json:"next_window,omitempty"`
	Scheduled     *JobResult   `json:"scheduled,omitempty"`
	ScheduledRuns []*JobResult `json:"scheduled_runs,omitempty"`
	Name          string       `json:"name"`
	Kind          string       `json:"kind"`
	Target        string       `json:"target"`
	Approval      string       `json:"approval,omitempty"`  // required
	Operators     []string     `json:"operators,omitempty"` // the job's own list
	Allowed       bool         `json:"allowed"`             // the caller may run, approve, reject
	Adhoc         *ServeAdhoc  `json:"adhoc,omitempty"`     // what a run may set, when anything
	Every         string       `json:"every,omitempty"`
	Timeout       string       `json:"timeout,omitempty"`
	Next          *time.Time   `json:"next,omitempty"`
	Queued        bool         `json:"queued"`
	Running       bool         `json:"running"`           // queued, running or stopping
	Pending       *JobResult   `json:"pending,omitempty"` // the run that waits for approval
	Active        *JobResult   `json:"active,omitempty"`  // queued or executing, without report
	Recent        []jobRecent  `json:"recent"`
	Last          *JobResult   `json:"last,omitempty"` // the last executed run
}

func (s *server) jobsView(id serveIdentity) []jobView {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]jobView, 0, len(s.cfg.Jobs))
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		v := jobView{Operators: j.Operators, Allowed: s.allowed(id, j), Name: j.Name, Kind: j.Kind, Target: j.Playbook, Running: s.running[j.Name] && s.pending[j.Name] == nil, Approval: j.Approval}
		if j.Adhoc.Limit || len(j.Adhoc.ExtraVars) > 0 {
			v.Adhoc = &j.Adhoc
		}
		if j.gated("") {
			v.Windows, v.WindowsTZ = j.Windows, j.WindowsTZ
			if now := time.Now(); !j.windowOpen(now) { // open now: no next start to show
				next := j.nextWindow(now)
				v.NextWindow = &next
			}
		}
		for _, r := range s.results[j.Name] {
			if r.Status == "scheduled" {
				scheduled := resultCopy(r)
				scheduled.Report = nil
				if v.Scheduled == nil {
					v.Scheduled = scheduled
				}
				v.ScheduledRuns = append(v.ScheduledRuns, scheduled)
			}
			if v.Running && (r.Status == "queued" || r.Status == "running" || r.Status == statusCancelling) {
				v.Queued = r.Status == "queued"
				v.Active = resultCopy(r)
				v.Active.Report = nil
			}
		}

		if len(v.ScheduledRuns) < 2 {
			v.ScheduledRuns = nil
		}
		if j.Kind == "comply" {
			v.Target = j.Profile
		}
		if j.Timeout > 0 {
			v.Timeout = j.Timeout.String()
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
		v.Recent = make([]jobRecent, 0, 20)
		for _, result := range s.results[j.Name] {
			if result.executed() {
				v.Recent = append(v.Recent, jobRecent{Status: result.Status, Mode: result.Mode, Started: result.Started})
				if len(v.Recent) == 20 {
					break
				}
			}
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

// lastHostResult excludes stopped, timed-out and check runs from host verdicts; caller holds mu.
func (s *server) lastHostResult(job string) *JobResult {
	for _, r := range s.results[job] {
		if r.Status == statusCancelled || r.Status == statusTimedOut || r.Mode == "check" {
			continue
		}
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

type hostResultView struct {
	ID         string    `json:"id"`
	Job        string    `json:"job"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	HostStatus string    `json:"host_status"`
	Started    time.Time `json:"started"`
	Summary    string    `json:"summary"`
}

func (s *server) hostResults(host string) []hostResultView {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]hostResultView, 0)
	for _, results := range s.results {
		for _, r := range results {
			if status, ok := r.Hosts[host]; ok {
				rows = append(rows, hostResultView{ID: r.ID, Job: r.Job, Kind: r.Kind, Status: r.Status, HostStatus: status, Started: r.Started, Summary: r.Summary})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Started.Equal(rows[j].Started) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].Started.After(rows[j].Started)
	})
	if len(rows) > 200 {
		rows = rows[:200]
	}
	return rows
}

func (s *server) hostsView() []hostView {
	s.mu.Lock()
	defer s.mu.Unlock()
	rank := map[string]int{"ok": 0, "changed": 1, "drift": 2, "failed": 3, "unreachable": 4}
	hosts := map[string]*hostView{}
	for job := range s.results {
		last := s.lastHostResult(job)
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

// activity counts executing and queued runs; caller holds mu.
func (s *server) activity() (running, queued int) {
	for _, runs := range s.results {
		for _, r := range runs {
			switch r.Status {
			case "running", statusCancelling:
				running++
			case "queued":
				queued++
			}
		}
	}
	return
}

func (s *server) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var b strings.Builder
	b.WriteString("# TYPE onigirazu_serve_job_last_status gauge\n# TYPE onigirazu_serve_job_last_run_timestamp_seconds gauge\n# TYPE onigirazu_serve_host_status gauge\n")
	b.WriteString("# TYPE onigirazu_serve_job_pending gauge\n# TYPE onigirazu_serve_paused gauge\n")
	paused := float64(0)
	if s.pause.Paused {
		paused = 1
	}
	promLine(&b, "onigirazu_serve_paused", s.cfg.Metrics.Labels, paused)
	b.WriteString("# TYPE onigirazu_serve_storage_error gauge\n")
	storageError := float64(0)
	if s.storeErr != "" {
		storageError = 1
	}
	promLine(&b, "onigirazu_serve_storage_error", s.cfg.Metrics.Labels, storageError)
	b.WriteString("# TYPE onigirazu_serve_running gauge\n# TYPE onigirazu_serve_concurrency gauge\n")
	b.WriteString("# TYPE onigirazu_serve_scheduled gauge\n")
	promLine(&b, "onigirazu_serve_scheduled", s.cfg.Metrics.Labels, float64(s.scheduledCount()))
	running, _ := s.activity()
	promLine(&b, "onigirazu_serve_running", s.cfg.Metrics.Labels, float64(running))
	promLine(&b, "onigirazu_serve_concurrency", s.cfg.Metrics.Labels, float64(s.cfg.Concurrency))
	for _, j := range s.cfg.Jobs {
		if j.Approval == "required" || j.CheckApproval == "required" {
			value := float64(0)
			if s.pending[j.Name] != nil {
				value = 1
			}
			promLine(&b, "onigirazu_serve_job_pending", withLabels(map[string]string{"job": j.Name}, s.cfg.Metrics.Labels), value)
		}
	}
	codes := map[string]float64{"ok": 0, "drift": 1, "changed": 1, "failed": 2, "error": 3, "unreachable": 3, statusCancelled: 4, statusTimedOut: 5}
	names := make([]string, 0, len(s.results))
	for n := range s.results {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		var last *JobResult
		for _, r := range s.results[n] {
			if r.Mode != "check" && r.executed() {
				last = r
				break
			}
		}
		if last == nil {
			continue
		}
		promLine(&b, "onigirazu_serve_job_last_status", withLabels(map[string]string{"job": n, "kind": last.Kind}, s.cfg.Metrics.Labels), codes[last.Status])
		promLine(&b, "onigirazu_serve_job_last_run_timestamp_seconds", withLabels(map[string]string{"job": n}, s.cfg.Metrics.Labels), float64(last.Started.Unix()))
		last = s.lastHostResult(n)
		if last == nil {
			continue
		}
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
	var workers sync.WaitGroup
	defer workers.Wait()
	defer stop()
	for i := 0; i < cfg.Concurrency; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); s.worker(ctx) }()
	}
	go s.scheduler(ctx)
	srv := &http.Server{Addr: cfg.Listen, Handler: s.mux(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	open := ""
	if cfg.Auth.Token == "" && len(cfg.Auth.Tokens) == 0 && cfg.Auth.UserHeader == "" {
		open = " — no auth configured, everyone is an operator"
	}
	parallel := ""
	if cfg.Concurrency > 1 {
		parallel = fmt.Sprintf(", concurrency %d", cfg.Concurrency)
	}
	fmt.Fprintf(out, "serve: %s on %s: %d job(s)%s, results in %s%s\n", cfg.Title, cfg.Listen, len(cfg.Jobs), parallel, cfg.DataDir, open)
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
