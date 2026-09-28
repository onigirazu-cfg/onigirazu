package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// layer is one generated role and the hosts it applies to
type layer struct {
	name, kind string
	hosts      []string
	// cond selects the layer's hosts in the stage plays ("" = all)
	cond  string
	tasks []layerTask
	files map[string][]byte // files/<path>
	tmpls map[string][]byte // templates/<path>
	// vars: host -> template variables the layer's templates use
	vars map[string]map[string]string
	pkgs map[string][]string // os -> packages
}

type layerTask struct {
	stage int
	node  *yaml.Node
}

// add puts a resource into the layer; perHost is the content each host
// has when the files differ (nil: one content for all)
func (l *layer) add(r *resource, perHost map[string][]byte) {
	if strings.HasPrefix(r.mod, "package:") {
		if l.pkgs == nil {
			l.pkgs = map[string][]string{}
		}
		os := strings.TrimPrefix(r.mod, "package:")
		l.pkgs[os] = append(l.pkgs[os], r.name)
		return
	}
	args := r.args
	mod := r.mod
	if r.file != nil && r.file.Masked != nil {
		// a file with secrets is a template of its masked content
		if perHost == nil {
			l.addTemplate(r, r.file.Masked, nil)
			return
		}
		src := strings.TrimPrefix(r.file.Path, "/") + ".j2"
		if l.tmpls == nil {
			l.tmpls = map[string][]byte{}
		}
		for h, c := range perHost {
			l.tmpls[h+"/"+src] = unmark(c)
		}
		args = cloneMapping(r.args)
		prependPair(args, "src", scalar("{{ inventory_hostname }}/"+src))
		l.tasks = append(l.tasks, layerTask{r.stage, mapping("name", scalar(r.name), "template", args)})
		return
	}
	if r.file != nil {
		src := strings.TrimPrefix(r.file.Path, "/")
		args = cloneMapping(r.args)
		if l.files == nil {
			l.files = map[string][]byte{}
		}
		if perHost == nil {
			l.files[src] = r.file.Content
			prependPair(args, "src", scalar(src))
		} else {
			for h, c := range perHost {
				l.files[h+"/"+src] = c
			}
			prependPair(args, "src", scalar("{{ inventory_hostname }}/"+src))
		}
	}
	l.tasks = append(l.tasks, layerTask{r.stage, mapping("name", scalar(r.name), mod, args)})
}

// addTemplate puts a file as a template whose variables differ per host
func (l *layer) addTemplate(r *resource, tmpl []byte, vars map[string]map[string]string) {
	src := strings.TrimPrefix(r.file.Path, "/") + ".j2"
	if l.tmpls == nil {
		l.tmpls = map[string][]byte{}
	}
	l.tmpls[src] = unmark(tmpl)
	if l.vars == nil {
		l.vars = map[string]map[string]string{}
	}
	for h, v := range vars {
		if l.vars[h] == nil {
			l.vars[h] = map[string]string{}
		}
		for k, val := range v {
			l.vars[h][k] = val
		}
	}
	args := cloneMapping(r.args)
	prependPair(args, "src", scalar(src))
	l.tasks = append(l.tasks, layerTask{r.stage, mapping("name", scalar(r.name), "template", args)})
}

// extract moves what the hosts share into a new layer: resources equal on
// all of them, and files with the same path, mode and owner whose content
// differs only by host names and addresses (a template) or otherwise (one
// file per host)
func extract(name, kind string, hosts []string, perHost map[string]map[string]*resource, snaps map[string]*Snapshot) *layer {
	if len(hosts) < 2 {
		return nil
	}
	l := &layer{name: name, kind: kind, hosts: append([]string(nil), hosts...)}
	first := perHost[hosts[0]]
	for _, r := range sortedResources(first) {
		same, meta := true, true
		for _, h := range hosts[1:] {
			o, ok := perHost[h][r.key]
			if !ok {
				same, meta = false, false
				break
			}
			if o.value != r.value {
				same = false
				if r.file == nil || o.file == nil || nodeString(o.args) != nodeString(r.args) {
					meta = false
				}
			}
		}
		switch {
		case same:
			l.add(r, nil)
		case meta && r.file != nil:
			contents := map[string][]byte{}
			for _, h := range hosts {
				contents[h] = contentOf(perHost[h][r.key].file)
			}
			if tmpl, vars, ok := templateOf(contents, snaps); ok {
				l.addTemplate(r, tmpl, vars)
			} else {
				l.add(r, contents)
			}
		default:
			continue
		}
		for _, h := range hosts {
			delete(perHost[h], r.key)
		}
	}
	if len(l.tasks) == 0 && len(l.pkgs) == 0 {
		return nil
	}
	return l
}

// host facts a template may use, longest value first when replacing
var templateVars = []string{"import_fqdn", "import_hostname", "import_ip"}

func hostFacts(s *Snapshot) map[string]string {
	return map[string]string{"import_fqdn": s.FQDN, "import_hostname": s.Hostname, "import_ip": s.IP}
}

var jinjaSyntax = regexp.MustCompile(`\{\{|\{%|\{#`)

// templateOf finds one template for files that differ only by the hosts'
// names and addresses. Replacing a host's values by variables is undone
// exactly by rendering with the same values, so the template gives back
// every host's file. Files that already hold Jinja syntax are not
// templated.
func templateOf(contents map[string][]byte, snaps map[string]*Snapshot) ([]byte, map[string]map[string]string, bool) {
	var tmpl string
	vars := map[string]map[string]string{}
	first := true
	for h, c := range contents {
		s := snaps[h]
		if s == nil || jinjaSyntax.Match(c) {
			return nil, nil, false
		}
		facts := hostFacts(s)
		// values that are empty or one another's substrings would not
		// round-trip safely
		used := map[string]string{}
		text := string(c)
		keys := append([]string(nil), templateVars...)
		sort.SliceStable(keys, func(i, j int) bool { return len(facts[keys[i]]) > len(facts[keys[j]]) })
		for _, k := range keys {
			v := facts[k]
			if len(v) < 3 || !strings.Contains(text, v) {
				continue
			}
			text = replaceOutsideMarks(text, v, "{{ "+k+" }}")
			used[k] = v
		}
		if len(used) == 0 {
			return nil, nil, false
		}
		// the template must render back to the file
		back := text
		for k, v := range used {
			back = strings.ReplaceAll(back, "{{ "+k+" }}", v)
		}
		if back != string(c) {
			return nil, nil, false
		}
		if first {
			tmpl, first = text, false
		} else if text != tmpl {
			return nil, nil, false
		}
		vars[h] = used
	}
	return []byte(tmpl), vars, true
}

// replaceOutsideMarks replaces old by new except inside secret marks
func replaceOutsideMarks(text, old, new string) string {
	var b strings.Builder
	last := 0
	for _, m := range secretMarks.FindAllStringIndex(text, -1) {
		b.WriteString(strings.ReplaceAll(text[last:m[0]], old, new))
		b.WriteString(text[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(strings.ReplaceAll(text[last:], old, new))
	return b.String()
}

// group is an inventory group of imported hosts
type group struct {
	name  string
	hosts []string
}

// groupsBySize are the inventory groups with at least two imported hosts,
// the biggest first (a resource goes to the biggest group sharing it)
func groupsBySize(hosts []string, hostGroups map[string][]string) []group {
	members := map[string][]string{}
	for _, h := range hosts {
		for _, g := range hostGroups[h] {
			if g == "all" || g == "ungrouped" {
				continue
			}
			members[g] = append(members[g], h)
		}
	}
	var out []group
	for g, hs := range members {
		if len(hs) >= 2 {
			out = append(out, group{g, hs})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].hosts) != len(out[j].hosts) {
			return len(out[i].hosts) > len(out[j].hosts)
		}
		return out[i].name < out[j].name
	})
	return out
}

// clusters groups hosts without an inventory group by how much they share
// (Jaccard similarity of their packages and services, at least 0.5)
func clusters(hosts []string, perHost map[string]map[string]*resource) [][]string {
	sig := func(h string) map[string]bool {
		out := map[string]bool{}
		for k := range perHost[h] {
			if strings.HasPrefix(k, "package ") || strings.HasPrefix(k, "service ") {
				out[k] = true
			}
		}
		return out
	}
	sim := func(a, b map[string]bool) float64 {
		inter, union := 0, len(a)
		for k := range b {
			if a[k] {
				inter++
			} else {
				union++
			}
		}
		if union == 0 {
			return 0
		}
		return float64(inter) / float64(union)
	}
	var out [][]string
	used := map[string]bool{}
	for i, h := range hosts {
		if used[h] {
			continue
		}
		c := []string{h}
		used[h] = true
		for _, o := range hosts[i+1:] {
			if !used[o] && sim(sig(h), sig(o)) >= 0.5 {
				c = append(c, o)
				used[o] = true
			}
		}
		if len(c) >= 2 {
			out = append(out, c)
		}
	}
	return out
}

// serverPackages name a cluster better than its first package does
var serverPackages = regexp.MustCompile(`^(nginx|apache2|httpd|haproxy|postgresql(-\d+)?|mysql-server|mariadb-server|redis(-server)?|mongodb-org|docker-ce|containerd|podman|bind9|named|postfix|dovecot-core|squid|tomcat\d*|jenkins|elasticsearch|rabbitmq-server|memcached|keepalived|openvpn|wireguard(-tools)?|zabbix-server-\w+|prometheus|grafana)$`)

// clusterName names a cluster after a service its hosts run, a server
// package they share, or their distribution
func clusterName(hosts []string, perHost map[string]map[string]*resource, snaps map[string]*Snapshot, i int) string {
	shared := func(prefix string, ok func(string) bool) string {
		var names []string
		for k := range perHost[hosts[0]] {
			if !strings.HasPrefix(k, prefix) || !ok(strings.TrimPrefix(k, prefix)) {
				continue
			}
			all := true
			for _, h := range hosts[1:] {
				if _, found := perHost[h][k]; !found {
					all = false
				}
			}
			if all {
				names = append(names, strings.TrimPrefix(k, prefix))
			}
		}
		sort.Strings(names)
		if len(names) == 0 {
			return ""
		}
		return names[0]
	}
	if n := shared("service ", func(string) bool { return true }); n != "" {
		return safeName(n) + "_hosts"
	}
	if n := shared("package ", serverPackages.MatchString); n != "" {
		return safeName(n) + "_hosts"
	}
	if s := snaps[hosts[0]]; s != nil && s.Distro != "" {
		same := true
		for _, h := range hosts[1:] {
			if o := snaps[h]; o == nil || o.Distro != s.Distro {
				same = false
			}
		}
		if same {
			return safeName(strings.Fields(s.Distro)[0]) + "_hosts"
		}
	}
	return fmt.Sprintf("cluster%d", i+1)
}

// uniqueNames keeps role names apart (a group and a host may share one)
func uniqueNames(layers []*layer) {
	seen := map[string]int{}
	for _, l := range layers {
		seen[l.name]++
		if n := seen[l.name]; n > 1 {
			l.name = fmt.Sprintf("%s_%d", l.name, n)
		}
	}
}

func (l *layer) write(dir string) error {
	base := filepath.Join(dir, "roles", l.name)
	for sub, files := range map[string]map[string][]byte{"files": l.files, "templates": l.tmpls} {
		for path, content := range files {
			dst := filepath.Join(base, sub, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, content, 0o644); err != nil {
				return err
			}
		}
	}
	for os, pkgs := range l.pkgs {
		sort.Strings(pkgs)
		names := make([]*yaml.Node, len(pkgs))
		for i, p := range pkgs {
			names[i] = scalar(p)
		}
		var task *yaml.Node
		if os == "debian" {
			task = mapping("name", scalar("Packages"), "apt", mapping("name", seq(names...), "state", scalar("present"),
				"update_cache", boolean(true), "cache_valid_time", integer(3600)))
		} else {
			task = mapping("name", scalar("Packages"), "package", mapping("name", seq(names...), "state", scalar("present")))
		}
		if len(l.pkgs) > 1 {
			family := map[string]string{"debian": "Debian", "redhat": "RedHat"}[os]
			addPair(task, "when", scalar(fmt.Sprintf("ansible_os_family == '%s'", family)))
		}
		l.tasks = append(l.tasks, layerTask{stagePackages, task})
	}
	var imports []*yaml.Node
	for i, st := range stages {
		var nodes []*yaml.Node
		for _, t := range l.tasks {
			if t.stage == i {
				nodes = append(nodes, t.node)
			}
		}
		if len(nodes) == 0 {
			continue
		}
		if err := writeYAML(filepath.Join(base, "tasks", st+".yml"), seq(nodes...)); err != nil {
			return err
		}
		imports = append(imports, mapping("import_tasks", scalar(st+".yml")))
	}
	return writeYAML(filepath.Join(base, "tasks", "main.yml"), seq(imports...))
}

// hasStage tells whether the layer has tasks for a stage
func (l *layer) hasStage(i int) bool {
	for _, t := range l.tasks {
		if t.stage == i {
			return true
		}
	}
	return false
}

// writeSite writes site.yml: one play per stage over all imported hosts,
// each running the stage of every layer on the layer's hosts
func writeSite(dir string, hosts []string, layers []*layer) error {
	var plays []*yaml.Node
	for i, st := range stages {
		var tasks []*yaml.Node
		for _, l := range layers {
			if !l.hasStage(i) {
				continue
			}
			t := mapping("name", scalar(l.name+" "+st), "include_role",
				mapping("name", scalar(l.name), "tasks_from", scalar(st)))
			if l.cond != "" {
				addPair(t, "when", scalar(l.cond))
			}
			tasks = append(tasks, t)
		}
		if len(tasks) == 0 {
			continue
		}
		plays = append(plays, mapping("name", scalar(st), "hosts", scalar(strings.Join(hosts, ":")),
			"become", boolean(true), "gather_facts", boolean(needsFacts(layers)), "tasks", seq(tasks...)))
	}
	return writeYAML(filepath.Join(dir, "site.yml"), seq(plays...))
}

// needsFacts: a layer holds packages for two OS families
func needsFacts(layers []*layer) bool {
	for _, l := range layers {
		if len(l.pkgs) > 1 {
			return true
		}
	}
	return false
}

// writeHostVars writes the template variables of every host
func writeHostVars(dir string, layers []*layer) error {
	vars := map[string]map[string]string{}
	for _, l := range layers {
		for h, v := range l.vars {
			if vars[h] == nil {
				vars[h] = map[string]string{}
			}
			for k, val := range v {
				vars[h][k] = val
			}
		}
	}
	for h, v := range vars {
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m := &yaml.Node{Kind: yaml.MappingNode}
		for _, k := range keys {
			addPair(m, k, scalar(v[k]))
		}
		if err := writeYAML(filepath.Join(dir, "host_vars", h+".yml"), m); err != nil {
			return err
		}
	}
	return nil
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
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

func integer(i int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprint(i)}
}

func boolean(b bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprint(b)}
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

func prependPair(m *yaml.Node, key string, v *yaml.Node) {
	m.Content = append([]*yaml.Node{scalar(key), v}, m.Content...)
}

func cloneMapping(m *yaml.Node) *yaml.Node {
	c := *m
	c.Content = append([]*yaml.Node(nil), m.Content...)
	return &c
}
