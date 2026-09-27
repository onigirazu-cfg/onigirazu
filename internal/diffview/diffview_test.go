package diffview

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestTaskDiff(t *testing.T) {
	task := types.TaskResult{Output: map[string]interface{}{"diff": []interface{}{
		map[string]interface{}{"before_header": "/etc/app", "before": "a\nb\n", "after": "a\nc\n"},
	}}}
	assert.Contains(t, TaskDiff(task), "-b\n+c\n")
	assert.Equal(t, "", TaskDiff(types.TaskResult{}))
	assert.Equal(t, []string{"a\n", "b" + "\n\\ No newline at end of file\n"}, diffLines("a\nb"))
}
