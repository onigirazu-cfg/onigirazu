package formatter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// YAMLFormatter provides custom YAML formatting with proper indentation
type YAMLFormatter struct {
	IndentSize int
}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{
		IndentSize: 2,
	}
}

// FormatTask formats a task with proper indentation for module parameters
func (f *YAMLFormatter) FormatTask(task *types.Task, baseIndent int) (string, error) {
	var lines []string
	indent := strings.Repeat(" ", baseIndent)

	// Task name
	if task.Name != "" {
		lines = append(lines, fmt.Sprintf("%s- name: %q", indent, task.Name))
	} else {
		lines = append(lines, fmt.Sprintf("%s-", indent))
	}

	// Module name
	if task.Module != "" {
		lines = append(lines, fmt.Sprintf("%s  module: %q", indent, task.Module))
	}

	// Module parameters with proper indentation
	if len(task.Args) > 0 {
		// Sort keys for consistent output
		var keys []string
		for key := range task.Args {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Add comment to separate module parameters
		lines = append(lines, fmt.Sprintf("%s  # Module parameters", indent))

		for _, key := range keys {
			value := task.Args[key]
			// Add extra indentation for module parameters (2 spaces instead of 4)
			formattedValue, err := f.formatValue(value, baseIndent+f.IndentSize*2)
			if err != nil {
				return "", fmt.Errorf("failed to format value for key %s: %w", key, err)
			}
			if strings.HasPrefix(formattedValue, "\n") {
				// For arrays and objects that start with newline
				lines = append(lines, fmt.Sprintf("%s  %s:%s", indent, key, formattedValue))
			} else {
				lines = append(lines, fmt.Sprintf("%s  %s: %s", indent, key, formattedValue))
			}
		}
	}

	// Task-level parameters
	taskLevelAdded := false

	// Helper function to add task-level comment
	addTaskLevelComment := func() {
		if !taskLevelAdded {
			lines = append(lines, fmt.Sprintf("%s  # Task parameters", indent))
			taskLevelAdded = true
		}
	}

	if task.When != "" {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  when: %q", indent, task.When))
	}

	if task.Register != "" {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  register: %q", indent, task.Register))
	}

	if task.IgnoreErrors {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  ignore_errors: true", indent))
	}

	if task.Timeout > 0 {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  timeout: %q", indent, task.Timeout.String()))
	}

	if task.Retries > 0 {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  retries: %d", indent, task.Retries))
	}

	if task.Delay > 0 {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  delay: %q", indent, task.Delay.String()))
	}

	if task.Until != "" {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  until: %q", indent, task.Until))
	}

	if task.ChangedWhen != "" {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  changed_when: %q", indent, task.ChangedWhen))
	}

	if task.FailedWhen != "" {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  failed_when: %q", indent, task.FailedWhen))
	}

	if task.Include != "" {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  include: %q", indent, task.Include))
	}

	if task.Serial {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  serial: true", indent))
	}

	if task.RetryDelay > 0 {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  retry_delay: %q", indent, task.RetryDelay.String()))
	}

	if len(task.Tags) > 0 {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  tags:", indent))
		for _, tag := range task.Tags {
			lines = append(lines, fmt.Sprintf("%s    - %q", indent, tag))
		}
	}

	if len(task.Notify) > 0 {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  notify:", indent))
		for _, notify := range task.Notify {
			lines = append(lines, fmt.Sprintf("%s    - %q", indent, notify))
		}
	}

	if task.Loop != nil {
		addTaskLevelComment()
		lines = append(lines, fmt.Sprintf("%s  loop:", indent))
		if task.Loop.Items != nil {
			lines = append(lines, fmt.Sprintf("%s    items:", indent))
			for _, item := range task.Loop.Items {
				formattedItem, err := f.formatValue(item, baseIndent+f.IndentSize*4)
				if err != nil {
					return "", fmt.Errorf("failed to format loop item: %w", err)
				}
				if strings.HasPrefix(formattedItem, "\n") {
					// For objects that start with newline
					lines = append(lines, fmt.Sprintf("%s      -%s", indent, formattedItem))
				} else {
					lines = append(lines, fmt.Sprintf("%s      - %s", indent, formattedItem))
				}
			}
		}
		if task.Loop.Variable != "" {
			lines = append(lines, fmt.Sprintf("%s    variable: %q", indent, task.Loop.Variable))
		}
	}

	return strings.Join(lines, "\n"), nil
}

// formatValue formats a value with proper YAML formatting
func (f *YAMLFormatter) formatValue(value interface{}, indent int) (string, error) {
	if value == nil {
		return "null", nil
	}

	switch v := value.(type) {
	case string:
		// Always quote strings for consistency with expected output
		return fmt.Sprintf("%q", v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%g", v), nil
	case time.Duration:
		return fmt.Sprintf("%q", v.String()), nil
	case []interface{}:
		if len(v) == 0 {
			return "[]", nil
		}
		var lines []string
		baseIndent := strings.Repeat(" ", indent)
		for _, item := range v {
			formattedItem, err := f.formatValue(item, indent+f.IndentSize)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("%s- %s", baseIndent, formattedItem))
		}
		return "\n" + strings.Join(lines, "\n"), nil
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}", nil
		}
		var lines []string
		baseIndent := strings.Repeat(" ", indent)

		// Sort keys for consistent output
		var keys []string
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			formattedValue, err := f.formatValue(v[key], indent+f.IndentSize)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("%s%s: %s", baseIndent, key, formattedValue))
		}
		return "\n" + strings.Join(lines, "\n"), nil
	default:
		// Fallback to standard YAML marshaling
		data, err := yaml.Marshal(value)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
}

// needsQuoting determines if a string needs to be quoted in YAML
func (f *YAMLFormatter) needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	// Check for YAML special values
	switch strings.ToLower(s) {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return true
	}

	// Check if it's a pure number
	if strings.ContainsAny(s, "0123456789") {
		// If it starts with 0 and has more digits, it's octal
		if strings.HasPrefix(s, "0") && len(s) > 1 {
			return true
		}
		// If it contains only digits, it's a number
		allDigits := true
		for _, r := range s {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
		// If it contains a dot, it might be a float
		if strings.Contains(s, ".") {
			return true
		}
	}

	// Check for special characters that might need quoting
	if strings.ContainsAny(s, ":{}[]|>*&!%#`@,") {
		return true
	}

	// Check for leading/trailing whitespace
	if strings.TrimSpace(s) != s {
		return true
	}

	// Check for template variables
	if strings.Contains(s, "{{") && strings.Contains(s, "}}") {
		return true
	}

	return false
}

// FormatPlaybook formats an entire playbook with proper indentation
func (f *YAMLFormatter) FormatPlaybook(playbook *types.Playbook) (string, error) {
	var lines []string

	if playbook.Name != "" {
		lines = append(lines, fmt.Sprintf("name: %q", playbook.Name))
		lines = append(lines, "")
	}

	if len(playbook.Vars) > 0 {
		lines = append(lines, "vars:")
		// Sort keys for consistent output
		var keys []string
		for key := range playbook.Vars {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := playbook.Vars[key]
			formattedValue, err := f.formatValue(value, f.IndentSize*2) // 4 spaces for nested content
			if err != nil {
				return "", fmt.Errorf("failed to format playbook var %s: %w", key, err)
			}
			if strings.HasPrefix(formattedValue, "\n") {
				// For arrays and objects that start with newline
				lines = append(lines, fmt.Sprintf("  %s:%s", key, formattedValue))
			} else {
				lines = append(lines, fmt.Sprintf("  %s: %s", key, formattedValue))
			}
		}
		lines = append(lines, "")
	}

	if len(playbook.Plays) > 0 {
		lines = append(lines, "plays:")
		for i, play := range playbook.Plays {
			if i > 0 {
				lines = append(lines, "")
			}
			playLines, err := f.FormatPlay(&play)
			if err != nil {
				return "", fmt.Errorf("failed to format play %d: %w", i, err)
			}
			lines = append(lines, playLines)
		}
	}

	return strings.Join(lines, "\n"), nil
}

// FormatPlay formats a play with proper indentation
func (f *YAMLFormatter) FormatPlay(play *types.Play) (string, error) {
	var lines []string
	indent := "  "

	if play.Name != "" {
		lines = append(lines, fmt.Sprintf("%s- name: %q", indent, play.Name))
	} else {
		lines = append(lines, fmt.Sprintf("%s-", indent))
	}

	if play.Hosts != "" {
		lines = append(lines, fmt.Sprintf("%s  hosts: %q", indent, play.Hosts))
	}

	if len(play.Vars) > 0 {
		lines = append(lines, fmt.Sprintf("%s  vars:", indent))
		for key, value := range play.Vars {
			formattedValue, err := f.formatValue(value, len(indent)+f.IndentSize*2)
			if err != nil {
				return "", fmt.Errorf("failed to format play var %s: %w", key, err)
			}
			lines = append(lines, fmt.Sprintf("%s    %s: %s", indent, key, formattedValue))
		}
	}

	if len(play.Tasks) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("%s  tasks:", indent))
		for i, task := range play.Tasks {
			if i > 0 {
				lines = append(lines, "")
			}
			taskLines, err := f.FormatTask(&task, len(indent)+f.IndentSize*2) // 6 spaces for tasks
			if err != nil {
				return "", fmt.Errorf("failed to format task %d: %w", i, err)
			}
			lines = append(lines, taskLines)
		}
	}

	return strings.Join(lines, "\n"), nil
}
