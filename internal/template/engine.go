package template

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"text/template"
	"bytes"
)

// Engine provides template rendering capabilities similar to Jinja2
type Engine struct {
	funcMap template.FuncMap
}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	engine := &Engine{
		funcMap: template.FuncMap{
			"default":    defaultFunc,
			"upper":      strings.ToUpper,
			"lower":      strings.ToLower,
			"title":      strings.Title,
			"trim":       strings.TrimSpace,
			"replace":    strings.ReplaceAll,
			"split":      strings.Split,
			"join":       strings.Join,
			"contains":   strings.Contains,
			"hasPrefix":  strings.HasPrefix,
			"hasSuffix":  strings.HasSuffix,
			"len":        lenFunc,
			"add":        addFunc,
			"sub":        subFunc,
			"mul":        mulFunc,
			"div":        divFunc,
			"mod":        modFunc,
			"eq":         eqFunc,
			"ne":         neFunc,
			"lt":         ltFunc,
			"le":         leFunc,
			"gt":         gtFunc,
			"ge":         geFunc,
			"and":        andFunc,
			"or":         orFunc,
			"not":        notFunc,
			"list":       listFunc,
			"dict":       dictFunc,
			"range":      rangeFunc,
			"toJson":     toJsonFunc,
			"fromJson":   fromJsonFunc,
		},
	}

	return engine
}

// Render renders a template string with variables
func (e *Engine) Render(ctx context.Context, templateStr string, variables map[string]interface{}) (string, error) {
	// Convert Jinja2-style syntax to Go template syntax
	converted, err := e.convertJinja2Syntax(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to convert template syntax: %w", err)
	}

	// Create and parse template
	tmpl, err := template.New("template").Funcs(e.funcMap).Parse(converted)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderFile renders a template file with variables
func (e *Engine) RenderFile(ctx context.Context, filePath string, variables map[string]interface{}) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	return e.Render(ctx, string(content), variables)
}

// ValidateTemplate validates template syntax
func (e *Engine) ValidateTemplate(templateStr string) error {
	converted, err := e.convertJinja2Syntax(templateStr)
	if err != nil {
		return fmt.Errorf("failed to convert template syntax: %w", err)
	}

	_, err = template.New("validation").Funcs(e.funcMap).Parse(converted)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	return nil
}

// RenderTaskArgs renders template variables in task arguments
func (e *Engine) RenderTaskArgs(ctx context.Context, args map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range args {
		renderedValue, err := e.renderValue(ctx, value, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render arg %s: %w", key, err)
		}
		result[key] = renderedValue
	}

	return result, nil
}

// renderValue recursively renders template variables in any value type
func (e *Engine) renderValue(ctx context.Context, value interface{}, variables map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return e.Render(ctx, v, variables)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			renderedVal, err := e.renderValue(ctx, val, variables)
			if err != nil {
				return nil, err
			}
			result[k] = renderedVal
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			renderedVal, err := e.renderValue(ctx, val, variables)
			if err != nil {
				return nil, err
			}
			result[i] = renderedVal
		}
		return result, nil
	default:
		return value, nil
	}
}

// convertJinja2Syntax converts Jinja2-style syntax to Go template syntax
func (e *Engine) convertJinja2Syntax(input string) (string, error) {
	// Convert {{ variable }} to {{ .variable }}
	varRegex := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s*\}\}`)
	result := varRegex.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		varName := strings.Trim(match, "{} ")

		// Don't modify if it already starts with a dot or contains function calls
		if strings.HasPrefix(varName, ".") || strings.Contains(varName, "(") {
			return match
		}

		// Add dot prefix for variable access
		return "{{ ." + varName + " }}"
	})

	// Convert {% if condition %} to {{ if condition }}
	ifRegex := regexp.MustCompile(`\{%\s*if\s+(.+?)\s*%\}`)
	result = ifRegex.ReplaceAllString(result, "{{ if $1 }}")

	// Convert {% else %} to {{ else }}
	result = strings.ReplaceAll(result, "{% else %}", "{{ else }}")

	// Convert {% elif condition %} to {{ else if condition }}
	elifRegex := regexp.MustCompile(`\{%\s*elif\s+(.+?)\s*%\}`)
	result = elifRegex.ReplaceAllString(result, "{{ else if $1 }}")

	// Convert {% endif %} to {{ end }}
	result = strings.ReplaceAll(result, "{% endif %}", "{{ end }}")

	// Convert {% for item in items %} to {{ range .items }}
	forRegex := regexp.MustCompile(`\{%\s*for\s+(\w+)\s+in\s+(\w+)\s*%\}`)
	result = forRegex.ReplaceAllString(result, "{{ range .$2 }}")

	// Convert {% endfor %} to {{ end }}
	result = strings.ReplaceAll(result, "{% endfor %}", "{{ end }}")

	return result, nil
}

// Template function implementations

func defaultFunc(value, defaultValue interface{}) interface{} {
	if value == nil || value == "" {
		return defaultValue
	}
	return value
}

func lenFunc(value interface{}) int {
	switch v := value.(type) {
	case string:
		return len(v)
	case []interface{}:
		return len(v)
	case map[string]interface{}:
		return len(v)
	default:
		return 0
	}
}

func addFunc(a, b interface{}) interface{} {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return va + vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va + vb
		}
	case string:
		if vb, ok := b.(string); ok {
			return va + vb
		}
	}
	return nil
}

func subFunc(a, b interface{}) interface{} {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return va - vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va - vb
		}
	}
	return nil
}

func mulFunc(a, b interface{}) interface{} {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return va * vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va * vb
		}
	}
	return nil
}

func divFunc(a, b interface{}) interface{} {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok && vb != 0 {
			return va / vb
		}
	case float64:
		if vb, ok := b.(float64); ok && vb != 0 {
			return va / vb
		}
	}
	return nil
}

func modFunc(a, b interface{}) interface{} {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok && vb != 0 {
			return va % vb
		}
	}
	return nil
}

func eqFunc(a, b interface{}) bool {
	return a == b
}

func neFunc(a, b interface{}) bool {
	return a != b
}

func ltFunc(a, b interface{}) bool {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return va < vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va < vb
		}
	case string:
		if vb, ok := b.(string); ok {
			return va < vb
		}
	}
	return false
}

func leFunc(a, b interface{}) bool {
	return ltFunc(a, b) || eqFunc(a, b)
}

func gtFunc(a, b interface{}) bool {
	return !leFunc(a, b)
}

func geFunc(a, b interface{}) bool {
	return !ltFunc(a, b)
}

func andFunc(a, b interface{}) bool {
	return toBool(a) && toBool(b)
}

func orFunc(a, b interface{}) bool {
	return toBool(a) || toBool(b)
}

func notFunc(a interface{}) bool {
	return !toBool(a)
}

func toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case nil:
		return false
	default:
		return true
	}
}

func listFunc(items ...interface{}) []interface{} {
	return items
}

func dictFunc(pairs ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(pairs)-1; i += 2 {
		if key, ok := pairs[i].(string); ok {
			result[key] = pairs[i+1]
		}
	}
	return result
}

func rangeFunc(start, end int) []int {
	result := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, i)
	}
	return result
}

func toJsonFunc(value interface{}) string {
	// This would need proper JSON marshaling
	return fmt.Sprintf("%v", value)
}

func fromJsonFunc(jsonStr string) interface{} {
	// This would need proper JSON unmarshaling
	return jsonStr
}
