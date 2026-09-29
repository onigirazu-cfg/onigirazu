package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/asyncjob"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AsyncStatusModule reports on a job started with async and poll: 0
type AsyncStatusModule struct {
	*BaseModule
}

// NewAsyncStatusModule creates the async_status module
func NewAsyncStatusModule() *AsyncStatusModule {
	return &AsyncStatusModule{BaseModule: NewBaseModule("async_status")}
}

func (m *AsyncStatusModule) GetDescription() string {
	return "Report the status of a task started with async and poll: 0"
}

func (m *AsyncStatusModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: time.Now(),
		Output: map[string]interface{}{}}
	jid := getStringArg(args, "jid", "")
	mode := getStringArg(args, "mode", "status")
	result.Output["ansible_job_id"] = jid
	if mode != "status" && mode != "cleanup" {
		result.Error = fmt.Sprintf("mode must be status or cleanup, got %q", mode)
		result.Failed = true
		return result, nil
	}
	if mode == "cleanup" {
		if !asyncjob.Cleanup(host.Name, jid) {
			result.Error = fmt.Sprintf("could not find job %s", jid)
			result.Failed = true
			return result, nil
		}
		result.Output["erased"] = "~/.ansible_async/" + jid
		result.Success = true
		return result, nil
	}
	done, finished, ok := asyncjob.Status(host.Name, jid)
	if !ok {
		result.Error = fmt.Sprintf("could not find job %s", jid)
		result.Failed = true
		return result, nil
	}
	if !finished {
		result.Output["started"] = true
		result.Output["finished"] = false
		result.Success = true
		return result, nil
	}
	for k, v := range done.Output {
		result.Output[k] = v
	}
	result.Output["ansible_job_id"] = jid
	result.Output["started"] = true
	result.Output["finished"] = true
	result.Changed = done.Changed
	result.Success, result.Failed, result.Error = done.Success, done.Failed, done.Error
	return result, nil
}
