package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// Stages: every role has one task file per stage and the playbook runs a
// stage on all hosts before the next, so a file in a shared role can be
// owned by an account of a host's own role
var stages = []string{"repositories", "accounts", "packages", "files", "system", "services"}

const (
	stageRepos = iota
	stageAccounts
	stagePackages
	stageFiles
	stageSystem
	stageServices
)

// Report says what an import took, left out and needs from the user
type Report struct {
	Hosts   []string
	Counts  map[string]int
	Skipped []Skip
	Secrets []Skip
	// SecretVars are the variables that stand for secret values
	SecretVars []SecretVar
	// Layers: the generated roles and their hosts
	Layers []LayerInfo
	// Verified: the new playbook was planned; Drift is what it would change
	Verified bool
	Drift    []Drift
	// Adopted: the check run recorded the resources in the managed state
	Adopted bool
}

// SecretVar is a variable the user provides for a secret value
type SecretVar struct {
	Var, Path string
	Line      int
	Hosts     []string
}

// LayerInfo describes one generated role
type LayerInfo struct {
	Role, Kind string // common | group | cluster | host
	Hosts      []string
	Tasks      int
}

// Options change how hosts are grouped
type Options struct {
	// Groups: host -> its inventory groups
	Groups map[string][]string
	// Baseline is a fresh host of the same system: what it has too is not
	// imported
	Baseline *Snapshot
}

// resource is one thing a host has, with the task that recreates it
type resource struct {
	key   string // identity: "file /etc/x", "package nginx"
	stage int
	// value: equal on two hosts when one task serves both
	value string
	name  string // task name (the package name for packages)
	mod   string // module; "package:<os>" for a package
	args  *yaml.Node
	// file content for copy
	file *File
	// what it counts as in the report
	count string
}

// Generate writes a playbook for the snapshots into dir
func Generate(snaps []*Snapshot, dir string) (*Report, error) {
	return GenerateWith(snaps, dir, Options{})
}

// GenerateWith writes roles for what all hosts share, what the hosts of an
// inventory group (or of a cluster of similar hosts) share, and what each
// host has alone, and site.yml running them stage by stage
func GenerateWith(snaps []*Snapshot, dir string, opts Options) (*Report, error) {
	rep := &Report{Counts: map[string]int{}}
	perHost := map[string]map[string]*resource{}
	byName := map[string]*Snapshot{}
	var hosts []string
	for _, s := range snaps {
		hosts = append(hosts, s.Host)
		byName[s.Host] = s
		res := map[string]*resource{}
		for _, r := range hostResources(s, rep) {
			res[r.key] = r
			rep.Counts[r.count]++
		}
		perHost[s.Host] = res
	}
	rep.Hosts = hosts
	if opts.Baseline != nil {
		base := map[string]string{}
		for _, r := range hostResources(opts.Baseline, &Report{Counts: map[string]int{}}) {
			base[r.key] = r.value
		}
		for _, h := range hosts {
			for k, r := range perHost[h] {
				if v, ok := base[k]; ok && v == r.value {
					delete(perHost[h], k)
					rep.Counts[r.count]--
					rep.Counts["same as the baseline"]++
				}
			}
		}
	}

	var layers []*layer
	if l := extract("common", "common", hosts, perHost, byName); l != nil {
		layers = append(layers, l)
	}
	grouped := map[string]bool{}
	for _, g := range groupsBySize(hosts, opts.Groups) {
		if l := extract(safeName(g.name), "group", g.hosts, perHost, byName); l != nil {
			l.cond = fmt.Sprintf("'%s' in group_names", g.name)
			layers = append(layers, l)
		}
		for _, h := range g.hosts {
			grouped[h] = true
		}
	}
	var loose []string
	for _, h := range hosts {
		if !grouped[h] {
			loose = append(loose, h)
		}
	}
	for i, c := range clusters(loose, perHost) {
		if l := extract(clusterName(c, perHost, byName, i), "cluster", c, perHost, byName); l != nil {
			layers = append(layers, l)
		}
	}
	for _, h := range hosts {
		if len(perHost[h]) == 0 {
			continue
		}
		l := &layer{name: roleName(h), kind: "host", hosts: []string{h}}
		if len(hosts) > 1 {
			l.cond = fmt.Sprintf("inventory_hostname == '%s'", h)
		}
		for _, r := range sortedResources(perHost[h]) {
			l.add(r, nil)
		}
		layers = append(layers, l)
	}
	uniqueNames(layers)
	for _, l := range layers {
		if l.cond == "" && l.kind == "cluster" {
			l.cond = fmt.Sprintf("inventory_hostname in [%s]", quoteList(l.hosts))
		}
		if err := l.write(dir); err != nil {
			return nil, err
		}
		rep.Layers = append(rep.Layers, LayerInfo{Role: l.name, Kind: l.kind, Hosts: l.hosts, Tasks: len(l.tasks)})
	}
	if err := writeHostVars(dir, layers); err != nil {
		return nil, err
	}
	if err := writeSite(dir, hosts, layers); err != nil {
		return nil, err
	}
	if err := writeSecretsExample(dir, rep.SecretVars); err != nil {
		return nil, err
	}
	return rep, nil
}

func (rep *Report) addSecretVar(sv SecretValue, path, host string) {
	for i := range rep.SecretVars {
		if rep.SecretVars[i].Var == sv.Var {
			rep.SecretVars[i].Hosts = append(rep.SecretVars[i].Hosts, host)
			return
		}
	}
	rep.SecretVars = append(rep.SecretVars, SecretVar{Var: sv.Var, Path: path, Line: sv.Line, Hosts: []string{host}})
}

// writeSecretsExample lists the secret variables, without values, for the
// user to fill (host_vars, group_vars, -e @secrets.yml or a vault lookup)
func writeSecretsExample(dir string, vars []SecretVar) error {
	if len(vars) == 0 {
		return nil
	}
	m := &yaml.Node{Kind: yaml.MappingNode}
	for _, v := range vars {
		k := scalar(v.Var)
		k.HeadComment = fmt.Sprintf("%s line %d (%s)", v.Path, v.Line, strings.Join(v.Hosts, ", "))
		m.Content = append(m.Content, k, scalar(""))
	}
	return writeYAML(filepath.Join(dir, "secrets.example.yml"), m)
}

// SecretValues are each host's secret values, for the check run only
func SecretValues(snaps []*Snapshot) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, s := range snaps {
		for _, f := range s.Files {
			for _, sv := range f.Secrets {
				if out[s.Host] == nil {
					out[s.Host] = map[string]string{}
				}
				out[s.Host][sv.Var] = sv.Value
			}
		}
	}
	return out
}

// roleName makes a role name out of a host name
func roleName(host string) string {
	return "host_" + safeName(host)
}

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func safeName(s string) string {
	return strings.Trim(unsafeChars.ReplaceAllString(s, "_"), "_")
}

func quoteList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "'" + s + "'"
	}
	return strings.Join(q, ", ")
}

// ownerOf is the owner name, or the uid when the host has no name for it
func ownerOf(name string, id int) string {
	if name == "" || name == "UNKNOWN" {
		return strconv.Itoa(id)
	}
	return name
}

// mode is an octal mode with a leading zero ("644" -> "0644")
func mode(m string) string {
	for len(m) < 4 {
		m = "0" + m
	}
	return m
}

// hostResources is everything one host has
func hostResources(s *Snapshot, rep *Report) []*resource {
	var out []*resource
	add := func(r *resource) { out = append(out, r) }
	localRepos := localRepoDirs(s.Files)
	for _, f := range s.Files {
		if f.Secret != "" {
			rep.Secrets = append(rep.Secrets, Skip{s.Host + ":" + f.Path, f.SecretReason})
			continue
		}
		for _, sv := range f.Secrets {
			rep.addSecretVar(sv, f.Path, s.Host)
		}
		stage := stageFiles
		if repoPaths.MatchString(f.Path) || underAny(f.Path, localRepos) {
			stage = stageRepos
		}
		add(fileResource(f, stage))
	}
	for _, sk := range s.Skipped {
		rep.Skipped = append(rep.Skipped, Skip{s.Host + ":" + sk.Path, sk.Reason})
	}

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
		// "account 1": groups before users in the stage
		add(&resource{key: "account 1 group " + g.Name, stage: stageAccounts, name: "Group " + g.Name, mod: "group",
			args: mapping("name", scalar(g.Name), "gid", integer(g.GID)), count: "groups"})
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
		add(&resource{key: "account 2 user " + u.Name, stage: stageAccounts, name: "User " + u.Name, mod: "user", args: args, count: "users"})
	}
	for _, p := range s.Packages {
		add(&resource{key: "package " + p, stage: stagePackages, name: p, mod: "package:" + s.OS,
			value: "package " + s.OS + " " + p, count: "packages"})
	}
	if s.Timezone != "" {
		add(&resource{key: "timezone", stage: stageSystem, name: "Timezone", mod: "timezone",
			args: mapping("name", scalar(s.Timezone)), count: "timezone"})
	}
	for _, m := range s.Mounts {
		if m.Path == "/" || strings.HasPrefix(m.Path, "/boot") || m.FSType == "swap" || m.Path == "none" ||
			m.FSType == "proc" || m.FSType == "sysfs" || (m.FSType == "tmpfs" && m.Path == "/tmp") {
			continue
		}
		add(&resource{key: "mount " + m.Path, stage: stageSystem, name: "Mount " + m.Path, mod: "mount",
			args: mapping("path", scalar(m.Path), "src", scalar(m.Src), "fstype", scalar(m.FSType),
				"opts", scalar(m.Opts), "state", scalar("mounted")), count: "mounts"})
	}
	out = append(out, serviceResources(s, rep)...)
	for _, r := range out {
		if r.value == "" {
			r.value = r.mod + " " + nodeString(r.args)
		}
	}
	return out
}

func fileResource(f File, stage int) *resource {
	owner, group := ownerOf(f.Owner, f.UID), ownerOf(f.Group, f.GID)
	r := &resource{key: "file " + f.Path, stage: stage}
	switch f.Kind {
	case "directory":
		r.name, r.mod, r.count = "Directory "+f.Path, "file", "directories"
		r.args = mapping("path", scalar(f.Path), "state", scalar("directory"),
			"owner", scalar(owner), "group", scalar(group), "mode", scalar(mode(f.Mode)))
	case "link":
		r.name, r.mod, r.count = "Link "+f.Path, "file", "links"
		r.args = mapping("path", scalar(f.Path), "src", scalar(f.Target), "state", scalar("link"), "force", boolean(true))
	default:
		ff := f
		r.name, r.mod, r.count, r.file = "File "+f.Path, "copy", "files", &ff
		// src is set when the layer is known
		r.args = mapping("dest", scalar(f.Path), "owner", scalar(owner), "group", scalar(group), "mode", scalar(mode(f.Mode)))
		sum := sha256.Sum256(contentOf(&ff))
		r.value = "copy " + nodeString(r.args) + " " + hex.EncodeToString(sum[:])
	}
	return r
}

// serviceResources keeps the services whose enablement differs from the
// vendor preset (what installing the packages alone would give)
func serviceResources(s *Snapshot, rep *Report) []*resource {
	if !s.Systemd {
		if len(s.Units) > 0 {
			rep.Skipped = append(rep.Skipped, Skip{s.Host, "systemd is not running: services not imported"})
		}
		return nil
	}
	names := make([]string, 0, len(s.Units))
	for n := range s.Units {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []*resource
	for _, n := range names {
		u := s.Units[n]
		name := strings.TrimSuffix(n, ".service")
		if strings.Contains(name, "@") {
			continue // templates and instances
		}
		var args *yaml.Node
		switch {
		case u.State == "enabled" && u.Preset != "enabled":
			args = mapping("name", scalar(name), "enabled", boolean(true))
			if s.Active[n] {
				addPair(args, "state", scalar("started"))
			}
		case u.State == "disabled" && u.Preset == "enabled":
			args = mapping("name", scalar(name), "enabled", boolean(false))
		case u.State == "masked":
			rep.Skipped = append(rep.Skipped, Skip{s.Host + ":" + n, "masked unit (not imported)"})
			continue
		default:
			continue
		}
		out = append(out, &resource{key: "service " + name, stage: stageServices, name: "Service " + name,
			mod: "service", args: args, count: "services"})
	}
	return out
}

func nodeString(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	data, _ := yaml.Marshal(n)
	return string(data)
}

func sortedResources(m map[string]*resource) []*resource {
	out := make([]*resource, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].stage != out[j].stage {
			return out[i].stage < out[j].stage
		}
		// parents before children
		return out[i].key < out[j].key
	})
	return out
}

// contentOf is what a file resource compares and writes: the content with
// its secrets masked
func contentOf(f *File) []byte {
	if f.Masked != nil {
		return f.Masked
	}
	return f.Content
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
