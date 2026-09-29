package expression

import (
	"encoding/json"
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

// pyJSON is Python's json.dumps as Ansible's to_json/to_nice_json call it:
// ", " and ": " separators, non-ASCII as \uXXXX, no HTML escaping. indent > 0
// is to_nice_json. Keys are sorted (Go maps keep no order).
func pyJSON(v interface{}, indent int) (string, error) {
	var b strings.Builder
	if err := writePyJSON(&b, v, indent, 0); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writePyJSON(b *strings.Builder, v interface{}, indent, level int) error {
	newline := func(l int) {
		if indent > 0 {
			b.WriteByte('\n')
			b.WriteString(strings.Repeat(" ", indent*l))
		}
	}
	sep := ", "
	if indent > 0 {
		sep = ","
	}
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
		return nil
	case bool:
		b.WriteString(strconv.FormatBool(x))
		return nil
	case string:
		b.WriteString(pyJSONString(x))
		return nil
	case float64:
		b.WriteString(pyFloat(x))
		return nil
	case float32:
		b.WriteString(pyFloat(float64(x)))
		return nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprint(b, v)
		return nil
	case reflect.Slice, reflect.Array:
		if rv.Len() == 0 {
			b.WriteString("[]")
			return nil
		}
		b.WriteByte('[')
		for i := 0; i < rv.Len(); i++ {
			if i > 0 {
				b.WriteString(sep)
			}
			newline(level + 1)
			if err := writePyJSON(b, rv.Index(i).Interface(), indent, level+1); err != nil {
				return err
			}
		}
		newline(level)
		b.WriteByte(']')
		return nil
	case reflect.Map:
		if rv.Len() == 0 {
			b.WriteString("{}")
			return nil
		}
		keys := rv.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface()) })
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteString(sep)
			}
			newline(level + 1)
			b.WriteString(pyJSONString(fmt.Sprint(k.Interface())))
			b.WriteString(": ")
			if err := writePyJSON(b, rv.MapIndex(k).Interface(), indent, level+1); err != nil {
				return err
			}
		}
		newline(level)
		b.WriteByte('}')
		return nil
	}
	// anything else: through encoding/json and back to plain values
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var plain interface{}
	if err := json.Unmarshal(raw, &plain); err != nil {
		return err
	}
	return writePyJSON(b, plain, indent, level)
}

func pyJSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r < 0x20 || (r >= 0x7f && r <= 0xffff):
			fmt.Fprintf(&b, `\u%04x`, r)
		case r > 0xffff:
			r -= 0x10000
			fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// fromJSON decodes JSON with whole numbers as ints, as Python does
func fromJSON(s string) (interface{}, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var out interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return plainNumbers(out), nil
}

func plainNumbers(v interface{}) interface{} {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return int(i)
		}
		f, _ := x.Float64()
		return f
	case []interface{}:
		for i := range x {
			x[i] = plainNumbers(x[i])
		}
	case map[string]interface{}:
		for k := range x {
			x[k] = plainNumbers(x[k])
		}
	}
	return v
}
