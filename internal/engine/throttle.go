package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// slots limit how many hosts run a task at once (throttle); nil: no limit
type slots chan struct{}

// acquire takes a slot and returns its release
func (s slots) acquire() func() {
	if s == nil {
		return func() {}
	}
	s <- struct{}{}
	return func() { <-s }
}

// throttleSlots reads the task's throttle (a number or a template of one;
// 0 or empty: no limit)
func (e *ExecutionEngine) throttleSlots(ctx context.Context, task *types.Task, variables map[string]interface{}) (slots, error) {
	value := strings.TrimSpace(task.Throttle)
	if value == "" {
		value = strings.TrimSpace(e.playThrottle) // the play's, when the task has none
	}
	if value == "" {
		return nil, nil
	}
	if strings.Contains(value, "{{") {
		rendered, err := e.templateEngine.Render(ctx, value, variables)
		if err != nil {
			return nil, fmt.Errorf("throttle: %w", err)
		}
		value = strings.TrimSpace(rendered)
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("throttle: expected a number, got %q", value)
	}
	if n == 0 {
		return nil, nil
	}
	return make(slots, n), nil
}

// playStrategyFree reads a play's strategy: linear (the default) or free
func playStrategyFree(strategy string) (bool, error) {
	switch strings.TrimSpace(strategy) {
	case "", "linear":
		return false, nil
	case "free":
		return true, nil
	}
	return false, fmt.Errorf("strategy: expected linear or free, got %q", strategy)
}
