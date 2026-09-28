package types

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWithLookupLoops(t *testing.T) {
	var tasks []Task
	src := `
- debug: {msg: "{{ item }}"}
  with_nested: [[1, 2], [a]]
- debug: {msg: "{{ item }}"}
  with_fileglob: "*.conf"
`
	if err := yaml.Unmarshal([]byte(src), &tasks); err != nil {
		t.Fatal(err)
	}
	if tasks[0].Loop == nil || tasks[0].Loop.Lookup != "nested" || tasks[1].Loop.Lookup != "fileglob" {
		t.Fatalf("loops: %+v %+v", tasks[0].Loop, tasks[1].Loop)
	}
	if tasks[0].Module != "debug" || tasks[0].Args["with_nested"] != nil {
		t.Errorf("with_nested must not become a module or an argument: %s %v", tasks[0].Module, tasks[0].Args)
	}
	err := yaml.Unmarshal([]byte("- debug: {msg: x}\n  with_bogus: [1]\n"), &tasks)
	if err == nil || !strings.Contains(err.Error(), "with_bogus is not supported") {
		t.Errorf("an unknown with_ must be an error, got %v", err)
	}
}
