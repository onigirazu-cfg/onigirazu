package expression

import (
	"strings"
)

// In Jinja a filter binds to the operand right before it: `a and b | length`
// is `a and (b | length)`. expr's pipe binds weaker than and/or, so every
// `X | f(args)` is rewritten into the call `f(X, args)`, X being the postfix
// expression before the pipe (a name with .attr/[i]/(...) parts, a literal,
// a parenthesized group or an earlier call). Tests work the same way:
// `X is [not] name(args)` becomes `[!]jinja_is(X, "name", args)`.

func isIdentByte(c byte) bool {
	return c == '_' || c == '.' || c == '?' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// skipQuoteBack returns the index of the opening quote of the string that
// ends at s[end]
func skipQuoteBack(s string, end int) int {
	q := s[end]
	for i := end - 1; i >= 0; i-- {
		if s[i] == q && (i == 0 || s[i-1] != '\\') {
			return i
		}
	}
	return 0
}

// operandStart finds where the postfix expression ending at s[end] starts
func operandStart(s string, end int) int {
	j := end
	for j >= 0 {
		c := s[j]
		switch {
		case c == ')' || c == ']' || c == '}':
			open := map[byte]byte{')': '(', ']': '[', '}': '{'}[c]
			depth, k := 0, j
			for ; k >= 0; k-- {
				ch := s[k]
				if ch == '\'' || ch == '"' {
					k = skipQuoteBack(s, k)
					continue
				}
				if ch == c {
					depth++
				} else if ch == open {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			j = k - 1
		case c == '\'' || c == '"':
			j = skipQuoteBack(s, j) - 1
		case isIdentByte(c):
			j--
		default:
			return j + 1
		}
	}
	return 0
}

// callEnd returns the index after the ")" matching the "(" at s[open]
func callEnd(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '\'', '"':
			q := s[i]
			for i++; i < len(s) && (s[i] != q || s[i-1] == '\\'); i++ {
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(s)
}

// nextOutsideQuotes finds the first occurrence of sep outside quotes at or
// after from; -1 when there is none
func nextOutsideQuotes(s, sep string, from int) int {
	for i := from; i < len(s); i++ {
		switch s[i] {
		case '\'', '"':
			q := s[i]
			for i++; i < len(s) && (s[i] != q || s[i-1] == '\\'); i++ {
			}
			continue
		}
		if strings.HasPrefix(s[i:], sep) {
			return i
		}
	}
	return -1
}

func trimRightSpace(s string, end int) int {
	for end >= 0 && s[end] == ' ' {
		end--
	}
	return end
}

// readName reads an identifier and an optional call after pos; it returns
// the name, the arguments (without parentheses) and where the whole ends
func readName(s string, pos int) (string, string, int) {
	for pos < len(s) && s[pos] == ' ' {
		pos++
	}
	start := pos
	for pos < len(s) && (s[pos] == '_' || (s[pos] >= 'a' && s[pos] <= 'z') || (s[pos] >= 'A' && s[pos] <= 'Z') || (s[pos] >= '0' && s[pos] <= '9')) {
		pos++
	}
	name := s[start:pos]
	args := ""
	if pos < len(s) && s[pos] == '(' {
		end := callEnd(s, pos)
		args = strings.TrimSpace(s[pos+1 : end-1])
		pos = end
	}
	return name, args, pos
}

// pipesToCalls rewrites filters into calls bound to their operand
func pipesToCalls(s string) string {
	from := 0
	for guard := 0; guard < 200; guard++ {
		i := nextOutsideQuotes(s, "|", from)
		if i < 0 {
			return s
		}
		if i+1 < len(s) && s[i+1] == '|' { // expr's ||
			from = i + 2
			continue
		}
		end := trimRightSpace(s, i-1)
		if end < 0 {
			return s
		}
		start := operandStart(s, end)
		name, args, after := readName(s, i+1)
		if name == "" || start > end {
			from = i + 1
			continue
		}
		call := name + "(" + s[start:end+1]
		if args != "" {
			call += ", " + args
		}
		call += ")"
		s = s[:start] + call + s[after:]
		from = 0
	}
	return s
}

// testsToCalls rewrites `X is [not] name(args)` into jinja_is calls; the
// defined/undefined tests are handled before
func testsToCalls(s string) string {
	from := 0
	for guard := 0; guard < 100; guard++ {
		i := nextOutsideQuotes(s, " is ", from)
		if i < 0 {
			return s
		}
		end := trimRightSpace(s, i-1)
		if end < 0 {
			return s
		}
		start := operandStart(s, end)
		pos := i + len(" is ")
		neg := false
		if strings.HasPrefix(s[pos:], "not ") {
			neg, pos = true, pos+len("not ")
		}
		name, args, after := readName(s, pos)
		if name == "" || start > end {
			from = i + 1
			continue
		}
		call := "jinja_is(" + s[start:end+1] + `, "` + name + `"`
		if args != "" {
			call += ", " + args
		}
		call += ")"
		if neg {
			call = "!" + call
		}
		s = s[:start] + call + s[after:]
		from = 0
	}
	return s
}

// notToBang gives the unary `not` Jinja's precedence (weaker than
// comparisons): `not a > 5` is `!(a > 5)`, up to the next and/or at the same
// depth. `x not in y` and `is not` are left alone.
func notToBang(s string) string {
	for guard := 0; guard < 100; guard++ {
		i := findUnaryNot(s)
		if i < 0 {
			return s
		}
		body := i + len("not ")
		end := len(s)
		depth := 0
	scan:
		for k := body; k < len(s); k++ {
			switch s[k] {
			case '\'', '"':
				q := s[k]
				for k++; k < len(s) && (s[k] != q || s[k-1] == '\\'); k++ {
				}
			case '(', '[', '{':
				depth++
			case ')', ']', '}':
				if depth == 0 {
					end = k
					break scan
				}
				depth--
			default:
				if depth == 0 && (strings.HasPrefix(s[k:], " and ") || strings.HasPrefix(s[k:], " or ")) {
					end = k
					break scan
				}
			}
		}
		s = s[:i] + "!(" + strings.TrimSpace(s[body:end]) + ")" + s[end:]
	}
	return s
}

// findUnaryNot returns the index of a `not ` used as a prefix operator
func findUnaryNot(s string) int {
	from := 0
	for {
		i := nextOutsideQuotes(s, "not ", from)
		if i < 0 {
			return -1
		}
		from = i + 1
		if i > 0 && isIdentByte(s[i-1]) {
			continue // part of a longer word
		}
		prev := trimRightSpace(s, i-1)
		if prev >= 0 {
			if isIdentByte(s[prev]) && !strings.HasSuffix(s[:prev+1], "and") && !strings.HasSuffix(s[:prev+1], "or") && !strings.HasSuffix(s[:prev+1], "not") {
				continue // `x not in y`, `is not`
			}
			if c := s[prev]; c == ')' || c == ']' || c == '\'' || c == '"' {
				continue
			}
		}
		return i
	}
}
