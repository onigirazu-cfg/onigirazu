package types

import (
	"fmt"
	"regexp"
	"strings"
)

// Ansible short forms of a task's module:
//
//	command: echo hi chdir=/tmp     free form, with the module's own key=value options
//	file: path=/tmp/x state=touch   key=value pairs
//	ping:                           no arguments
//	command: make                   together with "args: {chdir: /src}"

// freeFormModules take their main argument as a string
var freeFormModules = map[string]string{"command": "cmd", "shell": "cmd", "raw": "cmd", "script": "script", "meta": "free_form", "include_vars": "file"}

// options of the free-form modules that may be written inline as key=value
var inlineOption = regexp.MustCompile(`(^|\s)(chdir|creates|removes|executable|stdin)=("[^"]*"|'[^']*'|\S+)`)

// applyShortForm finds the module of a task written in a short form; it does
// nothing when the module is already known or the task has no single
// module key
func (t *Task) applyShortForm(taskMap map[string]interface{}, reserved map[string]bool) error {
	if local, ok := taskMap["local_action"]; ok {
		return t.applyLocalAction(local)
	}
	if t.Module != "" {
		return nil
	}
	var key string
	for k := range taskMap {
		if reserved[k] {
			continue
		}
		if key != "" {
			return nil // more than one candidate: let validation report it
		}
		key = k
	}
	if key == "" {
		return nil
	}

	args := map[string]interface{}{}
	switch v := taskMap[key].(type) {
	case nil:
	case map[string]interface{}:
		for k, val := range v {
			args[k] = val
		}
	case string:
		parsed, err := shortFormArgs(key, v)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		args = parsed
	default:
		return nil
	}
	// "args:" adds to (and overrides) the short form
	if extra, ok := taskMap["args"].(map[string]interface{}); ok {
		for k, val := range extra {
			args[k] = val
		}
	}
	t.Module, t.Args = key, args
	return nil
}

func shortFormArgs(module, s string) (map[string]interface{}, error) {
	args := map[string]interface{}{}
	if main, ok := freeFormModules[module]; ok {
		for _, m := range inlineOption.FindAllStringSubmatch(s, -1) {
			args[m[2]] = unquote(m[3])
		}
		rest := strings.TrimSpace(inlineOption.ReplaceAllString(s, "$1"))
		if main == "script" {
			path, scriptArgs, _ := strings.Cut(rest, " ")
			args["script"] = path
			if scriptArgs = strings.TrimSpace(scriptArgs); scriptArgs != "" {
				args["args"] = scriptArgs
			}
		} else if rest != "" {
			args[main] = rest
		}
		return args, nil
	}
	words, err := splitWords(s)
	if err != nil {
		return nil, err
	}
	for _, w := range words {
		k, v, ok := strings.Cut(w, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("expected key=value, got %q", w)
		}
		args[k] = v
	}
	return args, nil
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// splitWords splits on whitespace outside quotes and {{ }} and removes the
// quotes: msg="hello world" is one word, msg={{ a }} too
func splitWords(s string) ([]string, error) {
	var words []string
	var b strings.Builder
	var quote byte
	inWord, depth := false, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				b.WriteByte(c)
			}
		case (c == '"' || c == '\'') && depth == 0:
			quote, inWord = c, true
		case strings.HasPrefix(s[i:], "{{"):
			depth++
			b.WriteString("{{")
			i++
			inWord = true
		case strings.HasPrefix(s[i:], "}}") && depth > 0:
			depth--
			b.WriteString("}}")
			i++
		case (c == ' ' || c == '\t' || c == '\n') && depth == 0:
			if inWord {
				words = append(words, b.String())
				b.Reset()
				inWord = false
			}
		default:
			b.WriteByte(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in %q", s)
	}
	if inWord {
		words = append(words, b.String())
	}
	return words, nil
}

// applyLocalAction reads "local_action: command echo hi" or
// "local_action: {module: copy, ...}": the module runs on the control
// machine, as with delegate_to: localhost
func (t *Task) applyLocalAction(v interface{}) error {
	switch a := v.(type) {
	case string:
		module, rest, _ := strings.Cut(strings.TrimSpace(a), " ")
		args, err := shortFormArgs(module, strings.TrimSpace(rest))
		if err != nil {
			return fmt.Errorf("local_action: %w", err)
		}
		t.Module, t.Args = module, args
	case map[string]interface{}:
		module, _ := a["module"].(string)
		if module == "" {
			return fmt.Errorf("local_action: module is required")
		}
		t.Module, t.Args = module, map[string]interface{}{}
		for k, val := range a {
			if k != "module" {
				t.Args[k] = val
			}
		}
	default:
		return fmt.Errorf("local_action: expected a string or a map, got %T", v)
	}
	t.DelegateTo = "localhost"
	return nil
}
