package template

import (
	"context"
	"testing"
)

func TestRawBlocks(t *testing.T) {
	e := NewEngine()
	vars := map[string]interface{}{"name": "web"}
	for in, want := range map[string]string{
		"{% raw %}{{x}}{% endraw %}":                                 "{{x}}",
		"{{ name }}: {% raw %}{{ name }} {% if %}{% endraw %}":       "web: {{ name }} {% if %}",
		"a {%- raw -%} {{y}} {%- endraw %} b":                        "a{{y}} b",
		"{% raw %}1{% endraw %}{{ name }}{% raw %}{{2}}{% endraw %}": "1web{{2}}",
	} {
		got, err := e.Render(context.Background(), in, vars)
		if err != nil || got != want {
			t.Errorf("Render(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
