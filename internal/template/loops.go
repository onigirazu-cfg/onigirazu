package template

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
)

// Jinja for loops are expanded before anything else: the loop source is an
// expression, and the body is rendered once per item with the loop variable
// (and loop.index / index0 / first / last / length) added. As in Ansible's
// template module, the newline right after a for/endfor tag is dropped.

var (
	forTag = regexp.MustCompile(`\{%-?\s*for\s+(\w+)(?:\s*,\s*(\w+))?\s+in\s+(.+?)\s*-?%\}\n?`)
	// any for or endfor tag, to find the matching endfor of nested loops
	loopTag = regexp.MustCompile(`\{%-?\s*(for\s|endfor\s*-?%\})`)
	endTag  = regexp.MustCompile(`\{%-?\s*endfor\s*-?%\}\n?`)
	// `d.items()` and `d | dictsort` give key, value pairs
	itemsCall = regexp.MustCompile(`^(.+?)(?:\.items\(\)|\s*\|\s*dictsort)$`)
)

const loopMark = "\x01"

// expandLoops replaces every top-level for loop with a placeholder and
// returns the rendered loops
func (e *Engine) expandLoops(ctx context.Context, text string, variables map[string]interface{}) (string, []string, error) {
	var loops []string
	var out strings.Builder
	for {
		loc := forTag.FindStringSubmatchIndex(text)
		if loc == nil {
			out.WriteString(text)
			return out.String(), loops, nil
		}
		m := forTag.FindStringSubmatch(text)
		bodyStart := loc[1]
		bodyEnd, afterEnd, err := matchEndfor(text, bodyStart)
		if err != nil {
			return "", nil, err
		}
		rendered, err := e.renderLoop(ctx, m[1], m[2], m[3], text[bodyStart:bodyEnd], variables)
		if err != nil {
			return "", nil, err
		}
		out.WriteString(text[:loc[0]])
		fmt.Fprintf(&out, "%s%d%s", loopMark, len(loops), loopMark)
		loops = append(loops, rendered)
		text = text[afterEnd:]
	}
}

// matchEndfor returns where the body ends and where the text after the
// matching endfor tag starts
func matchEndfor(text string, from int) (int, int, error) {
	depth := 1
	pos := from
	for {
		loc := loopTag.FindStringIndex(text[pos:])
		if loc == nil {
			return 0, 0, fmt.Errorf("{%% for %%} without {%% endfor %%}")
		}
		start := pos + loc[0]
		if strings.Contains(text[start:pos+loc[1]], "endfor") {
			depth--
			if depth == 0 {
				end := endTag.FindStringIndex(text[start:])
				return start, start + end[1], nil
			}
		} else {
			depth++
		}
		pos += loc[1]
	}
}

func (e *Engine) renderLoop(ctx context.Context, name, valueName, source, body string, variables map[string]interface{}) (string, error) {
	pairs := false
	if m := itemsCall.FindStringSubmatch(strings.TrimSpace(source)); m != nil && valueName != "" {
		source, pairs = m[1], true
	}
	value, err := expression.Eval(source, variables)
	if err != nil {
		return "", fmt.Errorf("for loop over %q: %w", source, err)
	}

	type entry struct{ key, item interface{} }
	var entries []entry
	if dict, ok := value.(map[string]interface{}); ok && (pairs || valueName != "") {
		keys := make([]string, 0, len(dict))
		for k := range dict {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			entries = append(entries, entry{k, dict[k]})
		}
	} else {
		items, err := expression.Items(value)
		if err != nil {
			return "", fmt.Errorf("for loop over %q: %w", source, err)
		}
		for _, item := range items {
			entries = append(entries, entry{nil, item})
		}
	}

	var b strings.Builder
	for i, en := range entries {
		vars := make(map[string]interface{})
		for k, v := range variables {
			vars[k] = v
		}
		if valueName != "" {
			vars[name], vars[valueName] = en.key, en.item
		} else {
			vars[name] = en.item
		}
		vars["loop"] = map[string]interface{}{
			"index": i + 1, "index0": i, "length": len(entries),
			"first": i == 0, "last": i == len(entries)-1,
		}
		out, err := e.Render(ctx, body, vars)
		if err != nil {
			return "", err
		}
		b.WriteString(out)
	}
	return b.String(), nil
}

func restoreLoops(text string, loops []string) string {
	for i, v := range loops {
		text = strings.Replace(text, fmt.Sprintf("%s%d%s", loopMark, i, loopMark), v, 1)
	}
	return text
}

// setTag is {% set name = expression %}
var setTag = regexp.MustCompile(`\{%-?\s*set\s+(\w+)\s*=\s*(.+?)\s*-?%\}`)

// applySets evaluates the {% set %} tags of the top level in order and
// removes them; the variables they set hold for the rest of the text (for
// loops included). Tags inside a for loop belong to its body, which is
// rendered on its own for every item.
func applySets(text string, variables map[string]interface{}, trim bool) (string, map[string]interface{}, error) {
	if !setTag.MatchString(text) {
		return text, variables, nil
	}
	vars := make(map[string]interface{}, len(variables)+2)
	for k, v := range variables {
		vars[k] = v
	}
	var out strings.Builder
	for {
		set := setTag.FindStringSubmatchIndex(text)
		loop := forTag.FindStringIndex(text)
		if set == nil {
			out.WriteString(text)
			return out.String(), vars, nil
		}
		if len(loop) == 2 && loop[0] < set[0] {
			// keep the loop as it is: its sets are its own
			_, afterEnd, err := matchEndfor(text, loop[1])
			if err != nil {
				return "", nil, err
			}
			out.WriteString(text[:afterEnd])
			text = text[afterEnd:]
			continue
		}
		name, source := text[set[2]:set[3]], text[set[4]:set[5]]
		value, err := expression.Eval(source, vars)
		if err != nil {
			return "", nil, fmt.Errorf("set %s: %w", name, err)
		}
		vars[name] = value
		out.WriteString(text[:set[0]])
		end := set[1]
		if trim && end < len(text) && text[end] == '\n' {
			end++
		}
		text = text[end:]
	}
}
