package parser

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRoleFilesKeepWholePathTemplates(t *testing.T) {
	tasks := []types.Task{
		{Module: "copy", Args: map[string]interface{}{"src": "{{ role_path }}/files/sensor.conf", "dest": "/etc/x"}},
		{Module: "copy", Args: map[string]interface{}{"src": "sub/{{ name }}.conf", "dest": "/etc/y"}},
		{Module: "template", Args: map[string]interface{}{"src": "/abs/t.j2", "dest": "/etc/z"}},
	}
	resolveRoleFiles(tasks, "/roles/sensor")
	if tasks[0].Args["src"] != "{{ role_path }}/files/sensor.conf" {
		t.Errorf("a whole-path template is left alone: %v", tasks[0].Args["src"])
	}
	if tasks[1].Args["src"] != "sub/{{ name }}.conf" || tasks[1].Args["_role_dir"] != "/roles/sensor/files" {
		t.Errorf("a relative template keeps its text and remembers the role's files for later: %v %v", tasks[1].Args["src"], tasks[1].Args["_role_dir"])
	}
	if tasks[2].Args["src"] != "/abs/t.j2" {
		t.Errorf("absolute stays: %v", tasks[2].Args["src"])
	}
}
