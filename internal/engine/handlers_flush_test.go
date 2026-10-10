package engine

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

// a flush before the handler is known (meta: flush_handlers inside a role
// whose handlers load later) keeps the notification for the next flush
func TestHandlers_FlushKeepsUnansweredNotifications(t *testing.T) {
	h := &playHandlers{notified: map[string]map[string]bool{}, ended: map[string]bool{}}
	h.notified["later"] = map[string]bool{"h1": true}
	h.notified["now"] = map[string]bool{"h1": true}
	h.entries = []handlerEntry{{task: types.Task{Name: "now", Module: "debug"}}}
	out := h.takeNotified([]types.Host{{Name: "h1"}})
	assert.Len(t, out[0], 1)
	assert.False(t, h.notified["now"]["h1"], "handed out")
	assert.True(t, h.notified["later"]["h1"], "kept for a handler added later")
	h.entries = append(h.entries, handlerEntry{task: types.Task{Name: "later", Module: "debug"}})
	out = h.takeNotified([]types.Host{{Name: "h1"}})
	assert.Empty(t, out[0], "ran already")
	assert.Len(t, out[1], 1, "runs now")
	assert.False(t, h.notified["later"]["h1"])
}
