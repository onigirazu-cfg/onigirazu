package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type recordingCallback struct {
	*plugins.BaseCallbackPlugin
	mu     sync.Mutex
	events []string
}

func (r *recordingCallback) add(format string, args ...interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, fmt.Sprintf(format, args...))
	return nil
}

func (r *recordingCallback) OnPlaybookStart(ctx context.Context, p *types.Playbook) error {
	return r.add("playbook start")
}

func (r *recordingCallback) OnPlaybookEnd(ctx context.Context, p *types.Playbook, ok bool, d time.Duration) error {
	return r.add("playbook end %t", ok)
}

func (r *recordingCallback) OnPlayStart(ctx context.Context, p *types.Play) error {
	return r.add("play start %s", p.Name)
}

func (r *recordingCallback) OnPlayEnd(ctx context.Context, p *types.Play, ok bool, d time.Duration) error {
	return r.add("play end %s %t", p.Name, ok)
}

func (r *recordingCallback) OnTaskStart(ctx context.Context, t *types.Task, h types.Host) error {
	return r.add("task start %s %s", t.Name, h.Name)
}

func (r *recordingCallback) OnTaskEnd(ctx context.Context, t *types.Task, h types.Host, res types.TaskResult) error {
	_ = r.add("task end %s %s %t", t.Name, h.Name, res.Changed)
	return fmt.Errorf("callback errors are only logged")
}

func TestCallbackPlugins_ReceiveEvents(t *testing.T) {
	engine, mockConfig, _, mockInventory, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	hosts := twoHosts()[:1]
	mockInventory.hosts = hosts
	mockInventory.On("GetHosts", "all").Return(hosts, nil)
	mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{Success: true, Changed: true}, nil)

	cb := &recordingCallback{BaseCallbackPlugin: plugins.NewBaseCallbackPlugin("rec", "1", "")}
	engine.SetCallbacks([]plugins.CallbackPlugin{cb})
	_, err := engine.ExecutePlaybook(context.Background(), &types.Playbook{Plays: []types.Play{{
		Name: "p", Hosts: "all", Tasks: []types.Task{{Name: "t", Module: "command"}},
	}}})
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"playbook start", "play start p", "task start t h1", "task end t h1 true",
		"play end p true", "playbook end true",
	}, cb.events)
}
