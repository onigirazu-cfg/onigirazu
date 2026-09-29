package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/asyncjob"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// asyncOutcome fails a task that ran past its async seconds and marks a
// finished one as Ansible does
func asyncOutcome(result *types.TaskResult, err error, runCtx, ctx context.Context, async int) error {
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
		result.Failed, result.Success = true, false
		result.Error = fmt.Sprintf("async task did not complete within %d seconds", async)
		return errors.New(result.Error)
	}
	if result.Output == nil {
		result.Output = map[string]interface{}{}
	}
	result.Output["finished"] = true
	return err
}

// startAsyncJob starts a poll: 0 task in the background and records the
// launch, whose ansible_job_id async_status takes
func (e *ExecutionEngine) startAsyncJob(ctx context.Context, task, moduleTask *types.Task, host *types.Host,
	target types.Host, taskVars map[string]interface{}, playResult *types.PlayResult) error {
	async := task.Async
	jid := asyncjob.Start(host.Name, func() types.TaskResult {
		runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Duration(async)*time.Second)
		defer cancel()
		result, err := e.moduleRegistry.ExecuteTask(runCtx, moduleTask, target, taskVars)
		err = asyncOutcome(&result, err, runCtx, ctx, async)
		if err != nil && result.Error == "" {
			result.Failed, result.Success, result.Error = true, false, err.Error()
		}
		return result
	})
	return e.finishTask(task, host, types.TaskResult{
		TaskName: task.Name, Host: host.Name, Module: task.Module,
		Success: true, Changed: true, Timestamp: time.Now(),
		Output: map[string]interface{}{
			"ansible_job_id": jid, "started": true, "finished": false,
			"results_file": "~/.ansible_async/" + jid,
		},
	}, playResult)
}
