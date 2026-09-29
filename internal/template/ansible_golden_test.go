package template

import (
	"context"
	"os"
	"testing"
)

// Output of ansible-core 2.21 for testdata/ansible-conf.j2 (compat/cases/template.yml)
func TestTemplateMatchesAnsible(t *testing.T) {
	src, err := os.ReadFile("testdata/ansible-conf.j2")
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]interface{}{
		"ansible_managed": "Ansible managed",
		"enabled":         true,
		"users": []interface{}{
			map[string]interface{}{"name": "ann", "uid": 1, "admin": true},
			map[string]interface{}{"name": "bob", "uid": 2, "admin": false},
		},
	}
	for _, c := range []struct {
		golden       string
		trim, lstrip bool
	}{
		{"testdata/ansible-conf.trim.out", true, false},
		{"testdata/ansible-conf.notrim-lstrip.out", false, true},
	} {
		want, err := os.ReadFile(c.golden)
		if err != nil {
			t.Fatal(err)
		}
		got, err := NewEngine().Render(WithBlockOptions(context.Background(), c.trim, c.lstrip), string(src), vars)
		if err != nil {
			t.Fatalf("%s: %v", c.golden, err)
		}
		if got != string(want) {
			t.Errorf("%s:\n got %q\nwant %q", c.golden, got, want)
		}
	}
}
