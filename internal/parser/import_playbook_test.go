package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportPlaybook(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	write("site.yml", `
- import_playbook: playbooks/base.yml
  vars: {env: prod, level: 1}
  tags: base
- name: local
  hosts: all
  tasks:
    - debug: {msg: here}
`)
	write("playbooks/base.yml", `
- name: base
  hosts: web
  vars: {level: 2}
  vars_files: [vars.yml]
  tasks:
    - copy: {src: motd, dest: /etc/motd, mode: 0644}
- ansible.builtin.import_playbook: more/extra.yml
`)
	write("playbooks/vars.yml", "from_file: true\n")
	write("playbooks/files/motd", "hi\n")
	write("playbooks/more/extra.yml", `
- name: extra
  hosts: db
  tags: [x]
  tasks:
    - debug: {msg: extra}
`)
	p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
	pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	require.Len(t, pb.Plays, 3)

	base, extra, local := pb.Plays[0], pb.Plays[1], pb.Plays[2]
	assert.Equal(t, "base", base.Name)
	assert.Equal(t, "prod", base.Vars["env"])
	assert.Equal(t, 2, base.Vars["level"], "the play's own vars win")
	assert.Equal(t, true, base.Vars["from_file"], "vars_files relative to the imported file")
	assert.Contains(t, base.Tags, "base")
	assert.Equal(t, filepath.Join(dir, "playbooks", "files", "motd"), base.Tasks[0].Args["src"])
	assert.Equal(t, 420, base.Tasks[0].Args["mode"], "0644 as in a playbook without imports")

	assert.Equal(t, "extra", extra.Name)
	assert.ElementsMatch(t, []string{"x", "base"}, extra.Tags)
	assert.Equal(t, filepath.Join(dir, "playbooks", "more"), extra.BaseDir)
	assert.Equal(t, "local", local.Name)
	assert.Equal(t, dir, local.BaseDir)
}

func TestImportPlaybookErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yml"), []byte("- import_playbook: a.yml\n"), 0o644))
	p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
	_, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "a.yml"))
	assert.ErrorContains(t, err, "nested deeper")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.yml"), []byte("- import_playbook: c.yml\n  when: x\n"), 0o644))
	_, err = p.ParsePlaybook(context.Background(), filepath.Join(dir, "b.yml"))
	assert.ErrorContains(t, err, "when is not supported")
}
