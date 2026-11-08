package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestRoleListNormalisingVar(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  []interface{}
	}{
		{"[]", []interface{}{}}, {`"samba, galah"`, []interface{}{"samba", "galah"}}, {"true", []interface{}{"all"}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			e, _ := becomeExprEngine(t)
			registry := modules.NewRegistry()
			e.moduleRegistry = registry
			dir := t.TempDir()
			files := map[string]string{
				"roles/app_sensor/vars/main.yml": `sandbox_kinds: >-
  {{ [] if (sandbox | default([])) in [false, none, '', 'none']
     else (['all'] if (sandbox is sameas true)
     else ((sandbox.split(',') | map('trim') | list) if sandbox is string
     else sandbox)) }}
`,
				"roles/app_sensor/tasks/main.yml": `- name: conditional
  debug:
    msg: install
  when: sandbox_kinds | length > 0
- name: type
  debug:
    msg: "{{ sandbox_kinds | type_debug }}"
- name: value
  debug:
    msg: "{{ sandbox_kinds }}"
`,
				"play.yml": fmt.Sprintf(`- name: sensor
  hosts: all
  gather_facts: false
  vars:
    sandbox: %s
  roles:
    - app_sensor
`, tc.input),
			}
			for path, content := range files {
				path = filepath.Join(dir, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			}
			p := parser.NewEnhancedParser(e.templateEngine, e.logger)
			playbook, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "play.yml"))
			require.NoError(t, err)
			result, err := e.executePlayOn(context.Background(), &playbook.Plays[0], []types.Host{twoHosts()[0]})
			require.NoError(t, err)
			require.Len(t, result.Hosts, 1)
			tasks := result.Hosts[0].Tasks
			require.Len(t, tasks, 3)
			for _, task := range tasks {
				require.False(t, task.Failed, task.Error)
			}
			require.Equal(t, len(tc.want) == 0, tasks[0].Skipped)
			require.Equal(t, "list", tasks[1].Output["msg"])
			require.Equal(t, tc.want, tasks[2].Output["msg"])
		})
	}
}

func TestPublicRoleVariables(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		inventory, play, extra map[string]interface{}
		want                   string
	}{
		{name: "shared", want: "B=[1] 1 role later"},
		{name: "inventory", inventory: map[string]interface{}{"adef": []interface{}{1, 2}, "avar": "inventory"}, want: "B=[1, 2] 2 role later"},
		{name: "play", play: map[string]interface{}{"adef": []interface{}{1, 2, 3}, "avar": "play"}, want: "B=[1, 2, 3] 3 role later"},
		{name: "extra", extra: map[string]interface{}{"adef": []interface{}{9}, "avar": "extra"}, want: "B=[9] 1 extra later"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := becomeExprEngine(t)
			e.moduleRegistry = modules.NewRegistry()
			e.extraVars = tc.extra
			dir := t.TempDir()
			files := map[string]string{
				"roles/a/defaults/main.yml": "adef: [1]\nshared: earlier\n",
				"roles/a/vars/main.yml":     "avar: role\n",
				"roles/a/tasks/main.yml":    "- debug: {msg: 'A={{ adef }}'}\n",
				"roles/b/defaults/main.yml": "shared: later\n",
				"roles/b/vars/main.yml":     "bvar: '{{ adef | length }}'\n",
				"roles/b/tasks/main.yml":    "- debug: {msg: 'B={{ adef }} {{ bvar }} {{ avar }} {{ shared }}'}\n",
				"play.yml":                  "- hosts: all\n  gather_facts: false\n  roles: [a, b]\n  tasks:\n    - debug: {msg: 'B={{ adef }} {{ bvar }} {{ avar }} {{ shared }}'}\n",
			}
			for path, content := range files {
				path = filepath.Join(dir, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			}
			p := parser.NewEnhancedParser(e.templateEngine, e.logger)
			pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "play.yml"))
			require.NoError(t, err)
			pb.Plays[0].Vars = tc.play
			host := twoHosts()[0]
			host.Vars = tc.inventory
			result, err := e.executePlayOn(context.Background(), &pb.Plays[0], []types.Host{host})
			require.NoError(t, err)
			tasks := result.Hosts[0].Tasks
			require.Len(t, tasks, 3)
			for _, task := range tasks {
				require.False(t, task.Failed, task.Error)
			}
			require.Equal(t, tc.want, tasks[1].Output["msg"])
			require.Equal(t, tc.want, tasks[2].Output["msg"])
		})
	}
}

func TestIncludedRoleVisibility(t *testing.T) {
	for _, tc := range []struct {
		module, option string
		public         bool
	}{
		{"include_role", "", false}, {"include_role", ", public: true", false}, {"import_role", "", true}, {"import_role", ", public: false", false},
	} {
		t.Run(tc.module+tc.option, func(t *testing.T) {
			e, _ := becomeExprEngine(t)
			e.moduleRegistry = modules.NewRegistry()
			dir := t.TempDir()
			files := map[string]string{
				"roles/a/defaults/main.yml": "adef: [1]\n",
				"roles/a/vars/main.yml":     "avar: role\n",
				"roles/a/tasks/main.yml":    "- debug: {msg: '{{ adef | length }} {{ avar }}'}\n",
				"play.yml":                  fmt.Sprintf("- hosts: all\n  gather_facts: false\n  pre_tasks:\n    - debug: {msg: '{{ avar | default(\"private\") }}'}\n  tasks:\n    - %s: {name: a%s}\n    - debug: {msg: '{{ adef | default([]) | length }} {{ avar | default(\"private\") }}'}\n", tc.module, tc.option),
			}
			for path, content := range files {
				path = filepath.Join(dir, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			}
			p := parser.NewEnhancedParser(e.templateEngine, e.logger)
			pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "play.yml"))
			require.NoError(t, err)
			result, err := e.executePlayOn(context.Background(), &pb.Plays[0], []types.Host{twoHosts()[0]})
			require.NoError(t, err)
			tasks := result.Hosts[0].Tasks
			require.Len(t, tasks, 3)
			for _, task := range tasks {
				require.False(t, task.Failed, task.Error)
			}
			before, after := "private", "0 private"
			if tc.public {
				after = "1 role"
				if tc.module == "import_role" {
					before = "role"
				}
			}
			require.Equal(t, before, tasks[0].Output["msg"])
			require.Equal(t, after, tasks[2].Output["msg"])
		})
	}
}

func TestBlockRoleVisibilityAcrossHosts(t *testing.T) {
	for _, strategy := range []string{"linear", "free"} {
		t.Run(strategy, func(t *testing.T) {
			e, _ := becomeExprEngine(t)
			e.moduleRegistry = modules.NewRegistry()
			dir := t.TempDir()
			files := map[string]string{
				"roles/private/defaults/main.yml": "private_default: private default\n",
				"roles/private/vars/main.yml":     "private_var: private var\n",
				"roles/private/tasks/main.yml":    "- debug: {msg: '{{ private_default }} {{ private_var }}'}\n",
				"roles/public/defaults/main.yml":  "public_default: public default\n",
				"roles/public/vars/main.yml":      "public_var: public var\n",
				"roles/public/tasks/main.yml":     "- debug: {msg: '{{ public_default }} {{ public_var }}'}\n",
				"play.yml": fmt.Sprintf(`- hosts: all
  gather_facts: false
  strategy: %s
  pre_tasks:
    - debug: {msg: '{{ public_default }} {{ public_var }}'}
  tasks:
    - block:
        - include_role: {name: private, public: true}
        - import_role: {name: public}
        - debug: {msg: '{{ private_default | default("hidden") }} {{ private_var | default("hidden") }} {{ public_default }} {{ public_var }}'}
`, strategy),
			}
			for path, content := range files {
				path = filepath.Join(dir, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			}
			p := parser.NewEnhancedParser(e.templateEngine, e.logger)
			pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "play.yml"))
			require.NoError(t, err)
			result, err := e.executePlayOn(context.Background(), &pb.Plays[0], twoHosts())
			require.NoError(t, err)
			require.Len(t, result.Hosts, 2)
			for _, host := range result.Hosts {
				require.Len(t, host.Tasks, 4)
				for _, task := range host.Tasks {
					require.False(t, task.Failed, task.Error)
				}
				require.Equal(t, "public default public var", host.Tasks[0].Output["msg"])
				require.Equal(t, "private default private var", host.Tasks[1].Output["msg"])
				require.Equal(t, "hidden hidden public default public var", host.Tasks[3].Output["msg"])
			}
		})
	}
}

// runRolePlay writes files into a temp dir and runs the first play of play.yml
func runRolePlay(t *testing.T, files map[string]string) *types.PlayResult {
	t.Helper()
	e, _ := becomeExprEngine(t)
	e.moduleRegistry = modules.NewRegistry()
	dir := t.TempDir()
	for path, content := range files {
		path = filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	}
	p := parser.NewEnhancedParser(e.templateEngine, e.logger)
	playbook, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "play.yml"))
	require.NoError(t, err)
	result, err := e.executePlayOn(context.Background(), &playbook.Plays[0], []types.Host{twoHosts()[0]})
	require.NoError(t, err)
	return result
}

func TestRoleConditionSeesRoleDefaults(t *testing.T) {
	result := runRolePlay(t, map[string]string{
		"roles/a/defaults/main.yml": "enabled: true\n",
		"roles/a/tasks/main.yml":    "- debug: {msg: ran}\n",
		"play.yml":                  "- hosts: all\n  gather_facts: false\n  roles:\n    - {role: a, when: enabled}\n",
	})
	require.Len(t, result.Hosts, 1)
	require.Len(t, result.Hosts[0].Tasks, 1)
	require.False(t, result.Hosts[0].Tasks[0].Skipped)
	require.Equal(t, "ran", result.Hosts[0].Tasks[0].Output["msg"])
}

func TestNestedImportRoleDefaultsArePublic(t *testing.T) {
	result := runRolePlay(t, map[string]string{
		"roles/a/tasks/main.yml":    "- import_role: {name: b}\n",
		"roles/b/defaults/main.yml": "bdef: from-b\n",
		"roles/b/tasks/main.yml":    "- debug: {msg: in-b}\n",
		"play.yml": `- hosts: all
  gather_facts: false
  tasks:
    - debug: {msg: "before={{ bdef }}"}
    - import_role: {name: a}
`,
	})
	tasks := result.Hosts[0].Tasks
	require.NotEmpty(t, tasks)
	require.False(t, tasks[0].Failed, tasks[0].Error)
	require.Equal(t, "before=from-b", tasks[0].Output["msg"])
}
