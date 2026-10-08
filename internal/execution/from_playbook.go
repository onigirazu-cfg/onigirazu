package execution

import (
	"fmt"
	"regexp"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// loopItemSuffix is what the engine appends to the name of a loop item; the
// report has one entry for the whole loop
var loopItemSuffix = regexp.MustCompile(` \(item \d+\)$`)

// FromPlaybookResult builds the cached record of a run: tasks in run order
// with a result per host, the number of distinct hosts and the overall status
func FromPlaybookResult(result *types.PlaybookResult, playbookPath, playbookName string,
	start time.Time, duration time.Duration) *ExecutionResult {
	exec := &ExecutionResult{
		ExecutionID:    fmt.Sprintf("exec-%d", start.UnixNano()),
		Timestamp:      start,
		PlaybookPath:   playbookPath,
		PlaybookName:   playbookName,
		StartTime:      start,
		EndTime:        start.Add(duration),
		Duration:       duration,
		HostResults:    make(map[string]*HostResult),
		PlaybookResult: result,
	}

	// Tasks are matched across hosts by their key; without one, by their
	// position in the host's run. Names repeat (unnamed tasks share one).
	index := make(map[string]int) // play + task -> position in exec.Tasks
	for p, play := range result.Plays {
		for _, host := range play.Hosts {
			for n, t := range host.Tasks {
				key := fmt.Sprintf("%d\x00%d", p, n)
				if t.TaskKey != "" {
					key = fmt.Sprintf("%d\x00%s", p, t.TaskKey)
				}
				i, ok := index[key]
				if !ok {
					exec.Tasks = append(exec.Tasks, TaskResult{Name: loopItemSuffix.ReplaceAllString(t.TaskName, ""), HostResults: map[string]HostResult{}, ErrorsByType: map[string][]string{}, StartTime: t.Timestamp})
					i = len(exec.Tasks) - 1
					index[key] = i
				}
				task := &exec.Tasks[i]
				status := hostStatus(t)
				task.Total++
				switch status {
				case "failed":
					task.Failed++
					task.ErrorsByType["error"] = append(task.ErrorsByType["error"], host.Host)
				case "skipped":
					task.Skipped++
				case "ignored":
					task.Success++
				case "changed":
					task.Changed++
					task.Success++
				default:
					task.Success++
				}
				// the task's wall time: from its first start to its last end
				// over hosts and loop items
				if !t.Timestamp.IsZero() && (task.StartTime.IsZero() || t.Timestamp.Before(task.StartTime)) {
					task.StartTime = t.Timestamp
				}
				if end := t.Timestamp.Add(t.Duration); end.After(task.EndTime) {
					task.EndTime = end
				}
				if !task.StartTime.IsZero() && task.EndTime.After(task.StartTime) {
					task.Duration = task.EndTime.Sub(task.StartTime)
				}
				task.HostResults[host.Host] = HostResult{Hostname: host.Host, Status: status, Error: t.Error,
					Output: logger.TaskMessage(t), Timestamp: t.Timestamp}

				switch {
				case status == "skipped":
					exec.TotalSkipped++
				case status == "ignored":
					exec.TotalIgnored++
				case status == "failed":
					exec.TotalFailed++
				default:
					exec.TotalSuccess++
					if status == "changed" {
						exec.TotalChanged++
					}
				}
			}
			hr := exec.HostResults[host.Host]
			if hr == nil {
				hr = &HostResult{Hostname: host.Host, Status: "success", Timestamp: start}
				exec.HostResults[host.Host] = hr
			}
			if host.Failed {
				hr.Status = "failed"
			}
		}
	}
	exec.TotalHosts = len(exec.HostResults)

	failedHosts := 0
	for _, hr := range exec.HostResults {
		if hr.Status == "failed" {
			failedHosts++
		}
	}
	switch {
	case result.Failed && failedHosts == 0:
		exec.Status = "failed" // the run failed outside any host task (a play error)
	case failedHosts == 0:
		exec.Status = "success"
	case failedHosts == len(exec.HostResults):
		exec.Status = "failed"
	default:
		exec.Status = "partial_success"
	}
	return exec
}

func hostStatus(t types.TaskResult) string {
	switch {
	case t.Failed && t.Ignored:
		return "ignored"
	case t.Failed:
		return "failed"
	case t.Skipped:
		return "skipped"
	case t.Changed:
		return "changed"
	}
	return "success"
}
