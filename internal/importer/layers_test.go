package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// roleTasks reads a role's tasks, stage by stage
func roleTasks(t *testing.T, dir, role string) []map[string]interface{} {
	t.Helper()
	var all []map[string]interface{}
	for _, st := range stages {
		data, err := os.ReadFile(filepath.Join(dir, "roles", role, "tasks", st+".yml"))
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)
		var tasks []map[string]interface{}
		require.NoError(t, yaml.Unmarshal(data, &tasks))
		all = append(all, tasks...)
	}
	return all
}

func taskNames(tasks []map[string]interface{}) []string {
	var names []string
	for _, task := range tasks {
		names = append(names, task["name"].(string))
	}
	return names
}

func snap(host, ip string, extra ...string) *Snapshot {
	lines := append([]string{
		"OS\tdebian\tubuntu\t24.04",
		"HOST\t" + host + "\t" + host + ".example.org\t" + ip,
		"PKG\tcurl",
		"FILE\t/etc/motd.local\t644\troot\troot\t0\t0",
		"DATA\t" + b64("welcome\n"),
		"FILE\t/etc/app.conf\t644\troot\troot\t0\t0",
		"DATA\t" + b64("name "+host+".example.org\nbind "+ip+"\n"),
		"FILE\t/etc/seed\t600\troot\troot\t0\t0",
		"DATA\t" + b64("seed-"+ip[len(ip)-1:]+"\n"),
	}, extra...)
	s, err := Parse(host, strings.Join(append(lines, "END"), "\n"))
	if err != nil {
		panic(err)
	}
	return s
}

func TestGenerateLayers(t *testing.T) {
	web1 := snap("web1", "10.0.0.1", "PKG\tnginx", "PASSWD\tdeploy:x:1500:1500::/home/deploy:/bin/sh", "GROUP\tdeploy:x:1500:")
	web2 := snap("web2", "10.0.0.2", "PKG\tnginx", "PASSWD\tdeploy:x:1501:1501::/home/deploy:/bin/sh", "GROUP\tdeploy:x:1501:")
	db1 := snap("db1", "10.0.0.3", "PKG\tpostgresql")
	dir := t.TempDir()
	rep, err := GenerateWith([]*Snapshot{web1, web2, db1}, dir, Options{Groups: map[string][]string{
		"web1": {"all", "web"}, "web2": {"all", "web"}, "db1": {"all"}}})
	require.NoError(t, err)
	roles := map[string][]string{}
	for _, l := range rep.Layers {
		roles[l.Role] = l.Hosts
	}
	assert.Equal(t, map[string][]string{"common": {"web1", "web2", "db1"}, "web": {"web1", "web2"},
		"host_web1": {"web1"}, "host_web2": {"web2"}, "host_db1": {"db1"}}, roles)

	common := taskNames(roleTasks(t, dir, "common"))
	assert.Contains(t, common, "File /etc/motd.local")
	assert.Contains(t, common, "File /etc/app.conf", "a template for every host")
	assert.Contains(t, common, "File /etc/seed", "one file per host")
	tmpl, err := os.ReadFile(filepath.Join(dir, "roles", "common", "templates", "etc", "app.conf.j2"))
	require.NoError(t, err)
	assert.Equal(t, "name {{ import_fqdn }}\nbind {{ import_ip }}\n", string(tmpl))
	seed, err := os.ReadFile(filepath.Join(dir, "roles", "common", "files", "db1", "etc", "seed"))
	require.NoError(t, err)
	assert.Equal(t, "seed-3\n", string(seed))
	vars, err := os.ReadFile(filepath.Join(dir, "host_vars", "web2.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(vars), "import_ip: 10.0.0.2")

	web := roleTasks(t, dir, "web")
	require.Len(t, web, 1)
	assert.Equal(t, []interface{}{"nginx"}, web[0]["apt"].(map[string]interface{})["name"])
	// deploy has another uid on each host: host roles
	assert.Contains(t, taskNames(roleTasks(t, dir, "host_web1")), "User deploy")

	site, err := os.ReadFile(filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(site), "hosts: web1:web2:db1")
	assert.Contains(t, string(site), "in group_names")
	// accounts of the hosts run before the files of the shared role
	assert.Less(t, strings.Index(string(site), "name: accounts"), strings.Index(string(site), "name: files"))
}

func TestClusters(t *testing.T) {
	a := snap("a", "10.0.0.1", "PKG\tnginx")
	b := snap("b", "10.0.0.2", "PKG\tnginx")
	c := snap("c", "10.0.0.3", "PKG\tpostgresql", "PKG\tredis-server", "PKG\tvim")
	rep, err := GenerateWith([]*Snapshot{a, b, c}, t.TempDir(), Options{})
	require.NoError(t, err)
	var names []string
	for _, l := range rep.Layers {
		names = append(names, l.Role)
	}
	assert.Contains(t, names, "nginx_hosts")
}

func TestTemplateOfRefusesJinja(t *testing.T) {
	snaps := map[string]*Snapshot{"a": {Hostname: "alpha"}, "b": {Hostname: "bravo"}}
	_, _, ok := templateOf(map[string][]byte{"a": []byte("{{ x }} alpha"), "b": []byte("{{ x }} bravo")}, snaps)
	assert.False(t, ok)
	tmpl, _, ok := templateOf(map[string][]byte{"a": []byte("host alpha"), "b": []byte("host bravo")}, snaps)
	assert.True(t, ok)
	assert.Equal(t, "host {{ import_hostname }}", string(tmpl))
}
