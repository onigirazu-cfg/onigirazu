package documentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func createTestRoleMeta() *types.RoleMeta {
	return &types.RoleMeta{
		SchemaVersion: 1,
		Parameters: map[string]types.ParameterDef{
			"port": {
				Type:        "integer",
				Required:    true,
				Default:     8080,
				Description: "Service port number",
				Constraints: types.ParameterConstraints{
					Minimum: 1024,
					Maximum: 65535,
				},
			},
			"hostname": {
				Type:        "string",
				Required:    true,
				Description: "Hostname or IP address",
				Constraints: types.ParameterConstraints{
					MinLength: 3,
					MaxLength: 255,
					Pattern:   "^[a-zA-Z0-9.-]+$",
				},
			},
			"enabled": {
				Type:        "boolean",
				Required:    false,
				Default:     true,
				Description: "Whether the service is enabled",
			},
			"tags": {
				Type:        "array",
				Required:    false,
				Description: "Service tags",
				Constraints: types.ParameterConstraints{
					ItemsType: "string",
					MinItems:  0,
					MaxItems:  10,
				},
			},
			"metadata": {
				Type:        "object",
				Required:    false,
				Description: "Service metadata",
				Constraints: types.ParameterConstraints{
					RequiredFields: []string{"version"},
				},
			},
			"api_key": {
				Type:        "string",
				Required:    false,
				Description: "API key for authentication",
				ConditionalRequirement: &types.ConditionalRequirement{
					Condition:   "enable_auth=true",
					Description: "Required when authentication is enabled",
				},
			},
		},
	}
}

func TestNewParameterDocumenter(t *testing.T) {
	tests := []struct {
		name     string
		roleName string
		roleMeta *types.RoleMeta
	}{
		{"Valid", "web_server", createTestRoleMeta()},
		{"Nil meta", "app_server", nil},
		{"Empty name", "", createTestRoleMeta()},
	}

	for _, tt := range tests {
		doc := NewParameterDocumenter(tt.roleName, tt.roleMeta)
		if doc == nil {
			t.Error("documenter is nil")
		}
	}
}

func TestCountParameters(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	if doc.CountParameters() != 6 {
		t.Error("wrong parameter count")
	}
}

func TestGetStatistics(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	stats := doc.GetStatistics()

	if stats.Total != 6 {
		t.Errorf("total: expected 6, got %d", stats.Total)
	}
	if stats.Required != 2 {
		t.Errorf("required: expected 2, got %d", stats.Required)
	}
	if stats.StringParams != 2 {
		t.Errorf("strings: expected 2, got %d", stats.StringParams)
	}
}

func TestGenerateMarkdown(t *testing.T) {
	doc := NewParameterDocumenter("test_role", createTestRoleMeta())
	md := doc.GenerateMarkdown()

	tests := []string{
		"# test_role Role Parameters",
		"## Parameters",
		"| Parameter",
		"`port`",
		"`hostname`",
	}

	for _, expected := range tests {
		if !strings.Contains(md, expected) {
			t.Errorf("markdown missing: %s", expected)
		}
	}
}

func TestGenerateJSONSchema(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	schema, err := doc.GenerateJSONSchema()

	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["$schema"] == nil {
		t.Error("missing $schema")
	}
}

func TestGenerateCLIHelp(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	help := doc.GenerateCLIHelp()

	if !strings.Contains(help, "Role: test") {
		t.Error("missing role header")
	}
	if !strings.Contains(help, "port <integer>") {
		t.Error("missing port parameter")
	}
}

func TestGenerateWebDocumentation(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	html := doc.GenerateWebDocumentation()

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("missing HTML doctype")
	}
	if !strings.Contains(html, "test - Role Parameters") {
		t.Error("missing title")
	}
}

func TestExampleGeneration(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())

	tests := []struct {
		name     string
		paramDef types.ParameterDef
	}{
		{"String", types.ParameterDef{Type: "string"}},
		{"Integer", types.ParameterDef{Type: "integer"}},
		{"Boolean", types.ParameterDef{Type: "boolean"}},
		{"Array", types.ParameterDef{Type: "array"}},
		{"Object", types.ParameterDef{Type: "object"}},
	}

	for _, tt := range tests {
		result := doc.generateExampleValue("test", tt.paramDef)
		if result == "" {
			t.Errorf("%s: empty example", tt.name)
		}
	}
}

func TestConvertToJSON(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	jsonStr, err := doc.ConvertToJSON()

	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["roleName"] != "test" {
		t.Error("wrong roleName")
	}
}

func TestConvertToYAML(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	yaml := doc.ConvertToYAML()

	if !strings.Contains(yaml, "role: test") {
		t.Error("missing role line")
	}
	if !strings.Contains(yaml, "parameters:") {
		t.Error("missing parameters section")
	}
}

func TestExportExample(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	example := doc.ExportExample()

	if !strings.Contains(example, "- test:") {
		t.Error("missing role header")
	}
	if !strings.Contains(example, "port:") {
		t.Error("missing port")
	}
}

func TestValidateDocumentation(t *testing.T) {
	tests := []struct {
		name      string
		roleName  string
		roleMeta  *types.RoleMeta
		shouldErr bool
	}{
		{"Valid", "test", createTestRoleMeta(), false},
		{"Empty name", "", createTestRoleMeta(), true},
		{"No type", "test", &types.RoleMeta{Parameters: map[string]types.ParameterDef{"x": {}}}, true},
	}

	for _, tt := range tests {
		doc := NewParameterDocumenter(tt.roleName, tt.roleMeta)
		err := doc.ValidateDocumentation()
		if (err != nil) != tt.shouldErr {
			t.Errorf("%s: expected err=%v, got %v", tt.name, tt.shouldErr, err != nil)
		}
	}
}

func TestGenerateDocumentation(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())

	formats := []struct {
		format DocumentFormat
		check  func(string) bool
	}{
		{FormatMarkdown, func(s string) bool { return strings.Contains(s, "#") }},
		{FormatJSONSchema, func(s string) bool { return strings.Contains(s, "$schema") }},
		{FormatCLI, func(s string) bool { return strings.Contains(s, "Role:") }},
	}

	for _, ff := range formats {
		result, err := doc.GenerateDocumentation(ff.format)
		if err != nil {
			t.Errorf("format %s: error %v", ff.format, err)
		}
		if !ff.check(result) {
			t.Errorf("format %s: check failed", ff.format)
		}
	}
}

func TestInvalidFormat(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	_, err := doc.GenerateDocumentation("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestConditionalRequirement(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Required When:") {
		t.Error("missing conditional requirement")
	}
	if !strings.Contains(md, "enable_auth=true") {
		t.Error("missing condition")
	}
}

func TestEmptyParameters(t *testing.T) {
	doc := NewParameterDocumenter("test", &types.RoleMeta{
		Parameters: make(map[string]types.ParameterDef),
	})

	cli := doc.GenerateCLIHelp()
	if !strings.Contains(cli, "no configurable parameters") {
		t.Error("should mention no parameters")
	}
}

func TestConstraintsDocumentation(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Min Length") {
		t.Error("missing MinLength constraint")
	}
	if !strings.Contains(md, "Pattern") {
		t.Error("missing Pattern constraint")
	}
}

func TestEnumValues(t *testing.T) {
	meta := &types.RoleMeta{
		Parameters: map[string]types.ParameterDef{
			"service_type": {
				Type: "string",
				Constraints: types.ParameterConstraints{
					Enum: []interface{}{"http", "https"},
				},
			},
		},
	}

	doc := NewParameterDocumenter("test", meta)
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Allowed Values") {
		t.Error("missing Allowed Values")
	}
}

func TestSchemaVersion(t *testing.T) {
	meta := createTestRoleMeta()
	meta.SchemaVersion = 3

	doc := NewParameterDocumenter("test", meta)
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Schema Version: 3") {
		t.Error("missing schema version")
	}
}

func TestCustomValidators(t *testing.T) {
	meta := &types.RoleMeta{
		Parameters: map[string]types.ParameterDef{
			"config": {
				Type: "string",
				Validators: []types.CustomValidator{
					{Name: "readable", Description: "Must be readable"},
				},
			},
		},
	}

	doc := NewParameterDocumenter("test", meta)
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Validators:") {
		t.Error("missing Validators section")
	}
	if !strings.Contains(md, "readable") {
		t.Error("missing validator")
	}
}

func TestRequiredArray(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	schema, _ := doc.GenerateJSONSchema()

	var result map[string]interface{}
	json.Unmarshal([]byte(schema), &result)

	required, ok := result["required"].([]interface{})
	if !ok {
		t.Error("required not an array")
	}
	if len(required) != 2 {
		t.Errorf("expected 2 required, got %d", len(required))
	}
}

func TestHTMLStructure(t *testing.T) {
	doc := NewParameterDocumenter("test", createTestRoleMeta())
	html := doc.GenerateWebDocumentation()

	checks := []struct{ open, close string }{
		{"<html", "</html>"},
		{"<head>", "</head>"},
		{"<body>", "</body>"},
	}

	for _, c := range checks {
		if !strings.Contains(html, c.open) {
			t.Errorf("missing %s", c.open)
		}
		if !strings.Contains(html, c.close) {
			t.Errorf("missing %s", c.close)
		}
	}
}

func TestFloatType(t *testing.T) {
	meta := &types.RoleMeta{
		Parameters: map[string]types.ParameterDef{
			"timeout": {
				Type: "float",
				Constraints: types.ParameterConstraints{
					Minimum: 0.1,
					Maximum: 300.5,
				},
			},
		},
	}

	doc := NewParameterDocumenter("test", meta)
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "float") {
		t.Error("missing float type")
	}
}

func TestMultipleOfConstraint(t *testing.T) {
	multiple := 5
	meta := &types.RoleMeta{
		Parameters: map[string]types.ParameterDef{
			"batch_size": {
				Type: "integer",
				Constraints: types.ParameterConstraints{
					MultipleOf: &multiple,
				},
			},
		},
	}

	doc := NewParameterDocumenter("test", meta)
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Multiple Of") {
		t.Error("missing MultipleOf constraint")
	}
}

func TestComplexConstraints(t *testing.T) {
	meta := &types.RoleMeta{
		Parameters: map[string]types.ParameterDef{
			"items": {
				Type: "array",
				Constraints: types.ParameterConstraints{
					ItemsType:   "string",
					MinItems:    1,
					MaxItems:    100,
					UniqueItems: true,
				},
			},
		},
	}

	doc := NewParameterDocumenter("test", meta)
	md := doc.GenerateMarkdown()

	if !strings.Contains(md, "Unique Items") {
		t.Error("missing Unique Items")
	}
	if !strings.Contains(md, "Item Type") {
		t.Error("missing Item Type")
	}
}
