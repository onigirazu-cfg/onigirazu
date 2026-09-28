package template

import (
	"context"
	"reflect"
	"testing"
)

func TestOmitDropsArguments(t *testing.T) {
	e := NewEngine()
	vars := map[string]interface{}{"set": "512m"}
	args := map[string]interface{}{
		"name":    "x",
		"memory":  "{{ unset | default(omit) }}",
		"cpus":    "{{ set | default(omit) }}",
		"options": map[string]interface{}{"a": "{{ omit }}", "b": 1},
		"list":    []interface{}{"{{ unset | default(omit) }}", "y"},
	}
	got, err := e.RenderTaskArgs(context.Background(), args, vars)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"name":    "x",
		"cpus":    "512m",
		"options": map[string]interface{}{"b": 1},
		"list":    []interface{}{"y"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// omit only as a word: an attribute or a longer name stays
	out, err := e.Render(context.Background(), "{{ omit_me }}-{{ d.omit }}", map[string]interface{}{
		"omit_me": "a", "d": map[string]interface{}{"omit": "b"}})
	if err != nil || out != "a-b" {
		t.Errorf("Render = %q, %v", out, err)
	}
}
