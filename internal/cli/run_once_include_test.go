package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// run_once on include_tasks / include_role: as in Ansible the include runs
// on the first host of each serial batch only, its when is that host's, and
// the included tasks are not run_once themselves (their register stays on
// that host); import_tasks, import_role and a block pass run_once on to
// each of their tasks
func TestRunOnceIncludeRunsOnFirstHostOfEachBatch(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	write("hosts.yml", "all:\n  hosts:\n"+
		"    h1: {ansible_connection: local}\n    h2: {ansible_connection: local}\n"+
		"    h3: {ansible_connection: local}\n    h4: {ansible_connection: local}\n")
	write("inc.yml", `- debug: {msg: "inc {{ inventory_hostname }}"}
  register: increg
- block:
    - debug: {msg: "inc block {{ inventory_hostname }}"}
`)
	write("imp.yml", `- debug: {msg: "imp {{ inventory_hostname }}"}
  register: impreg
`)
	write("impblock.yml", `- block:
    - debug: {msg: "impblock {{ inventory_hostname }}"}
`)
	write("roles/ir/tasks/main.yml", `- debug: {msg: "irole {{ inventory_hostname }}"}
  register: irreg
`)
	write("roles/r/tasks/main.yml", `- debug: {msg: "role {{ inventory_hostname }}"}
`)
	write("site.yml", `- hosts: all
  serial: 2
  order: inventory
  gather_facts: false
  tasks:
    - include_tasks: inc.yml
      run_once: true
    - debug: {msg: "increg {{ inventory_hostname }} {{ increg is defined }}"}
    - name: when false on the first host of the batch
      include_tasks: inc.yml
      run_once: true
      when: inventory_hostname != ansible_play_batch[0]
    - include_role: {name: r}
      run_once: true
    - import_tasks: imp.yml
      run_once: true
    - debug: {msg: "impreg {{ inventory_hostname }} {{ impreg is defined }}"}
    - import_tasks: impblock.yml
      run_once: true
    - import_role: {name: ir}
      run_once: true
    - debug: {msg: "irreg {{ inventory_hostname }} {{ irreg is defined }}"}
    - block:
        - debug: {msg: "blk {{ inventory_hostname }}"}
          register: blkreg
      run_once: true
    - debug: {msg: "blkreg {{ inventory_hostname }} {{ blkreg is defined }}"}
  post_tasks:
    - include_tasks: inc.yml
      run_once: true
      when: ansible_play_batch[-1] == ansible_play_hosts_all[-1]
`)
	res, err := runPlaybook([]string{filepath.Join(dir, "site.yml"), "-i", filepath.Join(dir, "hosts.yml"),
		"--state", filepath.Join(dir, "state.json")})
	require.NoError(t, err)

	got := map[string]int{}
	for _, play := range res.Plays {
		for _, host := range play.Hosts {
			for _, task := range host.Tasks {
				if !task.Skipped {
					got[fmt.Sprint(task.Output["msg"])]++
				}
			}
		}
	}
	// the output of ansible-core 2.21 for the same playbook
	want := map[string]int{
		"inc h1": 1, "inc block h1": 1, "increg h1 True": 1, "increg h2 False": 1, "role h1": 1,
		"imp h1": 1, "impreg h1 True": 1, "impreg h2 True": 1,
		"impblock h1": 1, "irole h1": 1, "irreg h1 True": 1, "irreg h2 True": 1,
		"blk h1": 1, "blkreg h1 True": 1, "blkreg h2 True": 1,
		"inc h3": 2, "inc block h3": 2, "increg h3 True": 1, "increg h4 False": 1, "role h3": 1,
		"imp h3": 1, "impreg h3 True": 1, "impreg h4 True": 1,
		"impblock h3": 1, "irole h3": 1, "irreg h3 True": 1, "irreg h4 True": 1,
		"blk h3": 1, "blkreg h3 True": 1, "blkreg h4 True": 1,
	}
	assert.Equal(t, want, got)
}

// a task of an include with run_once that fails on its host fails that host
// only: the others carry on, as in Ansible
func TestRunOnceIncludeFailureLeavesOtherHosts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	write("hosts.yml", "all:\n  hosts:\n    h1: {ansible_connection: local}\n    h2: {ansible_connection: local}\n")
	write("fail.yml", "- fail: {msg: boom}\n")
	write("site.yml", `- hosts: all
  order: inventory
  gather_facts: false
  tasks:
    - include_tasks: fail.yml
      run_once: true
    - debug: {msg: "after {{ inventory_hostname }}"}
`)
	res, _ := runPlaybook([]string{filepath.Join(dir, "site.yml"), "-i", filepath.Join(dir, "hosts.yml"),
		"--state", filepath.Join(dir, "state.json")})
	require.NotNil(t, res)
	var after []string
	for _, play := range res.Plays {
		for _, host := range play.Hosts {
			for _, task := range host.Tasks {
				if msg, ok := task.Output["msg"].(string); ok && !task.Skipped && msg != "boom" {
					after = append(after, msg)
				}
			}
		}
	}
	assert.Equal(t, []string{"after h2"}, after)
}
