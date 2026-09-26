package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncludes_ExpandedEverywhere(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"site.yml": `plays:
  - name: p
    hosts: all
    tasks:
      - name: first
        debug: {msg: a}
      - include_tasks: tasks/web.yml
        when: web_enabled
        tags: [web]
    handlers:
      - include: handlers.yml
`,
		"tasks/web.yml": `- name: web task
  debug: {msg: "{{ item }}"}
  loop: [x, y]
  when: port is defined
- import_tasks: common.yml
`,
		"tasks/common.yml": `- name: nested
  debug: {msg: nested}
`,
		"handlers.yml": `- name: restart
  debug: {msg: restart}
`,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
	pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	tasks := pb.Plays[0].Tasks
	require.Len(t, tasks, 3)
	assert.Equal(t, "first", tasks[0].Name)
	assert.Equal(t, "web task", tasks[1].Name)
	assert.Equal(t, "(web_enabled) and (port is defined)", tasks[1].When)
	assert.Equal(t, []string{"web"}, tasks[1].Tags)
	assert.Equal(t, "nested", tasks[2].Name, "nested include resolved relative to tasks/")
	assert.Equal(t, "web_enabled", tasks[2].When)
	require.Len(t, pb.Plays[0].Handlers, 1)
	assert.Equal(t, "restart", pb.Plays[0].Handlers[0].Name)
}

func TestIncludes_InsideRoles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"site.yml":                    "plays:\n  - name: p\n    hosts: all\n    roles:\n      - name: web\n",
		"roles/web/tasks/main.yml":    "- include_tasks: install.yml\n",
		"roles/web/tasks/install.yml": "- name: install\n  debug: {msg: install}\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
	pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	require.Len(t, pb.Plays[0].RoleObjects, 1)
	tasks := pb.Plays[0].RoleObjects[0].Tasks
	require.Len(t, tasks, 1)
	assert.Equal(t, "install", tasks[0].Name)
}

func TestIncludeRole_LoadsTheRole(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"site.yml": "plays:\n  - name: p\n    hosts: all\n    tasks:\n" +
			"      - include_role: {name: web}\n        tags: [web]\n" +
			"      - import_role: {name: web, tasks_from: extra}\n",
		"roles/web/tasks/main.yml":    "- name: main\n  template: {src: a.j2, dest: /tmp/a}\n",
		"roles/web/tasks/extra.yml":   "- include_tasks: nested.yml\n",
		"roles/web/tasks/nested.yml":  "- name: nested\n  debug: {msg: n}\n",
		"roles/web/templates/a.j2":    "x",
		"roles/web/defaults/main.yml": "port: 80\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
	pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	tasks := pb.Plays[0].Tasks
	require.Len(t, tasks, 2)
	main := tasks[0].IncludedRole
	require.NotNil(t, main)
	assert.Equal(t, 80, main.Defaults["port"])
	assert.Equal(t, []string{"web"}, main.Tasks[0].Tags, "include tags reach the role's tasks")
	assert.Equal(t, filepath.Join(dir, "roles/web/templates/a.j2"), main.Tasks[0].Args["src"])
	extra := tasks[1].IncludedRole
	require.NotNil(t, extra)
	require.Len(t, extra.Tasks, 1)
	assert.Equal(t, "nested", extra.Tasks[0].Name, "tasks_from with its own includes")
}
