package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// PolicyEnvVar names the environment variable that points to a policy file
const PolicyEnvVar = "ONIGIRAZU_SECURITY_POLICY"

// PolicySearchPaths returns the files checked, in order, when no policy is given explicitly
func PolicySearchPaths() []string {
	paths := []string{"security-policy.json"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".onigirazu", "security-policy.json"))
	}
	return append(paths, "/etc/onigirazu/security-policy.json")
}

// LoadPolicy resolves and loads the security policy. An explicit path wins, then
// $ONIGIRAZU_SECURITY_POLICY, then the first existing file from PolicySearchPaths.
// Without a policy file the permissive DefaultSecurityConfig is returned and the
// source is empty.
func LoadPolicy(explicit string) (SecurityConfig, string, error) {
	if explicit == "" {
		explicit = os.Getenv(PolicyEnvVar)
	}
	if explicit != "" {
		cfg, err := LoadPolicyFile(explicit)
		return cfg, explicit, err
	}
	for _, path := range PolicySearchPaths() {
		if _, err := os.Stat(path); err == nil {
			cfg, err := LoadPolicyFile(path)
			return cfg, path, err
		}
	}
	return DefaultSecurityConfig(), "", nil
}

// LoadPolicyFile reads a JSON policy. Settings that are left out keep the permissive
// defaults. Keys starting with "_" are comments; any other unknown key is an error.
// max_timeout accepts a duration string ("30m") or nanoseconds.
func LoadPolicyFile(path string) (SecurityConfig, error) {
	cfg := DefaultSecurityConfig()

	data, err := os.ReadFile(path) // #nosec G304 -- policy path is chosen by the operator
	if err != nil {
		return cfg, fmt.Errorf("failed to read security policy: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg, fmt.Errorf("invalid security policy %s: %w", path, err)
	}

	known := policyKeys()
	var unknown []string
	for key := range raw {
		if strings.HasPrefix(key, "_") {
			delete(raw, key)
			continue
		}
		if !known[key] || unsupportedPolicyKeys[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return cfg, fmt.Errorf("invalid security policy %s: unknown or unsupported keys: %s", path, strings.Join(unknown, ", "))
	}

	if v, ok := raw["max_timeout"]; ok && bytes.HasPrefix(bytes.TrimSpace(v), []byte(`"`)) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return cfg, fmt.Errorf("invalid security policy %s: max_timeout: %w", path, err)
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return cfg, fmt.Errorf("invalid security policy %s: max_timeout: %w", path, err)
		}
		raw["max_timeout"] = json.RawMessage(fmt.Sprintf("%d", d.Nanoseconds()))
	}

	cleaned, err := json.Marshal(raw)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return cfg, fmt.Errorf("invalid security policy %s: %w", path, err)
	}
	return cfg, nil
}

// unsupportedPolicyKeys exist in SecurityConfig but are not enforced, so a policy
// must not rely on them
var unsupportedPolicyKeys = map[string]bool{
	"require_encryption":   true,
	"required_permissions": true,
	"audit_enabled":        true,
	"log_level":            true,
}

// policyKeys returns the JSON keys of SecurityConfig
func policyKeys() map[string]bool {
	keys := make(map[string]bool)
	t := reflect.TypeOf(SecurityConfig{})
	for i := 0; i < t.NumField(); i++ {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}
