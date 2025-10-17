package validator

import (
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestValidateStringParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramSchema map[string]types.ParameterDef
		vars        map[string]interface{}
		shouldPass  bool
		errorCount  int
	}{
		{
			name: "valid string parameter",
			paramSchema: map[string]types.ParameterDef{
				"hostname": {
					Type:     "string",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"hostname": "example.com",
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "missing required string parameter",
			paramSchema: map[string]types.ParameterDef{
				"hostname": {
					Type:     "string",
					Required: true,
				},
			},
			vars:       map[string]interface{}{},
			shouldPass: false,
			errorCount: 1,
		},
		{
			name: "string too short",
			paramSchema: map[string]types.ParameterDef{
				"domain": {
					Type:     "string",
					Required: true,
					Constraints: types.ParameterConstraints{
						MinLength: 5,
					},
				},
			},
			vars: map[string]interface{}{
				"domain": "a.b",
			},
			shouldPass: false,
			errorCount: 1,
		},
		{
			name: "string too long",
			paramSchema: map[string]types.ParameterDef{
				"domain": {
					Type:     "string",
					Required: true,
					Constraints: types.ParameterConstraints{
						MaxLength: 10,
					},
				},
			},
			vars: map[string]interface{}{
				"domain": "verylongdomainname.com",
			},
			shouldPass: false,
			errorCount: 1,
		},
		{
			name: "string pattern validation - valid",
			paramSchema: map[string]types.ParameterDef{
				"email": {
					Type:     "string",
					Required: true,
					Constraints: types.ParameterConstraints{
						Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
					},
				},
			},
			vars: map[string]interface{}{
				"email": "user@example.com",
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "string pattern validation - invalid",
			paramSchema: map[string]types.ParameterDef{
				"email": {
					Type:     "string",
					Required: true,
					Constraints: types.ParameterConstraints{
						Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
					},
				},
			},
			vars: map[string]interface{}{
				"email": "invalid-email",
			},
			shouldPass: false,
			errorCount: 1,
		},
		{
			name: "string enum validation - valid",
			paramSchema: map[string]types.ParameterDef{
				"environment": {
					Type:     "string",
					Required: true,
					Constraints: types.ParameterConstraints{
						Enum: []interface{}{"dev", "staging", "production"},
					},
				},
			},
			vars: map[string]interface{}{
				"environment": "production",
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "string enum validation - invalid",
			paramSchema: map[string]types.ParameterDef{
				"environment": {
					Type:     "string",
					Required: true,
					Constraints: types.ParameterConstraints{
						Enum: []interface{}{"dev", "staging", "production"},
					},
				},
			},
			vars: map[string]interface{}{
				"environment": "testing",
			},
			shouldPass: false,
			errorCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("expected Valid=%v, got %v", tt.shouldPass, result.Valid)
			}

			if len(result.Errors) != tt.errorCount {
				t.Errorf("expected %d errors, got %d: %v", tt.errorCount, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestValidateIntegerParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramSchema map[string]types.ParameterDef
		vars        map[string]interface{}
		shouldPass  bool
		errorCount  int
	}{
		{
			name: "valid integer parameter",
			paramSchema: map[string]types.ParameterDef{
				"port": {
					Type:     "integer",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"port": 8080,
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "integer below minimum",
			paramSchema: map[string]types.ParameterDef{
				"port": {
					Type:     "integer",
					Required: true,
					Constraints: types.ParameterConstraints{
						Minimum: 1024,
					},
				},
			},
			vars: map[string]interface{}{
				"port": 80,
			},
			shouldPass: false,
			errorCount: 1,
		},
		{
			name: "integer above maximum",
			paramSchema: map[string]types.ParameterDef{
				"port": {
					Type:     "integer",
					Required: true,
					Constraints: types.ParameterConstraints{
						Maximum: 65535,
					},
				},
			},
			vars: map[string]interface{}{
				"port": 70000,
			},
			shouldPass: false,
			errorCount: 1,
		},
		{
			name: "integer from float64",
			paramSchema: map[string]types.ParameterDef{
				"count": {
					Type:     "integer",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"count": float64(42),
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "integer multiple of validation - valid",
			paramSchema: map[string]types.ParameterDef{
				"threads": {
					Type:     "integer",
					Required: true,
					Constraints: types.ParameterConstraints{
						MultipleOf: 4,
					},
				},
			},
			vars: map[string]interface{}{
				"threads": 16,
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "integer multiple of validation - invalid",
			paramSchema: map[string]types.ParameterDef{
				"threads": {
					Type:     "integer",
					Required: true,
					Constraints: types.ParameterConstraints{
						MultipleOf: 4,
					},
				},
			},
			vars: map[string]interface{}{
				"threads": 15,
			},
			shouldPass: false,
			errorCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("expected Valid=%v, got %v", tt.shouldPass, result.Valid)
			}

			if len(result.Errors) != tt.errorCount {
				t.Errorf("expected %d errors, got %d: %v", tt.errorCount, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestValidateBooleanParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramSchema map[string]types.ParameterDef
		vars        map[string]interface{}
		shouldPass  bool
	}{
		{
			name: "valid boolean true",
			paramSchema: map[string]types.ParameterDef{
				"ssl": {
					Type:     "boolean",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"ssl": true,
			},
			shouldPass: true,
		},
		{
			name: "valid boolean false",
			paramSchema: map[string]types.ParameterDef{
				"ssl": {
					Type:     "boolean",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"ssl": false,
			},
			shouldPass: true,
		},
		{
			name: "boolean from string 'yes'",
			paramSchema: map[string]types.ParameterDef{
				"ssl": {
					Type:     "boolean",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"ssl": "yes",
			},
			shouldPass: true,
		},
		{
			name: "boolean from string 'no'",
			paramSchema: map[string]types.ParameterDef{
				"ssl": {
					Type:     "boolean",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"ssl": "no",
			},
			shouldPass: true,
		},
		{
			name: "invalid boolean",
			paramSchema: map[string]types.ParameterDef{
				"ssl": {
					Type:     "boolean",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"ssl": "maybe",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("expected Valid=%v, got %v", tt.shouldPass, result.Valid)
			}
		})
	}
}

func TestValidateArrayParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramSchema map[string]types.ParameterDef
		vars        map[string]interface{}
		shouldPass  bool
	}{
		{
			name: "valid array",
			paramSchema: map[string]types.ParameterDef{
				"hosts": {
					Type:     "array",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"hosts": []interface{}{"host1", "host2", "host3"},
			},
			shouldPass: true,
		},
		{
			name: "array too short",
			paramSchema: map[string]types.ParameterDef{
				"hosts": {
					Type:     "array",
					Required: true,
					Constraints: types.ParameterConstraints{
						MinItems: 3,
					},
				},
			},
			vars: map[string]interface{}{
				"hosts": []interface{}{"host1"},
			},
			shouldPass: false,
		},
		{
			name: "array too long",
			paramSchema: map[string]types.ParameterDef{
				"hosts": {
					Type:     "array",
					Required: true,
					Constraints: types.ParameterConstraints{
						MaxItems: 2,
					},
				},
			},
			vars: map[string]interface{}{
				"hosts": []interface{}{"host1", "host2", "host3"},
			},
			shouldPass: false,
		},
		{
			name: "array with unique items constraint - valid",
			paramSchema: map[string]types.ParameterDef{
				"tags": {
					Type:     "array",
					Required: true,
					Constraints: types.ParameterConstraints{
						UniqueItems: true,
					},
				},
			},
			vars: map[string]interface{}{
				"tags": []interface{}{"prod", "web", "api"},
			},
			shouldPass: true,
		},
		{
			name: "array with unique items constraint - invalid",
			paramSchema: map[string]types.ParameterDef{
				"tags": {
					Type:     "array",
					Required: true,
					Constraints: types.ParameterConstraints{
						UniqueItems: true,
					},
				},
			},
			vars: map[string]interface{}{
				"tags": []interface{}{"prod", "web", "prod"},
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("expected Valid=%v, got %v", tt.shouldPass, result.Valid)
			}
		})
	}
}

func TestValidateObjectParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramSchema map[string]types.ParameterDef
		vars        map[string]interface{}
		shouldPass  bool
	}{
		{
			name: "valid object",
			paramSchema: map[string]types.ParameterDef{
				"config": {
					Type:     "object",
					Required: true,
				},
			},
			vars: map[string]interface{}{
				"config": map[string]interface{}{
					"setting1": "value1",
					"setting2": 42,
				},
			},
			shouldPass: true,
		},
		{
			name: "object missing required field",
			paramSchema: map[string]types.ParameterDef{
				"config": {
					Type:     "object",
					Required: true,
					Constraints: types.ParameterConstraints{
						RequiredFields: []string{"database_url", "api_key"},
					},
				},
			},
			vars: map[string]interface{}{
				"config": map[string]interface{}{
					"database_url": "postgres://...",
				},
			},
			shouldPass: false,
		},
		{
			name: "object with all required fields",
			paramSchema: map[string]types.ParameterDef{
				"config": {
					Type:     "object",
					Required: true,
					Constraints: types.ParameterConstraints{
						RequiredFields: []string{"database_url", "api_key"},
					},
				},
			},
			vars: map[string]interface{}{
				"config": map[string]interface{}{
					"database_url": "postgres://...",
					"api_key":      "secret",
				},
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("expected Valid=%v, got %v", tt.shouldPass, result.Valid)
			}
		})
	}
}

func TestMergeWithDefaults(t *testing.T) {
	tests := []struct {
		name          string
		paramSchema   map[string]types.ParameterDef
		vars          map[string]interface{}
		expectedVars  map[string]interface{}
		expectedCount int
	}{
		{
			name: "merge with defaults - all defaults",
			paramSchema: map[string]types.ParameterDef{
				"port": {
					Type:    "integer",
					Default: 8080,
				},
				"ssl": {
					Type:    "boolean",
					Default: true,
				},
			},
			vars: map[string]interface{}{},
			expectedVars: map[string]interface{}{
				"port": 8080,
				"ssl":  true,
			},
			expectedCount: 2,
		},
		{
			name: "merge with defaults - override some",
			paramSchema: map[string]types.ParameterDef{
				"port": {
					Type:    "integer",
					Default: 8080,
				},
				"ssl": {
					Type:    "boolean",
					Default: true,
				},
			},
			vars: map[string]interface{}{
				"port": 9000,
			},
			expectedVars: map[string]interface{}{
				"port": 9000,
				"ssl":  true,
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.MergeWithDefaults(tt.vars)

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d parameters, got %d", tt.expectedCount, len(result))
			}

			for key, expected := range tt.expectedVars {
				actual, exists := result[key]
				if !exists {
					t.Errorf("expected parameter %q not found in result", key)
				}
				if actual != expected {
					t.Errorf("parameter %q: expected %v, got %v", key, expected, actual)
				}
			}
		})
	}
}

func TestOptionalParameters(t *testing.T) {
	paramSchema := map[string]types.ParameterDef{
		"hostname": {
			Type:     "string",
			Required: true,
		},
		"port": {
			Type:     "integer",
			Required: false,
			Default:  8080,
		},
		"ssl": {
			Type:     "boolean",
			Required: false,
		},
	}

	tests := []struct {
		name       string
		vars       map[string]interface{}
		shouldPass bool
		errorCount int
	}{
		{
			name: "only required parameter provided",
			vars: map[string]interface{}{
				"hostname": "example.com",
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "required and optional parameters",
			vars: map[string]interface{}{
				"hostname": "example.com",
				"port":     9000,
				"ssl":      true,
			},
			shouldPass: true,
			errorCount: 0,
		},
		{
			name: "missing required parameter",
			vars: map[string]interface{}{
				"port": 9000,
			},
			shouldPass: false,
			errorCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("expected Valid=%v, got %v", tt.shouldPass, result.Valid)
			}

			if len(result.Errors) != tt.errorCount {
				t.Errorf("expected %d errors, got %d: %v", tt.errorCount, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestParameterDescription(t *testing.T) {
	paramSchema := map[string]types.ParameterDef{
		"hostname": {
			Type:        "string",
			Required:    true,
			Description: "The hostname to configure",
		},
		"port": {
			Type:        "integer",
			Required:    false,
			Default:     8080,
			Description: "The port to listen on",
			Constraints: types.ParameterConstraints{
				Minimum: 1024,
				Maximum: 65535,
			},
		},
	}

	validator := NewParameterValidator(paramSchema)
	desc := validator.GetParameterDescription()

	if desc == "" {
		t.Error("expected non-empty description")
	}

	if !strings.Contains(desc, "hostname") {
		t.Error("expected description to contain 'hostname'")
	}

	if !strings.Contains(desc, "port") {
		t.Error("expected description to contain 'port'")
	}

	if !strings.Contains(desc, "required") {
		t.Error("expected description to contain 'required'")
	}
}

func TestConditionalRequirement(t *testing.T) {
	tests := []struct {
		name        string
		paramSchema map[string]types.ParameterDef
		vars        map[string]interface{}
		shouldPass  bool
		errorCount  int
		description string
	}{
		{
			name: "conditional requirement - condition false, parameter not required",
			paramSchema: map[string]types.ParameterDef{
				"enable_auth": {
					Type:     "boolean",
					Required: false,
				},
				"api_key": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "enable_auth=true",
						ErrorMsg:  "api_key is required when authentication is enabled",
					},
				},
			},
			vars: map[string]interface{}{
				"enable_auth": false,
			},
			shouldPass:  true,
			errorCount:  0,
			description: "api_key not required when enable_auth=false",
		},
		{
			name: "conditional requirement - condition true, parameter required but missing",
			paramSchema: map[string]types.ParameterDef{
				"enable_auth": {
					Type:     "boolean",
					Required: false,
				},
				"api_key": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "enable_auth=true",
						ErrorMsg:  "api_key is required when authentication is enabled",
					},
				},
			},
			vars: map[string]interface{}{
				"enable_auth": true,
			},
			shouldPass:  false,
			errorCount:  1,
			description: "api_key required when enable_auth=true",
		},
		{
			name: "conditional requirement - condition true, parameter provided",
			paramSchema: map[string]types.ParameterDef{
				"enable_auth": {
					Type:     "boolean",
					Required: false,
				},
				"api_key": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "enable_auth=true",
						ErrorMsg:  "api_key is required when authentication is enabled",
					},
				},
			},
			vars: map[string]interface{}{
				"enable_auth": true,
				"api_key":     "secret123",
			},
			shouldPass:  true,
			errorCount:  0,
			description: "api_key provided when required",
		},
		{
			name: "conditional requirement - complex condition with AND",
			paramSchema: map[string]types.ParameterDef{
				"backup_enabled": {
					Type:     "boolean",
					Required: false,
				},
				"backup_mode": {
					Type:     "string",
					Required: false,
				},
				"backup_schedule": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "backup_enabled=true && backup_mode=full",
						ErrorMsg:  "backup_schedule required for full backups",
					},
				},
			},
			vars: map[string]interface{}{
				"backup_enabled": true,
				"backup_mode":    "full",
			},
			shouldPass:  false,
			errorCount:  1,
			description: "backup_schedule required when both conditions met",
		},
		{
			name: "conditional requirement - complex condition with AND (not all true)",
			paramSchema: map[string]types.ParameterDef{
				"backup_enabled": {
					Type:     "boolean",
					Required: false,
				},
				"backup_mode": {
					Type:     "string",
					Required: false,
				},
				"backup_schedule": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "backup_enabled=true && backup_mode=full",
					},
				},
			},
			vars: map[string]interface{}{
				"backup_enabled": true,
				"backup_mode":    "incremental",
			},
			shouldPass:  true,
			errorCount:  0,
			description: "backup_schedule not required when not all AND conditions met",
		},
		{
			name: "conditional requirement - multiple conditional parameters",
			paramSchema: map[string]types.ParameterDef{
				"enable_ssl": {
					Type:     "boolean",
					Required: false,
				},
				"ssl_cert": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "enable_ssl=true",
						ErrorMsg:  "ssl_cert is required when SSL is enabled",
					},
				},
				"ssl_key": {
					Type:     "string",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "enable_ssl=true",
						ErrorMsg:  "ssl_key is required when SSL is enabled",
					},
				},
			},
			vars: map[string]interface{}{
				"enable_ssl": true,
				"ssl_cert":   "cert.pem",
				// ssl_key missing
			},
			shouldPass:  false,
			errorCount:  1,
			description: "multiple conditional parameters with some missing",
		},
		{
			name: "conditional requirement - numeric comparison",
			paramSchema: map[string]types.ParameterDef{
				"cluster_mode": {
					Type:     "boolean",
					Required: false,
				},
				"replica_count": {
					Type:     "integer",
					Required: false,
					ConditionalRequirement: &types.ConditionalRequirement{
						Condition: "cluster_mode=true",
						ErrorMsg:  "replica_count is required in cluster mode",
					},
				},
			},
			vars: map[string]interface{}{
				"cluster_mode": true,
			},
			shouldPass:  false,
			errorCount:  1,
			description: "numeric parameter required conditionally",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewParameterValidator(tt.paramSchema)
			result := validator.ValidateParameters(tt.vars)

			if result.Valid != tt.shouldPass {
				t.Errorf("validation result = %v, want %v (%s)", result.Valid, tt.shouldPass, tt.description)
			}

			if len(result.Errors) != tt.errorCount {
				t.Errorf("error count = %d, want %d (%s). Errors: %v", len(result.Errors), tt.errorCount, tt.description, result.Errors)
			}
		})
	}
}
