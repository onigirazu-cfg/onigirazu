// Package comply holds compliance profiles: named controls, each a verify
// check, with a severity and a remediation hint. `onigirazu comply` runs a
// profile against hosts through the verify module and scores the result.
package comply

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed profiles/*.yml
var bundled embed.FS

// Profile is a set of controls
type Profile struct {
	Name        string    `yaml:"name" json:"name"`
	Title       string    `yaml:"title" json:"title"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	Become      *bool     `yaml:"become,omitempty" json:"become,omitempty"` // default true
	Controls    []Control `yaml:"controls" json:"controls"`
}

// Control is one requirement, checked by one or more verify checks
type Control struct {
	ID          string   `yaml:"id" json:"id"`
	Title       string   `yaml:"title" json:"title"`
	Severity    string   `yaml:"severity" json:"severity"` // low, medium, high, critical
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	// Check is one verify check; Checks several (all must pass)
	Check       map[string]interface{}   `yaml:"check,omitempty" json:"check,omitempty"`
	Checks      []map[string]interface{} `yaml:"checks,omitempty" json:"checks,omitempty"`
	Remediation string                   `yaml:"remediation,omitempty" json:"remediation,omitempty"`
}

var severities = map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}

// SeverityRank orders severities; unknown ones rank as medium
func SeverityRank(s string) int {
	if r, ok := severities[strings.ToLower(s)]; ok {
		return r
	}
	return 2
}

// Bundled lists the profiles shipped in the binary
func Bundled() ([]Profile, error) {
	entries, err := fs.ReadDir(bundled, "profiles")
	if err != nil {
		return nil, err
	}
	var out []Profile
	for _, e := range entries {
		data, err := bundled.ReadFile("profiles/" + e.Name())
		if err != nil {
			return nil, err
		}
		p, err := parse(data, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Load reads a profile by bundled name or file path
func Load(name string) (*Profile, error) {
	if data, err := bundled.ReadFile("profiles/" + strings.TrimSuffix(name, ".yml") + ".yml"); err == nil {
		return parse(data, name)
	}
	data, err := os.ReadFile(name) // #nosec G304 -- the operator's own profile file
	if err != nil {
		if os.IsNotExist(err) {
			names, _ := Bundled()
			known := make([]string, 0, len(names))
			for _, p := range names {
				known = append(known, p.Name)
			}
			return nil, fmt.Errorf("profile %q: not a file and not one of %s", name, strings.Join(known, ", "))
		}
		return nil, err
	}
	return parse(data, filepath.Base(name))
}

func parse(data []byte, source string) (*Profile, error) {
	p := &Profile{}
	if err := yaml.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	if p.Name == "" {
		p.Name = strings.TrimSuffix(source, ".yml")
	}
	if len(p.Controls) == 0 {
		return nil, fmt.Errorf("%s: no controls", source)
	}
	seen := map[string]bool{}
	for i := range p.Controls {
		c := &p.Controls[i]
		if c.ID == "" {
			c.ID = fmt.Sprintf("%d", i+1)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("%s: control %s is defined twice", source, c.ID)
		}
		seen[c.ID] = true
		if c.Severity == "" {
			c.Severity = "medium"
		}
		if _, ok := severities[strings.ToLower(c.Severity)]; !ok {
			return nil, fmt.Errorf("%s: control %s: severity %q (low, medium, high, critical)", source, c.ID, c.Severity)
		}
		if c.Check != nil {
			c.Checks = append([]map[string]interface{}{c.Check}, c.Checks...)
			c.Check = nil
		}
		if len(c.Checks) == 0 {
			return nil, fmt.Errorf("%s: control %s has no check", source, c.ID)
		}
	}
	return p, nil
}

// Filter keeps the controls with one of the tags (any) and at least the
// severity; empty filters keep everything
func (p *Profile) Filter(tags []string, minSeverity string) *Profile {
	out := *p
	out.Controls = nil
	want := map[string]bool{}
	for _, t := range tags {
		want[strings.ToLower(t)] = true
	}
	for _, c := range p.Controls {
		if len(want) > 0 {
			hit := false
			for _, t := range c.Tags {
				if want[strings.ToLower(t)] {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		if minSeverity != "" && SeverityRank(c.Severity) < SeverityRank(minSeverity) {
			continue
		}
		out.Controls = append(out.Controls, c)
	}
	return &out
}

// Checks flattens the controls into the verify module's list; index maps
// each check back to its control
func (p *Profile) Checks() (checks []map[string]interface{}, index []int) {
	for i, c := range p.Controls {
		for _, ch := range c.Checks {
			checks = append(checks, ch)
			index = append(index, i)
		}
	}
	return checks, index
}
