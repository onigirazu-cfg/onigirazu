package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderedDaemonArgument(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	got, err := e.RenderTaskArgs(context.Background(), map[string]interface{}{"content": `{{ {'userland-proxy': false, 'log-driver': 'json-file', 'log-opts': {'max-size': '20m', 'max-file': '5'}} | combine({'userns-remap': 'default'} if flag else {}) | combine({'runtimes': {}} if other else {}) | to_json }}` + "\n"}, map[string]interface{}{"flag": false, "other": false})
	require.NoError(t, err)
	require.Equal(t, `{"userland-proxy": false, "log-driver": "json-file", "log-opts": {"max-size": "20m", "max-file": "5"}}`+"\n", got["content"])
}
func TestOrderedItemsLoop(t *testing.T) {
	e := NewEngine()
	defer e.Close()
	got, err := e.Render(context.Background(), `{% for k, v in {'z': 1, 'a': 2}.items() %}{{ k }}={{ v }};{% endfor %}`, nil)
	require.NoError(t, err)
	require.Equal(t, "z=1;a=2;", got)
}
