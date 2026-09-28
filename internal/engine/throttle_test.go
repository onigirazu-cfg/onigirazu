package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestThrottleSlots(t *testing.T) {
	e := &ExecutionEngine{templateEngine: template.NewEngine()}
	vars := map[string]interface{}{"par": 3}
	for value, want := range map[string]int{"": 0, "0": 0, "2": 2, "{{ par }}": 3} {
		s, err := e.throttleSlots(context.Background(), &types.Task{Throttle: value}, vars)
		require.NoError(t, err, value)
		assert.Equal(t, want, cap(s), value)
	}
	_, err := e.throttleSlots(context.Background(), &types.Task{Throttle: "many"}, vars)
	assert.Error(t, err)

	var none slots
	none.acquire()() // no limit: nothing blocks
}

func TestThrottleParsed(t *testing.T) {
	var task types.Task
	require.NoError(t, yaml.Unmarshal([]byte("name: x\nansible.builtin.command: 'true'\nthrottle: '{{ n }}'\ndelegate_to: localhost\n"), &task))
	assert.Equal(t, "{{ n }}", task.Throttle)
	assert.Equal(t, "command", task.Module)
}
