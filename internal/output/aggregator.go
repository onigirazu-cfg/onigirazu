package output

import (
	"sort"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ResultStatus represents result status
type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusFailed  ResultStatus = "failed"
	StatusChanged ResultStatus = "changed"
	StatusSkipped ResultStatus = "skipped"
)

// AggregatedResult represents a grouped execution result
type AggregatedResult struct {
	Status     ResultStatus
	Hosts      []AggregatedHost
	Count      int
	Percentage float64
}

// AggregatedHost represents a single host result with metadata
type AggregatedHost struct {
	Name         string
	Status       ResultStatus
	Duration     time.Duration
	Changed      bool
	ErrorMessage string
	Suggestions  []string
	Details      map[string]interface{}
}

// ResultAggregator aggregates and analyzes execution results
type ResultAggregator struct {
	results []AggregatedHost
	total   int
}

// NewResultAggregator creates a new aggregator
func NewResultAggregator() *ResultAggregator {
	return &ResultAggregator{
		results: make([]AggregatedHost, 0),
	}
}

// Add adds a result to the aggregator
func (ra *ResultAggregator) Add(host AggregatedHost) {
	ra.results = append(ra.results, host)
	ra.total++
}

// Aggregate groups results by status
func (ra *ResultAggregator) Aggregate() []AggregatedResult {
	groups := make(map[ResultStatus][]*AggregatedHost)

	// Group by status
	for i := range ra.results {
		status := ra.results[i].Status
		groups[status] = append(groups[status], &ra.results[i])
	}

	// Sort each group by hostname
	for status := range groups {
		sort.Slice(groups[status], func(i, j int) bool {
			return groups[status][i].Name < groups[status][j].Name
		})
	}

	// Build results array with order: Success, Changed, Failed, Skipped
	var aggregated []AggregatedResult
	statusOrder := []ResultStatus{StatusSuccess, StatusChanged, StatusFailed, StatusSkipped}

	for _, status := range statusOrder {
		hosts, exists := groups[status]
		if !exists || len(hosts) == 0 {
			continue
		}

		aggregated = append(aggregated, AggregatedResult{
			Status:     status,
			Hosts:      convertToAggregatedHosts(hosts),
			Count:      len(hosts),
			Percentage: float64(len(hosts)) / float64(ra.total) * 100,
		})
	}

	return aggregated
}

// convertToAggregatedHosts converts pointers to values
func convertToAggregatedHosts(hosts []*AggregatedHost) []AggregatedHost {
	result := make([]AggregatedHost, len(hosts))
	for i, h := range hosts {
		result[i] = *h
	}
	return result
}

// GetMetrics returns execution metrics
func (ra *ResultAggregator) GetMetrics() ExecutionMetrics {
	var totalDuration time.Duration
	var successCount, failedCount, changedCount int
	var fastestDuration, slowestDuration time.Duration

	for i, result := range ra.results {
		totalDuration += result.Duration

		if i == 0 {
			fastestDuration = result.Duration
			slowestDuration = result.Duration
		} else {
			if result.Duration < fastestDuration {
				fastestDuration = result.Duration
			}
			if result.Duration > slowestDuration {
				slowestDuration = result.Duration
			}
		}

		switch result.Status {
		case StatusSuccess:
			successCount++
		case StatusFailed:
			failedCount++
		case StatusChanged:
			changedCount++
		}
	}

	avgDuration := time.Duration(0)
	if ra.total > 0 {
		avgDuration = totalDuration / time.Duration(ra.total)
	}

	return ExecutionMetrics{
		Total:           ra.total,
		SuccessCount:    successCount,
		FailedCount:     failedCount,
		ChangedCount:    changedCount,
		TotalDuration:   totalDuration,
		AverageDuration: avgDuration,
		FastestDuration: fastestDuration,
		SlowestDuration: slowestDuration,
	}
}

// ExecutionMetrics holds execution statistics
type ExecutionMetrics struct {
	Total           int
	SuccessCount    int
	FailedCount     int
	ChangedCount    int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	FastestDuration time.Duration
	SlowestDuration time.Duration
}

// Filter filters results by status
func (ra *ResultAggregator) Filter(status ResultStatus) *ResultAggregator {
	filtered := NewResultAggregator()
	for _, result := range ra.results {
		if result.Status == status {
			filtered.Add(result)
		}
	}
	return filtered
}

// Sort sorts results by a field
func (ra *ResultAggregator) Sort(by string) {
	switch by {
	case "name":
		sort.Slice(ra.results, func(i, j int) bool {
			return ra.results[i].Name < ra.results[j].Name
		})
	case "duration":
		sort.Slice(ra.results, func(i, j int) bool {
			return ra.results[i].Duration > ra.results[j].Duration
		})
	case "status":
		sort.Slice(ra.results, func(i, j int) bool {
			return ra.results[i].Status < ra.results[j].Status
		})
	}
}

// FromPlayResult converts types.PlayResult into aggregated hosts
func FromPlayResult(pr types.PlayResult) *ResultAggregator {
	agg := NewResultAggregator()

	// Process each host result
	for _, hr := range pr.Hosts {
		host := AggregatedHost{
			Name:    hr.Host,
			Details: make(map[string]interface{}),
		}

		// Aggregate task statistics
		var totalDuration time.Duration
		var changedTasks int
		var errorMsg string

		for _, task := range hr.Tasks {
			totalDuration += task.Duration
			if task.Changed {
				changedTasks++
			}
			if task.Failed && errorMsg == "" {
				errorMsg = task.Error
			}
		}

		host.Duration = totalDuration

		// Determine status
		if hr.Failed {
			host.Status = StatusFailed
			host.ErrorMessage = errorMsg
		} else if changedTasks > 0 {
			host.Status = StatusChanged
			host.Changed = true
		} else {
			host.Status = StatusSuccess
		}

		host.Details["tasks_count"] = len(hr.Tasks)
		host.Details["changed_tasks"] = changedTasks

		agg.Add(host)
	}
	return agg
}

// FromTaskResults converts a slice of types.TaskResult into aggregated hosts
// Groups tasks by host
func FromTaskResults(tasks []types.TaskResult) *ResultAggregator {
	agg := NewResultAggregator()

	// Group tasks by host
	hostTasks := make(map[string][]types.TaskResult)
	for _, task := range tasks {
		hostTasks[task.Host] = append(hostTasks[task.Host], task)
	}

	// Process each host
	for hostname, hostTaskList := range hostTasks {
		host := AggregatedHost{
			Name:    hostname,
			Details: make(map[string]interface{}),
		}

		var totalDuration time.Duration
		var changedCount, failedCount int
		var lastError string

		for _, task := range hostTaskList {
			totalDuration += task.Duration
			if task.Changed {
				changedCount++
			}
			if task.Failed {
				failedCount++
				lastError = task.Error
			}
		}

		host.Duration = totalDuration

		// Determine status
		if failedCount > 0 {
			host.Status = StatusFailed
			host.ErrorMessage = lastError
		} else if changedCount > 0 {
			host.Status = StatusChanged
			host.Changed = true
		} else {
			host.Status = StatusSuccess
		}

		host.Details["task_count"] = len(hostTaskList)
		host.Details["changed_count"] = changedCount
		host.Details["failed_count"] = failedCount

		agg.Add(host)
	}
	return agg
}
