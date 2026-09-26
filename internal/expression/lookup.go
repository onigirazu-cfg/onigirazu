package expression

import (
	"context"
	"fmt"
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

func lookupItems(base interface{}, args []interface{}) ([]interface{}, error) {
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
	var out []interface{}
	switch plugin {
	case "env":
		for _, t := range terms {
			out = append(out, os.Getenv(fmt.Sprint(t)))
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
	default:
		return nil, fmt.Errorf("lookup plugin %q is not supported", plugin)
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
	items, err := lookupItems(p[0], p[1:])
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
	return lookupItems(p[0], p[1:])
}
