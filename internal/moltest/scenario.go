// Package moltest runs Molecule scenarios (molecule/<name>/molecule.yml)
// with onigirazu: docker or podman containers, the scenario's playbooks,
// an idempotence check. It backs the onigirazu-test command plugin.
package moltest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scenario is the part of molecule.yml onigirazu-test uses
type Scenario struct {
	Name       string `yaml:"-"`
	Dir        string `yaml:"-"` // molecule/<name>
	ProjectDir string `yaml:"-"` // the role (or project) directory

	Dependency struct {
		Name    string `yaml:"name"`
		Enabled *bool  `yaml:"enabled"`
		Options struct {
			RequirementsFile string `yaml:"requirements-file"`
			RoleFile         string `yaml:"role-file"`
		} `yaml:"options"`
	} `yaml:"dependency"`
	Driver struct {
		Name string `yaml:"name"`
	} `yaml:"driver"`
	Platforms   []Platform `yaml:"platforms"`
	Provisioner struct {
		Name      string            `yaml:"name"`
		Env       map[string]string `yaml:"env"`
		Playbooks map[string]string `yaml:"playbooks"`
		Inventory struct {
			GroupVars map[string]map[string]interface{} `yaml:"group_vars"`
			HostVars  map[string]map[string]interface{} `yaml:"host_vars"`
		} `yaml:"inventory"`
	} `yaml:"provisioner"`
	Verifier struct {
		Name    string `yaml:"name"`
		Enabled *bool  `yaml:"enabled"`
	} `yaml:"verifier"`
	ScenarioCfg struct {
		TestSequence []string `yaml:"test_sequence"`
	} `yaml:"scenario"`
}

// Platform is one container of the scenario
type Platform struct {
	Name            string            `yaml:"name"`
	Image           string            `yaml:"image"`
	Hostname        string            `yaml:"hostname"`
	PreBuildImage   *bool             `yaml:"pre_build_image"`
	Privileged      bool              `yaml:"privileged"`
	CgroupnsMode    string            `yaml:"cgroupns_mode"`
	OverrideCommand *bool             `yaml:"override_command"`
	Command         string            `yaml:"command"`
	Volumes         []string          `yaml:"volumes"`
	Tmpfs           []string          `yaml:"tmpfs"`
	CapAdd          []string          `yaml:"capabilities"`
	Env             map[string]string `yaml:"env"`
	Networks        []struct {
		Name string `yaml:"name"`
	} `yaml:"networks"`
	PublishedPorts []string `yaml:"published_ports"`
	Groups         []string `yaml:"groups"`
	Children       []string `yaml:"children"`
	Dockerfile     string   `yaml:"dockerfile"`
}

// Load reads molecule/<name>/molecule.yml under projectDir, with
// environment variables expanded as Molecule does
func Load(projectDir, name string) (*Scenario, error) {
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(projectDir, "molecule", name)
	raw, err := os.ReadFile(filepath.Join(dir, "molecule.yml")) // #nosec G304 -- the scenario the user named
	if err != nil {
		return nil, err
	}
	env := func(k string) (string, bool) {
		switch k {
		case "MOLECULE_SCENARIO_NAME":
			return name, true
		case "MOLECULE_SCENARIO_DIRECTORY":
			return dir, true
		case "MOLECULE_PROJECT_DIRECTORY":
			return projectDir, true
		}
		return os.LookupEnv(k)
	}
	s := &Scenario{}
	if err := yaml.Unmarshal([]byte(expandEnv(string(raw), env)), s); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, "molecule.yml"), err)
	}
	s.Name, s.Dir, s.ProjectDir = name, dir, projectDir
	if s.Driver.Name == "" {
		s.Driver.Name = "docker"
	}
	if s.Driver.Name != "docker" && s.Driver.Name != "podman" {
		return nil, fmt.Errorf("driver %q is not supported (docker, podman)", s.Driver.Name)
	}
	if len(s.Platforms) == 0 {
		return nil, fmt.Errorf("%s: no platforms", name)
	}
	for _, p := range s.Platforms {
		if p.Name == "" || p.Image == "" {
			return nil, fmt.Errorf("%s: every platform needs a name and an image", name)
		}
		if p.PreBuildImage != nil && !*p.PreBuildImage {
			return nil, fmt.Errorf("platform %s: pre_build_image: false (building the image) is not supported", p.Name)
		}
	}
	return s, nil
}

// Scenarios lists the scenario names under projectDir/molecule
func Scenarios(projectDir string) []string {
	matches, _ := filepath.Glob(filepath.Join(projectDir, "molecule", "*", "molecule.yml"))
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, filepath.Base(filepath.Dir(m)))
	}
	return names
}

var envRef = regexp.MustCompile(`\$\$|\$\{([A-Za-z_][A-Za-z0-9_]*)(:?-([^}]*))?\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// expandEnv replaces $VAR, ${VAR}, ${VAR:-default} and ${VAR-default}; $$ is
// a dollar sign
func expandEnv(s string, lookup func(string) (string, bool)) string {
	return envRef.ReplaceAllStringFunc(s, func(m string) string {
		if m == "$$" {
			return "$"
		}
		g := envRef.FindStringSubmatch(m)
		name := g[1]
		if name == "" {
			name = g[4]
		}
		v, ok := lookup(name)
		if g[2] != "" && (!ok || (strings.HasPrefix(g[2], ":") && v == "")) {
			return g[3]
		}
		return v
	})
}

// playbook returns the path of the scenario's playbook for step (converge,
// prepare, verify, side_effect, cleanup), or "" when there is none
func (s *Scenario) playbook(step string) string {
	name := step + ".yml"
	if p := s.Provisioner.Playbooks[step]; p != "" {
		name = p
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(s.Dir, name)
	}
	if _, err := os.Stat(name); err != nil {
		return ""
	}
	return name
}

// DefaultSequence is Molecule's test sequence
var DefaultSequence = []string{"dependency", "cleanup", "destroy", "syntax", "create", "prepare",
	"converge", "idempotence", "side_effect", "verify", "cleanup", "destroy"}

// Sequence is the scenario's test sequence
func (s *Scenario) Sequence() []string {
	if len(s.ScenarioCfg.TestSequence) > 0 {
		return s.ScenarioCfg.TestSequence
	}
	return DefaultSequence
}
