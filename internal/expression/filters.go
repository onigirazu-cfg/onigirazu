package expression

import (
	"crypto/md5"  // #nosec G501 -- the hash filter computes checksums a playbook asks for
	"crypto/sha1" // #nosec G505 -- the hash filter computes checksums a playbook asks for
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
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
	"from_yaml", "unique", "list", "capitalize", "title", "b64encode", "b64decode", "quote", "basename", "splitext", "wordcount", "center",
	"dirname", "sort", "sum", "max", "min", "reverse", "flatten", "join", "keys", "values", "abs", "round",
	"select", "reject", "mandatory", "password_hash", "random", "shuffle",
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
		expr.DisableBuiltin("sort"),
		// Jinja's string and join print as Python does (True, ['a'])
		expr.DisableBuiltin("string"),
		expr.DisableBuiltin("join"),
		expr.DisableBuiltin("split"),
		// Python's str.split(sep=None, maxsplit=-1): no separator splits on
		// runs of whitespace
		fn("split", func(p ...interface{}) (interface{}, error) {
			s := PyStr(p[0])
			maxsplit := -1
			if len(p) > 2 {
				if n, ok := asInt(p[2]); ok {
					maxsplit = n
				}
			}
			var parts []string
			if len(p) < 2 || p[1] == nil {
				fields := strings.Fields(s)
				if maxsplit >= 0 && len(fields) > maxsplit+1 {
					// keep the rest as one piece, from where field maxsplit starts
					rest := s
					parts = nil
					for i := 0; i < maxsplit; i++ {
						rest = strings.TrimLeft(rest, " \t\n\r\v\f")
						end := strings.IndexAny(rest, " \t\n\r\v\f")
						parts = append(parts, rest[:end])
						rest = rest[end:]
					}
					parts = append(parts, strings.TrimLeft(rest, " \t\n\r\v\f"))
				} else {
					parts = fields
				}
			} else {
				parts = strings.SplitN(s, PyStr(p[1]), maxsplit+1)
				if maxsplit < 0 {
					parts = strings.Split(s, PyStr(p[1]))
				}
			}
			out := make([]interface{}, len(parts))
			for i, v := range parts {
				out[i] = v
			}
			return out, nil
		}),
		expr.DisableBuiltin("lower"),
		expr.DisableBuiltin("upper"),
		// {{ flag | lower }} is the usual way to get "true" out of a boolean
		fn("lower", func(p ...interface{}) (interface{}, error) { return strings.ToLower(PyStr(p[0])), nil }),
		fn("upper", func(p ...interface{}) (interface{}, error) { return strings.ToUpper(PyStr(p[0])), nil }),
		fn("string", func(p ...interface{}) (interface{}, error) { return PyStr(p[0]), nil }),
		// "%-5s|" | format(x): Python printf-style formatting
		fn("format", func(p ...interface{}) (interface{}, error) { return pyFormat(PyStr(p[0]), p[1:]) }),
		fn("join", func(p ...interface{}) (interface{}, error) {
			items, err := Items(p[0])
			if err != nil {
				return nil, fmt.Errorf("join: %w", err)
			}
			sep := ""
			if len(p) > 1 {
				sep = PyStr(p[1])
			}
			parts := make([]string, len(items))
			for i, item := range items {
				parts[i] = PyStr(item)
			}
			return strings.Join(parts, sep), nil
		}),
		fn("to_json", func(p ...interface{}) (interface{}, error) { return pyJSON(p[0], 0) }),
		fn("to_nice_json", func(p ...interface{}) (interface{}, error) { return pyJSON(p[0], 4) }),
		fn("random", jinjaRandom),
		fn("shuffle", jinjaShuffle),
		fn("to_yaml", toYAML),
		fn("to_nice_yaml", toYAML),
		fn("from_json", func(p ...interface{}) (interface{}, error) { return fromJSON(str(p[0])) }),
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
		// Jinja's wordcount: runs of word characters
		fn("wordcount", func(p ...interface{}) (interface{}, error) {
			return len(wordRe.FindAllString(PyStr(p[0]), -1)), nil
		}),
		// Jinja's center(width=80): Python str.center
		fn("center", func(p ...interface{}) (interface{}, error) {
			s := PyStr(p[0])
			width := 80
			if len(p) > 1 {
				if w, ok := asInt(p[1]); ok {
					width = w
				}
			}
			n := len([]rune(s))
			if width <= n {
				return s, nil
			}
			pad := width - n
			left := pad/2 + (pad & width & 1) // CPython's str.center
			return strings.Repeat(" ", left) + s + strings.Repeat(" ", pad-left), nil
		}),
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
		// hash('sha1'): hex digest, as Ansible's hash filter (default sha1)
		fn("hash", func(p ...interface{}) (interface{}, error) {
			alg := "sha1"
			if len(p) > 1 {
				alg = strings.ToLower(str(p[1]))
			}
			var h hash.Hash
			switch alg {
			case "md5":
				h = md5.New() // #nosec G401 -- a checksum the playbook asked for
			case "sha1":
				h = sha1.New() // #nosec G401 -- a checksum the playbook asked for
			case "sha224":
				h = sha256.New224()
			case "sha256":
				h = sha256.New()
			case "sha384":
				h = sha512.New384()
			case "sha512":
				h = sha512.New()
			default:
				return nil, fmt.Errorf("hash: unsupported algorithm %q (md5, sha1, sha224, sha256, sha384, sha512)", alg)
			}
			h.Write([]byte(str(p[0])))
			return hex.EncodeToString(h.Sum(nil)), nil
		}),
		// Python's os.path.splitext: [root, ext]; a leading dot is not an extension
		fn("splitext", func(p ...interface{}) (interface{}, error) {
			s := str(p[0])
			base := s[strings.LastIndex(s, "/")+1:]
			i := strings.LastIndex(strings.TrimLeft(base, "."), ".")
			if i < 0 {
				return []interface{}{s, ""}, nil
			}
			cut := len(s) - len(base) + (len(base) - len(strings.TrimLeft(base, "."))) + i
			return []interface{}{s[:cut], s[cut:]}, nil
		}),
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
		// extract(key, container, morekey): container[key][morekey], as in
		// groups['web'] | map('extract', hostvars, 'ansible_host')
		fn("extract", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("extract needs a container")
			}
			value := index(p[1], p[0])
			for _, key := range p[2:] {
				value = index(value, key)
			}
			return value, nil
		}),
		fn("password_hash", passwordHash),
		fn("jinja_lookup", jinjaLookup),
		fn("jinja_defined", func(p ...interface{}) (interface{}, error) {
			vars, _ := p[0].(map[string]interface{})
			return jinjaDefined(vars, str(p[1]), p[2]), nil
		}),
		fn("jinja_query", jinjaQuery),
		fn("combine", jinjaCombine),
		fn("sort", jinjaSort),
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
			// map(attribute='x', default=d): "default", d after the name
			var def interface{}
			hasDef := len(p) > 3 && str(p[2]) == "default"
			if hasDef {
				def = p[3]
			}
			out := make([]interface{}, 0, len(items))
			for _, item := range items {
				v := attribute(item, str(p[1]))
				if v == nil && hasDef {
					v = def
				}
				out = append(out, v)
			}
			return out, nil
		}),
		fn("intersect", func(p ...interface{}) (interface{}, error) { return setOp(p, "intersect") }),
		fn("difference", func(p ...interface{}) (interface{}, error) { return setOp(p, "difference") }),
		fn("union", func(p ...interface{}) (interface{}, error) { return setOp(p, "union") }),
		fn("symmetric_difference", func(p ...interface{}) (interface{}, error) { return setOp(p, "symmetric_difference") }),
		fn("product", func(p ...interface{}) (interface{}, error) {
			// Ansible's product: the cartesian product of the lists, as lists
			lists := make([][]interface{}, 0, len(p))
			for _, x := range p {
				items, err := Items(x)
				if err != nil {
					return nil, err
				}
				lists = append(lists, items)
			}
			rows := [][]interface{}{{}}
			for _, l := range lists {
				var next [][]interface{}
				for _, prefix := range rows {
					for _, x := range l {
						next = append(next, append(append([]interface{}{}, prefix...), x))
					}
				}
				rows = next
			}
			out := make([]interface{}, len(rows))
			for i, r := range rows {
				out[i] = r
			}
			return out, nil
		}),
		fn("zip", func(p ...interface{}) (interface{}, error) {
			lists := make([][]interface{}, 0, len(p))
			n := -1
			for _, x := range p {
				items, err := Items(x)
				if err != nil {
					return nil, err
				}
				lists = append(lists, items)
				if n < 0 || len(items) < n {
					n = len(items)
				}
			}
			out := make([]interface{}, 0, n)
			for i := 0; i < n; i++ {
				row := make([]interface{}, len(lists))
				for j, l := range lists {
					row[j] = l[i]
				}
				out = append(out, row)
			}
			return out, nil
		}),
		fn("regex_escape", func(p ...interface{}) (interface{}, error) { return regexp.QuoteMeta(str(p[0])), nil }),
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
		fn("jinja_is", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("a test needs a name")
			}
			return jinjaTest(p[0], p[1:])
		}),
		fn("jinja_concat", func(p ...interface{}) (interface{}, error) {
			var b strings.Builder
			for _, v := range p {
				if v != nil {
					b.WriteString(PyStr(v))
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
	if ok, handled, err := ansibleTest(test, v, args[1:]); handled {
		return ok, err
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

// index reads container[key] from a map or a list; missing is nil
func index(container, key interface{}) interface{} {
	switch c := container.(type) {
	case map[string]interface{}:
		return c[fmt.Sprint(key)]
	case []interface{}:
		if i, ok := number(key); ok && int(i) >= 0 && int(i) < len(c) {
			return c[int(i)]
		}
	}
	return nil
}

// setOp is Ansible's intersect, difference, union and symmetric_difference:
// the order of the first list, no duplicates
func setOp(p []interface{}, op string) (interface{}, error) {
	if len(p) < 2 {
		return nil, fmt.Errorf("%s needs a second list", op)
	}
	a, err := Items(p[0])
	if err != nil {
		return nil, err
	}
	b, err := Items(p[1])
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	add := func(x interface{}) {
		if !contains(out, x) {
			out = append(out, x)
		}
	}
	switch op {
	case "intersect":
		for _, x := range a {
			if contains(b, x) {
				add(x)
			}
		}
	case "difference":
		for _, x := range a {
			if !contains(b, x) {
				add(x)
			}
		}
	case "union":
		for _, x := range append(append([]interface{}{}, a...), b...) {
			add(x)
		}
	case "symmetric_difference":
		for _, x := range a {
			if !contains(b, x) {
				add(x)
			}
		}
		for _, x := range b {
			if !contains(a, x) {
				add(x)
			}
		}
	}
	return out, nil
}

var wordRe = regexp.MustCompile(`\w+`)
