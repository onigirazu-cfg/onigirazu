package modules

import (
	"reflect"
	"testing"
)

// TestConfigModule_GetName tests the GetName method
func TestConfigModule_GetName(t *testing.T) {
	module := NewConfigModule()

	name := module.GetName()
	if name != "config" {
		t.Errorf("Expected name 'config', got '%s'", name)
	}
}

// TestConfigModule_GetDescription tests the GetDescription method
func TestConfigModule_GetDescription(t *testing.T) {
	module := NewConfigModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	expectedDesc := "Manage configuration files with validation and backup"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

// TestNewConfigModule tests module creation
func TestNewConfigModule(t *testing.T) {
	module := NewConfigModule()

	if module == nil {
		t.Fatalf("Expected non-nil module")
	}

	if module.GetName() != "config" {
		t.Errorf("Expected module name 'config', got '%s'", module.GetName())
	}
}

// TestConfigModule_DeepCopy tests the deepCopy helper function
func TestConfigModule_DeepCopy(t *testing.T) {
	module := NewConfigModule()

	tests := []struct {
		name     string
		original map[string]interface{}
	}{
		{
			name:     "empty map",
			original: map[string]interface{}{},
		},
		{
			name: "simple map",
			original: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		},
		{
			name: "nested map",
			original: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep value",
					},
				},
			},
		},
		{
			name: "mixed types",
			original: map[string]interface{}{
				"string": "text",
				"number": 123,
				"bool":   false,
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied := module.deepCopy(tt.original)

			// Verify the copy is equal
			if !reflect.DeepEqual(copied, tt.original) {
				t.Errorf("deepCopy() = %v, expected %v", copied, tt.original)
			}

			// Verify it's a deep copy (modifying copy doesn't affect original)
			if len(tt.original) > 0 {
				copied["new_key"] = "new_value"
				if _, exists := tt.original["new_key"]; exists {
					t.Errorf("Modifying copy affected original")
				}
			}
		})
	}
}

// TestConfigModule_GetNestedValue tests the getNestedValue helper function
func TestConfigModule_GetNestedValue(t *testing.T) {
	module := NewConfigModule()

	config := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep value",
			},
			"simple": "value",
		},
		"top": "top value",
	}

	tests := []struct {
		name     string
		key      string
		expected interface{}
	}{
		{
			name:     "top level key",
			key:      "top",
			expected: "top value",
		},
		{
			name:     "nested key - 2 levels",
			key:      "level1.simple",
			expected: "value",
		},
		{
			name:     "nested key - 3 levels",
			key:      "level1.level2.level3",
			expected: "deep value",
		},
		{
			name:     "non-existent key",
			key:      "nonexistent",
			expected: nil,
		},
		{
			name:     "non-existent nested key",
			key:      "level1.nonexistent",
			expected: nil,
		},
		{
			name:     "invalid path",
			key:      "top.invalid",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.getNestedValue(config, tt.key)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getNestedValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestConfigModule_DeleteNestedKey tests the deleteNestedKey helper function
func TestConfigModule_DeleteNestedKey(t *testing.T) {
	module := NewConfigModule()

	tests := []struct {
		name     string
		config   map[string]interface{}
		key      string
		expected bool
		verify   func(t *testing.T, config map[string]interface{})
	}{
		{
			name: "delete top level key",
			config: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			key:      "key1",
			expected: true,
			verify: func(t *testing.T, config map[string]interface{}) {
				if _, exists := config["key1"]; exists {
					t.Errorf("Key 'key1' should have been deleted")
				}
				if _, exists := config["key2"]; !exists {
					t.Errorf("Key 'key2' should still exist")
				}
			},
		},
		{
			name: "delete nested key",
			config: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "value",
					"other":  "keep",
				},
			},
			key:      "level1.level2",
			expected: true,
			verify: func(t *testing.T, config map[string]interface{}) {
				level1 := config["level1"].(map[string]interface{})
				if _, exists := level1["level2"]; exists {
					t.Errorf("Key 'level1.level2' should have been deleted")
				}
				if _, exists := level1["other"]; !exists {
					t.Errorf("Key 'level1.other' should still exist")
				}
			},
		},
		{
			name: "delete non-existent key",
			config: map[string]interface{}{
				"key1": "value1",
			},
			key:      "nonexistent",
			expected: false,
			verify: func(t *testing.T, config map[string]interface{}) {
				if len(config) != 1 {
					t.Errorf("Config should still have 1 key")
				}
			},
		},
		{
			name: "delete with invalid path",
			config: map[string]interface{}{
				"key1": "value1",
			},
			key:      "key1.invalid",
			expected: false,
			verify: func(t *testing.T, config map[string]interface{}) {
				if _, exists := config["key1"]; !exists {
					t.Errorf("Key 'key1' should still exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.deleteNestedKey(tt.config, tt.key)

			if result != tt.expected {
				t.Errorf("deleteNestedKey() = %v, expected %v", result, tt.expected)
			}

			if tt.verify != nil {
				tt.verify(t, tt.config)
			}
		})
	}
}

// TestConfigModule_MergeConfig tests the mergeConfig helper function
func TestConfigModule_MergeConfig(t *testing.T) {
	module := NewConfigModule()

	tests := []struct {
		name     string
		target   map[string]interface{}
		source   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:   "merge into empty target",
			target: map[string]interface{}{},
			source: map[string]interface{}{
				"key1": "value1",
			},
			expected: map[string]interface{}{
				"key1": "value1",
			},
		},
		{
			name: "merge non-overlapping keys",
			target: map[string]interface{}{
				"key1": "value1",
			},
			source: map[string]interface{}{
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "merge overlapping keys - source overwrites",
			target: map[string]interface{}{
				"key1": "old value",
			},
			source: map[string]interface{}{
				"key1": "new value",
			},
			expected: map[string]interface{}{
				"key1": "new value",
			},
		},
		{
			name: "merge nested maps",
			target: map[string]interface{}{
				"nested": map[string]interface{}{
					"key1": "value1",
				},
			},
			source: map[string]interface{}{
				"nested": map[string]interface{}{
					"key2": "value2",
				},
			},
			expected: map[string]interface{}{
				"nested": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
		{
			name: "merge deeply nested maps",
			target: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"key1": "value1",
					},
				},
			},
			source: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"key2": "value2",
					},
				},
			},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
		},
		{
			name: "merge replaces non-map with map",
			target: map[string]interface{}{
				"key1": "simple value",
			},
			source: map[string]interface{}{
				"key1": map[string]interface{}{
					"nested": "value",
				},
			},
			expected: map[string]interface{}{
				"key1": map[string]interface{}{
					"nested": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module.mergeConfig(tt.target, tt.source)

			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("mergeConfig() resulted in %v, expected %v", tt.target, tt.expected)
			}
		})
	}
}

// TestConfigModule_Validate tests the Validate method
func TestConfigModule_Validate(t *testing.T) {
	module := NewConfigModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid set action with path and values",
			args: map[string]interface{}{
				"path": "/etc/config.json",
				"values": map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "valid merge action with path and values",
			args: map[string]interface{}{
				"path":   "/etc/config.json",
				"action": "merge",
				"values": map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "valid restore action with path and backup_path",
			args: map[string]interface{}{
				"path":        "/etc/config.json",
				"action":      "restore",
				"backup_path": "/etc/config.json.backup",
			},
			wantErr: false,
		},
		{
			name: "valid get action with path only",
			args: map[string]interface{}{
				"path":   "/etc/config.json",
				"action": "get",
			},
			wantErr: false,
		},
		{
			name: "valid delete action with path only",
			args: map[string]interface{}{
				"path":   "/etc/config.json",
				"action": "delete",
			},
			wantErr: false,
		},
		{
			name: "default action (set) with path and values",
			args: map[string]interface{}{
				"path": "/etc/config.json",
				"values": map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
		{
			name:    "missing path",
			args:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "set action missing values",
			args: map[string]interface{}{
				"path":   "/etc/config.json",
				"action": "set",
			},
			wantErr: true,
			errMsg:  "values parameter is required for set action",
		},
		{
			name: "default action (set) missing values",
			args: map[string]interface{}{
				"path": "/etc/config.json",
			},
			wantErr: true,
			errMsg:  "values parameter is required for set action",
		},
		{
			name: "merge action missing values",
			args: map[string]interface{}{
				"path":   "/etc/config.json",
				"action": "merge",
			},
			wantErr: true,
			errMsg:  "values parameter is required for merge action",
		},
		{
			name: "restore action missing backup_path",
			args: map[string]interface{}{
				"path":   "/etc/config.json",
				"action": "restore",
			},
			wantErr: true,
			errMsg:  "backup_path parameter is required for restore action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}
