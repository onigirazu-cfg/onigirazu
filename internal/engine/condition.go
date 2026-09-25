package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
)

// conditionHolds evaluates a when/until/changed_when/failed_when condition
// against the host's variables. A bare expression and one {{ }} block are
// evaluated as expressions; other text with {{ }} is rendered as a template
// and the result read as a boolean.
func (e *ExecutionEngine) conditionHolds(ctx context.Context, condition string, variables map[string]interface{}) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
	}
	if _, whole := expression.Unwrap(condition); whole || !strings.Contains(condition, "{{") {
		return expression.Condition(condition, variables)
	}
	rendered, err := e.templateEngine.Render(ctx, condition, variables)
	if err != nil {
		return false, fmt.Errorf("condition %q: %w", condition, err)
	}
	return expression.Truthy(strings.TrimSpace(rendered)), nil
}
