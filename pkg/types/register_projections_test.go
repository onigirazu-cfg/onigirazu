package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRegisterProjectionsParse(t *testing.T) {
	var task Task
	if err := yaml.Unmarshal([]byte("name: echo\nshell: echo hello\nregister:\n  echo_result: _task.result\n  upper: _task.result.stdout | upper\n"), &task); err != nil {
		t.Fatal(err)
	}
	if task.Register != "" || task.RegisterVars["upper"] != "_task.result.stdout | upper" || len(task.RegisterVars) != 2 {
		t.Errorf("register map: %+v %q", task.RegisterVars, task.Register)
	}
	var plain Task
	_ = yaml.Unmarshal([]byte("shell: \"true\"\nregister: r\n"), &plain)
	if plain.Register != "r" || plain.RegisterVars != nil {
		t.Errorf("register string: %+v", plain)
	}
}

func TestArgumentSpecOptions(t *testing.T) {
	specs := map[string]interface{}{"main": map[string]interface{}{"options": map[string]interface{}{
		"port":  map[string]interface{}{"type": "int", "required": true, "description": "the port"},
		"mode":  map[string]interface{}{"type": "str", "default": "fast", "choices": []interface{}{"fast", "safe"}},
		"hosts": map[string]interface{}{"type": "list"},
	}}}
	opts := ArgumentSpecsEntry(specs, "main")
	defs := ArgumentSpecOptions(opts)
	if defs["port"].Type != "integer" || !defs["port"].Required || defs["mode"].Default != "fast" || len(defs["mode"].Constraints.Enum) != 2 || defs["hosts"].Type != "array" {
		t.Errorf("mapped: %+v", defs)
	}
	if ArgumentSpecsEntry(map[string]interface{}{"options": opts}, "main") == nil {
		t.Error("a bare options map is an entry too")
	}
}
