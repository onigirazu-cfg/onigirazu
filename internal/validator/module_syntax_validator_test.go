package validator

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewModuleSyntaxValidator(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	if validator == nil {
		t.Fatal("NewModuleSyntaxValidator returned nil")
	}

	if len(validator.validModules) != 4 {
		t.Errorf("Expected 4 valid modules, got %d", len(validator.validModules))
	}

	for _, module := range modules {
		if !validator.validModules[module] {
			t.Errorf("Module %s not registered", module)
		}
	}
}

func TestValidateTaskModuleExists(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	task := &types.Task{
		Name:   "Test Task",
		Module: "ping",
	}

	err := validator.ValidateTaskModule(task, 0, 0)
	if err != nil {
		t.Errorf("Valid module should not error: %v", err)
	}
}

func TestValidateTaskModuleNotFound(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	task := &types.Task{
		Name:   "Test Task",
		Module: "invalid_module",
	}

	err := validator.ValidateTaskModule(task, 0, 0)
	if err == nil {
		t.Error("Invalid module should error")
	}

	syntaxErr, ok := err.(*ModuleSyntaxError)
	if !ok {
		t.Errorf("Error should be ModuleSyntaxError, got %T", err)
	}

	if syntaxErr.PlayIndex != 1 {
		t.Errorf("PlayIndex should be 1, got %d", syntaxErr.PlayIndex)
	}

	if syntaxErr.TaskIndex != 1 {
		t.Errorf("TaskIndex should be 1, got %d", syntaxErr.TaskIndex)
	}

	if syntaxErr.ErrorType != "UnknownModule" {
		t.Errorf("ErrorType should be UnknownModule, got %s", syntaxErr.ErrorType)
	}
}

func TestValidateTaskModuleEmptyModule(t *testing.T) {
	modules := []string{"ping", "file"}
	validator := NewModuleSyntaxValidator(modules)

	task := &types.Task{
		Name:   "Test Task",
		Module: "",
	}

	err := validator.ValidateTaskModule(task, 1, 2)
	if err == nil {
		t.Error("Empty module should error")
	}

	if !stringContains(err.Error(), "module not specified") {
		t.Errorf("Error message should mention module not specified: %v", err)
	}
}

func TestModuleSyntaxErrorFormat(t *testing.T) {
	err := &ModuleSyntaxError{
		PlayIndex:  1,
		TaskIndex:  2,
		TaskName:   "Install packages",
		ModuleName: "pakage",
		ErrorType:  "UnknownModule",
		Message:    "unknown module 'pakage'",
		Suggestion: "Did you mean 'package'?",
	}

	errorMsg := err.Error()
	if !stringContains(errorMsg, "play #1") {
		t.Errorf("Error message should contain play number: %s", errorMsg)
	}

	if !stringContains(errorMsg, "task #2") {
		t.Errorf("Error message should contain task number: %s", errorMsg)
	}

	if !stringContains(errorMsg, "Install packages") {
		t.Errorf("Error message should contain task name: %s", errorMsg)
	}

	if !stringContains(errorMsg, "Did you mean 'package'?") {
		t.Errorf("Error message should contain suggestion: %s", errorMsg)
	}
}

func TestFindSimilarModulesPing(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command", "shell", "service"}
	validator := NewModuleSyntaxValidator(modules)

	suggestions := validator.findSimilarModules("pong", 3)

	if len(suggestions) == 0 {
		t.Error("Should find similar module for 'pong'")
	}

	if suggestions[0] != "ping" {
		t.Errorf("Expected 'ping' as first suggestion, got %s", suggestions[0])
	}
}

func TestFindSimilarModulesPackage(t *testing.T) {
	modules := []string{"package", "file", "copy", "apt", "yum"}
	validator := NewModuleSyntaxValidator(modules)

	suggestions := validator.findSimilarModules("pakage", 3)

	if len(suggestions) == 0 {
		t.Error("Should find similar module for 'pakage'")
	}

	if suggestions[0] != "package" {
		t.Errorf("Expected 'package' as first suggestion, got %s", suggestions[0])
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"abc", "xbc", 1},
		{"kitten", "sitting", 3},
		{"ping", "pong", 1},
		{"package", "pakage", 1},
	}

	for _, tt := range tests {
		result := levenshteinDistance(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestValidatePlayTasks(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		Tasks: []types.Task{
			{Name: "Task 1", Module: "ping"},
			{Name: "Task 2", Module: "file"},
			{Name: "Task 3", Module: "copy"},
		},
	}

	err := validator.ValidatePlayTasks(play, 0)
	if err != nil {
		t.Errorf("Valid play should not error: %v", err)
	}
}

func TestValidatePlayTasksWithInvalid(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		Tasks: []types.Task{
			{Name: "Task 1", Module: "ping"},
			{Name: "Task 2", Module: "invalid"},
			{Name: "Task 3", Module: "copy"},
		},
	}

	err := validator.ValidatePlayTasks(play, 0)
	if err == nil {
		t.Error("Play with invalid module should error")
	}

	if !stringContains(err.Error(), "task #2") {
		t.Errorf("Error should reference task #2: %v", err)
	}
}

func TestValidatePlaybookModules(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Play 1",
				Hosts: "all",
				Tasks: []types.Task{
					{Name: "Task 1", Module: "ping"},
					{Name: "Task 2", Module: "file"},
				},
			},
			{
				Name:  "Play 2",
				Hosts: "web",
				Tasks: []types.Task{
					{Name: "Task 1", Module: "copy"},
					{Name: "Task 2", Module: "command"},
				},
			},
		},
	}

	err := validator.ValidatePlaybookModules(playbook)
	if err != nil {
		t.Errorf("Valid playbook should not error: %v", err)
	}
}

func TestValidatePlaybookModulesWithInvalid(t *testing.T) {
	modules := []string{"ping", "file", "copy", "command"}
	validator := NewModuleSyntaxValidator(modules)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Play 1",
				Hosts: "all",
				Tasks: []types.Task{
					{Name: "Task 1", Module: "ping"},
					{Name: "Task 2", Module: "invalid_module"},
				},
			},
		},
	}

	err := validator.ValidatePlaybookModules(playbook)
	if err == nil {
		t.Error("Playbook with invalid module should error")
	}
}

func TestValidateMultipleModules(t *testing.T) {
	// Test with all common modules
	modules := []string{
		"ping", "file", "copy", "command", "shell", "service",
		"package", "user", "group", "template", "git", "debug",
		"set_fact", "stat", "find", "lineinfile", "systemd",
		"docker_container", "docker_image", "apt", "yum",
	}
	validator := NewModuleSyntaxValidator(modules)

	for _, module := range modules {
		task := &types.Task{
			Name:   "Test " + module,
			Module: module,
		}

		err := validator.ValidateTaskModule(task, 0, 0)
		if err != nil {
			t.Errorf("Module %s should be valid: %v", module, err)
		}
	}
}

func TestModuleNameTrimming(t *testing.T) {
	modules := []string{"ping"}
	validator := NewModuleSyntaxValidator(modules)

	task := &types.Task{
		Name:   "Test Task",
		Module: "  ping  ", // With whitespace
	}

	err := validator.ValidateTaskModule(task, 0, 0)
	if err != nil {
		t.Errorf("Module with whitespace should be trimmed: %v", err)
	}
}

func TestSuggestionMessage(t *testing.T) {
	modules := []string{"ping", "file", "package", "service"}
	validator := NewModuleSyntaxValidator(modules)

	tests := []struct {
		input          string
		shouldSuggest  bool
		expectedModule string
	}{
		{"pong", true, "ping"},
		{"pakage", true, "package"},
		{"servis", true, "service"},
		{"xyz", false, ""},
	}

	for _, tt := range tests {
		task := &types.Task{
			Name:   "Test",
			Module: tt.input,
		}

		err := validator.ValidateTaskModule(task, 0, 0)
		if err == nil {
			t.Errorf("Module %s should error", tt.input)
			continue
		}

		suggestion := validator.suggestModuleName(tt.input)
		if tt.shouldSuggest && suggestion == "" {
			t.Errorf("Module %s should have suggestion", tt.input)
		}

		if tt.shouldSuggest && !stringContains(suggestion, tt.expectedModule) {
			t.Errorf("Module %s suggestion should contain %s, got: %s", tt.input, tt.expectedModule, suggestion)
		}
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
