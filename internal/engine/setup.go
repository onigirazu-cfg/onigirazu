package engine

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// runSetup gathers the facts of a host again (setup, gather_facts): the
// host's variables get them, and the result holds the ones filter selects
func (e *ExecutionEngine) runSetup(ctx context.Context, task *types.Task, host *types.Host,
	args map[string]interface{}, playResult *types.PlayResult) error {
	start := time.Now()
	result := types.TaskResult{TaskName: task.Name, Host: host.Name, Module: task.Module,
		Success: true, Timestamp: start, Output: map[string]interface{}{}}
	factPath, _ := args["fact_path"].(string)
	sf, err := e.factsGatherer.Regather(ctx, *host, factPath)
	if err != nil {
		result.Success, result.Failed = false, true
		result.Error = fmt.Sprintf("failed to gather facts: %v", err)
	} else {
		facts := hostFacts(*host, sf)
		e.setFacts(host.Name, facts)
		result.Output["ansible_facts"] = filterFacts(facts, factFilters(args["filter"]))
	}
	result.Duration = time.Since(start)
	return e.finishTask(task, host, result, playResult)
}

// factFilters reads setup's filter: a pattern, a list of them, or a
// comma-separated string
func factFilters(value interface{}) []string {
	var out []string
	switch v := value.(type) {
	case string:
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	case []interface{}:
		for _, p := range v {
			out = append(out, factFilters(fmt.Sprint(p))...)
		}
	}
	return out
}

// filterFacts keeps the ansible_ facts whose names match a pattern (all of
// them without patterns)
func filterFacts(facts map[string]interface{}, patterns []string) map[string]interface{} {
	out := map[string]interface{}{}
	for name, value := range facts {
		if !strings.HasPrefix(name, "ansible_") || name == "ansible_facts" {
			continue
		}
		if len(patterns) == 0 {
			out[name] = value
			continue
		}
		for _, p := range patterns {
			if ok, _ := path.Match(p, name); ok || p == name {
				out[name] = value
				break
			}
		}
	}
	return out
}
