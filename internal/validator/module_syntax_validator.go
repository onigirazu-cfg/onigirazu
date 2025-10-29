package validator

import (
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ModuleSyntaxValidator validates module syntax and existence
type ModuleSyntaxValidator struct {
	validModules map[string]bool
}

// NewModuleSyntaxValidator creates a new module syntax validator
func NewModuleSyntaxValidator(availableModules []string) *ModuleSyntaxValidator {
	validator := &ModuleSyntaxValidator{
		validModules: make(map[string]bool),
	}

	for _, module := range availableModules {
		validator.validModules[module] = true
	}

	return validator
}

// ValidateTaskModule validates that a task's module exists and is valid
func (m *ModuleSyntaxValidator) ValidateTaskModule(task *types.Task, playIndex, taskIndex int) error {
	if task.Module == "" {
		return fmt.Errorf("play #%d, task #%d (%s): module not specified", playIndex+1, taskIndex+1, task.Name)
	}

	// Extract base module name (remove any underscore variations)
	moduleName := strings.TrimSpace(task.Module)

	// Check if module exists
	if !m.validModules[moduleName] {
		return &ModuleSyntaxError{
			PlayIndex:  playIndex + 1,
			TaskIndex:  taskIndex + 1,
			TaskName:   task.Name,
			ModuleName: moduleName,
			ErrorType:  "UnknownModule",
			Message:    fmt.Sprintf("unknown module '%s'", moduleName),
			Suggestion: m.suggestModuleName(moduleName),
		}
	}

	return nil
}

// ValidatePlayTasks validates all tasks in a play
func (m *ModuleSyntaxValidator) ValidatePlayTasks(play *types.Play, playIndex int) error {
	for i, task := range play.Tasks {
		if err := m.ValidateTaskModule(&task, playIndex, i); err != nil {
			return err
		}
	}

	return nil
}

// ValidatePlaybookModules validates all modules in playbook
func (m *ModuleSyntaxValidator) ValidatePlaybookModules(playbook *types.Playbook) error {
	for i, play := range playbook.Plays {
		if err := m.ValidatePlayTasks(&play, i); err != nil {
			return err
		}
	}

	return nil
}

// suggestModuleName suggests similar module names for misspelled modules
func (m *ModuleSyntaxValidator) suggestModuleName(moduleName string) string {
	suggestions := m.findSimilarModules(moduleName, 3)
	if len(suggestions) == 0 {
		return ""
	}

	if len(suggestions) == 1 {
		return fmt.Sprintf("Did you mean '%s'?", suggestions[0])
	}

	return fmt.Sprintf("Did you mean one of: %s?", strings.Join(suggestions, ", "))
}

// findSimilarModules finds modules similar to the given name (Levenshtein distance)
func (m *ModuleSyntaxValidator) findSimilarModules(moduleName string, maxDistance int) []string {
	var suggestions []string

	for validModule := range m.validModules {
		distance := levenshteinDistance(moduleName, validModule)
		if distance <= maxDistance && distance > 0 {
			suggestions = append(suggestions, validModule)
		}
	}

	// Sort by distance (closer matches first)
	// Simple bubble sort for small arrays
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if levenshteinDistance(moduleName, suggestions[i]) > levenshteinDistance(moduleName, suggestions[j]) {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	if a == b {
		return 0
	}

	d := make([][]int, len(a)+1)
	for i := range d {
		d[i] = make([]int, len(b)+1)
		d[i][0] = i
	}

	for j := 0; j <= len(b); j++ {
		d[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			d[i][j] = min(
				d[i-1][j]+1,                        // deletion
				min(d[i][j-1]+1, d[i-1][j-1]+cost), // insertion or substitution
			)
		}
	}

	return d[len(a)][len(b)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ModuleSyntaxError represents a module syntax validation error
type ModuleSyntaxError struct {
	PlayIndex  int
	TaskIndex  int
	TaskName   string
	ModuleName string
	ErrorType  string
	Message    string
	Suggestion string
}

// Error implements the error interface
func (e *ModuleSyntaxError) Error() string {
	msg := fmt.Sprintf("play #%d, task #%d (%s): %s", e.PlayIndex, e.TaskIndex, e.TaskName, e.Message)
	if e.Suggestion != "" {
		msg += fmt.Sprintf(" - %s", e.Suggestion)
	}
	return msg
}
