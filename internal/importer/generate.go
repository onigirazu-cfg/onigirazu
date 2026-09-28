package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Accounts from this uid/gid up to below maxID are people's and services'
// own; lower ones belong to the system and its packages
const (
	minID = 1000
	maxID = 60000
)

// repoPaths are files that must be in place before packages install
var repoPaths = regexp.MustCompile(`^/etc/apt/(sources\.list|keyrings|trusted\.gpg)|^/usr/share/keyrings/|^/etc/yum\.repos\.d/|^/etc/pki/rpm-gpg/|^/etc/dnf/`)

// Role is one generated role: the tasks and the files they copy
type Role struct {
	Name  string
	Tasks []*yaml.Node
	// Files: path under the role's files/ -> content
	Files map[string][]byte
}

// Report says what an import took, left out and needs from the user
type Report struct {
	Hosts   []string
	Counts  map[string]int
	Skipped []Skip
	Secrets []Skip
	// Verified: the new playbook was planned; Drift is what it would change
	Verified bool
	Drift    []Drift
}

// Generate writes a playbook for the snapshots into dir: site.yml and one
// role per host (phase 1: no shared layers yet)
func Generate(snaps []*Snapshot, dir string) (*Report, error) {
	rep := &Report{Counts: map[string]int{}}
	var plays []*yaml.Node
	for _, s := range snaps {
		role := hostRole(s, rep)
		if err := writeRole(dir, role); err != nil {
			return nil, err
		}
		rep.Hosts = append(rep.Hosts, s.Host)
		plays = append(plays, mapping(
			"name", scalar(s.Host),
			"hosts", scalar(s.Host),
			"become", boolean(true),
			"gather_facts", boolean(false),
			"roles", seq(scalar(role.Name)),
		))
	}
	if err := writeYAML(filepath.Join(dir, "site.yml"), seq(plays...)); err != nil {
		return nil, err
	}
	return rep, nil
}

// roleName makes a role name out of a host name
func roleName(host string) string {
	return "host_" + regexp.MustCompile(`[^A-Za-z0-9_]+`).ReplaceAllString(host, "_")
}

// hostRole is everything one host has, as tasks
func hostRole(s *Snapshot, rep *Report) *Role {
	r := &Role{Name: roleName(s.Host), Files: map[string][]byte{}}
	var repoFiles, files []File
	localRepos := localRepoDirs(s.Files)
	for _, f := range s.Files {
		if f.Secret != "" {
			rep.Secrets = append(rep.Secrets, Skip{s.Host + ":" + f.Path, f.SecretReason})
			continue
		}
		if repoPaths.MatchString(f.Path) || underAny(f.Path, localRepos) {
			repoFiles = append(repoFiles, f)
		} else {
			files = append(files, f)
		}
	}
	for _, sk := range s.Skipped {
		rep.Skipped = append(rep.Skipped, Skip{s.Host + ":" + sk.Path, sk.Reason})
	}

	r.addFiles(repoFiles, rep)
	r.addAccounts(s, rep)
	r.addPackages(s, rep)
	r.addFiles(files, rep)
	if s.Timezone != "" {
		r.task("Timezone", "timezone", mapping("name", scalar(s.Timezone)))
	}
	r.addMounts(s, rep)
	r.addServices(s, rep)
	return r
}

// fileURI finds the local directories repositories point at
// (deb file:/srv/repo ./, baseurl=file:///srv/repo)
var fileURI = regexp.MustCompile(`file:(?://)?(/[^\s\]"']+)`)

// localRepoDirs are the directories the repository files use as local
// repositories: they must be in place before the package cache updates
func localRepoDirs(files []File) []string {
	var dirs []string
	for _, f := range files {
		if f.Kind != "file" || !repoPaths.MatchString(f.Path) {
			continue
		}
		for _, m := range fileURI.FindAllStringSubmatch(string(f.Content), -1) {
			dirs = append(dirs, strings.TrimSuffix(filepath.Clean(m[1]), "/"))
		}
	}
	return dirs
}

func underAny(path string, dirs []string) bool {
	for _, d := range dirs {
		if path == d || strings.HasPrefix(path, d+"/") {
			return true
		}
	}
	return false
}

func (r *Role) task(name, module string, args *yaml.Node) {
	r.Tasks = append(r.Tasks, mapping("name", scalar(name), module, args))
}

// ownerOf is the owner name, or the uid when the host has no name for it
func ownerOf(name string, id int) string {
	if name == "" || name == "UNKNOWN" {
		return strconv.Itoa(id)
	}
	return name
}

func (r *Role) addFiles(files []File, rep *Report) {
	for _, f := range files {
		owner, group := ownerOf(f.Owner, f.UID), ownerOf(f.Group, f.GID)
		switch f.Kind {
		case "directory":
			r.task("Directory "+f.Path, "file", mapping("path", scalar(f.Path), "state", scalar("directory"),
				"owner", scalar(owner), "group", scalar(group), "mode", scalar(mode(f.Mode))))
			rep.Counts["directories"]++
		case "link":
			r.task("Link "+f.Path, "file", mapping("path", scalar(f.Path), "src", scalar(f.Target),
				"state", scalar("link"), "force", boolean(true)))
			rep.Counts["links"]++
		case "file":
			src := strings.TrimPrefix(f.Path, "/")
			r.Files[src] = f.Content
			r.task("File "+f.Path, "copy", mapping("src", scalar(src), "dest", scalar(f.Path),
				"owner", scalar(owner), "group", scalar(group), "mode", scalar(mode(f.Mode))))
			rep.Counts["files"]++
		}
	}
}

// mode is an octal mode with a leading zero ("644" -> "0644")
func mode(m string) string {
	for len(m) < 4 {
		m = "0" + m
	}
	return m
}

func (r *Role) addAccounts(s *Snapshot, rep *Report) {
	groupName := map[int]string{}
	for _, g := range s.Groups {
		groupName[g.GID] = g.Name
	}
	supplementary := map[string][]string{}
	for _, g := range s.Groups {
		for _, m := range g.Members {
			supplementary[m] = append(supplementary[m], g.Name)
		}
	}
	for _, g := range s.Groups {
		if g.GID < minID || g.GID >= maxID {
			continue
		}
		r.task("Group "+g.Name, "group", mapping("name", scalar(g.Name), "gid", integer(g.GID)))
		rep.Counts["groups"]++
	}
	for _, u := range s.Users {
		if u.UID < minID || u.UID >= maxID {
			continue
		}
		args := mapping("name", scalar(u.Name), "uid", integer(u.UID))
		if g, ok := groupName[u.GID]; ok {
			addPair(args, "group", scalar(g))
		}
		if sup := supplementary[u.Name]; len(sup) > 0 {
			sort.Strings(sup)
			addPair(args, "groups", scalar(strings.Join(sup, ",")))
		}
		if u.Comment != "" {
			addPair(args, "comment", scalar(u.Comment))
		}
		addPair(args, "home", scalar(u.Home))
		addPair(args, "shell", scalar(u.Shell))
		r.task("User "+u.Name, "user", args)
		rep.Counts["users"]++
	}
}

func (r *Role) addPackages(s *Snapshot, rep *Report) {
	if len(s.Packages) == 0 {
		return
	}
	names := make([]*yaml.Node, len(s.Packages))
	for i, p := range s.Packages {
		names[i] = scalar(p)
	}
	switch s.OS {
	case "debian":
		r.task("Packages", "apt", mapping("name", seq(names...), "state", scalar("present"),
			"update_cache", boolean(true), "cache_valid_time", integer(3600)))
	default:
		r.task("Packages", "package", mapping("name", seq(names...), "state", scalar("present")))
	}
	rep.Counts["packages"] += len(s.Packages)
}

func (r *Role) addMounts(s *Snapshot, rep *Report) {
	for _, m := range s.Mounts {
		switch {
		case m.Path == "/" || strings.HasPrefix(m.Path, "/boot") || m.FSType == "swap" || m.Path == "none" ||
			m.FSType == "proc" || m.FSType == "sysfs" || m.FSType == "tmpfs" && m.Path == "/tmp":
			continue
		}
		r.task("Mount "+m.Path, "mount", mapping("path", scalar(m.Path), "src", scalar(m.Src),
			"fstype", scalar(m.FSType), "opts", scalar(m.Opts), "state", scalar("mounted")))
		rep.Counts["mounts"]++
	}
}

// addServices keeps the services whose enablement differs from the
// vendor preset (what installing the packages alone would give)
func (r *Role) addServices(s *Snapshot, rep *Report) {
	if !s.Systemd {
		if len(s.Units) > 0 {
			rep.Skipped = append(rep.Skipped, Skip{s.Host, "systemd is not running: services not imported"})
		}
		return
	}
	names := make([]string, 0, len(s.Units))
	for n := range s.Units {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		u := s.Units[n]
		name := strings.TrimSuffix(n, ".service")
		if strings.Contains(name, "@") {
			continue // templates and instances
		}
		switch {
		case u.State == "enabled" && u.Preset != "enabled":
			args := mapping("name", scalar(name), "enabled", boolean(true))
			if s.Active[n] {
				addPair(args, "state", scalar("started"))
			}
			r.task("Service "+name, "service", args)
		case u.State == "disabled" && u.Preset == "enabled":
			r.task("Service "+name, "service", mapping("name", scalar(name), "enabled", boolean(false)))
		case u.State == "masked":
			rep.Skipped = append(rep.Skipped, Skip{s.Host + ":" + n, "masked unit (not imported)"})
			continue
		default:
			continue
		}
		rep.Counts["services"]++
	}
}

func writeRole(dir string, r *Role) error {
	base := filepath.Join(dir, "roles", r.Name)
	for path, content := range r.Files {
		dst := filepath.Join(base, "files", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return err
		}
	}
	return writeYAML(filepath.Join(base, "tasks", "main.yml"), seq(r.Tasks...))
}

func writeYAML(path string, node *yaml.Node) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, append([]byte("---\n"), data...), 0o644)
}

// YAML node helpers: the output keeps the order it is written in

func scalar(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	return n
}

func integer(i int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(i)}
}

func boolean(b bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(b)}
}

func seq(items ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.SequenceNode, Content: items}
}

// mapping takes key, value pairs; a value is a *yaml.Node or a string
func mapping(kv ...interface{}) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode}
	for i := 0; i+1 < len(kv); i += 2 {
		v, ok := kv[i+1].(*yaml.Node)
		if !ok {
			v = scalar(fmt.Sprint(kv[i+1]))
		}
		key, _ := kv[i].(string)
		addPair(m, key, v)
	}
	return m
}

func addPair(m *yaml.Node, key string, v *yaml.Node) {
	m.Content = append(m.Content, scalar(key), v)
}
