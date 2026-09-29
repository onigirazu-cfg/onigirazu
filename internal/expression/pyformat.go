package expression

import (
	"fmt"
	"math"
	"strings"
)

// pyFormat is Python's printf-style formatting, used by Jinja's format
// filter and by string % args: %[flags][width][.precision]type with the
// types s r d i u f F e E g G x X o c and %%
func pyFormat(format string, args []interface{}) (string, error) {
	var b strings.Builder
	n := 0
	next := func() (interface{}, error) {
		if n >= len(args) {
			return nil, fmt.Errorf("format: not enough arguments for %q", format)
		}
		n++
		return args[n-1], nil
	}
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		j := i + 1
		for j < len(format) && strings.IndexByte("-+ #0", format[j]) >= 0 {
			j++
		}
		for j < len(format) && (format[j] >= '0' && format[j] <= '9' || format[j] == '.') {
			j++
		}
		if j >= len(format) {
			return "", fmt.Errorf("format: incomplete format %q", format[i:])
		}
		spec, verb := format[i+1:j], format[j]
		i = j
		if verb == '%' {
			b.WriteByte('%')
			continue
		}
		arg, err := next()
		if err != nil {
			return "", err
		}
		switch verb {
		case 's':
			fmt.Fprintf(&b, "%"+spec+"s", PyStr(arg))
		case 'r', 'a':
			fmt.Fprintf(&b, "%"+spec+"s", pyRepr(arg))
		case 'd', 'i', 'u':
			v, ok := asInt(arg)
			if !ok {
				f, isNum := number(arg)
				if !isNum {
					return "", fmt.Errorf("format: %%%c needs a number, got %T", verb, arg)
				}
				v = int(f)
			}
			fmt.Fprintf(&b, "%"+spec+"d", v)
		case 'f', 'F', 'e', 'E', 'g', 'G':
			f, ok := number(arg)
			if !ok {
				return "", fmt.Errorf("format: %%%c needs a number, got %T", verb, arg)
			}
			if !strings.Contains(spec, ".") && verb != 'g' && verb != 'G' {
				spec += ".6"
			}
			fmt.Fprintf(&b, "%"+spec+string(rune(verb)), f)
		case 'x', 'X', 'o':
			v, ok := asInt(arg)
			if !ok {
				return "", fmt.Errorf("format: %%%c needs an integer, got %T", verb, arg)
			}
			if strings.Contains(spec, "#") && verb == 'o' {
				spec = strings.ReplaceAll(spec, "#", "")
				fmt.Fprintf(&b, "0o%"+spec+"o", v)
				continue
			}
			fmt.Fprintf(&b, "%"+spec+string(rune(verb)), v)
		case 'c':
			if s, ok := arg.(string); ok {
				fmt.Fprintf(&b, "%"+spec+"s", s)
			} else if v, ok := asInt(arg); ok {
				fmt.Fprintf(&b, "%"+spec+"c", rune(v))
			}
		default:
			return "", fmt.Errorf("format: unsupported format character %q", verb)
		}
	}
	if n < len(args) {
		return "", fmt.Errorf("format: not all arguments converted during string formatting")
	}
	return b.String(), nil
}

func jinjaMod(a, b interface{}) (interface{}, error) {
	if f, ok := a.(string); ok {
		return pyFormat(f, []interface{}{b})
	}
	if ai, ok := asInt(a); ok {
		if bi, ok := asInt(b); ok {
			if bi == 0 {
				return nil, fmt.Errorf("integer modulo by zero")
			}
			r := ai % bi
			if r != 0 && (r < 0) != (bi < 0) {
				r += bi
			}
			return r, nil
		}
	}
	af, ok1 := number(a)
	bf, ok2 := number(b)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot take %T modulo %T", a, b)
	}
	if bf == 0 {
		return nil, fmt.Errorf("float modulo by zero")
	}
	r := math.Mod(af, bf)
	if r != 0 && (r < 0) != (bf < 0) {
		r += bf
	}
	return r, nil
}
