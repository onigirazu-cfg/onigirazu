package template

import (
	"context"
	"testing"
)

func TestBracesInsideBlocks(t *testing.T) {
	e := NewEngine()
	vars := map[string]interface{}{"item": "web", "d": map[string]interface{}{"a": 1}}
	cases := map[string]string{
		"docker inspect --format '{{ '{{' }} .HostConfig.NanoCpus {{ '}}' }}' {{ item }}": "docker inspect --format '{{ .HostConfig.NanoCpus }}' web",
		"{{ {'a': {'b': 1}}['a']['b'] }}":                                                 "1",
		"{{ \"}}\" ~ item }}":                                                             "}}web",
		"x {{ item }} y {{ item | upper }}":                                               "x web y WEB",
	}
	for in, want := range cases {
		got, err := e.Render(context.Background(), in, vars)
		if err != nil || got != want {
			t.Errorf("Render(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
