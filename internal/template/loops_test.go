package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_ForLoops(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	vars := map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{"name": "a", "ip": "10.0.0.1"},
			map[string]interface{}{"name": "b", "ip": "10.0.0.2"},
		},
		"ports": []interface{}{80, 443},
		"env":   map[string]interface{}{"B": "2", "A": "1"},
		"item":  "global",
	}
	cases := map[string]string{
		"{% for s in servers %}\nserver {{ s.name }} {{ s.ip }}\n{% endfor %}\n":                  "server a 10.0.0.1\nserver b 10.0.0.2\n",
		"{% for p in ports %}{{ p }}{% if not loop.last %},{% endif %}{% endfor %}":               "80,443",
		"{% for k, v in env.items() %}{{ k }}={{ v }};{% endfor %}":                               "A=1;B=2;",
		"{% for s in servers %}{% for p in ports %}{{ s.name }}:{{ p }} {% endfor %}{% endfor %}": "a:80 a:443 b:80 b:443 ",
		"{{ item }} {% for item in ports %}{{ item }}{% endfor %} {{ item }}":                     "global 80443 global",
		"{% for x in missing | default([]) %}x{% endfor %}empty":                                  "empty",
		"{% for s in servers %}{{ loop.index }}/{{ loop.length }} {% endfor %}":                   "1/2 2/2 ",
	}
	for in, want := range cases {
		got, err := e.Render(context.Background(), in, vars)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	_, err := e.Render(context.Background(), "{% for x in ports %}no end", vars)
	assert.Error(t, err)
	_, err = e.Render(context.Background(), "{% for x in 42 %}{% endfor %}", vars)
	assert.Error(t, err)
}

func TestRender_WhitespaceControl(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	vars := map[string]interface{}{"xs": []interface{}{"a", "b"}, "n": 1}
	cases := map[string]string{
		// expected values are what Jinja2 renders with trim_blocks (Ansible)
		"list:\n  {%- for x in xs %}\n  - {{ x }}\n  {%- endfor %}\n":                       "list:  - a  - b",
		"# h\n{% for x in xs %}\n{{ x }}{% if loop.first %} p{% endif %}\n\n{% endfor %}\n": "# h\na p\nb\n",
		"n = {{- n -}} ;":              "n =1;",
		"a {%- if n %} b {%- endif %}": "a b",
	}
	for in, want := range cases {
		got, err := e.Render(context.Background(), in, vars)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
}
