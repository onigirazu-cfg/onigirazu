package template

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

// TestNewEngine tests basic engine creation
func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if engine.funcMap == nil {
		t.Error("funcMap should not be nil")
	}

	if engine.templateCache == nil {
		t.Error("templateCache should not be nil")
	}

	// Verify built-in functions are registered
	expectedFuncs := []string{
		"default", "upper", "lower", "title", "trim",
		"replace", "split", "join", "contains",
		"hasPrefix", "hasSuffix", "len",
		"add", "sub", "mul", "div", "mod",
		"eq", "ne", "lt", "le", "gt", "ge",
		"and", "or", "not",
		"list", "dict", "range",
		"toJson", "fromJson",
	}

	for _, funcName := range expectedFuncs {
		if _, exists := engine.funcMap[funcName]; !exists {
			t.Errorf("Expected function %s not found in funcMap", funcName)
		}
	}

	// Cleanup
	if err := engine.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

// TestRenderSimpleVariable tests rendering simple variables
func TestRenderSimpleVariable(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"name": "World",
		"age":  25,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "simple string variable",
			template: "Hello {{ name }}!",
			want:     "Hello World!",
		},
		{
			name:     "simple int variable",
			template: "Age: {{ age }}",
			want:     "Age: 25",
		},
		{
			name:     "multiple variables",
			template: "{{ name }} is {{ age }} years old",
			want:     "World is 25 years old",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(ctx, tt.template, variables)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderStringFunctions tests string manipulation functions
func TestRenderStringFunctions(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"text": "hello world",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "upper function",
			template: "{{ upper .text }}",
			want:     "HELLO WORLD",
		},
		{
			name:     "lower function",
			template: "{{ lower .text }}",
			want:     "hello world",
		},
		{
			name:     "title function",
			template: "{{ title .text }}",
			want:     "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(ctx, tt.template, variables)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderMathFunctions tests mathematical functions
func TestRenderMathFunctions(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"a": 10,
		"b": 3,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "add function",
			template: "{{ add .a .b }}",
			want:     "13",
		},
		{
			name:     "sub function",
			template: "{{ sub .a .b }}",
			want:     "7",
		},
		{
			name:     "mul function",
			template: "{{ mul .a .b }}",
			want:     "30",
		},
		{
			name:     "div function",
			template: "{{ div .a .b }}",
			want:     "3",
		},
		{
			name:     "mod function",
			template: "{{ mod .a .b }}",
			want:     "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(ctx, tt.template, variables)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderComparisonFunctions tests comparison functions
func TestRenderComparisonFunctions(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"a": 10,
		"b": 5,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "eq function - false",
			template: "{% if eq .a .b %}yes{% else %}no{% endif %}",
			want:     "no",
		},
		{
			name:     "ne function - true",
			template: "{% if ne .a .b %}yes{% else %}no{% endif %}",
			want:     "yes",
		},
		{
			name:     "lt function - false",
			template: "{% if lt .a .b %}yes{% else %}no{% endif %}",
			want:     "no",
		},
		{
			name:     "gt function - true",
			template: "{% if gt .a .b %}yes{% else %}no{% endif %}",
			want:     "yes",
		},
		{
			name:     "le function - false",
			template: "{% if le .a .b %}yes{% else %}no{% endif %}",
			want:     "no",
		},
		{
			name:     "ge function - true",
			template: "{% if ge .a .b %}yes{% else %}no{% endif %}",
			want:     "yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(ctx, tt.template, variables)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderLogicalFunctions tests logical functions
func TestRenderLogicalFunctions(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"true_val":  true,
		"false_val": false,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "and function - true",
			template: "{% if and .true_val .true_val %}yes{% else %}no{% endif %}",
			want:     "yes",
		},
		{
			name:     "and function - false",
			template: "{% if and .true_val .false_val %}yes{% else %}no{% endif %}",
			want:     "no",
		},
		{
			name:     "or function - true",
			template: "{% if or .true_val .false_val %}yes{% else %}no{% endif %}",
			want:     "yes",
		},
		{
			name:     "or function - false",
			template: "{% if or .false_val .false_val %}yes{% else %}no{% endif %}",
			want:     "no",
		},
		{
			name:     "not function - true",
			template: "{% if not .false_val %}yes{% else %}no{% endif %}",
			want:     "yes",
		},
		{
			name:     "not function - false",
			template: "{% if not .true_val %}yes{% else %}no{% endif %}",
			want:     "no",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(ctx, tt.template, variables)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderDefaultFunction tests the default function
func TestRenderDefaultFunction(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()

	tests := []struct {
		name      string
		template  string
		variables map[string]interface{}
		want      string
	}{
		{
			name:      "default with nil value",
			template:  "{{ default .missing \"fallback\" }}",
			variables: map[string]interface{}{},
			want:      "fallback",
		},
		{
			name:      "default with empty string",
			template:  "{{ default .empty \"fallback\" }}",
			variables: map[string]interface{}{"empty": ""},
			want:      "fallback",
		},
		{
			name:      "default with existing value",
			template:  "{{ default .value \"fallback\" }}",
			variables: map[string]interface{}{"value": "actual"},
			want:      "actual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(ctx, tt.template, tt.variables)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestConvertJinja2Syntax tests Jinja2 to Go template conversion
func TestConvertJinja2Syntax(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple variable",
			input: "{{ name }}",
			want:  "{{ .name }}",
		},
		{
			name:  "nested variable",
			input: "{{ user.name }}",
			want:  "{{ .user.name }}",
		},
		{
			name:  "if statement",
			input: "{% if condition %}yes{% endif %}",
			want:  "{{ if condition }}yes{{ end }}",
		},
		{
			name:  "if-else statement",
			input: "{% if condition %}yes{% else %}no{% endif %}",
			want:  "{{ if condition }}yes{{ else }}no{{ end }}",
		},
		{
			name:  "if-elif-else statement",
			input: "{% if a %}1{% elif b %}2{% else %}3{% endif %}",
			want:  "{{ if a }}1{{ else if b }}2{{ else }}3{{ end }}",
		},
		{
			name:  "for loop",
			input: "{% for item in items %}{{ item }}{% endfor %}",
			want:  "{{ range .items }}{{ .item }}{{ end }}",
		},
		{
			name:  "variable with function call",
			input: "{{ upper(name) }}",
			want:  "{{ upper(name) }}",
		},
		{
			name:  "variable already with dot",
			input: "{{ .name }}",
			want:  "{{ .name }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.convertJinja2Syntax(tt.input)
			if err != nil {
				t.Errorf("convertJinja2Syntax() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("convertJinja2Syntax() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderTaskArgs tests rendering task arguments recursively
func TestRenderTaskArgs(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"name": "test",
		"port": 8080,
	}

	tests := []struct {
		name string
		args map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "simple string rendering",
			args: map[string]interface{}{
				"message": "Hello {{ name }}",
			},
			want: map[string]interface{}{
				"message": "Hello test",
			},
		},
		{
			name: "nested map rendering",
			args: map[string]interface{}{
				"config": map[string]interface{}{
					"host": "{{ name }}.example.com",
					"port": "{{ port }}",
				},
			},
			want: map[string]interface{}{
				"config": map[string]interface{}{
					"host": "test.example.com",
					"port": "8080",
				},
			},
		},
		{
			name: "array rendering",
			args: map[string]interface{}{
				"items": []interface{}{
					"{{ name }}-1",
					"{{ name }}-2",
				},
			},
			want: map[string]interface{}{
				"items": []interface{}{
					"test-1",
					"test-2",
				},
			},
		},
		{
			name: "mixed types",
			args: map[string]interface{}{
				"string": "{{ name }}",
				"number": 42,
				"bool":   true,
			},
			want: map[string]interface{}{
				"string": "test",
				"number": 42,
				"bool":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderTaskArgs(ctx, tt.args, variables)
			if err != nil {
				t.Errorf("RenderTaskArgs() error = %v", err)
				return
			}

			// Deep comparison
			if !compareValues(got, tt.want) {
				t.Errorf("RenderTaskArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRenderFile tests rendering from file
func TestRenderFile(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	// Create temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tmpl")
	templateContent := "Hello {{ name }}!"

	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	ctx := context.Background()
	variables := map[string]interface{}{
		"name": "World",
	}

	got, err := engine.RenderFile(ctx, templatePath, variables)
	if err != nil {
		t.Errorf("RenderFile() error = %v", err)
		return
	}

	want := "Hello World!"
	if got != want {
		t.Errorf("RenderFile() = %q, want %q", got, want)
	}
}

// TestRenderFileNotFound tests error handling for missing file
func TestRenderFileNotFound(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{}

	_, err := engine.RenderFile(ctx, "/nonexistent/file.tmpl", variables)
	if err == nil {
		t.Error("RenderFile() should return error for nonexistent file")
	}
}

// TestValidateTemplate tests template validation
func TestValidateTemplate(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	tests := []struct {
		name      string
		template  string
		wantError bool
	}{
		{
			name:      "valid template",
			template:  "Hello {{ name }}!",
			wantError: false,
		},
		{
			name:      "valid template with function",
			template:  "{{ upper .name }}",
			wantError: false,
		},
		{
			name:      "invalid template - unclosed bracket",
			template:  "Hello {{ name }",
			wantError: true,
		},
		{
			name:      "invalid template - syntax error",
			template:  "{{ .undefined | invalid }}",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTemplate(tt.template)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTemplate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestTemplateCaching tests template caching functionality
func TestTemplateCaching(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"name": "World",
	}

	template := "Hello {{ name }}!"

	// First render - should cache
	_, err := engine.Render(ctx, template, variables)
	if err != nil {
		t.Errorf("First Render() error = %v", err)
	}

	// Get cache stats
	stats := engine.GetCacheStats()
	if stats.TotalEntries == 0 {
		t.Error("Template should be cached after first render")
	}

	// Second render - should use cache
	_, err = engine.Render(ctx, template, variables)
	if err != nil {
		t.Errorf("Second Render() error = %v", err)
	}

	// Clear cache
	if err := engine.ClearCache(ctx); err != nil {
		t.Errorf("ClearCache() error = %v", err)
	}

	stats = engine.GetCacheStats()
	if stats.TotalEntries != 0 {
		t.Error("Cache should be empty after ClearCache()")
	}
}

// TestNewEngineWithPlugins tests engine creation with plugin support
func TestNewEngineWithPlugins(t *testing.T) {
	// Create plugin manager
	loader := plugins.NewInMemoryLoader()
	manager := plugins.NewManager(loader)

	// Create engine with plugins
	engine := NewEngineWithPlugins(manager)
	defer engine.Close()

	if engine == nil {
		t.Fatal("NewEngineWithPlugins() returned nil")
	}

	if engine.pluginManager == nil {
		t.Error("pluginManager should not be nil")
	}

	// Verify built-in filters are registered
	filterPlugins := manager.List(plugins.PluginTypeFilter)
	if len(filterPlugins) == 0 {
		t.Error("Expected at least one filter plugin (built-in)")
	}
}

// TestSetPluginManager tests setting plugin manager after creation
func TestSetPluginManager(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	if engine.pluginManager != nil {
		t.Error("pluginManager should be nil initially")
	}

	// Create and set plugin manager
	loader := plugins.NewInMemoryLoader()
	manager := plugins.NewManager(loader)

	engine.SetPluginManager(manager)

	if engine.pluginManager == nil {
		t.Error("pluginManager should not be nil after SetPluginManager()")
	}
}

// TestHelperFunctions tests individual helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("defaultFunc", func(t *testing.T) {
		if got := defaultFunc(nil, "default"); got != "default" {
			t.Errorf("defaultFunc(nil, default) = %v, want default", got)
		}
		if got := defaultFunc("", "default"); got != "default" {
			t.Errorf("defaultFunc('', default) = %v, want default", got)
		}
		if got := defaultFunc("value", "default"); got != "value" {
			t.Errorf("defaultFunc(value, default) = %v, want value", got)
		}
	})

	t.Run("toTitle", func(t *testing.T) {
		if got := toTitle("hello world"); got != "Hello World" {
			t.Errorf("toTitle(hello world) = %v, want Hello World", got)
		}
		if got := toTitle(""); got != "" {
			t.Errorf("toTitle('') = %v, want ''", got)
		}
	})

	t.Run("lenFunc", func(t *testing.T) {
		if got := lenFunc("hello"); got != 5 {
			t.Errorf("lenFunc(hello) = %v, want 5", got)
		}
		if got := lenFunc([]interface{}{1, 2, 3}); got != 3 {
			t.Errorf("lenFunc([1,2,3]) = %v, want 3", got)
		}
		if got := lenFunc(map[string]interface{}{"a": 1, "b": 2}); got != 2 {
			t.Errorf("lenFunc(map) = %v, want 2", got)
		}
		if got := lenFunc(123); got != 0 {
			t.Errorf("lenFunc(123) = %v, want 0", got)
		}
	})

	t.Run("addFunc", func(t *testing.T) {
		if got := addFunc(5, 3); got != 8 {
			t.Errorf("addFunc(5, 3) = %v, want 8", got)
		}
		if got := addFunc(5.5, 3.5); got != 9.0 {
			t.Errorf("addFunc(5.5, 3.5) = %v, want 9.0", got)
		}
		if got := addFunc("hello", " world"); got != "hello world" {
			t.Errorf("addFunc(hello, world) = %v, want 'hello world'", got)
		}
		if got := addFunc(5, "3"); got != nil {
			t.Errorf("addFunc(5, '3') = %v, want nil", got)
		}
	})

	t.Run("divFunc", func(t *testing.T) {
		if got := divFunc(10, 2); got != 5 {
			t.Errorf("divFunc(10, 2) = %v, want 5", got)
		}
		if got := divFunc(10, 0); got != nil {
			t.Errorf("divFunc(10, 0) = %v, want nil (division by zero)", got)
		}
	})

	t.Run("toBool", func(t *testing.T) {
		if got := toBool(true); got != true {
			t.Errorf("toBool(true) = %v, want true", got)
		}
		if got := toBool(false); got != false {
			t.Errorf("toBool(false) = %v, want false", got)
		}
		if got := toBool(1); got != true {
			t.Errorf("toBool(1) = %v, want true", got)
		}
		if got := toBool(0); got != false {
			t.Errorf("toBool(0) = %v, want false", got)
		}
		if got := toBool("text"); got != true {
			t.Errorf("toBool('text') = %v, want true", got)
		}
		if got := toBool(""); got != false {
			t.Errorf("toBool('') = %v, want false", got)
		}
		if got := toBool(nil); got != false {
			t.Errorf("toBool(nil) = %v, want false", got)
		}
	})

	t.Run("listFunc", func(t *testing.T) {
		got := listFunc(1, 2, 3)
		if len(got) != 3 {
			t.Errorf("listFunc(1,2,3) length = %v, want 3", len(got))
		}
	})

	t.Run("dictFunc", func(t *testing.T) {
		got := dictFunc("key1", "value1", "key2", "value2")
		if len(got) != 2 {
			t.Errorf("dictFunc length = %v, want 2", len(got))
		}
		if got["key1"] != "value1" {
			t.Errorf("dictFunc[key1] = %v, want value1", got["key1"])
		}
	})

	t.Run("rangeFunc", func(t *testing.T) {
		got := rangeFunc(0, 5)
		if len(got) != 5 {
			t.Errorf("rangeFunc(0, 5) length = %v, want 5", len(got))
		}
		if got[0] != 0 || got[4] != 4 {
			t.Errorf("rangeFunc(0, 5) = %v, want [0,1,2,3,4]", got)
		}
	})
}

// TestRenderInvalidTemplate tests error handling for invalid templates
func TestRenderInvalidTemplate(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{}

	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "unclosed bracket",
			template: "{{ name }",
		},
		{
			name:     "invalid syntax",
			template: "{{ .undefined | invalid }}",
		},
		{
			name:     "undefined function",
			template: "{{ undefined_func .name }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Render(ctx, tt.template, variables)
			if err == nil {
				t.Error("Render() should return error for invalid template")
			}
		})
	}
}

// TestConcurrentRendering tests thread-safety of template rendering
func TestConcurrentRendering(t *testing.T) {
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	template := "Hello {{ name }}!"

	// Run multiple renders concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			variables := map[string]interface{}{
				"name": "World",
			}
			_, err := engine.Render(ctx, template, variables)
			if err != nil {
				t.Errorf("Concurrent Render() error = %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestCacheExpiration tests template cache expiration
func TestCacheExpiration(t *testing.T) {
	// This test would require waiting for TTL expiration
	// For now, we just verify the cache can be cleared
	engine := NewEngine()
	defer engine.Close()

	ctx := context.Background()
	variables := map[string]interface{}{
		"name": "World",
	}

	// Render to populate cache
	_, err := engine.Render(ctx, "Hello {{ name }}!", variables)
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	// Verify cache has entries
	stats := engine.GetCacheStats()
	if stats.TotalEntries == 0 {
		t.Error("Cache should have entries after render")
	}

	// Clear and verify
	if err := engine.ClearCache(ctx); err != nil {
		t.Errorf("ClearCache() error = %v", err)
	}

	stats = engine.GetCacheStats()
	if stats.TotalEntries != 0 {
		t.Error("Cache should be empty after clear")
	}
}

// Helper function to compare values deeply
func compareValues(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !compareValues(v, vb[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for i, v := range va {
			if !compareValues(v, vb[i]) {
				return false
			}
		}
		return true
	case string:
		vb, ok := b.(string)
		return ok && va == vb
	case int:
		vb, ok := b.(int)
		return ok && va == vb
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb
	default:
		return a == b
	}
}
