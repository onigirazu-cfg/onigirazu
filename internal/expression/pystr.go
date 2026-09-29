package expression

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// PyStr is Python's str() of a value: what Jinja prints for {{ x }} inside
// text, for ~ and for the string and join filters. Booleans are True and
// False, None is None, lists and dicts print as Python literals.
func PyStr(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return "None"
	}
	return pyRepr(v)
}

// pyRepr is Python's repr(): strings quoted, the rest as PyStr
func pyRepr(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		return pyQuote(x)
	case bool:
		if x {
			return "True"
		}
		return "False"
	case float64:
		return pyFloat(x)
	case float32:
		return pyFloat(float64(x))
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return "[]"
		}
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = pyRepr(rv.Index(i).Interface())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case reflect.Map:
		keys := rv.MapKeys()
		// Python keeps insertion order, which a Go map has lost: sorted keys
		// at least print the same every time
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface()) })
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = pyRepr(k.Interface()) + ": " + pyRepr(rv.MapIndex(k).Interface())
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(v)
}

// pyFloat prints a float as Python does: 3.0, 0.5, 1e+16, 1e-05
func pyFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	if f != 0 {
		if exp := math.Floor(math.Log10(math.Abs(f))); exp < -4 || exp >= 16 {
			return strconv.FormatFloat(f, 'e', -1, 64)
		}
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".") {
		s += ".0"
	}
	return s
}

// pyQuote is repr() of a string: single quotes unless the string has a single
// quote and no double quote
func pyQuote(s string) string {
	q := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		q = '"'
	}
	var b strings.Builder
	b.WriteByte(q)
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == rune(q):
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(q)
	return b.String()
}
