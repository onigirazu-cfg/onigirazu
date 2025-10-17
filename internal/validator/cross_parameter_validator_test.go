package validator

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestEqualityComparison tests equality comparisons
func TestEqualityComparison(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "string equality match",
			rules: []types.CrossParameterRule{
				{Rule: "service=http", ErrorMsg: "service must be http"},
			},
			params:   map[string]interface{}{"service": "http"},
			expected: true,
			errCount: 0,
		},
		{
			name: "string equality mismatch",
			rules: []types.CrossParameterRule{
				{Rule: "service=http", ErrorMsg: "service must be http"},
			},
			params:   map[string]interface{}{"service": "https"},
			expected: false,
			errCount: 1,
		},
		{
			name: "integer equality match",
			rules: []types.CrossParameterRule{
				{Rule: "port=8080", ErrorMsg: "port must be 8080"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "integer equality from float",
			rules: []types.CrossParameterRule{
				{Rule: "port=8080", ErrorMsg: "port must be 8080"},
			},
			params:   map[string]interface{}{"port": 8080.0},
			expected: true,
			errCount: 0,
		},
		{
			name: "boolean equality true",
			rules: []types.CrossParameterRule{
				{Rule: "ssl_enabled=true", ErrorMsg: "ssl must be enabled"},
			},
			params:   map[string]interface{}{"ssl_enabled": true},
			expected: true,
			errCount: 0,
		},
		{
			name: "boolean equality false",
			rules: []types.CrossParameterRule{
				{Rule: "debug_mode=false", ErrorMsg: "debug mode must be disabled"},
			},
			params:   map[string]interface{}{"debug_mode": false},
			expected: true,
			errCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestNotEqualComparison tests not equal comparisons
func TestNotEqualComparison(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "not equal match (string)",
			rules: []types.CrossParameterRule{
				{Rule: "service!=telnet", ErrorMsg: "service must not be telnet"},
			},
			params:   map[string]interface{}{"service": "ssh"},
			expected: true,
			errCount: 0,
		},
		{
			name: "not equal mismatch (string)",
			rules: []types.CrossParameterRule{
				{Rule: "service!=telnet", ErrorMsg: "service must not be telnet"},
			},
			params:   map[string]interface{}{"service": "telnet"},
			expected: false,
			errCount: 1,
		},
		{
			name: "not equal match (integer)",
			rules: []types.CrossParameterRule{
				{Rule: "port!=22", ErrorMsg: "port must not be 22"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestGreaterComparison tests greater than comparisons
func TestGreaterComparison(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "greater than match",
			rules: []types.CrossParameterRule{
				{Rule: "port>1024", ErrorMsg: "port must be > 1024"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "greater than equal mismatch",
			rules: []types.CrossParameterRule{
				{Rule: "port>8080", ErrorMsg: "port must be > 8080"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: false,
			errCount: 1,
		},
		{
			name: "greater than less mismatch",
			rules: []types.CrossParameterRule{
				{Rule: "port>9000", ErrorMsg: "port must be > 9000"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestLessComparison tests less than comparisons
func TestLessComparison(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "less than match",
			rules: []types.CrossParameterRule{
				{Rule: "port<65536", ErrorMsg: "port must be < 65536"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "less than equal mismatch",
			rules: []types.CrossParameterRule{
				{Rule: "port<8080", ErrorMsg: "port must be < 8080"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestGreaterEqualComparison tests >= comparisons
func TestGreaterEqualComparison(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "greater equal match",
			rules: []types.CrossParameterRule{
				{Rule: "port>=8080", ErrorMsg: "port must be >= 8080"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "greater equal greater",
			rules: []types.CrossParameterRule{
				{Rule: "port>=8000", ErrorMsg: "port must be >= 8000"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "greater equal mismatch",
			rules: []types.CrossParameterRule{
				{Rule: "port>=9000", ErrorMsg: "port must be >= 9000"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestLessEqualComparison tests <= comparisons
func TestLessEqualComparison(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "less equal match",
			rules: []types.CrossParameterRule{
				{Rule: "port<=8080", ErrorMsg: "port must be <= 8080"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "less equal less",
			rules: []types.CrossParameterRule{
				{Rule: "port<=9000", ErrorMsg: "port must be <= 9000"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: true,
			errCount: 0,
		},
		{
			name: "less equal mismatch",
			rules: []types.CrossParameterRule{
				{Rule: "port<=8000", ErrorMsg: "port must be <= 8000"},
			},
			params:   map[string]interface{}{"port": 8080},
			expected: false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestAndOperator tests && (AND) operator
func TestAndOperator(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "AND both true",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80", ErrorMsg: "HTTP must use port 80"},
			},
			params: map[string]interface{}{
				"service": "http",
				"port":    80,
			},
			expected: true,
			errCount: 0,
		},
		{
			name: "AND first false",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80", ErrorMsg: "HTTP must use port 80"},
			},
			params: map[string]interface{}{
				"service": "https",
				"port":    80,
			},
			expected: false,
			errCount: 1,
		},
		{
			name: "AND second false",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80", ErrorMsg: "HTTP must use port 80"},
			},
			params: map[string]interface{}{
				"service": "http",
				"port":    443,
			},
			expected: false,
			errCount: 1,
		},
		{
			name: "AND both false",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80", ErrorMsg: "HTTP must use port 80"},
			},
			params: map[string]interface{}{
				"service": "https",
				"port":    443,
			},
			expected: false,
			errCount: 1,
		},
		{
			name: "AND multiple conditions",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80 && ssl_enabled=false", ErrorMsg: "HTTP must use port 80 without SSL"},
			},
			params: map[string]interface{}{
				"service":     "http",
				"port":        80,
				"ssl_enabled": false,
			},
			expected: true,
			errCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestOrOperator tests || (OR) operator
func TestOrOperator(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "OR first true",
			rules: []types.CrossParameterRule{
				{Rule: "admin_override=true || password_set=true", ErrorMsg: "Must have admin override or password"},
			},
			params: map[string]interface{}{
				"admin_override": true,
				"password_set":   false,
			},
			expected: true,
			errCount: 0,
		},
		{
			name: "OR second true",
			rules: []types.CrossParameterRule{
				{Rule: "admin_override=true || password_set=true", ErrorMsg: "Must have admin override or password"},
			},
			params: map[string]interface{}{
				"admin_override": false,
				"password_set":   true,
			},
			expected: true,
			errCount: 0,
		},
		{
			name: "OR both true",
			rules: []types.CrossParameterRule{
				{Rule: "admin_override=true || password_set=true", ErrorMsg: "Must have admin override or password"},
			},
			params: map[string]interface{}{
				"admin_override": true,
				"password_set":   true,
			},
			expected: true,
			errCount: 0,
		},
		{
			name: "OR both false",
			rules: []types.CrossParameterRule{
				{Rule: "admin_override=true || password_set=true", ErrorMsg: "Must have admin override or password"},
			},
			params: map[string]interface{}{
				"admin_override": false,
				"password_set":   false,
			},
			expected: false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestMixedOperators tests mixing && and || operators
func TestMixedOperators(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "AND precedence over OR",
			rules: []types.CrossParameterRule{
				{Rule: "mode=production && replicas=3 || mode=test", ErrorMsg: "invalid mode/replica combo"},
			},
			params: map[string]interface{}{
				"mode":     "test",
				"replicas": 1,
			},
			expected: true,
			errCount: 0,
		},
		{
			name: "Complex expression",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80 || service=https && port=443", ErrorMsg: "invalid service/port"},
			},
			params: map[string]interface{}{
				"service": "https",
				"port":    443,
			},
			expected: true,
			errCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestMissingParameter tests behavior when parameter is missing
func TestMissingParameter(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "missing parameter in rule",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80", ErrorMsg: "invalid service/port"},
			},
			params: map[string]interface{}{
				"service": "http",
				// port is missing
			},
			expected: false,
			errCount: 1,
		},
		{
			name: "both parameters missing",
			rules: []types.CrossParameterRule{
				{Rule: "service=http && port=80", ErrorMsg: "invalid service/port"},
			},
			params:   map[string]interface{}{},
			expected: false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestEmptyRules tests behavior with no rules
func TestEmptyRules(t *testing.T) {
	validator := NewCrossParameterValidator([]types.CrossParameterRule{})
	result := validator.ValidateCrossParameters(map[string]interface{}{"test": "value"})

	if !result.Valid {
		t.Errorf("expected valid=true for empty rules, got false")
	}

	if len(result.CrossParamErrors) != 0 {
		t.Errorf("expected 0 errors for empty rules, got %d", len(result.CrossParamErrors))
	}
}

// TestMultipleRules tests multiple rules together
func TestMultipleRules(t *testing.T) {
	rules := []types.CrossParameterRule{
		{Rule: "service=https || service=http", ErrorMsg: "service must be http or https"},
		{Rule: "service=https && port=443", ErrorMsg: "HTTPS must use port 443"},
		{Rule: "enable_ssl=true && ssl_cert!=/notset", ErrorMsg: "SSL cert required when SSL enabled"},
	}

	params := map[string]interface{}{
		"service":    "https",
		"port":       443,
		"enable_ssl": true,
		"ssl_cert":   "/path/to/cert",
	}

	validator := NewCrossParameterValidator(rules)
	result := validator.ValidateCrossParameters(params)

	if !result.Valid {
		t.Errorf("expected valid=true, got false")
	}

	if len(result.CrossParamErrors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.CrossParamErrors))
	}
}

// TestTypeCoercion tests type coercion in comparisons
func TestTypeCoercion(t *testing.T) {
	tests := []struct {
		name     string
		rules    []types.CrossParameterRule
		params   map[string]interface{}
		expected bool
		errCount int
	}{
		{
			name: "float to int coercion (whole number)",
			rules: []types.CrossParameterRule{
				{Rule: "port=8080", ErrorMsg: "port must be 8080"},
			},
			params:   map[string]interface{}{"port": 8080.0},
			expected: true,
			errCount: 0,
		},
		{
			name: "string to int coercion",
			rules: []types.CrossParameterRule{
				{Rule: "port=8080", ErrorMsg: "port must be 8080"},
			},
			params:   map[string]interface{}{"port": "8080"},
			expected: true,
			errCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCrossParameterValidator(tt.rules)
			result := validator.ValidateCrossParameters(tt.params)

			if result.Valid != tt.expected {
				t.Errorf("expected valid=%v, got %v", tt.expected, result.Valid)
			}

			if len(result.CrossParamErrors) != tt.errCount {
				t.Errorf("expected %d errors, got %d", tt.errCount, len(result.CrossParamErrors))
			}
		})
	}
}

// TestValidateRuleExpression tests rule expression validation
func TestValidateRuleExpression(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		shouldErr bool
	}{
		{
			name:      "valid equality",
			expr:      "port=8080",
			shouldErr: false,
		},
		{
			name:      "valid AND",
			expr:      "service=http && port=80",
			shouldErr: false,
		},
		{
			name:      "valid OR",
			expr:      "mode=test || mode=prod",
			shouldErr: false,
		},
		{
			name:      "empty expression",
			expr:      "",
			shouldErr: true,
		},
		{
			name:      "unmatched parens",
			expr:      "port=(8080",
			shouldErr: true,
		},
		{
			name:      "no operator",
			expr:      "port 8080",
			shouldErr: true,
		},
		{
			name:      "valid comparison operators",
			expr:      "port>=8080",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRuleExpression(tt.expr)

			if tt.shouldErr && err == nil {
				t.Errorf("expected error, got nil")
			}

			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
