package engine

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestHandlers_SameNameRunsTheFirstOnly(t *testing.T) {
	h := &playHandlers{notified: map[string]map[string]bool{}, ended: map[string]bool{}}
	h.entries = []handlerEntry{
		{task: types.Task{Name: "dup", Module: "debug"}},
		{task: types.Task{Name: "dup", Module: "debug"}},
		{task: types.Task{Name: "other", Module: "debug", Listen: "topic"}},
		{task: types.Task{Name: "other", Module: "debug", Listen: "topic"}},
	}
	h.notified["dup"] = map[string]bool{"h1": true}
	h.notified["topic"] = map[string]bool{"h1": true}
	out := h.takeNotified([]types.Host{{Name: "h1"}})
	assert.Len(t, out[0], 1, "the first dup runs")
	assert.Empty(t, out[1], "the second dup does not")
	assert.Len(t, out[2], 1, "listeners all run")
	assert.Len(t, out[3], 1, "listeners all run")
}
