package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SafeApply are the rollout options of apply: a canary first batch, a soak
// after it, and rolling back an unhealthy batch
type SafeApply struct {
	Canary       string        // first batch size: a number or a percentage
	CanaryPause  time.Duration // soak after a healthy canary, then check again
	AutoRollback bool          // roll back a batch whose tasks failed, in every play
	Scope        string        // "batch" (default) or "run": what a rollback undoes
}

// Restorer undoes the changes of task results (rollback.RollbackExecutor)
type Restorer func(ctx context.Context, results []types.TaskResult) rollback.RestoreReport

// ErrRolledBack ends a run whose unhealthy batch was rolled back
var ErrRolledBack = errors.New("rolled back")

// SetSafeApply sets the rollout options
func (e *ExecutionEngine) SetSafeApply(s SafeApply) {
	e.safe = s
}

// SetRestorer sets how changes are undone when a batch is unhealthy
func (e *ExecutionEngine) SetRestorer(r Restorer) {
	e.restorer = r
}

// RolloutReports are the batches of the run's rollouts
func (e *ExecutionEngine) RolloutReports() []types.BatchReport {
	return e.rolloutReports
}

// rolloutEnabled: the play's batches are checked (health checks, a canary,
// or --auto-rollback)
func (e *ExecutionEngine) rolloutEnabled(play *types.Play) bool {
	return len(play.HealthCheck) > 0 || e.safe.AutoRollback || e.safe.Canary != ""
}

// onUnhealthy is what an unhealthy batch leads to: rollback, stop or continue
func (e *ExecutionEngine) onUnhealthy(play *types.Play) string {
	if play.OnUnhealthy != "" {
		return play.OnUnhealthy
	}
	if len(play.HealthCheck) > 0 || e.safe.AutoRollback {
		return "rollback"
	}
	return "stop"
}

// rolloutBatches are the batch sizes: the play's serial, with --canary as the
// first batch
func (e *ExecutionEngine) rolloutBatches(play *types.Play, total int) ([]int, bool, error) {
	if e.safe.Canary == "" || total == 0 {
		b, err := serialBatches(play.Serial, total)
		return b, false, err
	}
	first, err := batchSize(e.safe.Canary, total)
	if err != nil {
		return nil, false, fmt.Errorf("canary: %w", err)
	}
	if first >= total {
		return []int{total}, true, nil
	}
	rest, err := serialBatches(play.Serial, total-first)
	if err != nil {
		return nil, false, err
	}
	return append([]int{first}, rest...), true, nil
}

// runHealthChecks runs the checks on the hosts still in the run and returns
// the hosts that failed one; checks do not take hosts out of the run
func (e *ExecutionEngine) runHealthChecks(ctx context.Context, checks []types.Task, hosts []types.Host,
	vars map[string]interface{}, result *types.PlayResult) []string {
	bad := map[string]bool{}
	for i := range checks {
		check := checks[i]
		if check.Name == "" {
			check.Name = fmt.Sprintf("health check %d", i+1)
		} else {
			check.Name = "health check: " + check.Name
		}
		var active []types.Host
		for _, h := range e.activeHosts(hosts) {
			if !bad[h.Name] {
				active = append(active, h)
			}
		}
		if len(active) == 0 {
			break
		}
		if err := e.executeTask(ctx, &check, active, vars, result); err != nil {
			var hf *hostsFailedError
			if !errors.As(err, &hf) {
				for _, h := range active {
					bad[h.Name] = true
				}
				continue
			}
			for _, name := range hf.hosts {
				bad[name] = true
			}
		}
	}
	names := make([]string, 0, len(bad))
	for name := range bad {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// executeRollout runs a play batch by batch; after each batch its health is
// checked, and an unhealthy batch is rolled back (or stops the rollout)
func (e *ExecutionEngine) executeRollout(ctx context.Context, play *types.Play, hosts []types.Host,
	batches []int, canary bool) (*types.PlayResult, error) {
	result := &types.PlayResult{Name: play.Name, PlayName: play.Name, StartTime: time.Now(), Success: true}
	finish := func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}
	start := 0
	for i, size := range batches {
		batch := hosts[start : start+size]
		start += size
		label := fmt.Sprintf("batch %d/%d", i+1, len(batches))
		if canary && i == 0 {
			label = fmt.Sprintf("canary batch 1/%d", len(batches))
		}
		e.logger.Info("Play '%s': %s (%d hosts)", play.Name, label, len(batch))

		e.rolloutUnhealthy = nil
		batchResult, err := e.executePlayOn(ctx, play, batch)
		if batchResult != nil {
			result.Hosts = append(result.Hosts, batchResult.Hosts...)
			result.Tasks = append(result.Tasks, batchResult.Tasks...)
		}
		if err != nil && (errors.Is(err, ErrStoppedByUser) || ctx.Err() != nil) {
			result.Success = false
			finish()
			return result, err
		}

		report := types.BatchReport{Play: play.Name, Batch: i + 1, Batches: len(batches),
			Canary: canary && i == 0, Healthy: true}
		for _, h := range batch {
			report.Hosts = append(report.Hosts, h.Name)
		}
		unhealthy := e.batchUnhealthy(batch)
		reason := ""
		switch {
		case err != nil:
			reason = err.Error()
			unhealthy = report.Hosts
		case len(unhealthy) > 0 && len(e.rolloutUnhealthy) > 0:
			reason = "health checks failed"
		case len(unhealthy) > 0:
			reason = "tasks failed"
		}

		// a healthy canary soaks, then is checked again
		if len(unhealthy) == 0 && report.Canary && e.safe.CanaryPause > 0 && i < len(batches)-1 && len(play.HealthCheck) > 0 {
			e.logger.Info("Canary healthy; checking again in %s", e.safe.CanaryPause)
			if werr := e.wait(ctx, e.safe.CanaryPause); werr != nil {
				result.Success = false
				finish()
				return result, werr
			}
			leave := e.enterPlay(play)
			again := e.runHealthChecks(ctx, play.HealthCheck, batch, e.rolloutVars, result)
			leave()
			if len(again) > 0 {
				unhealthy, reason = again, "health checks failed after the canary pause"
			}
		}

		batchChanges := changesOf(batchResult)
		if len(unhealthy) == 0 {
			e.logger.Info("Play '%s': %s healthy", play.Name, label)
			e.rolloutApplied = append(e.rolloutApplied, batchChanges...)
			e.rolloutReports = append(e.rolloutReports, report)
			continue
		}

		result.Success = false
		report.Healthy = false
		report.UnhealthyHosts = unhealthy
		report.Reason = reason
		action := e.onUnhealthy(play)
		if action == "rollback" && e.config != nil && e.config.GetDryRun() {
			action = "stop" // check mode changed nothing
			reason += "; check mode, nothing to roll back"
		}
		e.logger.Warn("Play '%s': %s unhealthy (%s: %s); %s", play.Name, label, reason, strings.Join(unhealthy, ", "), action)

		switch action {
		case "continue":
			e.rolloutApplied = append(e.rolloutApplied, batchChanges...)
			e.rolloutReports = append(e.rolloutReports, report)
			continue
		case "rollback":
			undo := batchChanges
			if e.safe.Scope == "run" {
				undo = append(append([]types.TaskResult{}, e.rolloutApplied...), batchChanges...)
			}
			e.rollbackBatch(ctx, play, batch, undo, result, &report)
			e.rolloutReports = append(e.rolloutReports, report)
			finish()
			return result, fmt.Errorf("%s unhealthy (%s): %w", label, reason, ErrRolledBack)
		default: // stop
			e.rolloutReports = append(e.rolloutReports, report)
			finish()
			return result, fmt.Errorf("%s unhealthy (%s): rollout stopped", label, reason)
		}
	}
	finish()
	return result, nil
}

// batchUnhealthy are the batch's hosts that failed a task or a health check
func (e *ExecutionEngine) batchUnhealthy(batch []types.Host) []string {
	bad := map[string]bool{}
	for _, name := range e.rolloutUnhealthy {
		bad[name] = true
	}
	e.mutex.RLock()
	for _, h := range batch {
		if e.failedHosts[h.Name] {
			bad[h.Name] = true
		}
	}
	e.mutex.RUnlock()
	names := make([]string, 0, len(bad))
	for name := range bad {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// changesOf are the changed task results of a batch, in run order
func changesOf(r *types.PlayResult) []types.TaskResult {
	if r == nil {
		return nil
	}
	var out []types.TaskResult
	for _, h := range r.Hosts {
		for _, t := range h.Tasks {
			if t.Changed && !t.Failed {
				if t.Host == "" {
					t.Host = h.Host
				}
				out = append(out, t)
			}
		}
	}
	return out
}

// rollbackBatch undoes the changes, marks them rolled back in the result,
// and checks the batch's health again
func (e *ExecutionEngine) rollbackBatch(ctx context.Context, play *types.Play, batch []types.Host,
	undo []types.TaskResult, result *types.PlayResult, report *types.BatchReport) {
	if e.restorer == nil {
		report.RollbackErrors = []string{"no rollback available in this run"}
		return
	}
	e.logger.Warn("Rolling back %d change(s)", len(undo))
	// a new context: a canceled run still puts the batch back
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Minute)
	defer cancel()
	rep := e.restorer(rctx, undo)
	report.RolledBack = true
	report.Undone = rep.Undone
	report.Irreversible = rep.Irreversible
	report.RollbackErrors = rep.Failed
	e.markRolledBack(result, undo)
	if len(rep.Irreversible) > 0 {
		e.logger.Warn("Kept (no previous state): %s", strings.Join(rep.Irreversible, "; "))
	}
	for _, f := range rep.Failed {
		e.logger.Error("Rollback failed: %s", f)
	}
	e.logger.Warn("Rolled back %d change(s)", rep.Undone)

	if len(play.HealthCheck) > 0 {
		// the failed hosts are checked again too: they are the ones rolled back
		e.mutex.Lock()
		failed := e.failedHosts
		e.failedHosts = nil
		e.mutex.Unlock()
		leave := e.enterPlay(play)
		bad := e.runHealthChecks(rctx, play.HealthCheck, batch, e.rolloutVars, result)
		leave()
		e.mutex.Lock()
		e.failedHosts = failed
		e.mutex.Unlock()
		ok := len(bad) == 0
		report.HealthyAfterRollback = &ok
		if ok {
			e.logger.Info("Health checks pass after the rollback")
		} else {
			e.logger.Error("Health checks still fail after the rollback: %s", strings.Join(bad, ", "))
		}
	}
}

// markRolledBack flags the undone changes in the play result, so the run
// snapshot does not undo them again
func (e *ExecutionEngine) markRolledBack(result *types.PlayResult, undo []types.TaskResult) {
	type key struct{ host, task string }
	count := map[key]int{}
	for _, t := range undo {
		count[key{t.Host, t.TaskName}]++
	}
	for hi := range result.Hosts {
		h := &result.Hosts[hi]
		for ti := range h.Tasks {
			t := &h.Tasks[ti]
			k := key{h.Host, t.TaskName}
			if t.Changed && !t.Failed && count[k] > 0 {
				t.RolledBack = true
				count[k]--
			}
		}
	}
}

// wait sleeps for d, or until the run is stopped or canceled
func (e *ExecutionEngine) wait(ctx context.Context, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if e.control != nil && e.control.Stopped() {
			return ErrStoppedByUser
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return nil
}
