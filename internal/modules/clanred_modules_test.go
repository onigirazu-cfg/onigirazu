package modules

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func strp(s string) *string { return &s }

func TestEditIni(t *testing.T) {
	text := "top = 1\n\n[main]\n# comment\na = 1\nb=2\n\n[other]\na = 9\n"
	// set an existing option, keep the rest
	assert.Equal(t, "top = 1\n\n[main]\n# comment\na = 5\nb=2\n\n[other]\na = 9\n",
		editIni(text, "main", "a", strp("5"), true, false))
	// unchanged
	assert.Equal(t, text, editIni(text, "main", "a", strp("1"), true, false))
	// new option at the end of the section, before the blank line
	assert.Equal(t, "top = 1\n\n[main]\n# comment\na = 1\nb=2\nc=3\n\n[other]\na = 9\n",
		editIni(text, "main", "c", strp("3"), true, true))
	// new section at the end
	assert.Equal(t, text+"\n[new]\nx = y\n", editIni(text, "new", "x", strp("y"), true, false))
	// before the first section
	assert.Equal(t, "top = 1\nz = 0\n\n[main]\n# comment\na = 1\nb=2\n\n[other]\na = 9\n",
		editIni(text, "", "z", strp("0"), true, false))
	// absent option, absent section
	assert.Equal(t, "top = 1\n\n[main]\n# comment\nb=2\n\n[other]\na = 9\n", editIni(text, "main", "a", nil, false, false))
	assert.Equal(t, "top = 1\n\n[main]\n# comment\na = 1\nb=2\n\n", editIni(text, "other", "", nil, false, false))
	// empty file
	assert.Equal(t, "[s]\nk = v\n", editIni("", "s", "k", strp("v"), true, false))
	// sections with slashes (sssd)
	sssd := "[domain/ad.example]\nid_provider = ad\n"
	assert.Equal(t, "[domain/ad.example]\nid_provider = ad\ndyndns_update = True\n",
		editIni(sssd, "domain/ad.example", "dyndns_update", strp("True"), true, false))
}

func TestIniFileLocal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.ini")
	require.NoError(t, os.WriteFile(path, []byte("[a]\nk = 1\n"), 0o600))
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	m := NewIniFileModule()
	args := func() map[string]interface{} {
		return map[string]interface{}{"path": path, "section": "a", "option": "k", "value": "2"}
	}
	r, err := m.Execute(context.Background(), host, args())
	require.NoError(t, err)
	require.True(t, r.Success, r.Error)
	assert.True(t, r.Changed)
	data, _ := os.ReadFile(path)
	assert.Equal(t, "[a]\nk = 2\n", string(data))
	info, _ := os.Stat(path)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "the mode stays")
	r, _ = m.Execute(context.Background(), host, args())
	assert.False(t, r.Changed)
}

func TestSlurpLocal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s")
	require.NoError(t, os.WriteFile(path, []byte("secret\x00bytes"), 0o600))
	r, err := NewSlurpModule().Execute(context.Background(), types.Host{Name: "localhost", Address: "127.0.0.1"},
		map[string]interface{}{"src": path})
	require.NoError(t, err)
	require.True(t, r.Success, r.Error)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("secret\x00bytes")), r.Output["content"])
	r, _ = NewSlurpModule().Execute(context.Background(), types.Host{Name: "localhost", Address: "127.0.0.1"},
		map[string]interface{}{"src": path + ".none"})
	assert.False(t, r.Success)
}

func TestUfwRuleCommand(t *testing.T) {
	assert.Equal(t, "allow proto 'tcp' from '10.0.0.0/8' to 'any' port '3128' comment 'yggdrasil proxy'",
		ufwRuleCommand("allow", map[string]interface{}{"proto": "tcp", "src": "10.0.0.0/8", "port": "3128", "comment": "yggdrasil proxy"}))
	assert.Equal(t, "limit from 'any' to 'any' port 'ssh'", ufwRuleCommand("limit", map[string]interface{}{"port": "ssh"}))
	assert.Equal(t, "route delete allow in on 'eth0' from 'any' to 'any'",
		ufwRuleCommand("allow", map[string]interface{}{"route": true, "delete": true, "direction": "in", "interface": "eth0"}))
}

func TestComposeV2Args(t *testing.T) {
	args := map[string]interface{}{"project_src": "/srv/app", "build": "always", "pull": "missing",
		"files": []interface{}{"a.yml", "b.yml"}, "profiles": []interface{}{"web"}, "env_files": "x.env",
		"remove_orphans": true, "recreate": "never", "wait": true, "wait_timeout": 60}
	composeV2Args(args)
	assert.Equal(t, "/srv/app", args["project_dir"])
	assert.Equal(t, "up -d --build --pull missing --no-recreate --remove-orphans --wait --wait-timeout 60", upCommand(args))
	p := newComposeProject("/srv/app", args)
	assert.Equal(t, []string{"-f", "'a.yml'", "-f", "'b.yml'", "--profile", "'web'", "--env-file", "'x.env'"}, p.flags)
	assert.Equal(t, "up -d --no-build", upCommand(map[string]interface{}{"build": "never", "pull": "policy"}))
	assert.Equal(t, "up -d --build --pull always 'web'", upCommand(map[string]interface{}{"build": true, "pull": true, "services": []interface{}{"web"}}))
	assert.Equal(t, "down -v --remove-orphans", downCommand(map[string]interface{}{"remove_volumes": true, "remove_orphans": true}))
	assert.Equal(t, "docker_compose", types.ShortModuleName("community.docker.docker_compose_v2"))
	assert.Equal(t, "docker_compose", types.ShortModuleName("docker_compose_v2"))
	assert.Equal(t, "ufw", types.ShortModuleName("community.general.ufw"))
	assert.Equal(t, "timezone", types.ShortModuleName("community.general.timezone"))
}

func TestPipSpec(t *testing.T) {
	m := pipSpec.FindStringSubmatch("Requests[socks]>=2.0")
	require.NotNil(t, m)
	assert.Equal(t, "Requests", m[1])
	assert.Equal(t, ">=", m[3])
	assert.Equal(t, "2.0", m[4])
	assert.Equal(t, "zope-interface", normalizePip("Zope.Interface"))
}

func TestParseGetent(t *testing.T) {
	got := parseGetent("root:x:0:0:root:/root:/bin/bash\n", "passwd", "")
	assert.Equal(t, []interface{}{"x", "0", "0", "root", "/root", "/bin/bash"}, got["root"])
	hosts := parseGetent("127.0.0.1       localhost ip6-localhost\n", "hosts", "")
	assert.Equal(t, []interface{}{"localhost", "ip6-localhost"}, hosts["127.0.0.1"])
}
