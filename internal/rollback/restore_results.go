package rollback

import (
	"context"
	"fmt"
	"sort"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RestoreReport says what a rollback of task results did
type RestoreReport struct {
	Undone       int      // changes put back
	Failed       []string // changes that could not be put back, with the reason
	Irreversible []string // changes kept: no previous state
}

// RestoreResults undoes the changes of task results in the middle of a run
// (a batch that broke the service), newest first, with the operations a run
// snapshot would hold. Unchanged results are skipped.
func (re *RollbackExecutor) RestoreResults(ctx context.Context, results []types.TaskResult) RestoreReport {
	var report RestoreReport
	var resources []ResourceSnapshot
	for i, t := range results {
		if !t.Changed || t.Failed {
			continue
		}
		res, ok := ResourceFromResult(t, t.Host, i+1)
		if !ok || !res.Reversible || res.RollbackOp == nil {
			report.Irreversible = append(report.Irreversible, fmt.Sprintf("%s: %s (%s)", t.Host, t.TaskName, t.Module))
			continue
		}
		resources = append(resources, res)
	}
	sort.SliceStable(resources, func(i, j int) bool {
		return resources[i].RollbackOp.Order > resources[j].RollbackOp.Order
	})
	for i := range resources {
		res := &resources[i]
		if err := re.executeRollbackOperation(ctx, res); err != nil {
			report.Failed = append(report.Failed, fmt.Sprintf("%s: %s: %v", res.Host, res.Identifier, err))
			continue
		}
		report.Undone++
	}
	return report
}
