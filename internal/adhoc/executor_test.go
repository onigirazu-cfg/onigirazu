package adhoc

import (
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

// TestCommand_Structure tests Command structure
func TestCommand_Structure(t *testing.T) {
	cmd := &Command{
		Module: "debug",
		Args: map[string]interface{}{
			"msg": "hello",
		},
		Format: FormatAnsibleLike,
		Raw:    "debug msg='hello'",
	}

	assert.Equal(t, "debug", cmd.Module, "Module should be set correctly")
	assert.Equal(t, map[string]interface{}{"msg": "hello"}, cmd.Args, "Args should be set correctly")
	assert.Equal(t, FormatAnsibleLike, cmd.Format, "Format should be set correctly")
	assert.Equal(t, "debug msg='hello'", cmd.Raw, "Raw command should be set correctly")
}

// TestOptions_Structure tests Options structure
func TestOptions_Structure(t *testing.T) {
	opts := Options{
		Check:    true,
		Diff:     false,
		Timeout:  30 * time.Second,
		Parallel: 5,
		Output:   "text",
		Verbose:  true,
		NoColor:  false,
	}

	assert.True(t, opts.Check, "Check should be set")
	assert.False(t, opts.Diff, "Diff should be false")
	assert.Equal(t, 30*time.Second, opts.Timeout, "Timeout should be 30 seconds")
	assert.Equal(t, 5, opts.Parallel, "Parallel should be 5")
	assert.Equal(t, "text", opts.Output, "Output should be text")
	assert.True(t, opts.Verbose, "Verbose should be true")
}

// TestResult_Structure tests Result structure
func TestResult_Structure(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	result := &Result{
		Host:     host,
		Task:     nil,
		Result:   nil,
		Duration: 2 * time.Second,
		Error:    nil,
	}

	assert.Equal(t, host, result.Host, "Host should be set")
	assert.Equal(t, 2*time.Second, result.Duration, "Duration should be 2 seconds")
	assert.NoError(t, result.Error, "Error should be nil")
}

// TestSummary_Structure tests Summary structure
func TestSummary_Structure(t *testing.T) {
	summary := &Summary{
		Total:    10,
		Success:  8,
		Failed:   2,
		Changed:  5,
		Skipped:  0,
		Duration: 10 * time.Second,
		Results:  nil,
	}

	assert.Equal(t, 10, summary.Total, "Total should be 10")
	assert.Equal(t, 8, summary.Success, "Success should be 8")
	assert.Equal(t, 2, summary.Failed, "Failed should be 2")
	assert.Equal(t, 5, summary.Changed, "Changed should be 5")
	assert.Equal(t, 0, summary.Skipped, "Skipped should be 0")
	assert.Equal(t, 10*time.Second, summary.Duration, "Duration should be 10 seconds")
}

// TestCommandFormat_Types tests different command format types
func TestCommandFormat_Types(t *testing.T) {
	testCases := []struct {
		name     string
		format   CommandFormat
		expected CommandFormat
		desc     string
	}{
		{name: "Ansible-like", format: FormatAnsibleLike, expected: 0, desc: "Ansible-like format"},
		{name: "Natural Language", format: FormatNaturalLanguage, expected: 1, desc: "Natural language format"},
		{name: "JSON", format: FormatJSON, expected: 2, desc: "JSON format"},
		{name: "YAML", format: FormatYAML, expected: 3, desc: "YAML format"},
		{name: "Simple", format: FormatSimple, expected: 4, desc: "Simple format"},
	}

	// Track all formats to ensure uniqueness
	seen := make(map[CommandFormat]bool)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.format, "Format should have correct iota value for %s", tc.desc)
			assert.False(t, seen[tc.format], "Format should be unique: %s", tc.desc)
		})
		seen[tc.format] = true
	}

	// Verify all 5 formats are defined
	assert.Equal(t, 5, len(seen), "All 5 CommandFormat types should be defined")
}

// TestCommand_EmptyModule tests command validation
func TestCommand_EmptyModule(t *testing.T) {
	cmd := &Command{
		Module: "",
		Args:   map[string]interface{}{},
	}

	assert.Empty(t, cmd.Module, "Module should be empty")
}

// TestOptions_Defaults tests default options
func TestOptions_Defaults(t *testing.T) {
	opts := Options{}

	// Verify defaults
	assert.False(t, opts.Check, "Check should default to false")
	assert.False(t, opts.Verbose, "Verbose should default to false")
	assert.Equal(t, 0, opts.Parallel, "Parallel should default to 0")
	assert.Equal(t, time.Duration(0), opts.Timeout, "Timeout should default to 0")
}

// TestResult_WithError tests result with error
func TestResult_WithError(t *testing.T) {
	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
	}

	testErr := assert.AnError

	result := &Result{
		Host:     host,
		Task:     nil,
		Result:   nil,
		Duration: time.Second,
		Error:    testErr,
	}

	assert.Error(t, result.Error, "Error should be set")
	assert.Equal(t, testErr, result.Error, "Error should match")
}

// TestSummary_SuccessRatio tests calculating success ratio
func TestSummary_SuccessRatio(t *testing.T) {
	testCases := []struct {
		name     string
		success  int
		failed   int
		expected float64
	}{
		{
			name:     "all successful",
			success:  10,
			failed:   0,
			expected: 1.0,
		},
		{
			name:     "all failed",
			success:  0,
			failed:   10,
			expected: 0.0,
		},
		{
			name:     "mixed results",
			success:  5,
			failed:   5,
			expected: 0.5,
		},
		{
			name:     "one of ten",
			success:  1,
			failed:   9,
			expected: 0.1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			total := tc.success + tc.failed
			if total == 0 {
				assert.Fail(t, "Total must be greater than 0")
				return
			}

			ratio := float64(tc.success) / float64(total)
			assert.Equal(t, tc.expected, ratio, "Success ratio should match for: %s", tc.name)
		})
	}
}

// TestExecutor_Creation tests that executor can be created (basic structure test)
func TestExecutor_Creation(t *testing.T) {
	// This test verifies that the Executor type exists and has the expected fields
	// Full executor testing would require a complete logger mock implementation
	executor := &Executor{
		moduleRegistry: nil,
		inventoryMgr:   nil,
		logger:         nil,
	}

	assert.NotNil(t, executor, "Executor should be creatable as a type")
}

// MockLogger implements interfaces.Logger for testing
type MockLogger struct{}

func (m *MockLogger) Info(msg string, args ...interface{})             {}
func (m *MockLogger) Warn(msg string, args ...interface{})             {}
func (m *MockLogger) Error(msg string, args ...interface{})            {}
func (m *MockLogger) Debug(msg string, args ...interface{})            {}
func (m *MockLogger) Trace(msg string, args ...interface{})            {}
func (m *MockLogger) Fatal(msg string, args ...interface{})            {}
func (m *MockLogger) Printf(format string, args ...interface{}) string { return "" }

// TestCommand_ComplexArgs tests command with complex arguments
func TestCommand_ComplexArgs(t *testing.T) {
	complexArgs := map[string]interface{}{
		"simple":  "value",
		"number":  42,
		"boolean": true,
		"nested": map[string]interface{}{
			"key": "value",
		},
		"list": []interface{}{"a", "b", "c"},
	}

	cmd := &Command{
		Module: "test",
		Args:   complexArgs,
	}

	assert.Equal(t, "value", cmd.Args["simple"], "Simple arg should match")
	assert.Equal(t, 42, cmd.Args["number"], "Number arg should match")
	assert.Equal(t, true, cmd.Args["boolean"], "Boolean arg should match")

	nested, ok := cmd.Args["nested"].(map[string]interface{})
	assert.True(t, ok, "Nested arg should be map")
	assert.Equal(t, "value", nested["key"], "Nested value should match")

	list, ok := cmd.Args["list"].([]interface{})
	assert.True(t, ok, "List arg should be slice")
	assert.Len(t, list, 3, "List should have 3 elements")
}

// TestOptions_AllFormats tests options with different formats
func TestOptions_AllFormats(t *testing.T) {
	formats := []struct {
		name   string
		format CommandFormat
	}{
		{"AnsibleLike", FormatAnsibleLike},
		{"NaturalLanguage", FormatNaturalLanguage},
		{"JSON", FormatJSON},
		{"YAML", FormatYAML},
		{"Simple", FormatSimple},
	}

	for _, fmt := range formats {
		t.Run(fmt.name, func(t *testing.T) {
			cmd := &Command{
				Module: "test",
				Format: fmt.format,
			}

			assert.Equal(t, fmt.format, cmd.Format, "Format should match for: %s", fmt.name)
		})
	}
}

// TestSummary_EmptyResults tests summary with empty results
func TestSummary_EmptyResults(t *testing.T) {
	summary := &Summary{
		Total:   0,
		Success: 0,
		Failed:  0,
		Results: make([]*Result, 0),
	}

	assert.Equal(t, 0, summary.Total, "Total should be 0")
	assert.Equal(t, 0, len(summary.Results), "Results should be empty")
}

// TestResult_MultipleHosts tests results for multiple hosts
func TestResult_MultipleHosts(t *testing.T) {
	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
		{Name: "host3", Address: "192.168.1.3"},
	}

	results := make([]*Result, len(hosts))
	for i, host := range hosts {
		results[i] = &Result{
			Host:     host,
			Duration: time.Second * time.Duration(i+1),
		}
	}

	assert.Len(t, results, 3, "Should have 3 results")
	assert.Equal(t, "host1", results[0].Host.Name, "First host should be host1")
	assert.Equal(t, "host2", results[1].Host.Name, "Second host should be host2")
	assert.Equal(t, "host3", results[2].Host.Name, "Third host should be host3")
}

// TestCommand_RawStorage tests raw command storage
func TestCommand_RawStorage(t *testing.T) {
	rawCommands := []string{
		"debug msg='hello'",
		"copy src='/tmp/file' dest='/home/'",
		"shell echo 'test'",
	}

	for _, raw := range rawCommands {
		t.Run(raw, func(t *testing.T) {
			cmd := &Command{
				Raw: raw,
			}

			assert.Equal(t, raw, cmd.Raw, "Raw command should be stored correctly")
		})
	}
}

// TestOptions_OutputFormats tests different output formats
func TestOptions_OutputFormats(t *testing.T) {
	formats := []string{"text", "json", "yaml", "table", "csv"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			opts := Options{
				Output: format,
			}

			assert.Equal(t, format, opts.Output, "Output format should match")
		})
	}
}

// BenchmarkCommand_Creation benchmarks command creation
func BenchmarkCommand_Creation(b *testing.B) {
	args := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Command{
			Module: "test",
			Args:   args,
			Format: FormatAnsibleLike,
		}
	}
}

// BenchmarkResult_Creation benchmarks result creation
func BenchmarkResult_Creation(b *testing.B) {
	host := types.Host{
		Name:    "test",
		Address: "127.0.0.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Result{
			Host:     host,
			Duration: time.Second,
		}
	}
}

// BenchmarkSummary_Creation benchmarks summary creation
func BenchmarkSummary_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Summary{
			Total:    100,
			Success:  80,
			Failed:   20,
			Duration: 10 * time.Second,
		}
	}
}
