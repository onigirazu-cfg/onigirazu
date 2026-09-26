package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestPrintDiffs(t *testing.T) {
	task := types.TaskResult{TaskName: "conf", Output: map[string]interface{}{"diff": []interface{}{
		map[string]interface{}{"before_header": "/etc/app", "before": "a\nb\n", "after": "a\nc\n"},
	}}}
	result := &types.PlaybookResult{Plays: []types.PlayResult{{Hosts: []types.HostResult{{Host: "web1", Tasks: []types.TaskResult{task, {TaskName: "none"}}}}}}}
	var out bytes.Buffer
	printDiffs(&out, result)
	assert.Contains(t, out.String(), "--- web1: conf")
	assert.Contains(t, out.String(), "-b\n+c\n")
	assert.NotContains(t, out.String(), "none")
	assert.Equal(t, []string{"a\n", "b" + "\n\\ No newline at end of file\n"}, diffLines("a\nb"))
}
