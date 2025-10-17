package documentation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ParameterDocumenter generates documentation from parameter schemas
type ParameterDocumenter struct {
	roleName   string
	roleMeta   *types.RoleMeta
	parameters map[string]types.ParameterDef
}

// DocumentFormat specifies the output format for documentation
type DocumentFormat string

const (
	FormatMarkdown   DocumentFormat = "markdown"
	FormatJSON       DocumentFormat = "json"
	FormatJSONSchema DocumentFormat = "jsonschema"
	FormatCLI        DocumentFormat = "cli"
)

// DocumentationStats holds statistics about parameters
type DocumentationStats struct {
	Total           int
	Required        int
	Optional        int
	WithDefaults    int
	WithConstraints int
	WithValidators  int
	WithConditional int
	StringParams    int
	IntegerParams   int
	BooleanParams   int
	ArrayParams     int
	ObjectParams    int
}

// NewParameterDocumenter creates a new parameter documenter
func NewParameterDocumenter(roleName string, roleMeta *types.RoleMeta) *ParameterDocumenter {
	if roleMeta == nil {
		roleMeta = &types.RoleMeta{
			Parameters: make(map[string]types.ParameterDef),
		}
	}

	if roleMeta.Parameters == nil {
		roleMeta.Parameters = make(map[string]types.ParameterDef)
	}

	return &ParameterDocumenter{
		roleName:   roleName,
		roleMeta:   roleMeta,
		parameters: roleMeta.Parameters,
	}
}

// CountParameters returns the total number of parameters
func (pd *ParameterDocumenter) CountParameters() int {
	return len(pd.parameters)
}

// GetStatistics returns comprehensive documentation statistics
func (pd *ParameterDocumenter) GetStatistics() DocumentationStats {
	stats := DocumentationStats{
		Total: len(pd.parameters),
	}

	for _, param := range pd.parameters {
		if param.Required {
			stats.Required++
		} else {
			stats.Optional++
		}

		if param.Default != nil {
			stats.WithDefaults++
		}

		if !isZeroConstraints(param.Constraints) {
			stats.WithConstraints++
		}

		if len(param.Validators) > 0 {
			stats.WithValidators++
		}

		if param.ConditionalRequirement != nil {
			stats.WithConditional++
		}

		switch param.Type {
		case "string":
			stats.StringParams++
		case "integer":
			stats.IntegerParams++
		case "boolean":
			stats.BooleanParams++
		case "array":
			stats.ArrayParams++
		case "object":
			stats.ObjectParams++
		}
	}

	return stats
}

// GenerateMarkdown generates comprehensive Markdown documentation
func (pd *ParameterDocumenter) GenerateMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s Role Parameters\n\n", pd.roleName))

	if pd.roleMeta.SchemaVersion > 0 {
		sb.WriteString(fmt.Sprintf("> **Schema Version:** %d\n\n", pd.roleMeta.SchemaVersion))
	}

	if len(pd.parameters) > 0 {
		sb.WriteString("## Parameters\n\n")
		sb.WriteString("| Parameter | Type | Required | Description |\n")
		sb.WriteString("|-----------|------|----------|-------------|\n")

		for name, param := range pd.parameters {
			req := "No"
			if param.Required {
				req = "Yes"
			}
			desc := param.Description
			if desc == "" {
				desc = "-"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s |\n", name, param.Type, req, desc))
		}
		sb.WriteString("\n")
	}

	if len(pd.parameters) > 0 {
		sb.WriteString("## Parameter Details\n\n")
		for name, param := range pd.parameters {
			sb.WriteString(pd.parameterToMarkdown(name, param))
			sb.WriteString("\n")
		}
	}

	if len(pd.parameters) > 0 {
		sb.WriteString("## Usage Examples\n\n")
		sb.WriteString("```yaml\n")
		sb.WriteString(fmt.Sprintf("- %s:\n", pd.roleName))

		for name, param := range pd.parameters {
			example := pd.generateExampleValue(name, param)
			sb.WriteString(fmt.Sprintf("    %s: %s\n", name, example))
		}

		sb.WriteString("```\n\n")
	}

	if pd.hasConstraints() {
		sb.WriteString("## Constraints\n\n")
		sb.WriteString("This section documents parameter constraints and validation rules.\n\n")

		for name, param := range pd.parameters {
			if !isZeroConstraints(param.Constraints) {
				sb.WriteString(pd.constraintsToMarkdown(name, param))
			}
		}
	}

	return sb.String()
}

// parameterToMarkdown converts a single parameter to Markdown format
func (pd *ParameterDocumenter) parameterToMarkdown(name string, param types.ParameterDef) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### `%s`\n\n", name))

	sb.WriteString("**Type:** `" + param.Type + "`  \n")

	if param.Required {
		sb.WriteString("**Required:** Yes  \n")
	} else {
		sb.WriteString("**Required:** No  \n")
	}

	if param.ConditionalRequirement != nil {
		sb.WriteString(fmt.Sprintf("**Required When:** `%s`  \n", param.ConditionalRequirement.Condition))
		if param.ConditionalRequirement.Description != "" {
			sb.WriteString(fmt.Sprintf("*%s*  \n", param.ConditionalRequirement.Description))
		}
	}

	if param.Default != nil {
		sb.WriteString(fmt.Sprintf("**Default:** `%v`  \n", param.Default))
	}

	if param.Description != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", param.Description))
	}

	if !isZeroConstraints(param.Constraints) {
		sb.WriteString("\n**Constraints:**\n")
		sb.WriteString(pd.constraintsToMarkdown(name, param))
	}

	if len(param.Validators) > 0 {
		sb.WriteString("\n**Validators:**\n")
		for _, v := range param.Validators {
			sb.WriteString(fmt.Sprintf("- `%s`: %s\n", v.Name, v.Description))
		}
	}

	return sb.String()
}

// constraintsToMarkdown converts constraints to Markdown format
func (pd *ParameterDocumenter) constraintsToMarkdown(name string, param types.ParameterDef) string {
	var sb strings.Builder
	constraints := param.Constraints

	switch param.Type {
	case "string":
		if constraints.Pattern != "" {
			sb.WriteString(fmt.Sprintf("- **Pattern:** `%s`\n", constraints.Pattern))
		}
		if constraints.MinLength > 0 {
			sb.WriteString(fmt.Sprintf("- **Min Length:** %d\n", constraints.MinLength))
		}
		if constraints.MaxLength > 0 {
			sb.WriteString(fmt.Sprintf("- **Max Length:** %d\n", constraints.MaxLength))
		}
		if len(constraints.Enum) > 0 {
			sb.WriteString("- **Allowed Values:** ")
			for i, val := range constraints.Enum {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("`%v`", val))
			}
			sb.WriteString("\n")
		}

	case "integer", "float":
		if constraints.Minimum != nil {
			sb.WriteString(fmt.Sprintf("- **Minimum:** %v\n", constraints.Minimum))
		}
		if constraints.Maximum != nil {
			sb.WriteString(fmt.Sprintf("- **Maximum:** %v\n", constraints.Maximum))
		}
		if constraints.MultipleOf != nil {
			sb.WriteString(fmt.Sprintf("- **Multiple Of:** %v\n", constraints.MultipleOf))
		}

	case "array":
		if constraints.ItemsType != "" {
			sb.WriteString(fmt.Sprintf("- **Item Type:** `%s`\n", constraints.ItemsType))
		}
		if constraints.MinItems > 0 {
			sb.WriteString(fmt.Sprintf("- **Min Items:** %d\n", constraints.MinItems))
		}
		if constraints.MaxItems > 0 {
			sb.WriteString(fmt.Sprintf("- **Max Items:** %d\n", constraints.MaxItems))
		}
		if constraints.UniqueItems {
			sb.WriteString("- **Unique Items:** Yes\n")
		}

	case "object":
		if len(constraints.RequiredFields) > 0 {
			sb.WriteString("- **Required Fields:** ")
			for i, field := range constraints.RequiredFields {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("`%s`", field))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GenerateJSONSchema generates JSON-Schema representation
func (pd *ParameterDocumenter) GenerateJSONSchema() (string, error) {
	schema := map[string]interface{}{
		"$schema":    "http://json-schema.org/draft-07/schema#",
		"type":       "object",
		"title":      fmt.Sprintf("%s Parameters", pd.roleName),
		"properties": pd.generateProperties(),
		"required":   pd.getRequiredParameters(),
	}

	if pd.roleMeta.SchemaVersion > 0 {
		schema["version"] = pd.roleMeta.SchemaVersion
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// generateProperties generates JSON-Schema properties
func (pd *ParameterDocumenter) generateProperties() map[string]interface{} {
	properties := make(map[string]interface{})

	for name, param := range pd.parameters {
		properties[name] = pd.parameterToJSONSchema(param)
	}

	return properties
}

// parameterToJSONSchema converts a parameter to JSON-Schema format
func (pd *ParameterDocumenter) parameterToJSONSchema(param types.ParameterDef) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        param.Type,
		"description": param.Description,
	}

	if param.Default != nil {
		schema["default"] = param.Default
	}

	switch param.Type {
	case "string":
		if param.Constraints.Pattern != "" {
			schema["pattern"] = param.Constraints.Pattern
		}
		if param.Constraints.MinLength > 0 {
			schema["minLength"] = param.Constraints.MinLength
		}
		if param.Constraints.MaxLength > 0 {
			schema["maxLength"] = param.Constraints.MaxLength
		}
		if len(param.Constraints.Enum) > 0 {
			schema["enum"] = param.Constraints.Enum
		}

	case "integer", "number":
		if param.Constraints.Minimum != nil {
			schema["minimum"] = param.Constraints.Minimum
		}
		if param.Constraints.Maximum != nil {
			schema["maximum"] = param.Constraints.Maximum
		}
		if param.Constraints.MultipleOf != nil {
			schema["multipleOf"] = param.Constraints.MultipleOf
		}

	case "array":
		if param.Constraints.ItemsType != "" {
			schema["items"] = map[string]string{"type": param.Constraints.ItemsType}
		}
		if param.Constraints.MinItems > 0 {
			schema["minItems"] = param.Constraints.MinItems
		}
		if param.Constraints.MaxItems > 0 {
			schema["maxItems"] = param.Constraints.MaxItems
		}
		if param.Constraints.UniqueItems {
			schema["uniqueItems"] = true
		}

	case "object":
		if len(param.Constraints.RequiredFields) > 0 {
			schema["required"] = param.Constraints.RequiredFields
		}
	}

	return schema
}

// getRequiredParameters returns list of required parameters
func (pd *ParameterDocumenter) getRequiredParameters() []string {
	var required []string

	for name, param := range pd.parameters {
		if param.Required {
			required = append(required, name)
		}
	}

	return required
}

// GenerateCLIHelp generates interactive CLI help text
func (pd *ParameterDocumenter) GenerateCLIHelp() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Role: %s\n\n", pd.roleName))

	if len(pd.parameters) == 0 {
		sb.WriteString("This role has no configurable parameters.\n")
		return sb.String()
	}

	sb.WriteString("Parameters:\n\n")

	for name, param := range pd.parameters {
		req := ""
		if param.Required {
			req = " [REQUIRED]"
		}
		sb.WriteString(fmt.Sprintf("  %s <%s>%s\n", name, param.Type, req))

		if param.Description != "" {
			sb.WriteString(fmt.Sprintf("    %s\n", param.Description))
		}

		if param.Default != nil {
			sb.WriteString(fmt.Sprintf("    Default: %v\n", param.Default))
		}

		if !isZeroConstraints(param.Constraints) {
			sb.WriteString(pd.constraintsSummary(param))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// constraintsSummary generates a summary of constraints for CLI help
func (pd *ParameterDocumenter) constraintsSummary(param types.ParameterDef) string {
	var sb strings.Builder
	constraints := param.Constraints

	switch param.Type {
	case "string":
		if constraints.MinLength > 0 || constraints.MaxLength > 0 {
			sb.WriteString(fmt.Sprintf("    Length: %d-%d\n", constraints.MinLength, constraints.MaxLength))
		}
		if len(constraints.Enum) > 0 {
			sb.WriteString("    Values: ")
			for i, val := range constraints.Enum {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%v", val))
			}
			sb.WriteString("\n")
		}

	case "integer":
		if constraints.Minimum != nil || constraints.Maximum != nil {
			sb.WriteString(fmt.Sprintf("    Range: %v-%v\n", constraints.Minimum, constraints.Maximum))
		}

	case "array":
		if constraints.MinItems > 0 || constraints.MaxItems > 0 {
			sb.WriteString(fmt.Sprintf("    Items: %d-%d\n", constraints.MinItems, constraints.MaxItems))
		}
	}

	return sb.String()
}

// GenerateWebDocumentation generates HTML documentation
func (pd *ParameterDocumenter) GenerateWebDocumentation() string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
`)
	sb.WriteString(fmt.Sprintf("  <title>%s - Role Parameters</title>\n", pd.roleName))
	sb.WriteString(`  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 1200px; margin: 0 auto; padding: 20px; background-color: #f5f5f5; }
    .container { background: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
    h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
    h2 { color: #34495e; margin-top: 30px; }
    .parameter { margin: 20px 0; padding: 15px; background-color: #f9f9f9; border-left: 4px solid #3498db; border-radius: 4px; }
    .parameter-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
    .parameter-name { font-weight: bold; font-family: "Monaco", "Courier New", monospace; color: #2980b9; }
    .parameter-type { display: inline-block; background-color: #e8f4f8; padding: 2px 8px; border-radius: 3px; font-size: 0.9em; color: #16a085; font-family: "Monaco", "Courier New", monospace; }
    .required { display: inline-block; background-color: #ffe8e8; color: #c0392b; padding: 2px 8px; border-radius: 3px; font-size: 0.85em; font-weight: bold; margin-left: 10px; }
    .optional { display: inline-block; background-color: #e8f8e8; color: #27ae60; padding: 2px 8px; border-radius: 3px; font-size: 0.85em; margin-left: 10px; }
    .description { color: #555; margin: 10px 0; }
    .constraint { background-color: #fff; padding: 10px; margin: 5px 0; border-left: 3px solid #f39c12; font-size: 0.95em; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin: 20px 0; }
    .stat-card { background-color: #ecf0f1; padding: 15px; border-radius: 6px; text-align: center; }
    .stat-number { font-size: 2em; font-weight: bold; color: #3498db; }
    .stat-label { color: #555; font-size: 0.9em; margin-top: 5px; }
    .example { background-color: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 4px; font-family: "Monaco", "Courier New", monospace; overflow-x: auto; margin: 10px 0; }
  </style>
</head>
<body>
  <div class="container">
`)

	sb.WriteString(fmt.Sprintf("    <h1>%s - Role Parameters</h1>\n", pd.roleName))

	if pd.roleMeta.SchemaVersion > 0 {
		sb.WriteString(fmt.Sprintf("    <p><strong>Schema Version:</strong> %d</p>\n", pd.roleMeta.SchemaVersion))
	}

	stats := pd.GetStatistics()
	sb.WriteString("    <h2>Statistics</h2>\n")
	sb.WriteString("    <div class=\"stats\">\n")
	sb.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><div class=\"stat-number\">%d</div><div class=\"stat-label\">Total Parameters</div></div>\n", stats.Total))
	sb.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><div class=\"stat-number\">%d</div><div class=\"stat-label\">Required</div></div>\n", stats.Required))
	sb.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><div class=\"stat-number\">%d</div><div class=\"stat-label\">With Constraints</div></div>\n", stats.WithConstraints))
	sb.WriteString(fmt.Sprintf("      <div class=\"stat-card\"><div class=\"stat-number\">%d</div><div class=\"stat-label\">With Validators</div></div>\n", stats.WithValidators))
	sb.WriteString("    </div>\n")

	if len(pd.parameters) > 0 {
		sb.WriteString("    <h2>Parameters</h2>\n")
		for name, param := range pd.parameters {
			sb.WriteString("    <div class=\"parameter\">\n")
			sb.WriteString("      <div class=\"parameter-header\">\n")
			sb.WriteString(fmt.Sprintf("        <span class=\"parameter-name\">%s</span>\n", name))
			sb.WriteString(fmt.Sprintf("        <span class=\"parameter-type\">%s</span>\n", param.Type))
			if param.Required {
				sb.WriteString("        <span class=\"required\">REQUIRED</span>\n")
			} else {
				sb.WriteString("        <span class=\"optional\">OPTIONAL</span>\n")
			}
			sb.WriteString("      </div>\n")

			if param.Description != "" {
				sb.WriteString(fmt.Sprintf("      <div class=\"description\">%s</div>\n", param.Description))
			}

			if param.Default != nil {
				sb.WriteString(fmt.Sprintf("      <div><strong>Default:</strong> <code>%v</code></div>\n", param.Default))
			}

			if param.ConditionalRequirement != nil {
				sb.WriteString(fmt.Sprintf("      <div><strong>Required When:</strong> <code>%s</code></div>\n", param.ConditionalRequirement.Condition))
			}

			if !isZeroConstraints(param.Constraints) {
				sb.WriteString("      <div><strong>Constraints:</strong></div>\n")
				constraints := param.Constraints
				switch param.Type {
				case "string":
					if constraints.MinLength > 0 || constraints.MaxLength > 0 {
						sb.WriteString(fmt.Sprintf("      <div class=\"constraint\">Length: %d-%d</div>\n", constraints.MinLength, constraints.MaxLength))
					}
					if constraints.Pattern != "" {
						sb.WriteString(fmt.Sprintf("      <div class=\"constraint\">Pattern: <code>%s</code></div>\n", constraints.Pattern))
					}
				case "integer", "number":
					if constraints.Minimum != nil || constraints.Maximum != nil {
						sb.WriteString(fmt.Sprintf("      <div class=\"constraint\">Range: %v-%v</div>\n", constraints.Minimum, constraints.Maximum))
					}
				case "array":
					if constraints.MinItems > 0 || constraints.MaxItems > 0 {
						sb.WriteString(fmt.Sprintf("      <div class=\"constraint\">Items: %d-%d</div>\n", constraints.MinItems, constraints.MaxItems))
					}
				}
			}

			sb.WriteString("    </div>\n")
		}
	}

	if len(pd.parameters) > 0 {
		sb.WriteString("    <h2>Example</h2>\n")
		sb.WriteString("    <div class=\"example\">\n")
		sb.WriteString(fmt.Sprintf("- %s:\n", pd.roleName))
		for name, param := range pd.parameters {
			example := pd.generateExampleValue(name, param)
			sb.WriteString(fmt.Sprintf("    %s: %s\n", name, example))
		}
		sb.WriteString("    </div>\n")
	}

	sb.WriteString(`  </div>
</body>
</html>
`)

	return sb.String()
}

// generateExampleValue generates an example value based on parameter type and constraints
func (pd *ParameterDocumenter) generateExampleValue(name string, param types.ParameterDef) string {
	constraints := param.Constraints

	switch param.Type {
	case "string":
		if len(constraints.Enum) > 0 {
			return fmt.Sprintf("'%v'", constraints.Enum[0])
		}
		if constraints.Pattern != "" {
			return fmt.Sprintf("'example_%s'", name)
		}
		if param.Default != nil {
			return fmt.Sprintf("'%v'", param.Default)
		}
		return fmt.Sprintf("'%s_value'", name)

	case "integer":
		if constraints.Minimum != nil {
			return fmt.Sprintf("%v", constraints.Minimum)
		}
		if param.Default != nil {
			return fmt.Sprintf("%v", param.Default)
		}
		return "0"

	case "number", "float":
		if constraints.Minimum != nil {
			return fmt.Sprintf("%v", constraints.Minimum)
		}
		if param.Default != nil {
			return fmt.Sprintf("%v", param.Default)
		}
		return "0.0"

	case "boolean":
		if param.Default != nil {
			return fmt.Sprintf("%v", param.Default)
		}
		return "true"

	case "array":
		if constraints.ItemsType != "" {
			return fmt.Sprintf("[]  # Array of %s", constraints.ItemsType)
		}
		return "[]"

	case "object":
		return "{}"

	default:
		return "null"
	}
}

// GenerateDocumentation generates documentation in the specified format
func (pd *ParameterDocumenter) GenerateDocumentation(format DocumentFormat) (string, error) {
	switch format {
	case FormatMarkdown:
		return pd.GenerateMarkdown(), nil
	case FormatJSONSchema:
		return pd.GenerateJSONSchema()
	case FormatCLI:
		return pd.GenerateCLIHelp(), nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

// ValidateDocumentation validates if documentation can be generated
func (pd *ParameterDocumenter) ValidateDocumentation() error {
	if pd.roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	for name, param := range pd.parameters {
		if param.Type == "" {
			return fmt.Errorf("parameter %s has no type defined", name)
		}
	}

	return nil
}

// ConvertToJSON converts documentation to JSON format
func (pd *ParameterDocumenter) ConvertToJSON() (string, error) {
	doc := map[string]interface{}{
		"roleName": pd.roleName,
		"version":  pd.roleMeta.SchemaVersion,
		"parameters": func() []map[string]interface{} {
			var result []map[string]interface{}
			for name, param := range pd.parameters {
				p := map[string]interface{}{
					"name":        name,
					"type":        param.Type,
					"required":    param.Required,
					"description": param.Description,
				}
				if param.Default != nil {
					p["default"] = param.Default
				}
				result = append(result, p)
			}
			return result
		}(),
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ConvertToYAML converts documentation to YAML-compatible format
func (pd *ParameterDocumenter) ConvertToYAML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("role: %s\n", pd.roleName))
	if pd.roleMeta.SchemaVersion > 0 {
		sb.WriteString(fmt.Sprintf("version: %d\n", pd.roleMeta.SchemaVersion))
	}
	sb.WriteString("parameters:\n")

	for name, param := range pd.parameters {
		sb.WriteString(fmt.Sprintf("  %s:\n", name))
		sb.WriteString(fmt.Sprintf("    type: %s\n", param.Type))
		sb.WriteString(fmt.Sprintf("    required: %v\n", param.Required))
		if param.Description != "" {
			sb.WriteString(fmt.Sprintf("    description: %s\n", strconv.Quote(param.Description)))
		}
		if param.Default != nil {
			sb.WriteString(fmt.Sprintf("    default: %v\n", param.Default))
		}
	}

	return sb.String()
}

// ExportExample exports a complete example in YAML format
func (pd *ParameterDocumenter) ExportExample() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("- %s:\n", pd.roleName))

	for name, param := range pd.parameters {
		example := pd.generateExampleValue(name, param)
		sb.WriteString(fmt.Sprintf("    %s: %s\n", name, example))
	}

	return sb.String()
}

// hasConstraints checks if any parameter has constraints
func (pd *ParameterDocumenter) hasConstraints() bool {
	for _, param := range pd.parameters {
		if !isZeroConstraints(param.Constraints) {
			return true
		}
	}
	return false
}

// isZeroConstraints checks if constraints are empty
func isZeroConstraints(c types.ParameterConstraints) bool {
	return c.Pattern == "" &&
		c.MinLength == 0 &&
		c.MaxLength == 0 &&
		len(c.Enum) == 0 &&
		c.Minimum == nil &&
		c.Maximum == nil &&
		c.MultipleOf == nil &&
		c.ItemsType == "" &&
		c.MinItems == 0 &&
		c.MaxItems == 0 &&
		!c.UniqueItems &&
		len(c.RequiredFields) == 0
}
