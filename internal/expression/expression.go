// Package expression evaluates the Jinja-style expressions of playbooks:
// conditions (x == "y" and r.rc == 0, x is defined, a in [..]), loop sources
// and {{ }} blocks in templates.
package expression

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var (
	// one {{ }} block around a whole expression
	wholeTemplate = regexp.MustCompile(`^\{\{\s*((?:[^{}]|\{[^{]|\}[^}])*?)\s*\}\}$`)
	// subject of an "is [not] defined" test: a name with .attr or [key] parts
	definedTest = regexp.MustCompile(`([A-Za-z_]\w*(?:\.[A-Za-z_]\w*|\[[^\]]+\])*)\s+is\s+(not\s+defined|undefined|defined)\b`)
	// Jinja filters without arguments that map onto expr builtins
	bareFilter = regexp.MustCompile(`\|\s*(length|count|lower|upper|int|float|string|trim|bool|first|last)\b(\s*\()?`)
	jinjaWord  = regexp.MustCompile(`\b(True|False|None)\b`)
	jinjaWords = strings.NewReplacer("True", "true", "False", "false", "None", "nil")
	quoted     = regexp.MustCompile(`"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'`)
)

// Unwrap returns the expression inside a condition that is one {{ }} block
func Unwrap(s string) (string, bool) {
	if m := wholeTemplate.FindStringSubmatch(strings.TrimSpace(s)); m != nil {
		return m[1], true
	}
	return "", false
}

// Condition evaluates an expression as a condition
func Condition(expression string, variables map[string]interface{}) (bool, error) {
	out, err := Eval(expression, variables)
	if err != nil {
		return false, fmt.Errorf("condition %q: %w", expression, err)
	}
	return Truthy(out), nil
}

// Eval evaluates an expression; "{{ }}" around it is optional. Undefined
// names evaluate to nil.
func Eval(expression string, variables map[string]interface{}) (interface{}, error) {
	expression = strings.TrimSpace(expression)
	if inner, ok := Unwrap(expression); ok {
		expression = inner
	}
	program, err := compile(expression)
	if err != nil {
		return nil, err
	}
	env := make(map[string]interface{}, len(variables))
	for k, v := range variables {
		env[k] = v
	}
	return expr.Run(program, env)
}

// programs caches compiled expressions; playbooks repeat the same few
var programs sync.Map // string -> *vm.Program

func compile(expression string) (*vm.Program, error) {
	if p, ok := programs.Load(expression); ok {
		return p.(*vm.Program), nil
	}
	program, err := expr.Compile(translate(expression),
		expr.Env(map[string]interface{}{}),
		expr.AllowUndefinedVariables(),
		expr.Function("bool", func(params ...interface{}) (interface{}, error) {
			return Truthy(params[0]), nil
		}),
		expr.Function("default", func(params ...interface{}) (interface{}, error) {
			if len(params) < 2 {
				return nil, fmt.Errorf("default needs a value")
			}
			if params[0] == nil || params[0] == "" {
				return params[1], nil
			}
			return params[0], nil
		}),
	)
	if err != nil {
		return nil, err
	}
	programs.Store(expression, program)
	return program, nil
}

// Items turns the value of a loop expression into loop items
func Items(value interface{}) ([]interface{}, error) {
	if value == nil {
		return []interface{}{}, nil
	}
	if items, ok := value.([]interface{}); ok {
		return items, nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("loop needs a list, got %T", value)
	}
	items := make([]interface{}, rv.Len())
	for i := range items {
		items[i] = rv.Index(i).Interface()
	}
	return items, nil
}

// translate rewrites the Jinja parts of a condition into expr syntax.
// String literals are left alone.
func translate(condition string) string {
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
	return jinjaWord.ReplaceAllStringFunc(code, jinjaWords.Replace)
}

// Truthy follows Jinja: empty, zero, false and missing values are false
func Truthy(v interface{}) bool {
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
