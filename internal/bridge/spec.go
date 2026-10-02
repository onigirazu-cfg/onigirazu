package bridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Option is one argument of a module, from ansible-doc
type Option struct {
	Type        string        `json:"type"`
	Required    bool          `json:"required"`
	Choices     []interface{} `json:"choices"`
	Aliases     []string      `json:"aliases"`
	Default     interface{}   `json:"default"`
	Elements    string        `json:"elements"`
	Description interface{}   `json:"description"`
}

// Spec is what ansible-doc says about a module
type Spec struct {
	Module      string            `json:"module"`
	Description interface{}       `json:"short_description"`
	Options     map[string]Option `json:"options"`
}

// Names is the option names and aliases
func (s *Spec) Names() map[string]string {
	names := map[string]string{}
	for name, o := range s.Options {
		names[name] = name
		for _, a := range o.Aliases {
			names[a] = name
		}
	}
	return names
}

// Text is a description as one string (ansible-doc gives a string or a list)
func Text(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []interface{}:
		parts := make([]string, len(x))
		for i, p := range x {
			parts[i] = fmt.Sprint(p)
		}
		return strings.Join(parts, " ")
	}
	return ""
}

var (
	specMu    sync.Mutex
	specCache = map[string]*Spec{}
	docVer    string
)

// docBinary is ansible-doc next to the configured ansible-playbook, or from
// PATH
func docBinary() string {
	mu.RLock()
	pb := cfg.AnsiblePlaybook
	mu.RUnlock()
	if pb != "" && strings.ContainsRune(pb, filepath.Separator) {
		return filepath.Join(filepath.Dir(pb), "ansible-doc")
	}
	return "ansible-doc"
}

// ErrNoAnsibleDoc means the arguments cannot be checked: no ansible-doc
var ErrNoAnsibleDoc = errors.New("ansible-doc is not installed")

// SpecFor returns the argument spec of a module through the bridge. Specs
// are cached per ansible-core version in the user cache directory, so lint
// runs offline once a module was looked up.
func SpecFor(module string) (*Spec, error) {
	specMu.Lock()
	defer specMu.Unlock()
	if s, ok := specCache[module]; ok {
		return s, nil
	}
	bin := docBinary()
	if _, err := exec.LookPath(bin); err != nil {
		return nil, ErrNoAnsibleDoc
	}
	if docVer == "" {
		out, err := exec.Command(bin, "--version").Output() // #nosec G204 -- ansible-doc from the configuration
		if err != nil {
			return nil, fmt.Errorf("ansible-doc --version: %w", err)
		}
		docVer = strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	}
	cacheFile := ""
	if dir, err := os.UserCacheDir(); err == nil {
		sum := sha256.Sum256([]byte(docVer))
		cacheFile = filepath.Join(dir, "onigirazu", "ansible-doc", hex.EncodeToString(sum[:6]), module+".json")
		if data, err := os.ReadFile(cacheFile); err == nil { // #nosec G304 -- our own cache
			s := &Spec{}
			if json.Unmarshal(data, s) == nil {
				specCache[module] = s
				return s, nil
			}
		}
	}
	var stderr bytes.Buffer
	cmd := exec.Command(bin, "-j", module) // #nosec G204 -- a module name from the playbook, no shell
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ansible-doc %s: %v %s", module, err, strings.TrimSpace(stderr.String()))
	}
	var doc map[string]struct {
		Doc struct {
			ShortDescription interface{}       `json:"short_description"`
			Options          map[string]Option `json:"options"`
		} `json:"doc"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("ansible-doc %s: %w", module, err)
	}
	if len(doc) == 0 {
		return nil, fmt.Errorf("ansible-core does not know module %s (is its collection installed?)", module)
	}
	s := &Spec{Module: module, Options: map[string]Option{}}
	for _, d := range doc {
		s.Description = d.Doc.ShortDescription
		for k, o := range d.Doc.Options {
			s.Options[k] = o
		}
	}
	specCache[module] = s
	if cacheFile != "" {
		if data, err := json.Marshal(s); err == nil && os.MkdirAll(filepath.Dir(cacheFile), 0o750) == nil {
			_ = os.WriteFile(cacheFile, data, 0o600)
		}
	}
	return s, nil
}

// Problem is one finding about a bridged task's arguments
type Problem struct {
	Error   bool // an error, else a warning
	Message string
}

// CheckArgs checks a task's arguments against the spec: unknown names (with
// the closest known one), missing required options, literal values outside
// the choices. Templated values are left to run time.
func (s *Spec) CheckArgs(args map[string]interface{}) []Problem {
	var out []Problem
	names := s.Names()
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.HasPrefix(k, "_") {
			continue
		}
		if _, ok := names[k]; !ok {
			msg := fmt.Sprintf("module %s has no argument %q", s.Module, k)
			if near := closest(k, names); near != "" {
				msg += fmt.Sprintf(" (did you mean %q?)", near)
			}
			out = append(out, Problem{Error: true, Message: msg})
		}
	}
	given := map[string]interface{}{}
	for k, v := range args {
		if canon, ok := names[k]; ok {
			given[canon] = v
		}
	}
	opts := make([]string, 0, len(s.Options))
	for k := range s.Options {
		opts = append(opts, k)
	}
	sort.Strings(opts)
	for _, name := range opts {
		o := s.Options[name]
		v, has := given[name]
		if o.Required && !has {
			out = append(out, Problem{Error: true, Message: fmt.Sprintf("module %s needs argument %q", s.Module, name)})
			continue
		}
		if !has || len(o.Choices) == 0 {
			continue
		}
		str, isStr := v.(string)
		if isStr && strings.Contains(str, "{{") {
			continue
		}
		if !inChoices(v, o.Choices) {
			out = append(out, Problem{Error: true, Message: fmt.Sprintf("module %s: %s is %v, not one of %v", s.Module, name, v, o.Choices)})
		}
	}
	return out
}

func inChoices(v interface{}, choices []interface{}) bool {
	for _, c := range choices {
		if fmt.Sprint(c) == fmt.Sprint(v) {
			return true
		}
		// yes/no/true/false for boolean choices
		if b, ok := c.(bool); ok {
			switch strings.ToLower(fmt.Sprint(v)) {
			case "yes", "true", "on", "1":
				if b {
					return true
				}
			case "no", "false", "off", "0":
				if !b {
					return true
				}
			}
		}
	}
	return false
}

// closest is the known name nearest to name (edit distance at most 2)
func closest(name string, names map[string]string) string {
	best, bestD := "", 3
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if d := distance(name, k); d < bestD {
			best, bestD = k, d
		}
	}
	return best
}

func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
