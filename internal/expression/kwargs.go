package expression

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// kwPrefix marks a keyword argument name in a filter call: sort(attribute='x')
// arrives as sort(list, "__kw__attribute", "x")
const kwPrefix = "__kw__"

// splitKwargs separates positional arguments from keyword arguments
func splitKwargs(p []interface{}) ([]interface{}, map[string]interface{}) {
	pos := make([]interface{}, 0, len(p))
	kw := map[string]interface{}{}
	for i := 0; i < len(p); i++ {
		if s, ok := p[i].(string); ok && strings.HasPrefix(s, kwPrefix) && i+1 < len(p) {
			kw[strings.TrimPrefix(s, kwPrefix)] = p[i+1]
			i++
			continue
		}
		pos = append(pos, p[i])
	}
	return pos, kw
}

// jinjaSort is Jinja's sort(reverse=false, case_sensitive=false,
// attribute=none); attribute may name several comma separated keys, and
// dotted paths
func jinjaSort(p ...interface{}) (interface{}, error) {
	pos, kw := splitKwargs(p)
	if len(pos) == 0 {
		return nil, fmt.Errorf("sort needs a list")
	}
	items, err := Items(pos[0])
	if err != nil {
		return nil, fmt.Errorf("sort: %w", err)
	}
	opt := func(name string, i int) interface{} {
		if v, ok := kw[name]; ok {
			return v
		}
		if i < len(pos) {
			return pos[i]
		}
		return nil
	}
	reverse, caseSensitive := Truthy(opt("reverse", 1)), Truthy(opt("case_sensitive", 2))
	var attrs []string
	if a := opt("attribute", 3); a != nil {
		for _, name := range strings.Split(fmt.Sprint(a), ",") {
			attrs = append(attrs, strings.TrimSpace(name))
		}
	}
	key := func(v interface{}) []interface{} {
		if len(attrs) == 0 {
			return []interface{}{v}
		}
		keys := make([]interface{}, len(attrs))
		for i, a := range attrs {
			keys[i] = attribute(v, a)
		}
		return keys
	}
	out := append([]interface{}{}, items...)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := key(out[i]), key(out[j])
		for n := range ki {
			if c := compareSortKeys(ki[n], kj[n], caseSensitive); c != 0 {
				if reverse {
					return c > 0
				}
				return c < 0
			}
		}
		return false
	})
	return out, nil
}

func compareSortKeys(a, b interface{}, caseSensitive bool) int {
	fa, aNum := toFloat(a)
	fb, bNum := toFloat(b)
	if aNum && bNum {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		}
		return 0
	}
	sa, sb := fmt.Sprint(a), fmt.Sprint(b)
	if !caseSensitive {
		sa, sb = strings.ToLower(sa), strings.ToLower(sb)
	}
	return strings.Compare(sa, sb)
}

func toFloat(v interface{}) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}

// jinjaCombine is Ansible's combine(*dicts, recursive=false,
// list_merge='replace')
func jinjaCombine(p ...interface{}) (interface{}, error) {
	pos, kw := splitKwargs(p)
	recursive := Truthy(kw["recursive"])
	listMerge := "replace"
	if v, ok := kw["list_merge"]; ok {
		listMerge = fmt.Sprint(v)
	}
	switch listMerge {
	case "replace", "keep", "append", "prepend":
	default:
		return nil, fmt.Errorf("combine: list_merge %q is not supported (replace, keep, append, prepend)", listMerge)
	}
	out := map[string]interface{}{}
	for _, v := range pos {
		// combine also takes lists of dictionaries
		dicts := []interface{}{v}
		if l, ok := v.([]interface{}); ok {
			dicts = l
		}
		for _, d := range dicts {
			m, ok := d.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("combine needs dictionaries, got %T", d)
			}
			out = mergeDicts(out, m, recursive, listMerge)
		}
	}
	return out, nil
}

func mergeDicts(dst, src map[string]interface{}, recursive bool, listMerge string) map[string]interface{} {
	out := make(map[string]interface{}, len(dst)+len(src))
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range src {
		old, exists := out[k]
		if !exists {
			out[k] = v
			continue
		}
		om, oldIsMap := old.(map[string]interface{})
		nm, newIsMap := v.(map[string]interface{})
		if recursive && oldIsMap && newIsMap {
			out[k] = mergeDicts(om, nm, recursive, listMerge)
			continue
		}
		ol, oldIsList := old.([]interface{})
		nl, newIsList := v.([]interface{})
		if oldIsList && newIsList {
			switch listMerge {
			case "keep":
				continue
			case "append":
				out[k] = append(append([]interface{}{}, ol...), nl...)
				continue
			case "prepend":
				out[k] = append(append([]interface{}{}, nl...), ol...)
				continue
			}
		}
		out[k] = v
	}
	return out
}
