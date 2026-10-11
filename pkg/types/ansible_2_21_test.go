package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTaskKeywords221(t *testing.T) {
	var task Task
	if err := yaml.Unmarshal([]byte("command: /bin/true\nloop: [1, 2, 3]\nbreak_when: item == 2\nignore_errors: \"{{ item > 1 }}\"\nignore_unreachable: true\n"), &task); err != nil {
		t.Fatal(err)
	}
	if task.BreakWhen != "item == 2" || task.IgnoreErrorsExpr != "{{ item > 1 }}" || task.IgnoreErrors || !task.IgnoreUnreachable {
		t.Errorf("keywords: %+v", task)
	}
	var plain Task
	_ = yaml.Unmarshal([]byte("command: x\nignore_errors: nope\n"), &plain)
	if plain.IgnoreErrors || plain.IgnoreErrorsExpr != "" {
		t.Errorf("an invalid ignore_errors is false: %+v", plain)
	}
}
