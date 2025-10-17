package validator

import (
	"testing"
)

func TestEqualityCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "String equality - true",
			condition: "environment=production",
			parameters: map[string]interface{}{
				"environment": "production",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "String equality - false",
			condition: "environment=production",
			parameters: map[string]interface{}{
				"environment": "staging",
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "Boolean equality - true",
			condition: "ssl_enabled=true",
			parameters: map[string]interface{}{
				"ssl_enabled": true,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Boolean string equality - true",
			condition: "ssl_enabled=true",
			parameters: map[string]interface{}{
				"ssl_enabled": "true",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer equality - true",
			condition: "port=8080",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer string equality - true",
			condition: "port=8080",
			parameters: map[string]interface{}{
				"port": "8080",
			},
			expected:  true,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInequalityCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "String inequality - true",
			condition: "environment!=production",
			parameters: map[string]interface{}{
				"environment": "staging",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "String inequality - false",
			condition: "environment!=production",
			parameters: map[string]interface{}{
				"environment": "production",
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "Undefined parameter inequality - true",
			condition: "missing_param!=null",
			parameters: map[string]interface{}{
				"other": "value",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer inequality - true",
			condition: "port!=8080",
			parameters: map[string]interface{}{
				"port": 8081,
			},
			expected:  true,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGreaterThanCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "Integer greater - true",
			condition: "port>1024",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer greater - false",
			condition: "port>8080",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "Float greater - true",
			condition: "threshold>0.5",
			parameters: map[string]interface{}{
				"threshold": 0.75,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "String as number greater - true",
			condition: "replicas>2",
			parameters: map[string]interface{}{
				"replicas": "5",
			},
			expected:  true,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLessThanCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "Integer less - true",
			condition: "port<10000",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer less - false",
			condition: "port<8080",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  false,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGreaterEqualCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "Integer >= - true (greater)",
			condition: "port>=1024",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer >= - true (equal)",
			condition: "port>=8080",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer >= - false",
			condition: "port>=9000",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  false,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLessEqualCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "Integer <= - true (less)",
			condition: "port<=10000",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer <= - true (equal)",
			condition: "port<=8080",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Integer <= - false",
			condition: "port<=5000",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  false,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAndOperatorCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "AND - both true",
			condition: "enable_auth=true && auth_type=ldap",
			parameters: map[string]interface{}{
				"enable_auth": true,
				"auth_type":   "ldap",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "AND - first false",
			condition: "enable_auth=true && auth_type=ldap",
			parameters: map[string]interface{}{
				"enable_auth": false,
				"auth_type":   "ldap",
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "AND - second false",
			condition: "enable_auth=true && auth_type=ldap",
			parameters: map[string]interface{}{
				"enable_auth": true,
				"auth_type":   "oauth2",
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "AND - both false",
			condition: "enable_auth=true && auth_type=ldap",
			parameters: map[string]interface{}{
				"enable_auth": false,
				"auth_type":   "oauth2",
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "AND with numeric comparisons",
			condition: "port>=8000 && port<=9000",
			parameters: map[string]interface{}{
				"port": 8080,
			},
			expected:  true,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOrOperatorCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "OR - both true",
			condition: "environment=production || environment=staging",
			parameters: map[string]interface{}{
				"environment": "production",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "OR - first true",
			condition: "environment=production || environment=staging",
			parameters: map[string]interface{}{
				"environment": "production",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "OR - second true",
			condition: "environment=production || environment=staging",
			parameters: map[string]interface{}{
				"environment": "staging",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "OR - both false",
			condition: "environment=production || environment=staging",
			parameters: map[string]interface{}{
				"environment": "development",
			},
			expected:  false,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMixedOperatorsCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "AND has higher precedence than OR",
			condition: "a=1 || b=2 && c=3",
			parameters: map[string]interface{}{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "AND then OR evaluation",
			condition: "environment=production && enable_auth=true",
			parameters: map[string]interface{}{
				"environment": "production",
				"enable_auth": true,
			},
			expected:  true,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMissingParameterCondition(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:      "Missing parameter with equality",
			condition: "missing_param=value",
			parameters: map[string]interface{}{
				"other": "value",
			},
			expected:  false,
			shouldErr: false,
		},
		{
			name:      "Missing parameter with inequality",
			condition: "missing_param!=value",
			parameters: map[string]interface{}{
				"other": "value",
			},
			expected:  true,
			shouldErr: false,
		},
		{
			name:      "Missing parameter with comparison",
			condition: "missing_param>10",
			parameters: map[string]interface{}{
				"other": "value",
			},
			expected:  false,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
				return
			}

			if !tt.shouldErr && result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEmptyAndInvalidConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		shouldErr bool
	}{
		{
			name:      "Empty condition",
			condition: "",
			shouldErr: true,
		},
		{
			name:      "Whitespace only condition",
			condition: "   ",
			shouldErr: true,
		},
		{
			name:      "Invalid operator placement",
			condition: "&&",
			shouldErr: true,
		},
		{
			name:      "No valid operator",
			condition: "parameter",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(map[string]interface{}{
				"parameter": "value",
			})
			_, err := evaluator.EvaluateCondition(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("EvaluateCondition() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
		})
	}
}

func TestRealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		condition   string
		parameters  map[string]interface{}
		expected    bool
		description string
	}{
		{
			name:      "API key required when auth enabled",
			condition: "enable_auth=true",
			parameters: map[string]interface{}{
				"enable_auth": true,
			},
			expected:    true,
			description: "Check if auth is enabled",
		},
		{
			name:      "Backup schedule when backup enabled",
			condition: "backup_enabled=true && backup_mode=full",
			parameters: map[string]interface{}{
				"backup_enabled": true,
				"backup_mode":    "full",
			},
			expected:    true,
			description: "Both backup enabled and mode is full",
		},
		{
			name:      "Replica count for cluster mode",
			condition: "cluster_mode=true && replicas>=3",
			parameters: map[string]interface{}{
				"cluster_mode": true,
				"replicas":     5,
			},
			expected:    true,
			description: "Cluster mode with sufficient replicas",
		},
		{
			name:      "Database port validation",
			condition: "db_type=postgres && db_port=5432 || db_type=mysql && db_port=3306",
			parameters: map[string]interface{}{
				"db_type": "postgres",
				"db_port": 5432,
			},
			expected:    true,
			description: "PostgreSQL on port 5432",
		},
		{
			name:      "SSL certificate when HTTPS",
			condition: "protocol=https && ssl_enabled=true",
			parameters: map[string]interface{}{
				"protocol":    "https",
				"ssl_enabled": true,
			},
			expected:    true,
			description: "HTTPS with SSL enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if err != nil {
				t.Errorf("EvaluateCondition() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v (scenario: %s)", result, tt.expected, tt.description)
			}
		})
	}
}

func TestValidateConditionExpression(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		shouldErr bool
	}{
		{
			name:      "Valid equality",
			condition: "parameter=value",
			shouldErr: false,
		},
		{
			name:      "Valid with spaces",
			condition: "parameter = value",
			shouldErr: false,
		},
		{
			name:      "Valid AND expression",
			condition: "param1=value1 && param2=value2",
			shouldErr: false,
		},
		{
			name:      "Valid OR expression",
			condition: "param1=value1 || param2=value2",
			shouldErr: false,
		},
		{
			name:      "Invalid - empty",
			condition: "",
			shouldErr: true,
		},
		{
			name:      "Invalid - operator at start",
			condition: "&&param=value",
			shouldErr: true,
		},
		{
			name:      "Invalid - operator at end",
			condition: "param=value&&",
			shouldErr: true,
		},
		{
			name:      "Invalid - missing parameter",
			condition: "=value",
			shouldErr: true,
		},
		{
			name:      "Invalid - missing value",
			condition: "param=",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConditionExpression(tt.condition)

			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateConditionExpression() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
		})
	}
}

func TestTypeCoercionInConditions(t *testing.T) {
	tests := []struct {
		name       string
		condition  string
		parameters map[string]interface{}
		expected   bool
	}{
		{
			name:      "Float to bool coercion",
			condition: "is_active=true",
			parameters: map[string]interface{}{
				"is_active": "true",
			},
			expected: true,
		},
		{
			name:      "Integer string comparison",
			condition: "count>5",
			parameters: map[string]interface{}{
				"count": "10",
			},
			expected: true,
		},
		{
			name:      "Mixed int and float",
			condition: "value=3.0",
			parameters: map[string]interface{}{
				"value": 3,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(tt.parameters)
			result, err := evaluator.EvaluateCondition(tt.condition)

			if err != nil {
				t.Errorf("EvaluateCondition() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("EvaluateCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}
