package expression

import (
	"reflect"
	"regexp"
	"strconv"
)

var (
	// a.b, a['b'], a["b"], a[0]: a path with literal steps only
	staticPath = regexp.MustCompile(`^[A-Za-z_]\w*(?:\.[A-Za-z_]\w*|\[(?:'[^']*'|"[^"]*"|\d+)\])*$`)
	pathStep   = regexp.MustCompile(`\.?([A-Za-z_]\w*)|\[(?:'([^']*)'|"([^"]*)"|(\d+))\]`)
)

// LookupPath walks a variable path with literal steps (a.b, a['b'], a[0]).
// It tells a variable that holds None (value nil, found true) from one that
// does not exist (found false). known is false for a path it cannot walk,
// such as a[i].
func LookupPath(path string, variables map[string]interface{}) (value interface{}, found, known bool) {
	if !staticPath.MatchString(path) {
		return nil, false, false
	}
	var cur interface{} = variables
	for _, m := range pathStep.FindAllStringSubmatch(path, -1) {
		if cur == nil {
			return nil, false, true
		}
		rv := reflect.ValueOf(cur)
		switch rv.Kind() {
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String || m[4] != "" {
				return nil, false, true
			}
			v := rv.MapIndex(reflect.ValueOf(m[1] + m[2] + m[3]).Convert(rv.Type().Key()))
			if !v.IsValid() {
				return nil, false, true
			}
			cur = v.Interface()
		case reflect.Slice, reflect.Array:
			i, err := strconv.Atoi(m[4])
			if err != nil || i >= rv.Len() {
				return nil, false, true
			}
			cur = rv.Index(i).Interface()
		default:
			return nil, false, false
		}
	}
	return cur, true, true
}

// jinjaDefined is "path is defined": a variable that holds None is defined
func jinjaDefined(variables map[string]interface{}, path string, fallback interface{}) bool {
	if _, found, known := LookupPath(path, variables); known {
		return found
	}
	return fallback != nil
}
