package engine

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func hostsNamed(names ...string) []types.Host {
	var hosts []types.Host
	for _, n := range names {
		hosts = append(hosts, types.Host{Name: n})
	}
	return hosts
}

func TestContinueAfter(t *testing.T) {
	boom := errors.New("boom")
	failedOn := func(names ...string) error { return &hostsFailedError{hosts: names, err: boom} }
	newEngine := func(p failurePolicy) *ExecutionEngine {
		return &ExecutionEngine{logger: logger.NewEnhanced("error", logger.LogFormat("text"), io.Discard), policy: p}
	}
	ctx := context.Background()
	hosts := hostsNamed("a", "b", "c", "d")

	e := newEngine(failurePolicy{maxFailPct: -1, batchSize: 4})
	result := &types.PlayResult{Success: true}
	assert.NoError(t, e.continueAfter(ctx, failedOn("a"), hosts, result))
	assert.False(t, result.Success)
	assert.Equal(t, hostsNamed("b", "c", "d"), e.activeHosts(hosts))

	// no host left
	assert.Error(t, e.continueAfter(ctx, failedOn("b", "c", "d"), hosts, nil))
	assert.Error(t, newEngine(failurePolicy{maxFailPct: 0, batchSize: 4}).continueAfter(ctx, failedOn("a"), hosts, nil))

	assert.Error(t, newEngine(failurePolicy{anyErrorsFatal: true, maxFailPct: -1, batchSize: 4}).continueAfter(ctx, failedOn("a"), hosts, nil))

	// 1 of 4 is 25%: within 30, over 20
	assert.NoError(t, newEngine(failurePolicy{maxFailPct: 30, batchSize: 4}).continueAfter(ctx, failedOn("a"), hosts, nil))
	assert.Error(t, newEngine(failurePolicy{maxFailPct: 20, batchSize: 4}).continueAfter(ctx, failedOn("a"), hosts, nil))

	// inside a block the failure goes to rescue; errors without hosts stop
	inBlock := context.WithValue(ctx, inBlockKey{}, true)
	assert.Error(t, newEngine(failurePolicy{maxFailPct: -1, batchSize: 4}).continueAfter(inBlock, failedOn("a"), hosts, nil))
	assert.Error(t, newEngine(failurePolicy{maxFailPct: -1, batchSize: 4}).continueAfter(ctx, boom, hosts, nil))
}

func TestTaskTagsInherited(t *testing.T) {
	e := &ExecutionEngine{inheritedTags: []string{"play", "role"}}
	assert.Equal(t, []string{"play", "role", "own"}, e.taskTags(&types.Task{Tags: []string{"own"}}))
	assert.Equal(t, []string{"own"}, (&ExecutionEngine{}).taskTags(&types.Task{Tags: []string{"own"}}))
}
