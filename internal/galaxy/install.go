// Package galaxy installs the roles and collections a requirements.yml
// names, as ansible-galaxy install -r does, for what onigirazu can use:
// collections from git (their roles), roles from git or from Ansible Galaxy
// (by their source repository). Galaxy collections of modules
// (community.general, ansible.posix, ...) are skipped: onigirazu has its own
// modules.
package galaxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Requirement is one entry of requirements.yml
type Requirement struct {
	Name    string `yaml:"name"`
	Src     string `yaml:"src"`
	Source  string `yaml:"source"`
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
	Scm     string `yaml:"scm"`
}

// Requirements is requirements.yml: a list of roles (old form) or
// collections: and roles:
type Requirements struct {
	Collections []Requirement
	Roles       []Requirement
}

// Load reads a requirements file; entries may be strings
func Load(path string) (*Requirements, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the user's requirements file
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	req := &Requirements{}
	if len(node.Content) == 0 {
		return req, nil
	}
	root := node.Content[0]
	decode := func(n *yaml.Node) ([]Requirement, error) {
		var out []Requirement
		for _, item := range n.Content {
			if item.Kind == yaml.ScalarNode {
				out = append(out, Requirement{Name: item.Value})
				continue
			}
			var r Requirement
			if err := item.Decode(&r); err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	}
	switch root.Kind {
	case yaml.SequenceNode: // roles only
		roles, err := decode(root)
		if err != nil {
			return nil, err
		}
		req.Roles = roles
	case yaml.MappingNode:
		for i := 0; i+1 < len(root.Content); i += 2 {
			list, err := decode(root.Content[i+1])
			if err != nil {
				return nil, err
			}
			switch root.Content[i].Value {
			case "collections":
				req.Collections = list
			case "roles":
				req.Roles = list
			}
		}
	}
	return req, nil
}

// Options of an install
type Options struct {
	RolesPath       string
	CollectionsPath string
	Force           bool
	// GalaxyAPI is the Galaxy server (tests use a fake one)
	GalaxyAPI string
	Log       io.Writer
}

// Install installs what can be used; it returns one line per entry
func Install(ctx context.Context, req *Requirements, opts Options) error {
	if opts.GalaxyAPI == "" {
		opts.GalaxyAPI = "https://galaxy.ansible.com"
	}
	logf := func(format string, a ...interface{}) {
		if opts.Log != nil {
			fmt.Fprintf(opts.Log, format+"\n", a...)
		}
	}
	for _, c := range req.Collections {
		repo := gitSource(c)
		if repo == "" {
			logf("- %s: skipped (a Galaxy collection of modules; onigirazu has its own)", c.Name)
			continue
		}
		dest, err := installCollection(ctx, repo, c.Version, opts)
		if err != nil {
			return fmt.Errorf("collection %s: %w", c.Name, err)
		}
		logf("- %s %s installed in %s", repo, orLatest(c.Version), dest)
	}
	for _, r := range req.Roles {
		name, repo := r.Name, gitSource(r)
		if repo == "" {
			src := r.Src
			if src == "" {
				src = r.Name
			}
			var latest string
			var err error
			repo, latest, err = galaxyRoleRepo(ctx, opts.GalaxyAPI, src)
			if err != nil {
				return fmt.Errorf("role %s: %w", src, err)
			}
			if r.Version == "" {
				r.Version = latest // as ansible-galaxy: the newest release
			}
			if name == "" {
				name = src
			}
		}
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(repo), ".git")
		}
		dest := filepath.Join(opts.RolesPath, name)
		if err := clone(ctx, repo, r.Version, dest, opts.Force); err != nil {
			return fmt.Errorf("role %s: %w", name, err)
		}
		logf("- role %s %s installed in %s", name, orLatest(r.Version), dest)
	}
	return nil
}

func orLatest(v string) string {
	if v == "" {
		return "(default branch)"
	}
	return v
}

// gitSource is the repository of an entry that comes from git (a Galaxy
// name has neither :// nor git@)
func gitSource(r Requirement) string {
	for _, s := range []string{r.Src, r.Source, r.Name} {
		s = strings.TrimPrefix(s, "git+")
		if strings.HasPrefix(s, "git@") || strings.Contains(s, "://") {
			return s
		}
	}
	return ""
}

// installCollection clones a collection and moves it where its galaxy.yml
// says: <path>/ansible_collections/<namespace>/<name>
func installCollection(ctx context.Context, repo, version string, opts Options) (string, error) {
	tmp, err := os.MkdirTemp("", "onigirazu-collection-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	src := filepath.Join(tmp, "c")
	if err := clone(ctx, repo, version, src, true); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(src, "galaxy.yml"))
	if err != nil {
		return "", fmt.Errorf("no galaxy.yml: %w", err)
	}
	var meta struct {
		Namespace string `yaml:"namespace"`
		Name      string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil || meta.Namespace == "" || meta.Name == "" {
		return "", fmt.Errorf("galaxy.yml needs namespace and name")
	}
	base := opts.CollectionsPath
	if filepath.Base(base) != "ansible_collections" {
		base = filepath.Join(base, "ansible_collections")
	}
	dest := filepath.Join(base, meta.Namespace, meta.Name)
	if _, err := os.Stat(dest); err == nil {
		if !opts.Force && sameVersion(dest, version) {
			return dest, nil
		}
		if err := os.RemoveAll(dest); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(src, dest); err != nil {
		return "", err
	}
	return dest, writeVersion(dest, version)
}

// clone checks out repo at version into dest (a shallow clone; the .git
// directory goes, the version is kept in .onigirazu-version)
func clone(ctx context.Context, repo, version, dest string, force bool) error {
	if _, err := os.Stat(dest); err == nil {
		if !force && sameVersion(dest, version) {
			return nil
		}
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	args := []string{"clone", "--quiet", "--depth", "1"}
	if version != "" {
		args = append(args, "--branch", version)
	}
	args = append(args, "--", repo, dest)
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- git with the requirement's repository
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone %s: %v %s", repo, err, strings.TrimSpace(string(out)))
	}
	if err := os.RemoveAll(filepath.Join(dest, ".git")); err != nil {
		return err
	}
	return writeVersion(dest, version)
}

func writeVersion(dest, version string) error {
	return os.WriteFile(filepath.Join(dest, ".onigirazu-version"), []byte(version+"\n"), 0o644)
}

func sameVersion(dest, version string) bool {
	data, err := os.ReadFile(filepath.Join(dest, ".onigirazu-version")) // #nosec G304 -- written by install
	return err == nil && version != "" && strings.TrimSpace(string(data)) == version
}

// galaxyRoleRepo asks Ansible Galaxy where a role (owner.name) lives and
// its newest release ("" when it has none)
func galaxyRoleRepo(ctx context.Context, api, role string) (string, string, error) {
	owner, name, ok := strings.Cut(role, ".")
	if !ok {
		return "", "", fmt.Errorf("a Galaxy role is owner.name")
	}
	u := strings.TrimSuffix(api, "/") + "/api/v1/roles/?owner__username=" + url.QueryEscape(owner) + "&name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("galaxy: HTTP %d", resp.StatusCode)
	}
	var body struct {
		Results []struct {
			GithubUser    string `json:"github_user"`
			GithubRepo    string `json:"github_repo"`
			SummaryFields struct {
				Versions []struct {
					Name string `json:"name"`
				} `json:"versions"`
			} `json:"summary_fields"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", err
	}
	if len(body.Results) == 0 || body.Results[0].GithubUser == "" {
		return "", "", fmt.Errorf("not found on Galaxy")
	}
	r := body.Results[0]
	latest := ""
	for _, v := range r.SummaryFields.Versions {
		if latest == "" || newerVersion(v.Name, latest) {
			latest = v.Name
		}
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", r.GithubUser, r.GithubRepo), latest, nil
}

// newerVersion compares release names such as v1.2.10 and 1.2.9
func newerVersion(a, b string) bool {
	pa := strings.FieldsFunc(strings.TrimPrefix(a, "v"), func(r rune) bool { return r == '.' || r == '-' })
	pb := strings.FieldsFunc(strings.TrimPrefix(b, "v"), func(r rune) bool { return r == '.' || r == '-' })
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		if ea == nil && eb == nil {
			if na != nb {
				return na > nb
			}
			continue
		}
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return len(pa) > len(pb)
}
