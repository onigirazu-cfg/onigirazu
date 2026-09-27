package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockOptions(t *testing.T) {
	e := NewEngine()
	tmpl := "a\n  {% if x %}\nb\n  {% endif %}\nc\n"
	vars := map[string]interface{}{"x": true}

	out, err := e.Render(context.Background(), tmpl, vars)
	require.NoError(t, err)
	assert.Equal(t, "a\n  b\n  c\n", out) // trim_blocks only (default)

	out, err = e.Render(WithBlockOptions(context.Background(), true, true), tmpl, vars)
	require.NoError(t, err)
	assert.Equal(t, "a\nb\nc\n", out)

	out, err = e.Render(WithBlockOptions(context.Background(), false, false), tmpl, vars)
	require.NoError(t, err)
	assert.Equal(t, "a\n  \nb\n  \nc\n", out)
}

func TestBlockOptionsLoop(t *testing.T) {
	e := NewEngine()
	tmpl := "list:\n  {% for i in items %}\n- {{ i }}\n  {% endfor %}\nend\n"
	vars := map[string]interface{}{"items": []interface{}{"a", "b"}}
	out, err := e.Render(WithBlockOptions(context.Background(), true, true), tmpl, vars)
	require.NoError(t, err)
	assert.Equal(t, "list:\n- a\n- b\nend\n", out)
}
