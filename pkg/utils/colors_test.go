package utils

import (
	"os"
	"strings"
	"testing"
)

func TestNewColorScheme(t *testing.T) {
	scheme := NewColorScheme()

	if scheme.Success == nil {
		t.Error("Expected Success color function to be set")
	}
	if scheme.Error == nil {
		t.Error("Expected Error color function to be set")
	}
	if scheme.Warning == nil {
		t.Error("Expected Warning color function to be set")
	}
	if scheme.Info == nil {
		t.Error("Expected Info color function to be set")
	}
}

func TestMakeColorFunc(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled

	// Test with colors enabled
	ColorsEnabled = true
	colorFunc := makeColorFunc(Red)
	result := colorFunc("test")
	if !strings.Contains(result, "test") {
		t.Error("Expected result to contain 'test'")
	}
	if !strings.Contains(result, Red) {
		t.Error("Expected result to contain color code")
	}
	if !strings.Contains(result, Reset) {
		t.Error("Expected result to contain reset code")
	}

	// Test with colors disabled
	ColorsEnabled = false
	result = colorFunc("test")
	if result != "test" {
		t.Errorf("Expected 'test', got '%s'", result)
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestEnableColors(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled

	EnableColors(true)
	if !ColorsEnabled {
		t.Error("Expected colors to be enabled")
	}

	EnableColors(false)
	if ColorsEnabled {
		t.Error("Expected colors to be disabled")
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestColorize(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled
	ColorsEnabled = true

	tests := []struct {
		name            string
		text            string
		colorType       string
		shouldHaveColor bool
	}{
		{"success", "test", "success", true},
		{"ok", "test", "ok", true},
		{"error", "test", "error", true},
		{"failed", "test", "failed", true},
		{"warning", "test", "warning", true},
		{"warn", "test", "warn", true},
		{"info", "test", "info", true},
		{"debug", "test", "debug", true},
		{"changed", "test", "changed", true},
		{"skipped", "test", "skipped", true},
		{"header", "test", "header", true},
		{"highlight", "test", "highlight", true},
		{"dim", "test", "dim", true},
		{"bold", "test", "bold", true},
		{"task", "test", "task", true},
		{"host", "test", "host", true},
		{"module", "test", "module", true},
		{"play", "test", "play", true},
		{"unknown", "test", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Colorize(tt.text, tt.colorType)
			if !strings.Contains(result, tt.text) {
				t.Errorf("Expected result to contain '%s'", tt.text)
			}
			if tt.shouldHaveColor && result == tt.text {
				t.Errorf("Expected result to have color applied")
			}
			if !tt.shouldHaveColor && result != tt.text {
				t.Errorf("Expected result to be unchanged for unknown type")
			}
		})
	}

	// Test with colors disabled
	ColorsEnabled = false
	result := Colorize("test", "success")
	if result != "test" {
		t.Errorf("Expected 'test' when colors disabled, got '%s'", result)
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestGetStatusSymbol(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled
	ColorsEnabled = true

	tests := []struct {
		name   string
		status string
		symbol string
	}{
		{"success", "success", SuccessSymbol},
		{"ok", "ok", SuccessSymbol},
		{"error", "error", ErrorSymbol},
		{"failed", "failed", ErrorSymbol},
		{"warning", "warning", WarningSymbol},
		{"warn", "warn", WarningSymbol},
		{"info", "info", InfoSymbol},
		{"changed", "changed", ChangedSymbol},
		{"skipped", "skipped", SkippedSymbol},
		{"unknown", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusSymbol(tt.status)
			if tt.status == "unknown" {
				if result != tt.status {
					t.Errorf("Expected '%s', got '%s'", tt.status, result)
				}
			} else {
				if !strings.Contains(result, tt.symbol) {
					t.Errorf("Expected result to contain symbol '%s'", tt.symbol)
				}
			}
		})
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestFormatTaskResult(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled
	ColorsEnabled = true

	tests := []struct {
		name     string
		taskName string
		host     string
		status   string
		changed  bool
		contains []string
	}{
		{
			name:     "success without change",
			taskName: "install package",
			host:     "server1",
			status:   "success",
			changed:  false,
			contains: []string{"install package", "server1", SuccessSymbol},
		},
		{
			name:     "success with change",
			taskName: "update config",
			host:     "server2",
			status:   "success",
			changed:  true,
			contains: []string{"update config", "server2", SuccessSymbol, "changed"},
		},
		{
			name:     "failed",
			taskName: "deploy app",
			host:     "server3",
			status:   "failed",
			changed:  false,
			contains: []string{"deploy app", "server3", ErrorSymbol},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTaskResult(tt.taskName, tt.host, tt.status, tt.changed)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
				}
			}
		})
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestFormatPlayHeader(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled
	ColorsEnabled = true

	result := FormatPlayHeader("Deploy Application", 5)

	if !strings.Contains(result, "Deploy Application") {
		t.Error("Expected result to contain play name")
	}
	if !strings.Contains(result, "5 hosts") {
		t.Error("Expected result to contain host count")
	}
	if !strings.Contains(result, "PLAY") {
		t.Error("Expected result to contain 'PLAY'")
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestFormatTaskHeader(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled
	ColorsEnabled = true

	result := FormatTaskHeader("Install nginx", "apt")

	if !strings.Contains(result, "Install nginx") {
		t.Error("Expected result to contain task name")
	}
	if !strings.Contains(result, "apt") {
		t.Error("Expected result to contain module name")
	}
	if !strings.Contains(result, "TASK") {
		t.Error("Expected result to contain 'TASK'")
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestProgressBar(t *testing.T) {
	// Save original state
	origEnabled := ColorsEnabled

	// Test with colors enabled
	ColorsEnabled = true
	result := ProgressBar(5, 10, 20)
	if !strings.Contains(result, "5/10") {
		t.Error("Expected result to contain progress numbers")
	}
	if !strings.Contains(result, "50.0%") {
		t.Error("Expected result to contain percentage")
	}

	// Test with colors disabled
	ColorsEnabled = false
	result = ProgressBar(5, 10, 20)
	if result != "[5/10]" {
		t.Errorf("Expected '[5/10]', got '%s'", result)
	}

	// Test with zero width
	ColorsEnabled = true
	result = ProgressBar(5, 10, 0)
	if result != "[5/10]" {
		t.Errorf("Expected '[5/10]' for zero width, got '%s'", result)
	}

	// Test edge cases
	result = ProgressBar(0, 10, 20)
	if !strings.Contains(result, "0/10") {
		t.Error("Expected result to contain '0/10'")
	}

	result = ProgressBar(10, 10, 20)
	if !strings.Contains(result, "10/10") {
		t.Error("Expected result to contain '10/10'")
	}
	if !strings.Contains(result, "100.0%") {
		t.Error("Expected result to contain '100.0%'")
	}

	// Restore original state
	ColorsEnabled = origEnabled
}

func TestShouldEnableColors(t *testing.T) {
	// Save original environment
	origNoColor := os.Getenv("NO_COLOR")
	origForceColor := os.Getenv("FORCE_COLOR")

	// Test NO_COLOR
	os.Setenv("NO_COLOR", "1")
	os.Unsetenv("FORCE_COLOR")
	if shouldEnableColors() {
		t.Error("Expected colors to be disabled when NO_COLOR is set")
	}

	// Test FORCE_COLOR
	os.Unsetenv("NO_COLOR")
	os.Setenv("FORCE_COLOR", "1")
	if !shouldEnableColors() {
		t.Error("Expected colors to be enabled when FORCE_COLOR is set")
	}

	// Restore original environment
	if origNoColor != "" {
		os.Setenv("NO_COLOR", origNoColor)
	} else {
		os.Unsetenv("NO_COLOR")
	}
	if origForceColor != "" {
		os.Setenv("FORCE_COLOR", origForceColor)
	} else {
		os.Unsetenv("FORCE_COLOR")
	}
}

func TestColorConstants(t *testing.T) {
	// Test that color constants are defined
	constants := []string{
		Reset, Bold, Dim,
		Black, Red, Green, Yellow, Blue, Magenta, Cyan, White,
		BrightBlack, BrightRed, BrightGreen, BrightYellow,
		BrightBlue, BrightMagenta, BrightCyan, BrightWhite,
		BgBlack, BgRed, BgGreen, BgYellow, BgBlue, BgMagenta, BgCyan, BgWhite,
	}

	for _, constant := range constants {
		if constant == "" {
			t.Error("Expected color constant to be non-empty")
		}
		if !strings.HasPrefix(constant, "\033[") {
			t.Errorf("Expected color constant to start with ANSI escape sequence, got '%s'", constant)
		}
	}
}

func TestStatusSymbols(t *testing.T) {
	// Test that status symbols are defined
	symbols := []string{
		SuccessSymbol, ErrorSymbol, WarningSymbol,
		InfoSymbol, ChangedSymbol, SkippedSymbol,
	}

	for _, symbol := range symbols {
		if symbol == "" {
			t.Error("Expected status symbol to be non-empty")
		}
	}
}

func TestGlobalColorScheme(t *testing.T) {
	// Test that global Colors is initialized
	if Colors == nil {
		t.Error("Expected global Colors to be initialized")
	}

	// Test all color functions are set
	if Colors.Success == nil {
		t.Error("Expected Success color function to be set")
	}
	if Colors.Error == nil {
		t.Error("Expected Error color function to be set")
	}
	if Colors.Warning == nil {
		t.Error("Expected Warning color function to be set")
	}
	if Colors.Info == nil {
		t.Error("Expected Info color function to be set")
	}
	if Colors.Debug == nil {
		t.Error("Expected Debug color function to be set")
	}
	if Colors.Changed == nil {
		t.Error("Expected Changed color function to be set")
	}
	if Colors.Skipped == nil {
		t.Error("Expected Skipped color function to be set")
	}
	if Colors.Failed == nil {
		t.Error("Expected Failed color function to be set")
	}
	if Colors.Header == nil {
		t.Error("Expected Header color function to be set")
	}
	if Colors.Highlight == nil {
		t.Error("Expected Highlight color function to be set")
	}
	if Colors.Dim == nil {
		t.Error("Expected Dim color function to be set")
	}
	if Colors.Bold == nil {
		t.Error("Expected Bold color function to be set")
	}
	if Colors.TaskName == nil {
		t.Error("Expected TaskName color function to be set")
	}
	if Colors.HostName == nil {
		t.Error("Expected HostName color function to be set")
	}
	if Colors.ModuleName == nil {
		t.Error("Expected ModuleName color function to be set")
	}
	if Colors.PlayName == nil {
		t.Error("Expected PlayName color function to be set")
	}
}
