// Package managed keeps the managed state of a playbook: the resources its
// tasks keep on every host (files, packages, services, users, groups), which
// task claims each of them and what was there before. A resource no task
// claims any more is an orphan: onigirazu removes it when it created it and
// puts it back as it was when it only took it over.
package managed

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Version of the state file format
const Version = 1

// Origins of a resource
const (
	// OriginCreated: it was not there before onigirazu made it
	OriginCreated = "created"
	// OriginAdopted: it was there; onigirazu only changed or kept it
	OriginAdopted = "adopted"
	// OriginUnknown: nothing was captured (check mode, no_log, errors)
	OriginUnknown = "unknown"
)

// Actions for an orphan
const (
	ActionDestroy = "destroy"
	ActionRestore = "restore"
	ActionForget  = "forget"
)

// Record is one resource on one host
type Record struct {
	Host string `json:"host"`
	Type string `json:"type"`
	ID   string `json:"id"`
	// Tasks are the keys of the tasks that claim it; none: an orphan
	Tasks []string `json:"tasks"`
	// TaskName is the name of the task that claimed it last
	TaskName       string                 `json:"task_name,omitempty"`
	Origin         string                 `json:"origin"`
	Before         map[string]interface{} `json:"before,omitempty"`
	PreventDestroy bool                   `json:"prevent_destroy,omitempty"`
	FirstApplied   time.Time              `json:"first_applied"`
	LastApplied    time.Time              `json:"last_applied"`
}

// Address names a record: host type id
func (r *Record) Address() string {
	return r.Host + " " + r.Type + " " + r.ID
}

// Action is what happens to the record once no task claims it
func (r *Record) Action() string {
	switch {
	case r.PreventDestroy:
		return ActionForget
	case r.Origin == OriginCreated:
		return ActionDestroy
	case r.Origin == OriginAdopted && restorable(r):
		return ActionRestore
	}
	return ActionForget
}

// restorable: the before state is enough to put the resource back
func restorable(r *Record) bool {
	switch r.Type {
	case "file":
		kind, _ := r.Before["kind"].(string)
		_, hasContent := r.Before["content"]
		return kind == "directory" || (kind == "file" && hasContent)
	case "service":
		return r.Before != nil
	}
	// an adopted package stays installed; an adopted account is kept
	return false
}

// State is the managed state of one playbook
type State struct {
	Version   int       `json:"version"`
	Playbook  string    `json:"playbook"`
	Serial    int       `json:"serial"`
	Updated   time.Time `json:"updated"`
	Resources []*Record `json:"resources"`
}

// Path is where the state of a playbook lives: .onigirazu/<name>.state.json
// next to it, so every playbook of a directory has its own
func Path(playbook string) string {
	base := strings.TrimSuffix(filepath.Base(playbook), filepath.Ext(playbook))
	return filepath.Join(filepath.Dir(playbook), ".onigirazu", base+".state.json")
}

// Load reads a state file; a missing file is an empty state
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the playbook's own state file
	if errors.Is(err, os.ErrNotExist) {
		return &State{Version: Version}, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("state %s: %w", path, err)
	}
	if s.Version > Version {
		return nil, fmt.Errorf("state %s has version %d, this onigirazu reads up to %d", path, s.Version, Version)
	}
	return &s, nil
}

// Save writes the state atomically, readable by the owner only (it holds
// the previous content of files)
func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	s.Version = Version
	s.sort()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *State) sort() {
	sort.Slice(s.Resources, func(i, j int) bool {
		a, b := s.Resources[i], s.Resources[j]
		if a.Host != b.Host {
			return a.Host < b.Host
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.ID < b.ID
	})
}

// Find returns the record of host/type/id
func (s *State) Find(host, typ, id string) *Record {
	for _, r := range s.Resources {
		if r.Host == host && r.Type == typ && r.ID == id {
			return r
		}
	}
	return nil
}

// Remove forgets a record; false when there is none
func (s *State) Remove(host, typ, id string) bool {
	for i, r := range s.Resources {
		if r.Host == host && r.Type == typ && r.ID == id {
			s.Resources = append(s.Resources[:i], s.Resources[i+1:]...)
			return true
		}
	}
	return false
}

// Orphans are the records no task claims
func (s *State) Orphans() []*Record {
	var out []*Record
	for _, r := range s.Resources {
		if len(r.Tasks) == 0 {
			out = append(out, r)
		}
	}
	return out
}

// Clone is a deep enough copy for a dry update (plan)
func (s *State) Clone() *State {
	c := *s
	c.Resources = make([]*Record, len(s.Resources))
	for i, r := range s.Resources {
		rc := *r
		rc.Tasks = append([]string(nil), r.Tasks...)
		c.Resources[i] = &rc
	}
	return &c
}

// Scopes is what a run saw of the playbook (engine.ManagedScopes)
type Scopes struct {
	Keys map[string]bool
	Kept []string
}

// Run describes a finished run for Update
type Run struct {
	Result *types.PlaybookResult
	Scopes Scopes
	// Complete: the whole playbook was run (no --tags, --skip-tags or
	// --start-at-task, not canceled); only then can claims be dropped
	Complete bool
	Now      time.Time
}

// Update records what the run's tasks manage and drops the claims of tasks
// that left the playbook or no longer manage a resource. It returns the
// orphans of the hosts the run touched.
func (s *State) Update(run Run) []*Record {
	if run.Now.IsZero() {
		run.Now = time.Now().UTC()
	}
	s.Serial++
	s.Updated = run.Now
	hosts := map[string]*hostRun{}
	var order []string
	for _, play := range run.Result.Plays {
		for _, h := range play.Hosts {
			hr := hosts[h.Host]
			if hr == nil {
				hr = &hostRun{executed: map[string]bool{}, declared: map[string]map[string]bool{}}
				hosts[h.Host] = hr
				order = append(order, h.Host)
			}
			for _, t := range h.Tasks {
				hr.add(t)
			}
		}
	}
	for _, host := range order {
		hr := hosts[host]
		for _, d := range hr.claims {
			s.claim(host, d, run.Now)
		}
		for _, a := range hr.absent {
			s.Remove(host, a.Type, a.ID)
		}
		if !run.Complete || hr.unsafe {
			continue
		}
		for _, r := range s.Resources {
			if r.Host != host {
				continue
			}
			kept := r.Tasks[:0]
			for _, key := range r.Tasks {
				if keepClaim(key, r, hr, run.Scopes) {
					kept = append(kept, key)
				}
			}
			r.Tasks = kept
		}
	}
	var orphans []*Record
	for _, r := range s.Orphans() {
		if hosts[r.Host] != nil {
			orphans = append(orphans, r)
		}
	}
	return orphans
}

// keepClaim: the task still exists (or may: its scope was skipped) and, if
// it ran on the host, it still declared the resource
func keepClaim(key string, r *Record, hr *hostRun, scopes Scopes) bool {
	if hr.executed[key] {
		return hr.declared[key][r.Type+" "+r.ID]
	}
	if scopes.Keys[key] {
		return true // did not run here (when, loop without items, handler)
	}
	for _, k := range scopes.Kept {
		if key == k || strings.HasPrefix(key, k+"/") {
			return true
		}
	}
	return false
}

type claim struct {
	key, task string
	res       types.ManagedResource
}

type hostRun struct {
	claims   []claim
	absent   []types.ManagedResource
	executed map[string]bool
	declared map[string]map[string]bool
	// unsafe: a task failed or was rolled back; claims are only added
	unsafe bool
}

func (hr *hostRun) add(t types.TaskResult) {
	if (t.Failed && !t.Ignored) || t.RolledBack {
		hr.unsafe = true
		return
	}
	if t.TaskKey == "" || t.Skipped || t.Failed {
		return
	}
	hr.executed[t.TaskKey] = true
	if hr.declared[t.TaskKey] == nil {
		hr.declared[t.TaskKey] = map[string]bool{}
	}
	for _, res := range t.Resources {
		if res.Absent {
			hr.absent = append(hr.absent, res)
			continue
		}
		hr.declared[t.TaskKey][res.Type+" "+res.ID] = true
		hr.claims = append(hr.claims, claim{key: t.TaskKey, task: t.TaskName, res: res})
	}
}

func (s *State) claim(host string, c claim, now time.Time) {
	r := s.Find(host, c.res.Type, c.res.ID)
	if r == nil {
		r = &Record{Host: host, Type: c.res.Type, ID: c.res.ID, Origin: originOf(c.res),
			Before: c.res.Before, FirstApplied: now}
		s.Resources = append(s.Resources, r)
	} else if r.Origin == OriginUnknown && c.res.Before != nil {
		r.Origin, r.Before = originOf(c.res), c.res.Before
	}
	found := false
	for _, k := range r.Tasks {
		found = found || k == c.key
	}
	if !found {
		r.Tasks = append(r.Tasks, c.key)
	}
	r.TaskName = c.task
	r.LastApplied = now
	r.PreventDestroy = c.res.PreventDestroy
}

// originOf tells from the capture whether the task found the resource
func originOf(res types.ManagedResource) string {
	b := res.Before
	if b == nil {
		return OriginUnknown
	}
	switch res.Type {
	case "file":
		if b["kind"] == "absent" {
			return OriginCreated
		}
	case "package":
		if list, ok := b["installed"].([]interface{}); !ok || len(list) == 0 {
			return OriginCreated
		}
	case "user", "group":
		if exists, ok := b["exists"].(bool); ok && !exists {
			return OriginCreated
		}
	}
	return OriginAdopted
}
