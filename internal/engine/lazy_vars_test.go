package engine

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/template"
)

func TestRenderLazyVars(t *testing.T) {
	e := &ExecutionEngine{
		templateEngine: template.NewEngine(),
		hostVars:       map[string]map[string]interface{}{"h": {"registered": "{{ keep }}"}},
		extraVars:      map[string]interface{}{"cli": "{{ keep }}"},
	}
	vars := map[string]interface{}{
		"ssh_port":   2222,
		"port":       "{{ ssh_port | default(22) }}",
		"chained":    "p{{ port }}",
		"users":      []interface{}{map[string]interface{}{"name": "{{ port }}"}},
		"registered": "{{ keep }}",
		"cli":        "{{ keep }}",
		"broken":     "{{ .Names }}",
		"versions":   []interface{}{"3.12", "3.11"},
		"global":     "{{ versions }}",
	}
	e.renderLazyVars(context.Background(), "h", vars)
	if vars["port"] != "2222" || vars["chained"] != "p2222" {
		t.Errorf("port = %v, chained = %v", vars["port"], vars["chained"])
	}
	if got := vars["users"].([]interface{})[0].(map[string]interface{})["name"]; got != "2222" {
		t.Errorf("nested = %v", got)
	}
	if vars["registered"] != "{{ keep }}" || vars["cli"] != "{{ keep }}" {
		t.Errorf("runtime and -e values must stay: %v, %v", vars["registered"], vars["cli"])
	}
	if got, ok := vars["global"].([]interface{}); !ok || len(got) != 2 {
		t.Errorf("a whole-expression list must stay a list: %#v", vars["global"])
	}
	if vars["broken"] != "{{ .Names }}" {
		t.Errorf("a value that does not render stays: %v", vars["broken"])
	}
}
