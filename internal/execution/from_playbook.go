package execution

import (
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

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

	index := make(map[string]int) // play + task name -> position in exec.Tasks
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			for _, t := range host.Tasks {
				key := play.Name + "\x00" + t.TaskName
				i, ok := index[key]
				if !ok {
					exec.Tasks = append(exec.Tasks, TaskResult{Name: t.TaskName, HostResults: map[string]HostResult{}, ErrorsByType: map[string][]string{}, StartTime: t.Timestamp})
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
				if t.Duration > task.Duration {
					task.Duration = t.Duration
				}
				task.HostResults[host.Host] = HostResult{Hostname: host.Host, Status: status, Error: t.Error, Timestamp: t.Timestamp}

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
