package expression

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/expr-lang/expr"
	"gopkg.in/yaml.v3"
)

// Ansible/Jinja filters that expr has no builtin for. A filter with
// arguments is called through expr's pipe: `x | f(a)` is f(x, a).

// filterNames are the filters that may be written without parentheses
var filterNames = []string{
	"length", "count", "lower", "upper", "int", "float", "string", "trim", "bool", "first", "last",
	"dict2items", "items2dict", "to_json", "to_nice_json", "to_yaml", "to_nice_yaml", "from_json",
	"from_yaml", "unique", "list", "capitalize", "title", "b64encode", "b64decode", "quote", "basename",
	"dirname", "sort", "sum", "max", "min", "reverse", "flatten", "join", "keys", "values", "abs", "round",
	"select", "reject", "mandatory",
}

func filterFunctions() []expr.Option {
	fn := func(name string, f func(params ...interface{}) (interface{}, error)) expr.Option {
		return expr.Function(name, f)
	}
	str := func(v interface{}) string {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprint(v)
	}
	return []expr.Option{
		expr.DisableBuiltin("map"),
		fn("to_json", func(p ...interface{}) (interface{}, error) {
			out, err := json.Marshal(p[0])
			return string(out), err
		}),
		fn("to_nice_json", func(p ...interface{}) (interface{}, error) {
			out, err := json.MarshalIndent(p[0], "", "    ")
			return string(out), err
		}),
		fn("to_yaml", toYAML),
		fn("to_nice_yaml", toYAML),
		fn("from_json", func(p ...interface{}) (interface{}, error) {
			var out interface{}
			err := json.Unmarshal([]byte(str(p[0])), &out)
			return out, err
		}),
		fn("from_yaml", func(p ...interface{}) (interface{}, error) {
			var out interface{}
			err := yaml.Unmarshal([]byte(str(p[0])), &out)
			return out, err
		}),
		fn("regex_replace", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("regex_replace needs a pattern")
			}
			re, err := regexp.Compile(str(p[1]))
			if err != nil {
				return nil, err
			}
			repl := ""
			if len(p) > 2 {
				// Python's \1 back references are $1 in Go
				repl = regexp.MustCompile(`\\(\d+)`).ReplaceAllString(str(p[2]), "$${$1}")
			}
			return re.ReplaceAllString(str(p[0]), repl), nil
		}),
		fn("regex_search", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("regex_search needs a pattern")
			}
			re, err := regexp.Compile(str(p[1]))
			if err != nil {
				return nil, err
			}
			if m := re.FindString(str(p[0])); m != "" || re.MatchString(str(p[0])) {
				return m, nil
			}
			return nil, nil
		}),
		fn("regex_findall", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("regex_findall needs a pattern")
			}
			re, err := regexp.Compile(str(p[1]))
			if err != nil {
				return nil, err
			}
			out := []interface{}{}
			for _, m := range re.FindAllString(str(p[0]), -1) {
				out = append(out, m)
			}
			return out, nil
		}),
		fn("unique", func(p ...interface{}) (interface{}, error) {
			items, err := Items(p[0])
			if err != nil {
				return nil, err
			}
			out := []interface{}{}
			for _, item := range items {
				if !contains(out, item) {
					out = append(out, item)
				}
			}
			return out, nil
		}),
		fn("list", func(p ...interface{}) (interface{}, error) { return Items(p[0]) }),
		fn("capitalize", func(p ...interface{}) (interface{}, error) {
			s := strings.ToLower(str(p[0]))
			if s == "" {
				return s, nil
			}
			r := []rune(s)
			r[0] = unicode.ToUpper(r[0])
			return string(r), nil
		}),
		fn("title", func(p ...interface{}) (interface{}, error) {
			words := strings.Fields(str(p[0]))
			for i, w := range words {
				r := []rune(strings.ToLower(w))
				r[0] = unicode.ToUpper(r[0])
				words[i] = string(r)
			}
			return strings.Join(words, " "), nil
		}),
		fn("b64encode", func(p ...interface{}) (interface{}, error) {
			return base64.StdEncoding.EncodeToString([]byte(str(p[0]))), nil
		}),
		fn("b64decode", func(p ...interface{}) (interface{}, error) {
			out, err := base64.StdEncoding.DecodeString(str(p[0]))
			return string(out), err
		}),
		fn("quote", func(p ...interface{}) (interface{}, error) {
			return "'" + strings.ReplaceAll(str(p[0]), "'", `'"'"'`) + "'", nil
		}),
		fn("basename", func(p ...interface{}) (interface{}, error) { return path.Base(str(p[0])), nil }),
		fn("dirname", func(p ...interface{}) (interface{}, error) { return path.Dir(str(p[0])), nil }),
		fn("ternary", func(p ...interface{}) (interface{}, error) {
			if len(p) < 3 {
				return nil, fmt.Errorf("ternary needs two values")
			}
			if Truthy(p[0]) {
				return p[1], nil
			}
			return p[2], nil
		}),
		fn("range", func(p ...interface{}) (interface{}, error) {
			n := make([]int, len(p))
			for i, v := range p {
				f, ok := number(v)
				if !ok {
					return nil, fmt.Errorf("range needs numbers")
				}
				n[i] = int(f)
			}
			start, stop, step := 0, 0, 1
			switch len(n) {
			case 1:
				stop = n[0]
			case 2:
				start, stop = n[0], n[1]
			case 3:
				start, stop, step = n[0], n[1], n[2]
			default:
				return nil, fmt.Errorf("range takes 1 to 3 numbers")
			}
			if step == 0 {
				return nil, fmt.Errorf("range step is 0")
			}
			out := []interface{}{}
			for i := start; (step > 0 && i < stop) || (step < 0 && i > stop); i += step {
				out = append(out, i)
			}
			return out, nil
		}),
		fn("combine", func(p ...interface{}) (interface{}, error) {
			out := map[string]interface{}{}
			for _, v := range p {
				m, ok := v.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("combine needs dictionaries, got %T", v)
				}
				for k, val := range m {
					out[k] = val
				}
			}
			return out, nil
		}),
		fn("mandatory", func(p ...interface{}) (interface{}, error) {
			if p[0] == nil {
				return nil, fmt.Errorf("mandatory variable is not defined")
			}
			return p[0], nil
		}),
		// map('filter', args...) applies a filter; map_attribute('x') is
		// map(attribute='x')
		fn("map", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("map needs a filter name")
			}
			items, err := Items(p[0])
			if err != nil {
				return nil, err
			}
			out := make([]interface{}, 0, len(items))
			for _, item := range items {
				v, err := applyFilter(str(p[1]), item, p[2:])
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			}
			return out, nil
		}),
		fn("map_attribute", func(p ...interface{}) (interface{}, error) {
			items, err := Items(p[0])
			if err != nil {
				return nil, err
			}
			out := make([]interface{}, 0, len(items))
			for _, item := range items {
				out = append(out, attribute(item, str(p[1])))
			}
			return out, nil
		}),
		fn("select", func(p ...interface{}) (interface{}, error) { return selectItems(p, false, false) }),
		fn("reject", func(p ...interface{}) (interface{}, error) { return selectItems(p, true, false) }),
		fn("selectattr", func(p ...interface{}) (interface{}, error) { return selectItems(p, false, true) }),
		fn("rejectattr", func(p ...interface{}) (interface{}, error) { return selectItems(p, true, true) }),
		// sorted, so rendered files do not change from run to run
		fn("keys", func(p ...interface{}) (interface{}, error) {
			items, err := dictPairs(p[0])
			out := make([]interface{}, len(items))
			for i, item := range items {
				out[i] = item["key"]
			}
			return out, err
		}),
		fn("values", func(p ...interface{}) (interface{}, error) {
			items, err := dictPairs(p[0])
			out := make([]interface{}, len(items))
			for i, item := range items {
				out[i] = item["value"]
			}
			return out, err
		}),
		fn("jinja_concat", func(p ...interface{}) (interface{}, error) {
			var b strings.Builder
			for _, v := range p {
				if v != nil {
					b.WriteString(str(v))
				}
			}
			return b.String(), nil
		}),
	}
}

func toYAML(p ...interface{}) (interface{}, error) {
	out, err := yaml.Marshal(p[0])
	return string(out), err
}

// attribute reads item.name, or item.a.b for a dotted name
func attribute(item interface{}, name string) interface{} {
	for _, part := range strings.Split(name, ".") {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil
		}
		item = m[part]
	}
	return item
}

// applyFilter runs a filter by name, for map('name')
func applyFilter(name string, value interface{}, args []interface{}) (interface{}, error) {
	env := map[string]interface{}{"v": value}
	call := name + "(v"
	for i, a := range args {
		key := fmt.Sprintf("a%d", i)
		env[key] = a
		call += ", " + key
	}
	return Eval(call+")", env)
}

// selectItems is select/reject(test, arg) and selectattr/rejectattr(attr,
// test, arg); without a test an item is kept when it is truthy
func selectItems(p []interface{}, reject, byAttr bool) (interface{}, error) {
	items, err := Items(p[0])
	if err != nil {
		return nil, err
	}
	rest := p[1:]
	attr := ""
	if byAttr {
		if len(rest) == 0 {
			return nil, fmt.Errorf("selectattr needs an attribute")
		}
		attr, rest = fmt.Sprint(rest[0]), rest[1:]
	}
	out := []interface{}{}
	for _, item := range items {
		v := item
		if byAttr {
			v = attribute(item, attr)
		}
		ok, err := jinjaTest(v, rest)
		if err != nil {
			return nil, err
		}
		if ok != reject {
			out = append(out, item)
		}
	}
	return out, nil
}

func jinjaTest(v interface{}, args []interface{}) (bool, error) {
	if len(args) == 0 {
		return Truthy(v), nil
	}
	test := fmt.Sprint(args[0])
	var arg interface{}
	if len(args) > 1 {
		arg = args[1]
	}
	switch test {
	case "defined":
		return v != nil, nil
	case "undefined", "none":
		return v == nil, nil
	case "truthy":
		return Truthy(v), nil
	case "falsy":
		return !Truthy(v), nil
	case "equalto", "==", "eq", "sameas":
		return equal(v, arg), nil
	case "!=", "ne":
		return !equal(v, arg), nil
	case "in":
		return contains(arg, v), nil
	case "contains":
		return contains(v, arg), nil
	case "match", "search", "regex":
		re, err := regexp.Compile(fmt.Sprint(arg))
		if err != nil {
			return false, err
		}
		s := fmt.Sprint(v)
		if test == "match" {
			loc := re.FindStringIndex(s)
			return len(loc) == 2 && loc[0] == 0, nil
		}
		return re.MatchString(s), nil
	case "string":
		_, ok := v.(string)
		return ok, nil
	case "number":
		_, ok := number(v)
		return ok, nil
	case ">", "gt", "<", "lt", ">=", "ge", "<=", "le":
		a, ok1 := number(v)
		b, ok2 := number(arg)
		if !ok1 || !ok2 {
			return false, nil
		}
		switch test {
		case ">", "gt":
			return a > b, nil
		case "<", "lt":
			return a < b, nil
		case ">=", "ge":
			return a >= b, nil
		}
		return a <= b, nil
	}
	return false, fmt.Errorf("unknown test %q", test)
}
