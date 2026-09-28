package expression

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Lookups run on the control machine, as in Ansible. lookup() joins a list
// result with commas; query() (and q()) returns the list. Relative paths
// start from playbook_dir.

// lookupVars names the variables a lookup sees in the expression env
const lookupVars = "__lookup_vars"

// RenderTemplate renders a Jinja template with variables; the template
// package sets it (lookup('template'))
var RenderTemplate func(text string, vars map[string]interface{}) (string, error)

func lookupItems(base, varsArg interface{}, args []interface{}) ([]interface{}, error) {
	vars, _ := varsArg.(map[string]interface{})
	if len(args) == 0 {
		return nil, fmt.Errorf("lookup needs a plugin name")
	}
	dir, _ := base.(string)
	resolve := func(p string) string {
		if filepath.IsAbs(p) || dir == "" {
			return p
		}
		return filepath.Join(dir, p)
	}
	plugin, terms := fmt.Sprint(args[0]), args[1:]
	// lookup('ansible.builtin.env', ...) is lookup('env', ...)
	plugin = strings.TrimPrefix(plugin, "ansible.builtin.")
	var out []interface{}
	switch plugin {
	case "env":
		for _, t := range terms {
			out = append(out, os.Getenv(fmt.Sprint(t)))
		}
	case "template":
		if RenderTemplate == nil {
			return nil, fmt.Errorf("lookup template: no template engine")
		}
		for _, t := range terms {
			data, err := os.ReadFile(resolve(fmt.Sprint(t))) // #nosec G304 -- the playbook's own template
			if err != nil {
				return nil, fmt.Errorf("lookup template: %w", err)
			}
			text, err := RenderTemplate(string(data), vars)
			if err != nil {
				return nil, fmt.Errorf("lookup template %s: %w", t, err)
			}
			out = append(out, text)
		}
	case "file":
		for _, t := range terms {
			data, err := os.ReadFile(resolve(fmt.Sprint(t))) // #nosec G304 -- the playbook names the file
			if err != nil {
				return nil, fmt.Errorf("lookup file: %w", err)
			}
			out = append(out, strings.TrimRight(string(data), "\n"))
		}
	case "pipe", "lines":
		for _, t := range terms {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprint(t)) // #nosec G204 -- the playbook's own command
			cmd.Dir = dir
			data, err := cmd.Output()
			cancel()
			if err != nil {
				return nil, fmt.Errorf("lookup %s %q: %w", plugin, t, err)
			}
			text := strings.TrimRight(string(data), "\n")
			if plugin == "pipe" {
				out = append(out, text)
			} else if text != "" {
				for _, line := range strings.Split(text, "\n") {
					out = append(out, line)
				}
			}
		}
	case "fileglob":
		for _, t := range terms {
			matches, err := filepath.Glob(resolve(fmt.Sprint(t)))
			if err != nil {
				return nil, err
			}
			sort.Strings(matches)
			for _, m := range matches {
				if info, err := os.Stat(m); err == nil && !info.IsDir() {
					out = append(out, m)
				}
			}
		}
	case "first_found":
		var candidates []interface{}
		for _, t := range terms {
			if items, err := Items(t); err == nil && isListValue(t) {
				candidates = append(candidates, items...)
			} else {
				candidates = append(candidates, t)
			}
		}
		for _, c := range candidates {
			if p := resolve(fmt.Sprint(c)); fileExists(p) {
				return []interface{}{p}, nil
			}
		}
		return nil, fmt.Errorf("lookup first_found: none of %v exists", candidates)
	case "dict":
		for _, t := range terms {
			items, err := Dict2Items(t)
			if err != nil {
				return nil, err
			}
			out = append(out, items...)
		}
	case "items", "list":
		for _, t := range terms {
			if isListValue(t) {
				items, _ := Items(t)
				out = append(out, items...)
			} else {
				out = append(out, t)
			}
		}
	case "nested":
		return nestedItems(terms)
	case "together":
		return togetherItems(terms)
	case "subelements":
		return subelementItems(terms)
	case "indexed_items":
		for i, item := range flattenTerms(terms) {
			out = append(out, []interface{}{i, item})
		}
	case "random_choice":
		items := flattenTerms(terms)
		if len(items) == 0 {
			return []interface{}{}, nil
		}
		return []interface{}{items[rand.IntN(len(items))]}, nil // #nosec G404 -- not for security
	default:
		return nil, fmt.Errorf("lookup plugin %q is not supported", plugin)
	}
	return out, nil
}

// LookupItems runs a lookup as query() does: with_<plugin> loops get their
// items from it. dir is where relative paths start.
func LookupItems(dir string, vars map[string]interface{}, plugin string, terms []interface{}) ([]interface{}, error) {
	return lookupItems(dir, vars, append([]interface{}{plugin}, terms...))
}

// flattenTerms is the terms with lists opened one level
func flattenTerms(terms []interface{}) []interface{} {
	var out []interface{}
	for _, t := range terms {
		if isListValue(t) {
			items, _ := Items(t)
			out = append(out, items...)
		} else {
			out = append(out, t)
		}
	}
	return out
}

// termLists reads every term as a list (a single value is a list of one)
func termLists(terms []interface{}) [][]interface{} {
	lists := make([][]interface{}, len(terms))
	for i, t := range terms {
		if isListValue(t) {
			lists[i], _ = Items(t)
		} else {
			lists[i] = []interface{}{t}
		}
	}
	return lists
}

// nestedItems is every combination of the lists, the first list outermost
func nestedItems(terms []interface{}) ([]interface{}, error) {
	lists := termLists(terms)
	if len(lists) == 0 {
		return nil, fmt.Errorf("lookup nested: no lists")
	}
	combos := [][]interface{}{{}}
	for _, list := range lists {
		var next [][]interface{}
		for _, c := range combos {
			for _, v := range list {
				next = append(next, append(append([]interface{}{}, c...), v))
			}
		}
		combos = next
	}
	out := make([]interface{}, len(combos))
	for i, c := range combos {
		out[i] = c
	}
	return out, nil
}

// togetherItems zips the lists; a shorter list gives none (nil)
func togetherItems(terms []interface{}) ([]interface{}, error) {
	lists := termLists(terms)
	longest := 0
	for _, l := range lists {
		longest = max(longest, len(l))
	}
	out := make([]interface{}, 0, longest)
	for i := 0; i < longest; i++ {
		row := make([]interface{}, len(lists))
		for j, l := range lists {
			if i < len(l) {
				row[j] = l[i]
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// subelementItems pairs every element of a list of dicts with each item of
// one of its lists: [element, subitem]. A third term {skip_missing: true}
// skips elements without the key.
func subelementItems(terms []interface{}) ([]interface{}, error) {
	if len(terms) < 2 {
		return nil, fmt.Errorf("lookup subelements: needs a list and a key")
	}
	elements, err := Items(terms[0])
	if err != nil {
		return nil, fmt.Errorf("lookup subelements: %w", err)
	}
	key := fmt.Sprint(terms[1])
	skipMissing := false
	if len(terms) > 2 {
		if flags, ok := terms[2].(map[string]interface{}); ok {
			skipMissing = Truthy(flags["skip_missing"])
		}
	}
	var out []interface{}
	for _, e := range elements {
		value := interface{}(nil)
		if m, ok := e.(map[string]interface{}); ok {
			value = m
			for _, part := range strings.Split(key, ".") {
				mm, ok := value.(map[string]interface{})
				if !ok {
					value = nil
					break
				}
				value = mm[part]
			}
		}
		if value == nil {
			if skipMissing {
				continue
			}
			return nil, fmt.Errorf("lookup subelements: %q is missing in an element", key)
		}
		subs, err := Items(value)
		if err != nil {
			return nil, fmt.Errorf("lookup subelements: %q is not a list", key)
		}
		for _, sub := range subs {
			out = append(out, []interface{}{e, sub})
		}
	}
	return out, nil
}

func isListValue(v interface{}) bool {
	_, err := Items(v)
	_, isString := v.(string)
	return err == nil && v != nil && !isString
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// jinjaLookup is lookup(): one value, or the values joined with commas
func jinjaLookup(p ...interface{}) (interface{}, error) {
	items, err := lookupItems(p[0], p[1], p[2:])
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		return items[0], nil
	}
	parts := make([]string, len(items))
	for i, v := range items {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, ","), nil
}

func jinjaQuery(p ...interface{}) (interface{}, error) {
	return lookupItems(p[0], p[1], p[2:])
}
