package audit

import (
	"strings"
	"testing"
)

func TestFilterSensitiveVariables_BasicFiltering(t *testing.T) {
	vars := map[string]interface{}{
		"username": "admin",
		"password": "secretpass123",
		"api_key":  "key12345",
		"normal":   "value",
	}

	filtered := filterSensitiveVariables(vars)

	// Check that sensitive fields are redacted
	if filtered["password"] != "***REDACTED***" {
		t.Errorf("Expected password to be redacted, got %v", filtered["password"])
	}
	if filtered["api_key"] != "***REDACTED***" {
		t.Errorf("Expected api_key to be redacted, got %v", filtered["api_key"])
	}

	// Check that normal fields are preserved
	if filtered["username"] != "admin" {
		t.Errorf("Expected username to be preserved, got %v", filtered["username"])
	}
	if filtered["normal"] != "value" {
		t.Errorf("Expected normal to be preserved, got %v", filtered["normal"])
	}
}

func TestFilterSensitiveVariables_CaseInsensitive(t *testing.T) {
	vars := map[string]interface{}{
		"PASSWORD":    "secret",
		"Api_Key":     "key",
		"PRIVATE_KEY": "key123",
		"pwd":         "pass",
	}

	filtered := filterSensitiveVariables(vars)

	for key, val := range filtered {
		if val != "***REDACTED***" {
			t.Errorf("Expected %s to be redacted (case-insensitive), got %v", key, val)
		}
	}
}

func TestFilterSensitiveVariables_SubstringMatching(t *testing.T) {
	vars := map[string]interface{}{
		"db_password":          "dbpass",
		"user_token":           "token123",
		"secret_value":         "secret",
		"authorization_header": "Bearer xxx",
		"admin_credentials":    "creds",
	}

	filtered := filterSensitiveVariables(vars)

	expected := map[string]bool{
		"db_password":          true,
		"user_token":           true,
		"secret_value":         true,
		"authorization_header": true,
		"admin_credentials":    true,
	}

	for key, shouldRedact := range expected {
		val := filtered[key]
		isRedacted := val == "***REDACTED***"
		if isRedacted != shouldRedact {
			t.Errorf("Expected %s redacted=%v, got redacted=%v (value=%v)", key, shouldRedact, isRedacted, val)
		}
	}
}

func TestFilterSensitiveVariables_NestedMaps(t *testing.T) {
	vars := map[string]interface{}{
		"normal": "value",
		"config": map[string]interface{}{
			"db": map[string]interface{}{
				"host":     "localhost",
				"password": "dbpass",
			},
			"api": map[string]interface{}{
				"key":    "mykey",
				"secret": "mysecret",
			},
		},
	}

	filtered := filterSensitiveVariables(vars)

	// Check nested password
	if configVal, ok := filtered["config"].(map[string]interface{}); ok {
		if dbVal, ok := configVal["db"].(map[string]interface{}); ok {
			if dbVal["password"] != "***REDACTED***" {
				t.Errorf("Expected nested password to be redacted, got %v", dbVal["password"])
			}
			if dbVal["host"] != "localhost" {
				t.Errorf("Expected nested host to be preserved, got %v", dbVal["host"])
			}
		} else {
			t.Errorf("Expected config.db to be a map")
		}

		if apiVal, ok := configVal["api"].(map[string]interface{}); ok {
			if apiVal["secret"] != "***REDACTED***" {
				t.Errorf("Expected nested secret to be redacted, got %v", apiVal["secret"])
			}
		} else {
			t.Errorf("Expected config.api to be a map")
		}
	} else {
		t.Errorf("Expected config to be a map")
	}
}

func TestFilterSensitiveVariables_Lists(t *testing.T) {
	vars := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name":     "user1",
				"password": "pass1",
			},
			map[string]interface{}{
				"name":     "user2",
				"password": "pass2",
			},
		},
	}

	filtered := filterSensitiveVariables(vars)

	if users, ok := filtered["users"].([]interface{}); ok {
		if len(users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(users))
		}

		for i, user := range users {
			if userMap, ok := user.(map[string]interface{}); ok {
				if userMap["password"] != "***REDACTED***" {
					t.Errorf("Expected user %d password to be redacted, got %v", i, userMap["password"])
				}
				if userMap["name"] != strings.Replace([]string{"user1", "user2"}[i], "", "", -1) {
					t.Errorf("Expected user %d name to be preserved", i)
				}
			}
		}
	} else {
		t.Errorf("Expected users to be a list")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	patterns := []string{"password", "token", "secret", "api_key", "apikey"}

	tests := []struct {
		key         string
		shouldMatch bool
	}{
		{"password", true},
		{"db_password", true},
		{"PASSWORD", true},
		{"token", true},
		{"api_token", true},
		{"secret", true},
		{"api_key", true},
		{"apikey", true},
		{"username", false},
		{"hostname", false},
		{"key", false}, // Should not match "api_key" without context
	}

	for _, tt := range tests {
		result := isSensitiveKey(strings.ToLower(tt.key), patterns)
		if result != tt.shouldMatch {
			t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, result, tt.shouldMatch)
		}
	}
}

func TestFilterSensitiveList(t *testing.T) {
	patterns := []string{"password"}

	list := []interface{}{
		map[string]interface{}{
			"name":     "item1",
			"password": "secret1",
		},
		"string_item",
		123,
	}

	filtered := filterSensitiveList(list, patterns)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 items after filtering, got %d", len(filtered))
	}

	if item, ok := filtered[0].(map[string]interface{}); ok {
		if item["password"] != "***REDACTED***" {
			t.Errorf("Expected password in list item to be redacted")
		}
	}

	if item, ok := filtered[1].(string); ok {
		if item != "string_item" {
			t.Errorf("Expected string item to be preserved")
		}
	}
}
