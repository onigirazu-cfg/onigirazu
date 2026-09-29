package expression

import (
	"fmt"
	"math"
	"strings"
)

// jinjaMul is Jinja's *: a string or list times a number repeats it,
// numbers multiply (ints stay ints)
func jinjaMul(a, b interface{}) (interface{}, error) {
	if n, ok := asInt(b); ok {
		switch x := a.(type) {
		case string:
			return strings.Repeat(x, max(n, 0)), nil
		case []interface{}:
			out := []interface{}{}
			for i := 0; i < n; i++ {
				out = append(out, x...)
			}
			return out, nil
		}
	}
	if _, ok := b.(string); ok {
		if _, isNum := asInt(a); isNum {
			return jinjaMul(b, a)
		}
	}
	if ai, ok := asInt(a); ok {
		if bi, ok := asInt(b); ok {
			return ai * bi, nil
		}
	}
	if af, ok := number(a); ok {
		if bf, ok := number(b); ok {
			return af * bf, nil
		}
	}
	return nil, fmt.Errorf("cannot multiply %T and %T", a, b)
}

// jinjaFloorDiv is Jinja's //: rounds toward minus infinity, ints stay ints
func jinjaFloorDiv(a, b interface{}) (interface{}, error) {
	if ai, ok := asInt(a); ok {
		if bi, ok := asInt(b); ok {
			if bi == 0 {
				return nil, fmt.Errorf("integer division by zero")
			}
			q := ai / bi
			if (ai%bi != 0) && ((ai < 0) != (bi < 0)) {
				q--
			}
			return q, nil
		}
	}
	af, ok1 := number(a)
	bf, ok2 := number(b)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot divide %T by %T", a, b)
	}
	if bf == 0 {
		return nil, fmt.Errorf("float floor division by zero")
	}
	return math.Floor(af / bf), nil
}

// floorDivToCalls turns a // b into jinja_floordiv(a, b): expr reads // as a
// comment and would drop the rest of the expression. Operands are names with
// attributes, indexes and calls, numbers, or parenthesized expressions.
func floorDivToCalls(s string) string {
	for {
		i := indexOutsideQuotes(s, "//")
		if i < 0 {
			return s
		}
		end := i
		for end > 0 && s[end-1] == ' ' {
			end--
		}
		start := fdOperandStart(s, end)
		// a unary minus belongs to the operand: -7 // 2 is (-7) // 2
		if start > 0 && s[start-1] == '-' {
			k := start - 1
			for k > 0 && s[k-1] == ' ' {
				k--
			}
			if k == 0 || strings.ContainsRune("(,[+-*/%=<>!", rune(s[k-1])) {
				start--
			}
		}
		r := i + 2
		for r < len(s) && s[r] == ' ' {
			r++
		}
		rend := fdOperandEnd(s, r)
		if start == end || r == rend {
			return s // not an operator we understand; leave it to fail
		}
		s = s[:start] + "jinja_floordiv(" + s[start:end] + ", " + s[r:rend] + ")" + s[rend:]
	}
}

func indexOutsideQuotes(s, sub string) int {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case strings.HasPrefix(s[i:], sub):
			return i
		}
	}
	return -1
}

func isOperandChar(c byte) bool {
	return c == '_' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// fdOperandStart scans back from end over one operand
func fdOperandStart(s string, end int) int {
	j := end
	for j > 0 {
		switch c := s[j-1]; {
		case c == ')' || c == ']':
			j = matchBack(s, j-1)
		case isOperandChar(c):
			j--
		default:
			return j
		}
	}
	return j
}

// fdOperandEnd scans forward from start over one operand
func fdOperandEnd(s string, start int) int {
	j := start
	if j < len(s) && s[j] == '-' {
		j++
	}
	for j < len(s) {
		switch c := s[j]; {
		case c == '(' || c == '[':
			j = matchForward(s, j) + 1
		case isOperandChar(c):
			j++
		default:
			return j
		}
	}
	return j
}

func matchBack(s string, close int) int {
	open := map[byte]byte{')': '(', ']': '['}[s[close]]
	depth := 0
	for k := close; k >= 0; k-- {
		switch s[k] {
		case s[close]:
			depth++
		case open:
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return 0
}

func matchForward(s string, open int) int {
	closeCh := map[byte]byte{'(': ')', '[': ']'}[s[open]]
	depth := 0
	for k := open; k < len(s); k++ {
		switch s[k] {
		case s[open]:
			depth++
		case closeCh:
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return len(s) - 1
}
