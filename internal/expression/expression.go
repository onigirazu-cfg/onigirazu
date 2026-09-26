// Package expression evaluates the Jinja-style expressions of playbooks:
// conditions (x == "y" and r.rc == 0, x is defined, a in [..]), loop sources
// and {{ }} blocks in templates.
package expression

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
)

var (
	// one {{ }} block around a whole expression
	wholeTemplate = regexp.MustCompile(`^\{\{\s*((?:[^{}]|\{[^{]|\}[^}])*?)\s*\}\}$`)
	// subject of an "is [not] defined" test: a name with .attr or [key] parts
	definedTest = regexp.MustCompile(`([A-Za-z_]\w*(?:\.[A-Za-z_]\w*|\[[^\]]+\])*)\s+is\s+(not\s+defined|undefined|defined)\b`)
	// Jinja filters without arguments that map onto expr builtins
	bareFilter = regexp.MustCompile(`\|\s*(` + strings.Join(filterNames, "|") + `)\b(\s*\()?`)
	// map(attribute='x') has a keyword argument, which expr does not
	mapAttribute = regexp.MustCompile(`\bmap\(\s*attribute\s*=\s*`)
	// d.keys() and d.values() are Python methods
	dictMethod = regexp.MustCompile(`\.(keys|values)\(\)`)
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
	if cached, ok := programs.Load(expression); ok {
		if program, ok := cached.(*vm.Program); ok {
			return program, nil
		}
	}
	options := append([]expr.Option{
		expr.Env(map[string]interface{}{}),
		expr.AllowUndefinedVariables(),
		expr.Patch(inPatch{}),
		expr.Function("jinja_in", func(params ...interface{}) (interface{}, error) {
			return contains(params[1], params[0]), nil
		}),
		expr.Function("dict2items", func(params ...interface{}) (interface{}, error) {
			return Dict2Items(params[0])
		}),
		expr.Function("items2dict", func(params ...interface{}) (interface{}, error) {
			return items2dict(params[0])
		}),
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
	}, filterFunctions()...)
	program, err := expr.Compile(translate(expression), options...)
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
	// inline if has the lowest precedence, then ~ (string concatenation)
	if value, cond, otherwise, ok := inlineIf(condition); ok {
		return "bool(" + translate(cond) + ") ? (" + translate(value) + ") : (" + translate(otherwise) + ")"
	}
	if parts := splitTop(condition, "~"); len(parts) > 1 {
		for i, part := range parts {
			parts[i] = translate(part)
		}
		return "jinja_concat(" + strings.Join(parts, ", ") + ")"
	}
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
	code = mapAttribute.ReplaceAllString(code, "map_attribute(")
	code = dictMethod.ReplaceAllString(code, " | $1()")
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

// inPatch sends every "in" through jinja_in, which also tests substrings
type inPatch struct{}

func (inPatch) Visit(node *ast.Node) {
	if b, ok := (*node).(*ast.BinaryNode); ok && b.Operator == "in" {
		ast.Patch(node, &ast.CallNode{
			Callee:    &ast.IdentifierNode{Value: "jinja_in"},
			Arguments: []ast.Node{b.Left, b.Right},
		})
	}
}

// contains is Jinja's "needle in haystack": substring, list item or map key
func contains(haystack, needle interface{}) bool {
	switch h := haystack.(type) {
	case nil:
		return false
	case string:
		n, ok := needle.(string)
		return ok && strings.Contains(h, n)
	case map[string]interface{}:
		_, ok := h[fmt.Sprint(needle)]
		return ok
	}
	rv := reflect.ValueOf(haystack)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if equal(rv.Index(i).Interface(), needle) {
				return true
			}
		}
	case reflect.Map:
		return rv.MapIndex(reflect.ValueOf(needle)).IsValid()
	}
	return false
}

// equal compares loosely like expr's ==: numbers by value
func equal(a, b interface{}) bool {
	if fa, ok := number(a); ok {
		fb, ok := number(b)
		return ok && fa == fb
	}
	return reflect.DeepEqual(a, b)
}

func number(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// Dict2Items turns a map into a list of {key, value} maps sorted by key
func Dict2Items(value interface{}) ([]interface{}, error) {
	pairs, err := dictPairs(value)
	items := make([]interface{}, len(pairs))
	for i, pair := range pairs {
		items[i] = pair
	}
	return items, err
}

func dictPairs(value interface{}) ([]map[string]interface{}, error) {
	if value == nil {
		return nil, nil
	}
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dict2items needs a dictionary, got %T", value)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, map[string]interface{}{"key": k, "value": m[k]})
	}
	return pairs, nil
}

func items2dict(value interface{}) (map[string]interface{}, error) {
	items, err := Items(value)
	if err != nil {
		return nil, fmt.Errorf("items2dict: %w", err)
	}
	out := make(map[string]interface{}, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("items2dict needs a list of {key, value}, got %T", item)
		}
		out[fmt.Sprint(m["key"])] = m["value"]
	}
	return out, nil
}

// inlineIf splits Jinja's "a if cond else b"; without else, b is ""
func inlineIf(code string) (string, string, string, bool) {
	parts := splitTop(code, " if ")
	if len(parts) != 2 {
		return "", "", "", false
	}
	cond, otherwise := parts[1], `""`
	if rest := splitTop(parts[1], " else "); len(rest) == 2 {
		cond, otherwise = rest[0], rest[1]
	}
	return parts[0], cond, otherwise, true
}

// splitTop splits code at sep outside quotes and brackets
func splitTop(code, sep string) []string {
	var parts []string
	depth, start := 0, 0
	var quote byte
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case depth == 0 && strings.HasPrefix(code[i:], sep):
			parts = append(parts, code[start:i])
			i += len(sep) - 1
			start = i + 1
		}
	}
	return append(parts, code[start:])
}
