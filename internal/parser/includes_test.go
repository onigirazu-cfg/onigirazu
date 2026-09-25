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
