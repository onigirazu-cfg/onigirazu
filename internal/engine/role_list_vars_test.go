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
