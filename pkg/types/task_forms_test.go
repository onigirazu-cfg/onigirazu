package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTask_NotifyAsStringOrList(t *testing.T) {
	var one, many Task
	require.NoError(t, yaml.Unmarshal([]byte("name: a\ncopy: {dest: /x}\nnotify: restart app\n"), &one))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\ncopy: {dest: /x}\nnotify: [a, b]\n"), &many))
	assert.Equal(t, []string{"restart app"}, one.Notify)
	assert.Equal(t, []string{"a", "b"}, many.Notify)
}

func TestRoleReference_AnsibleForms(t *testing.T) {
	var roles []RoleReference
	require.NoError(t, yaml.Unmarshal([]byte(`
- common
- role: web
  port: 8080
- name: db
  vars: {engine: pg}
`), &roles))
	require.Len(t, roles, 3)
	assert.Equal(t, "common", roles[0].Name)
	assert.Equal(t, "web", roles[1].Name)
	assert.Equal(t, 8080, roles[1].Vars["port"])
	assert.Equal(t, "db", roles[2].Name)
	assert.Equal(t, "pg", roles[2].Vars["engine"])
}

func TestPlay_HostsStringOrList(t *testing.T) {
	var s, l Play
	require.NoError(t, yaml.Unmarshal([]byte("name: a\nhosts: web\ntasks: []\n"), &s))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\nhosts: [web, db]\ntasks: []\n"), &l))
	assert.Equal(t, "web", s.Hosts)
	assert.Equal(t, "web,db", l.Hosts)
}

func TestTask_WithDict(t *testing.T) {
	var lit, expr Task
	require.NoError(t, yaml.Unmarshal([]byte("name: a\ndebug: {msg: x}\nwith_dict: {b: 2, a: 1}\n"), &lit))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\ndebug: {msg: x}\nwith_dict: \"{{ users }}\"\n"), &expr))
	require.NotNil(t, lit.Loop)
	assert.Equal(t, []interface{}{
		map[string]interface{}{"key": "a", "value": 1},
		map[string]interface{}{"key": "b", "value": 2},
	}, lit.Loop.Items)
	require.NotNil(t, expr.Loop)
	assert.Equal(t, "(users) | dict2items", expr.Loop.Expr)
}

func TestTask_ShortForms(t *testing.T) {
	cases := map[string]struct {
		module string
		args   map[string]interface{}
	}{
		"command: echo hi there":                      {"command", map[string]interface{}{"cmd": "echo hi there"}},
		"command: make install chdir=/src creates=/x": {"command", map[string]interface{}{"cmd": "make install", "chdir": "/src", "creates": "/x"}},
		"shell: 'echo a | tr a b'":                    {"shell", map[string]interface{}{"cmd": "echo a | tr a b"}},
		"file: path=/tmp/x state=touch":               {"file", map[string]interface{}{"path": "/tmp/x", "state": "touch"}},
		`debug: msg="hello {{ who }}"`:                {"debug", map[string]interface{}{"msg": "hello {{ who }}"}},
		"debug: msg={{ a | default('x y') }}":         {"debug", map[string]interface{}{"msg": "{{ a | default('x y') }}"}},
		"ping:":                                       {"ping", map[string]interface{}{}},
		"script: /opt/run.sh --fast now":              {"script", map[string]interface{}{"script": "/opt/run.sh", "args": "--fast now"}},
		"command: make\nargs: {chdir: /src}":          {"command", map[string]interface{}{"cmd": "make", "chdir": "/src"}},
		"copy: {dest: /x}\nargs: {mode: '0600'}":      {"copy", map[string]interface{}{"dest": "/x", "mode": "0600"}},
	}
	for src, want := range cases {
		var task Task
		require.NoError(t, yaml.Unmarshal([]byte("name: t\n"+src+"\n"), &task), src)
		assert.Equal(t, want.module, task.Module, src)
		assert.Equal(t, want.args, task.Args, src)
	}
	var bad Task
	assert.Error(t, yaml.Unmarshal([]byte("name: t\nfile: path=/x touch\n"), &bad))
}

func TestPlay_GatherFactsDefault(t *testing.T) {
	cases := map[string]bool{
		"name: a\nhosts: all\ntasks: []\n":                      true,
		"name: a\nhosts: all\ngather_facts: false\ntasks: []\n": false,
		"name: a\nhosts: all\ngather_facts: no\ntasks: []\n":    false,
		"name: a\nhosts: all\ngather_facts: yes\ntasks: []\n":   true,
	}
	for src, want := range cases {
		var p Play
		require.NoError(t, yaml.Unmarshal([]byte(src), &p), src)
		assert.Equal(t, want, p.GatherFacts, src)
	}
}

func TestTask_BooleanKeywords(t *testing.T) {
	var yes, no, none Task
	require.NoError(t, yaml.Unmarshal([]byte("name: a\nping:\nbecome: yes\nignore_errors: 'on'\n"), &yes))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\nping:\nbecome: no\n"), &no))
	require.NoError(t, yaml.Unmarshal([]byte("name: c\nping:\n"), &none))
	assert.True(t, yes.Become && yes.BecomeSet && yes.IgnoreErrors)
	assert.True(t, !no.Become && no.BecomeSet)
	assert.False(t, none.BecomeSet)
}

func TestRoleDependency_Forms(t *testing.T) {
	var meta RoleMeta
	require.NoError(t, yaml.Unmarshal([]byte("dependencies:\n  - common\n  - role: web\n    port: 80\n"), &meta))
	require.Len(t, meta.Dependencies, 2)
	assert.Equal(t, "common", meta.Dependencies[0].Name)
	assert.Equal(t, "web", meta.Dependencies[1].Name)
	assert.Equal(t, 80, meta.Dependencies[1].Vars["port"])
}

func TestTask_LocalAction(t *testing.T) {
	var s, m Task
	require.NoError(t, yaml.Unmarshal([]byte("name: a\nlocal_action: command echo hi chdir=/tmp\n"), &s))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\nlocal_action: {module: copy, dest: /tmp/x, content: y}\n"), &m))
	assert.Equal(t, "command", s.Module)
	assert.Equal(t, map[string]interface{}{"cmd": "echo hi", "chdir": "/tmp"}, s.Args)
	assert.Equal(t, "localhost", s.DelegateTo)
	assert.Equal(t, "copy", m.Module)
	assert.Equal(t, map[string]interface{}{"dest": "/tmp/x", "content": "y"}, m.Args)
	assert.Equal(t, "localhost", m.DelegateTo)
}

func TestTask_WithSequence(t *testing.T) {
	cases := map[string][]interface{}{
		"start=1 end=3":           {"1", "2", "3"},
		"start=0 end=10 stride=5": {"0", "5", "10"},
		"count=2 format=web%02d":  {"web01", "web02"},
		"start=3 end=1 stride=-1": {"3", "2", "1"},
	}
	for spec, want := range cases {
		var task Task
		require.NoError(t, yaml.Unmarshal([]byte("name: a\ndebug: {msg: x}\nwith_sequence: "+spec+"\n"), &task), spec)
		require.NotNil(t, task.Loop, spec)
		assert.Equal(t, want, task.Loop.Items, spec)
	}
}
