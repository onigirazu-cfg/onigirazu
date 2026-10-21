package types

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBecomeRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"maybe", "null", "[true]", "{enabled: true}", "1"} {
		for _, level := range []string{"task", "block", "play"} {
			t.Run(level+"/"+value, func(t *testing.T) {
				data := "become: " + value + "\n"
				var target interface{}
				switch level {
				case "play":
					target = &Play{}
				case "block":
					data += "block:\n  - command: true\n"
					target = &Task{}
				default:
					data += "command: true\n"
					target = &Task{}
				}
				err := yaml.Unmarshal([]byte(data), target)
				if err == nil || !strings.Contains(err.Error(), "become:") || !strings.Contains(err.Error(), "is not a boolean") {
					t.Fatalf("expected become boolean error, got %v", err)
				}
			})
		}
	}
}

func TestBecomeValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  string
		become bool
		expr   string
		set    bool
	}{
		{name: "absent"},
		{name: "true", value: "true", become: true, set: true},
		{name: "false", value: "false", set: true},
		{name: "yes", value: "yes", become: true, set: true},
		{name: "off", value: "off", set: true},
		{name: "quoted", value: `"TRUE"`, become: true, set: true},
		{name: "template", value: `"{{ x }}"`, expr: "{{ x }}", set: true},
		{name: "comparison", value: `"{{ ansible_connection != 'local' }}"`, expr: "{{ ansible_connection != 'local' }}", set: true},
	} {
		for _, level := range []string{"task", "block", "play"} {
			t.Run(level+"/"+tc.name, func(t *testing.T) {
				data := ""
				if tc.set {
					data = "become: " + tc.value + "\n"
				}
				if level == "play" {
					var play Play
					if err := yaml.Unmarshal([]byte(data+"hosts: all\n"), &play); err != nil {
						t.Fatal(err)
					}
					if play.Become != tc.become || play.BecomeExpr != tc.expr {
						t.Fatalf("become = %v, expression = %q", play.Become, play.BecomeExpr)
					}
					return
				}
				if level == "block" {
					data += "block:\n  - command: true\n"
				} else {
					data += "command: true\n"
				}
				var task Task
				if err := yaml.Unmarshal([]byte(data), &task); err != nil {
					t.Fatal(err)
				}
				if task.Become != tc.become || task.BecomeExpr != tc.expr || task.BecomeSet != tc.set {
					t.Fatalf("become = %v, expression = %q, set = %v", task.Become, task.BecomeExpr, task.BecomeSet)
				}
			})
		}
	}
}

func TestPlayBecomeExprSurvivesAlias(t *testing.T) {
	var plays []Play
	src := "- &base\n  hosts: all\n  become: \"{{ use_sudo }}\"\n- *base\n"
	if err := yaml.Unmarshal([]byte(src), &plays); err != nil {
		t.Fatal(err)
	}
	for i, p := range plays {
		if p.BecomeExpr != "{{ use_sudo }}" {
			t.Fatalf("play %d: BecomeExpr = %q", i, p.BecomeExpr)
		}
	}
}
