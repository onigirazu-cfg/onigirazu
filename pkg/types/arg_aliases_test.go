package types

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCanonicalArgs(t *testing.T) {
	var tasks []Task
	src := `
- apt: {pkg: [git, make], state: present}
- ansible.builtin.lineinfile: {dest: /etc/x, line: y}
- file: {path: /a, dest: /b}
- systemd: {unit: ssh, daemon-reload: true}
- sysctl: {key: vm.swappiness, val: 10}
`
	if err := yaml.Unmarshal([]byte(src), &tasks); err != nil {
		t.Fatal(err)
	}
	want := []map[string]interface{}{
		{"name": []interface{}{"git", "make"}, "state": "present"},
		{"path": "/etc/x", "line": "y"},
		{"path": "/a"},
		{"name": "ssh", "daemon_reload": true},
		{"name": "vm.swappiness", "value": 10},
	}
	for i, w := range want {
		if !reflect.DeepEqual(tasks[i].Args, w) {
			t.Errorf("task %d (%s): args %v, want %v", i, tasks[i].Module, tasks[i].Args, w)
		}
	}
}
