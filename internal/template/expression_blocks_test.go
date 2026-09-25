package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_ExpressionBlocks(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	vars := map[string]interface{}{
		"r":     map[string]interface{}{"results": []interface{}{1, 2}, "rc": 0},
		"name":  "web",
		"port":  8080,
		"ratio": 2.5,
		"tags":  []interface{}{"a", "b"},
		"brace": "{{ not a template }}",
	}
	cases := map[string]string{
		"{{ r.results | length }}":                  "2",
		"{{ name | upper }}-{{ port }}":             "WEB-8080",
		`{{ r.rc == 0 }}`:                           "true",
		`{{ missing | default("x") }}`:              "x",
		"{{ ratio }}":                               "2.5",
		"{{ tags }}":                                `["a","b"]`,
		"{{ brace }}":                               "{{ not a template }}",
		"{{ .name }} {{ name }}":                    "web web",
		"{% if port %}on{% endif %} {{ port + 1 }}": "on 8081",
	}
	for in, want := range cases {
		got, err := e.Render(context.Background(), in, vars)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
}

func TestRenderTaskArgs_ListsAndMapsKeepTheirType(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	vars := map[string]interface{}{
		"pkgs": []interface{}{"curl", "git"},
		"env":  map[string]interface{}{"A": "1"},
		"port": 8080,
	}
	out, err := e.RenderTaskArgs(context.Background(), map[string]interface{}{
		"name":  "{{ pkgs }}",
		"env":   "{{ env }}",
		"port":  "{{ port }}",
		"mixed": "pkgs: {{ pkgs }}",
	}, vars)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"curl", "git"}, out["name"])
	assert.Equal(t, map[string]interface{}{"A": "1"}, out["env"])
	assert.Equal(t, "8080", out["port"])
	assert.Equal(t, `pkgs: ["curl","git"]`, out["mixed"])

func TestRender_UndefinedVariableFails(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	vars := map[string]interface{}{"name": "web"}

	for _, tmpl := range []string{"{{ missing }}", "port={{ port }}", "{{ .missing }}", "{{ name }}-{{ nope.x }}"} {
		_, err := e.Render(context.Background(), tmpl, vars)
		assert.Error(t, err, tmpl)
	}
	for tmpl, want := range map[string]string{
		`{{ missing | default("x") }}`:  "x",
		`{{ default .missing "x" }}`:    "x",
		`{% if missing %}a{% endif %}b`: "b",
		"literal <no value> {{ name }}": "literal <no value> web",
	} {
		got, err := e.Render(context.Background(), tmpl, vars)
		require.NoError(t, err, tmpl)
		assert.Equal(t, want, got, tmpl)
	}
}
