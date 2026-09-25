package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

// Conditions (when, until) use Ansible syntax: a bare expression such as
// `os == "RedHat" and r.rc == 0`, `x is defined`, `env in ['a', 'b']`,
// `items | length > 0`. A condition that contains {{ }} is rendered as a
// template first and the result read as a boolean.

var (
	// subject of an "is [not] defined" test: a name with .attr or [key] parts
	definedTest = regexp.MustCompile(`([A-Za-z_]\w*(?:\.[A-Za-z_]\w*|\[[^\]]+\])*)\s+is\s+(not\s+defined|undefined|defined)\b`)
	// Jinja filters without arguments that map onto expr builtins
	bareFilter    = regexp.MustCompile(`\|\s*(length|count|lower|upper|int|float|string|trim|bool|first|last)\b(\s*\()?`)
	jinjaWords    = strings.NewReplacer("True", "true", "False", "false", "None", "nil")
	wholeTemplate = regexp.MustCompile(`^\{\{\s*((?:[^{}]|\{[^{]|\}[^}])*?)\s*\}\}$`)
	quoted        = regexp.MustCompile(`"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'`)
)

// conditionHolds evaluates a when/until condition against the host's variables
func (e *ExecutionEngine) conditionHolds(ctx context.Context, condition string, variables map[string]interface{}) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
	}
	// "{{ expr }}" is the same expression in template braces
	if m := wholeTemplate.FindStringSubmatch(condition); m != nil {
		return evalCondition(m[1], variables)
	}
	if strings.Contains(condition, "{{") {
		rendered, err := e.templateEngine.Render(ctx, condition, variables)
		if err != nil {
			return false, fmt.Errorf("condition %q: %w", condition, err)
		}
		return truthy(strings.TrimSpace(rendered)), nil
	}
	return evalCondition(condition, variables)
}

func evalCondition(condition string, variables map[string]interface{}) (bool, error) {
	program, err := expr.Compile(translateCondition(condition),
		expr.Env(map[string]interface{}{}),
		expr.AllowUndefinedVariables(),
		expr.Function("bool", func(params ...interface{}) (interface{}, error) {
			return truthy(params[0]), nil
		}),
	)
	if err != nil {
		return false, fmt.Errorf("condition %q: %w", condition, err)
	}
	env := make(map[string]interface{}, len(variables))
	for k, v := range variables {
		env[k] = v
	}
	out, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("condition %q: %w", condition, err)
	}
	return truthy(out), nil
}

// translateCondition rewrites the Jinja parts of a condition into expr syntax.
// String literals are left alone.
func translateCondition(condition string) string {
	var b strings.Builder
	last := 0
	for _, loc := range quoted.FindAllStringIndex(condition, -1) {
		b.WriteString(translateCode(condition[last:loc[0]]))
		b.WriteString(condition[loc[0]:loc[1]])
		last = loc[1]
	}
	b.WriteString(translateCode(condition[last:]))
	return b.String()
}

func translateCode(code string) string {
	code = definedTest.ReplaceAllStringFunc(code, func(m string) string {
		parts := definedTest.FindStringSubmatch(m)
		// optional chaining: a missing parent is "not defined", not an error
		subject := strings.ReplaceAll(parts[1], ".", "?.")
		if parts[2] == "defined" {
			return "(" + subject + " != nil)"
		}
		return "(" + subject + " == nil)"
	})
	code = bareFilter.ReplaceAllStringFunc(code, func(m string) string {
		parts := bareFilter.FindStringSubmatch(m)
		if parts[2] != "" { // already called with arguments
			return m
		}
		name := map[string]string{"length": "len", "count": "len"}[parts[1]]
		if name == "" {
			name = parts[1]
		}
		return "| " + name + "()"
	})
	return jinjaWordsIn(code)
}

// jinjaWordsIn replaces True/False/None only as whole words
func jinjaWordsIn(code string) string {
	return regexp.MustCompile(`\b(True|False|None)\b`).ReplaceAllStringFunc(code, jinjaWords.Replace)
}

// truthy follows Jinja: empty, zero, false and missing values are false
func truthy(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "", "false", "no", "0", "none", "<no value>":
			return false
		}
		return true
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	case []interface{}:
		return len(t) > 0
	case map[string]interface{}:
		return len(t) > 0
	}
	return true
}
