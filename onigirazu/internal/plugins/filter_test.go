package plugins

import (
	"testing"
)

func TestUpperFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{"lowercase", "hello", "HELLO", false},
		{"uppercase", "HELLO", "HELLO", false},
		{"mixed", "HeLLo", "HELLO", false},
		{"empty", "", "", false},
		{"invalid type", 123, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UpperFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpperFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("UpperFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLowerFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{"uppercase", "HELLO", "hello", false},
		{"lowercase", "hello", "hello", false},
		{"mixed", "HeLLo", "hello", false},
		{"empty", "", "", false},
		{"invalid type", 123, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := LowerFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("LowerFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("LowerFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTitleFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{"lowercase", "hello world", "Hello World", false},
		{"uppercase", "HELLO WORLD", "Hello World", false},
		{"mixed", "hELLo WoRLd", "Hello World", false},
		{"single word", "hello", "Hello", false},
		{"empty", "", "", false},
		{"invalid type", 123, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TitleFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("TitleFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("TitleFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTrimFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{"leading spaces", "  hello", "hello", false},
		{"trailing spaces", "hello  ", "hello", false},
		{"both sides", "  hello  ", "hello", false},
		{"no spaces", "hello", "hello", false},
		{"empty", "", "", false},
		{"invalid type", 123, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TrimFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("TrimFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("TrimFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReplaceFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		args     []interface{}
		expected interface{}
		wantErr  bool
	}{
		{"simple replace", "hello world", []interface{}{"world", "go"}, "hello go", false},
		{"multiple occurrences", "hello hello", []interface{}{"hello", "hi"}, "hi hi", false},
		{"no match", "hello", []interface{}{"world", "go"}, "hello", false},
		{"empty string", "", []interface{}{"world", "go"}, "", false},
		{"missing args", "hello", []interface{}{"world"}, nil, true},
		{"invalid type", 123, []interface{}{"world", "go"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReplaceFilter(tt.input, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ReplaceFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		args     []interface{}
		expected interface{}
		wantErr  bool
	}{
		{"nil input", nil, []interface{}{"default"}, "default", false},
		{"empty string", "", []interface{}{"default"}, "default", false},
		{"non-empty string", "value", []interface{}{"default"}, "value", false},
		{"zero int", 0, []interface{}{"default"}, 0, false},
		{"missing args", nil, []interface{}{}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DefaultFilter(tt.input, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("DefaultFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLengthFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{"string", "hello", 5, false},
		{"empty string", "", 0, false},
		{"slice", []interface{}{1, 2, 3}, 3, false},
		{"string slice", []string{"a", "b"}, 2, false},
		{"map", map[string]interface{}{"a": 1, "b": 2}, 2, false},
		{"invalid type", 123, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := LengthFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("LengthFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("LengthFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJoinFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		args     []interface{}
		expected interface{}
		wantErr  bool
	}{
		{"string slice", []string{"a", "b", "c"}, []interface{}{","}, "a,b,c", false},
		{"interface slice", []interface{}{"a", "b", "c"}, []interface{}{"-"}, "a-b-c", false},
		{"empty slice", []string{}, []interface{}{","}, "", false},
		{"single element", []string{"a"}, []interface{}{","}, "a", false},
		{"missing args", []string{"a", "b"}, []interface{}{}, nil, true},
		{"invalid type", "hello", []interface{}{","}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JoinFilter(tt.input, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("JoinFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("JoinFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSplitFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		args     []interface{}
		expected []string
		wantErr  bool
	}{
		{"comma separated", "a,b,c", []interface{}{","}, []string{"a", "b", "c"}, false},
		{"space separated", "a b c", []interface{}{" "}, []string{"a", "b", "c"}, false},
		{"single element", "a", []interface{}{","}, []string{"a"}, false},
		{"empty string", "", []interface{}{","}, []string{""}, false},
		{"missing args", "a,b", []interface{}{}, nil, true},
		{"invalid type", 123, []interface{}{","}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitFilter(tt.input, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				resultSlice, ok := result.([]string)
				if !ok {
					t.Errorf("SplitFilter() result is not []string")
					return
				}
				if len(resultSlice) != len(tt.expected) {
					t.Errorf("SplitFilter() length = %d, want %d", len(resultSlice), len(tt.expected))
					return
				}
				for i, v := range resultSlice {
					if v != tt.expected[i] {
						t.Errorf("SplitFilter()[%d] = %v, want %v", i, v, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestBuiltinFiltersPlugin(t *testing.T) {
	plugin := NewBuiltinFiltersPlugin()

	if plugin.GetName() != "builtin" {
		t.Errorf("Expected name 'builtin', got '%s'", plugin.GetName())
	}

	if plugin.GetType() != PluginTypeFilter {
		t.Errorf("Expected type '%s', got '%s'", PluginTypeFilter, plugin.GetType())
	}

	filters := plugin.GetFilters()
	expectedFilters := []string{"upper", "lower", "title", "trim", "replace", "default", "length", "join", "split"}

	for _, name := range expectedFilters {
		if _, exists := filters[name]; !exists {
			t.Errorf("Expected filter '%s' not found", name)
		}
	}
}

func TestBaseFilterPlugin(t *testing.T) {
	plugin := NewBaseFilterPlugin("test", "1.0.0", "Test filter plugin")

	if plugin.GetName() != "test" {
		t.Errorf("Expected name 'test', got '%s'", plugin.GetName())
	}

	if plugin.GetVersion() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", plugin.GetVersion())
	}

	// Test AddFilter
	testFilter := func(input interface{}, args ...interface{}) (interface{}, error) {
		return "test", nil
	}
	plugin.AddFilter("test", testFilter)

	filters := plugin.GetFilters()
	if _, exists := filters["test"]; !exists {
		t.Error("Filter 'test' not found after AddFilter")
	}

	// Test RemoveFilter
	plugin.RemoveFilter("test")
	filters = plugin.GetFilters()
	if _, exists := filters["test"]; exists {
		t.Error("Filter 'test' still exists after RemoveFilter")
	}
}

func BenchmarkUpperFilter(b *testing.B) {
	input := "hello world this is a test string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UpperFilter(input)
	}
}

func BenchmarkLowerFilter(b *testing.B) {
	input := "HELLO WORLD THIS IS A TEST STRING"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LowerFilter(input)
	}
}

func BenchmarkReplaceFilter(b *testing.B) {
	input := "hello world hello world"
	args := []interface{}{"world", "go"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReplaceFilter(input, args...)
	}
}

func BenchmarkJoinFilter(b *testing.B) {
	input := []string{"a", "b", "c", "d", "e"}
	args := []interface{}{","}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JoinFilter(input, args...)
	}
}

func BenchmarkSplitFilter(b *testing.B) {
	input := "a,b,c,d,e"
	args := []interface{}{","}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SplitFilter(input, args...)
	}
}
