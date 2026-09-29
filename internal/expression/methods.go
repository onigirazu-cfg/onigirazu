package expression

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr/ast"
)

// pythonMethods are the methods of strings, dicts and lists templates call
// as in Python: "{{ opts.split() }}", "{{ d.get('k', 1) }}"
var pythonMethods = map[string]bool{
	"split": true, "rsplit": true, "strip": true, "lstrip": true, "rstrip": true,
	"lower": true, "upper": true, "title": true, "capitalize": true,
	"startswith": true, "endswith": true, "replace": true, "find": true,
	"count": true, "join": true, "splitlines": true, "isdigit": true,
	"get": true, "items": true, "index": true,
}

// methodPatch turns obj.method(args) into jinja_method(obj, "method", args)
type methodPatch struct{}

func (methodPatch) Visit(node *ast.Node) {
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}
	member, ok := call.Callee.(*ast.MemberNode)
	if !ok {
		return
	}
	name, ok := member.Property.(*ast.StringNode)
	if !ok || !pythonMethods[name.Value] {
		return
	}
	args := append([]ast.Node{member.Node, &ast.StringNode{Value: name.Value}}, call.Arguments...)
	ast.Patch(node, &ast.CallNode{Callee: &ast.IdentifierNode{Value: "jinja_method"}, Arguments: args})
}

// callMethod runs a Python method on a value
func callMethod(params ...interface{}) (interface{}, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("method call without a name")
	}
	obj, name, args := params[0], fmt.Sprint(params[1]), params[2:]
	switch v := obj.(type) {
	case string:
		return stringMethod(v, name, args)
	case map[string]interface{}:
		switch name {
		case "get":
			if len(args) == 0 {
				return nil, fmt.Errorf("get needs a key")
			}
			if value, ok := v[fmt.Sprint(args[0])]; ok {
				return value, nil
			}
			if len(args) > 1 {
				return args[1], nil
			}
			return nil, nil
		case "items":
			return Dict2Items(v)
		}
	case []interface{}:
		switch name {
		case "index":
			for i, item := range v {
				if len(args) > 0 && equal(item, args[0]) {
					return i, nil
				}
			}
			return nil, fmt.Errorf("value is not in the list")
		case "count":
			n := 0
			for _, item := range v {
				if len(args) > 0 && equal(item, args[0]) {
					n++
				}
			}
			return n, nil
		}
	}
	return nil, fmt.Errorf("%T has no method %s", obj, name)
}

func stringMethod(s, name string, args []interface{}) (interface{}, error) {
	arg := func(i int) string {
		if i < len(args) && args[i] != nil {
			return fmt.Sprint(args[i])
		}
		return ""
	}
	switch name {
	case "split", "rsplit":
		var parts []string
		limit := -1
		if len(args) > 1 {
			if n, ok := asInt(args[1]); ok && n >= 0 {
				limit = n + 1
			}
		}
		switch {
		case arg(0) == "" && limit < 0:
			parts = strings.Fields(s)
		case arg(0) == "":
			parts = strings.Fields(s)
			if len(parts) > limit {
				parts = append(parts[:limit-1], strings.Join(parts[limit-1:], " "))
			}
		case name == "rsplit" && limit > 0:
			parts = strings.Split(s, arg(0))
			if len(parts) > limit {
				cut := len(parts) - limit + 1
				parts = append([]string{strings.Join(parts[:cut], arg(0))}, parts[cut:]...)
			}
		default:
			parts = strings.SplitN(s, arg(0), limit)
		}
		return stringsToList(parts), nil
	case "strip", "lstrip", "rstrip":
		cut := arg(0)
		if cut == "" {
			cut = " \t\n\r\v\f"
		}
		switch name {
		case "strip":
			return strings.Trim(s, cut), nil
		case "lstrip":
			return strings.TrimLeft(s, cut), nil
		}
		return strings.TrimRight(s, cut), nil
	case "lower":
		return strings.ToLower(s), nil
	case "upper":
		return strings.ToUpper(s), nil
	case "title":
		return titleCase(s), nil
	case "capitalize":
		if s == "" {
			return s, nil
		}
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:]), nil
	case "startswith", "endswith":
		candidates := []string{arg(0)}
		if list, ok := firstArg(args).([]interface{}); ok {
			candidates = candidates[:0]
			for _, c := range list {
				candidates = append(candidates, fmt.Sprint(c))
			}
		}
		for _, c := range candidates {
			if (name == "startswith" && strings.HasPrefix(s, c)) || (name == "endswith" && strings.HasSuffix(s, c)) {
				return true, nil
			}
		}
		return false, nil
	case "replace":
		n := -1
		if len(args) > 2 {
			if c, ok := asInt(args[2]); ok {
				n = c
			}
		}
		return strings.Replace(s, arg(0), arg(1), n), nil
	case "find":
		return strings.Index(s, arg(0)), nil
	case "count":
		return strings.Count(s, arg(0)), nil
	case "join":
		items, err := Items(firstArg(args))
		if err != nil {
			return nil, err
		}
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = PyStr(item)
		}
		return strings.Join(parts, s), nil
	case "splitlines":
		return stringsToList(strings.Split(strings.TrimRight(s, "\n"), "\n")), nil
	case "isdigit":
		if s == "" {
			return false, nil
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return false, nil
			}
		}
		return true, nil
	}
	return nil, fmt.Errorf("string has no method %s", name)
}

func firstArg(args []interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

func stringsToList(parts []string) []interface{} {
	out := make([]interface{}, len(parts))
	for i, p := range parts {
		out[i] = p
	}
	return out
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}
