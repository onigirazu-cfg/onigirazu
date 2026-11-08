package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Keep rendering failures local to the host/task evaluation, including the
// result conditions evaluated after the module has run.
type unrenderedKey struct{}

// vars must be a per-call copy: rendering and removing failed keys mutate it.
func (e *ExecutionEngine) lazyEvaluationContext(ctx context.Context, host string, vars map[string]interface{}) context.Context {
	bad := e.renderLazyVars(ctx, host, vars)
	for name := range bad {
		delete(vars, name)
	}
	if len(bad) == 0 && ctx.Value(unrenderedKey{}) == nil {
		return ctx
	}
	return context.WithValue(ctx, unrenderedKey{}, bad)
}

var safeUnrenderedUse = regexp.MustCompile(`^\s*(?:is\s+(?:not\s+)?(?:defined|undefined)\b|\|\s*(?:ansible\.(?:builtin|legacy)\.)?(?:default|d)\s*\()`)
var evaluationTemplate = regexp.MustCompile(`(?s)\{\{(.*?)\}\}|\{%(.*?)%\}`)

func identifierStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// checkUnrenderedExpression finds the original render error after evaluation
// fails. Quoted text, attribute/filter/test names and safe probes are ignored.
// A present value overrides a failed variable from an earlier context.
func checkUnrenderedExpression(ctx context.Context, code string, vars map[string]interface{}) error {
	bad, _ := ctx.Value(unrenderedKey{}).(map[string]unrenderedVar)
	if len(bad) == 0 {
		return nil
	}
	previous := ""
	for i := 0; i < len(code); {
		c := code[i]
		if c == '\'' || c == '"' {
			i++
			for i < len(code) && code[i] != c {
				if code[i] == '\\' {
					i++
				}
				i++
			}
			if i < len(code) {
				i++
			}
			previous = "quoted"
			continue
		}
		if identifierStart(c) {
			start := i
			for i < len(code) && (identifierStart(code[i]) || code[i] >= '0' && code[i] <= '9') {
				i++
			}
			name := code[start:i]
			_, present := vars[name]
			if u, ok := bad[name]; ok && !present && previous != "." && previous != "|" && previous != "is" && !safeUnrenderedUse.MatchString(code[i:]) {
				return fmt.Errorf("variable %q: %w", name, u.err)
			}
			previous = name
			continue
		}
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			previous = string(c)
		}
		i++
	}
	return nil
}

// Only template blocks in rendered fields contain expressions; literal text
// such as a path or an environment value may happen to match a variable name.
func checkUnrenderedTemplates(ctx context.Context, value interface{}, vars map[string]interface{}) error {
	switch v := value.(type) {
	case string:
		for _, m := range evaluationTemplate.FindAllStringSubmatch(v, -1) {
			code := m[1]
			if strings.HasPrefix(m[0], "{%") {
				code = m[2]
			}
			if err := checkUnrenderedExpression(ctx, code, vars); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for _, x := range v {
			if err := checkUnrenderedTemplates(ctx, x, vars); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, x := range v {
			if err := checkUnrenderedTemplates(ctx, x, vars); err != nil {
				return err
			}
		}
	}
	return nil
}

// Enrich only failed evaluations: undefined/default tests and short-circuiting
// must be handled by the evaluator, without inspecting unevaluated branches.
func enrichExpressionError(ctx context.Context, code string, vars map[string]interface{}, err error) error {
	if err != nil {
		if renderErr := checkUnrenderedExpression(ctx, code, vars); renderErr != nil {
			return renderErr
		}
	}
	return err
}

func enrichTemplateError(ctx context.Context, value interface{}, vars map[string]interface{}, err error) error {
	if err != nil {
		if renderErr := checkUnrenderedTemplates(ctx, value, vars); renderErr != nil {
			return renderErr
		}
	}
	return err
}

func (e *ExecutionEngine) renderEvaluation(ctx context.Context, text string, vars map[string]interface{}) (string, error) {
	value, err := e.templateEngine.Render(ctx, text, vars)
	return value, enrichTemplateError(ctx, text, vars, err)
}

func (e *ExecutionEngine) renderEvaluationArgs(ctx context.Context, args map[string]interface{}, vars map[string]interface{}) (map[string]interface{}, error) {
	value, err := e.templateEngine.RenderTaskArgs(ctx, args, vars)
	return value, enrichTemplateError(ctx, args, vars, err)
}
