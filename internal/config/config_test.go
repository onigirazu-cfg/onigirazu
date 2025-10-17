package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Test default values
	if config.GetMaxConcurrency() != 10 {
		t.Errorf("Expected MaxConcurrency to be 10, got %d", config.GetMaxConcurrency())
	}

	if config.GetDefaultTimeout() != 30*time.Second {
		t.Errorf("Expected DefaultTimeout to be 30s, got %v", config.GetDefaultTimeout())
	}

	if config.GetRetryAttempts() != 3 {
		t.Errorf("Expected RetryAttempts to be 3, got %d", config.GetRetryAttempts())
	}

	if config.GetRetryDelay() != 5*time.Second {
		t.Errorf("Expected RetryDelay to be 5s, got %v", config.GetRetryDelay())
	}

	if config.GetLogLevel() != "info" {
		t.Errorf("Expected LogLevel to be 'info', got %s", config.GetLogLevel())
	}

	if config.GetLogFormat() != "text" {
		t.Errorf("Expected LogFormat to be 'text', got %s", config.GetLogFormat())
	}

	if !config.IsShellCommandsAllowed() {
		t.Error("Expected ShellCommandsAllowed to be true")
	}

	if !config.IsCachingEnabled() {
		t.Error("Expected CachingEnabled to be true")
	}

	if config.GetCacheTTL() != 5*time.Minute {
		t.Errorf("Expected CacheTTL to be 5m, got %v", config.GetCacheTTL())
	}

	if !config.IsChecksumEnabled() {
		t.Error("Expected ChecksumEnabled to be true")
	}
}

func TestConfigSetters(t *testing.T) {
	config := DefaultConfig()

	// Test setting values
	config.MaxConcurrency = 20
	if config.GetMaxConcurrency() != 20 {
		t.Errorf("Expected MaxConcurrency to be 20, got %d", config.GetMaxConcurrency())
	}

	config.DefaultTimeout = 60 * time.Second
	if config.GetDefaultTimeout() != 60*time.Second {
		t.Errorf("Expected DefaultTimeout to be 60s, got %v", config.GetDefaultTimeout())
	}

	config.LogLevel = "debug"
	if config.GetLogLevel() != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got %s", config.GetLogLevel())
	}

	config.AllowShellCommands = false
	if config.IsShellCommandsAllowed() {
		t.Error("Expected AllowShellCommands to be false")
	}
}

func TestBlockedCommands(t *testing.T) {
	config := DefaultConfig()

	blockedCommands := config.GetBlockedCommands()
	expectedCommands := []string{"rm -rf", "format", "mkfs", "dd if=", ":(){ :|:& };:"}

	if len(blockedCommands) != len(expectedCommands) {
		t.Errorf("Expected %d blocked commands, got %d", len(expectedCommands), len(blockedCommands))
	}

	for i, expected := range expectedCommands {
		if i >= len(blockedCommands) || blockedCommands[i] != expected {
			t.Errorf("Expected blocked command %d to be '%s', got '%s'", i, expected, blockedCommands[i])
		}
	}
}

// ============================================================================
// HELPER FUNCTION TESTS
// ============================================================================

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "uses default when env not set",
			key:          "TEST_GET_ENV_STRING_NOTSET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "uses env value when set",
			key:          "TEST_GET_ENV_STRING_SET",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "uses default for empty env",
			key:          "TEST_GET_ENV_STRING_EMPTY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvString(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "uses default when env not set",
			key:          "TEST_GET_ENV_INT_NOTSET",
			defaultValue: 42,
			envValue:     "",
			expected:     42,
		},
		{
			name:         "parses valid int",
			key:          "TEST_GET_ENV_INT_VALID",
			defaultValue: 42,
			envValue:     "100",
			expected:     100,
		},
		{
			name:         "uses default for invalid int",
			key:          "TEST_GET_ENV_INT_INVALID",
			defaultValue: 42,
			envValue:     "not-a-number",
			expected:     42,
		},
		{
			name:         "handles negative numbers",
			key:          "TEST_GET_ENV_INT_NEGATIVE",
			defaultValue: 42,
			envValue:     "-5",
			expected:     -5,
		},
		{
			name:         "handles zero",
			key:          "TEST_GET_ENV_INT_ZERO",
			defaultValue: 42,
			envValue:     "0",
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "uses default when env not set",
			key:          "TEST_GET_ENV_BOOL_NOTSET",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
		{
			name:         "parses true",
			key:          "TEST_GET_ENV_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "parses false",
			key:          "TEST_GET_ENV_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "parses 1 as true",
			key:          "TEST_GET_ENV_BOOL_1",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "parses 0 as false",
			key:          "TEST_GET_ENV_BOOL_0",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "uses default for invalid bool",
			key:          "TEST_GET_ENV_BOOL_INVALID",
			defaultValue: false,
			envValue:     "maybe",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "uses default when env not set",
			key:          "TEST_GET_ENV_DUR_NOTSET",
			defaultValue: 30 * time.Second,
			envValue:     "",
			expected:     30 * time.Second,
		},
		{
			name:         "parses valid duration",
			key:          "TEST_GET_ENV_DUR_VALID",
			defaultValue: 30 * time.Second,
			envValue:     "1m",
			expected:     1 * time.Minute,
		},
		{
			name:         "parses seconds",
			key:          "TEST_GET_ENV_DUR_SECONDS",
			defaultValue: 30 * time.Second,
			envValue:     "45s",
			expected:     45 * time.Second,
		},
		{
			name:         "uses default for invalid duration",
			key:          "TEST_GET_ENV_DUR_INVALID",
			defaultValue: 30 * time.Second,
			envValue:     "not-a-duration",
			expected:     30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvDuration(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// CONFIG VALIDATION TESTS
// ============================================================================

func TestValidate(t *testing.T) {
	tests := []struct {
		name            string
		setupConfig     func(*Config)
		expectNoError   bool
		expectedMinCon  int
		expectedMinTime time.Duration
	}{
		{
			name: "fixes invalid MaxConcurrency",
			setupConfig: func(c *Config) {
				c.MaxConcurrency = -5
			},
			expectNoError:   true,
			expectedMinCon:  1,
			expectedMinTime: 0,
		},
		{
			name: "fixes invalid DefaultTimeout",
			setupConfig: func(c *Config) {
				c.DefaultTimeout = -10 * time.Second
			},
			expectNoError:   true,
			expectedMinCon:  0,
			expectedMinTime: 30 * time.Second,
		},
		{
			name: "allows valid config",
			setupConfig: func(c *Config) {
				c.MaxConcurrency = 5
				c.DefaultTimeout = 20 * time.Second
			},
			expectNoError:   true,
			expectedMinCon:  5,
			expectedMinTime: 20 * time.Second,
		},
		{
			name: "fixes negative RetryAttempts",
			setupConfig: func(c *Config) {
				c.RetryAttempts = -1
			},
			expectNoError:  true,
			expectedMinCon: 0,
		},
		{
			name: "fixes negative RetryDelay",
			setupConfig: func(c *Config) {
				c.RetryDelay = -5 * time.Second
			},
			expectNoError:  true,
			expectedMinCon: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setupConfig(cfg)

			err := cfg.Validate()

			if tt.expectNoError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.expectedMinCon > 0 && cfg.MaxConcurrency != tt.expectedMinCon {
				t.Errorf("Expected MaxConcurrency %d, got %d", tt.expectedMinCon, cfg.MaxConcurrency)
			}

			if tt.expectedMinTime > 0 && cfg.DefaultTimeout != tt.expectedMinTime {
				t.Errorf("Expected DefaultTimeout %v, got %v", tt.expectedMinTime, cfg.DefaultTimeout)
			}
		})
	}
}

// ============================================================================
// GETTER METHOD TESTS - PERFORMANCE
// ============================================================================

func TestPerformanceGetters(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name        string
		testFunc    func() bool
		expectedVal bool
	}{
		{
			name:        "IsParallelEnabled",
			testFunc:    cfg.IsParallelEnabled,
			expectedVal: false,
		},
		{
			name:        "IsDryRun",
			testFunc:    cfg.IsDryRun,
			expectedVal: false,
		},
		{
			name:        "IsCheckMode",
			testFunc:    cfg.IsCheckMode,
			expectedVal: false,
		},
		{
			name:        "GetDryRun",
			testFunc:    cfg.GetDryRun,
			expectedVal: false,
		},
		{
			name:        "GetCheckMode",
			testFunc:    cfg.GetCheckMode,
			expectedVal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.testFunc(); result != tt.expectedVal {
				t.Errorf("Expected %v, got %v", tt.expectedVal, result)
			}
		})
	}

	// Test strategy getter
	if cfg.GetParallelStrategy() != "linear" {
		t.Errorf("Expected parallel strategy 'linear', got %s", cfg.GetParallelStrategy())
	}
}

// ============================================================================
// GETTER METHOD TESTS - UI/UX
// ============================================================================

func TestUIUXGetters(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"IsVerbose", func() interface{} { return cfg.IsVerbose() }, false},
		{"ShouldShowDiff", func() interface{} { return cfg.ShouldShowDiff() }, false},
		{"IsColorOutputEnabled", func() interface{} { return cfg.IsColorOutputEnabled() }, true},
		{"IsProgressBarEnabled", func() interface{} { return cfg.IsProgressBarEnabled() }, true},
		{"IsInteractiveModeEnabled", func() interface{} { return cfg.IsInteractiveModeEnabled() }, false},
		{"GetOutputFormat", func() interface{} { return cfg.GetOutputFormat() }, "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// GETTER METHOD TESTS - MONITORING
// ============================================================================

func TestMonitoringGetters(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"IsMetricsEnabled", func() interface{} { return cfg.IsMetricsEnabled() }, false},
		{"GetMetricsPort", func() interface{} { return cfg.GetMetricsPort() }, 9090},
		{"GetMetricsPath", func() interface{} { return cfg.GetMetricsPath() }, "/metrics"},
		{"IsProfilingEnabled", func() interface{} { return cfg.IsProfilingEnabled() }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// GETTER METHOD TESTS - SSH/CONNECTION
// ============================================================================

func TestSSHConnectionGetters(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"GetSSHTimeout", func() interface{} { return cfg.GetSSHTimeout() }, 30 * time.Second},
		{"GetSSHKeepAlive", func() interface{} { return cfg.GetSSHKeepAlive() }, 60 * time.Second},
		{"GetSSHMaxSessions", func() interface{} { return cfg.GetSSHMaxSessions() }, 10},
		{"IsConnectionReuseEnabled", func() interface{} { return cfg.IsConnectionReuseEnabled() }, true},
		{"IsSSHStrictHostKeyEnabled", func() interface{} { return cfg.IsSSHStrictHostKeyEnabled() }, false},
		{"GetSSHKnownHostsFile", func() interface{} { return cfg.GetSSHKnownHostsFile() }, ""},
		{"GetDefaultInsecureIgnoreHostKey", func() interface{} { return cfg.GetDefaultInsecureIgnoreHostKey() }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// GETTER METHOD TESTS - VAULT
// ============================================================================

func TestVaultGetters(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"IsVaultEnabled", func() interface{} { return cfg.IsVaultEnabled() }, false},
		{"GetVaultAddress", func() interface{} { return cfg.GetVaultAddress() }, ""},
		{"GetVaultToken", func() interface{} { return cfg.GetVaultToken() }, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// GETTER METHOD TESTS - MODULE SYNTAX
// ============================================================================

func TestModuleSyntaxGetters(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"GetPreferredModuleSyntax", func() interface{} { return cfg.GetPreferredModuleSyntax() }, "nested"},
		{"IsModuleSyntaxEnforced", func() interface{} { return cfg.IsModuleSyntaxEnforced() }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// GETTER METHOD TESTS - COMBINED
// ============================================================================

func TestFilePathGetters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StateFile = "/path/to/state"
	cfg.ConfigFile = "/path/to/config.yml"

	if cfg.GetStateFile() != "/path/to/state" {
		t.Errorf("Expected /path/to/state, got %s", cfg.GetStateFile())
	}

	if cfg.ConfigFile != "/path/to/config.yml" {
		t.Errorf("Expected /path/to/config.yml, got %s", cfg.ConfigFile)
	}
}

// ============================================================================
// LOAD CONFIG TESTS
// ============================================================================

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Should return same as DefaultConfig
	if cfg.GetMaxConcurrency() != 10 {
		t.Errorf("Expected MaxConcurrency to be 10, got %d", cfg.GetMaxConcurrency())
	}
}

func TestLoadConfigEmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Errorf("Expected no error for empty path, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config to be returned, got nil")
	}

	// Should return default config
	if cfg.GetMaxConcurrency() != 10 {
		t.Errorf("Expected default MaxConcurrency to be 10, got %d", cfg.GetMaxConcurrency())
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.yml")
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected default config to be returned, got nil")
	}
}

func TestLoadConfigValidFile(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write a valid YAML config
	configContent := `max_concurrency: 20
default_timeout: 60s
retry_attempts: 5
log_level: debug
enable_caching: false
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config to be returned, got nil")
	}

	if cfg.MaxConcurrency != 20 {
		t.Errorf("Expected MaxConcurrency to be 20, got %d", cfg.MaxConcurrency)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got %s", cfg.LogLevel)
	}

	if cfg.EnableCaching != false {
		t.Errorf("Expected EnableCaching to be false, got %v", cfg.EnableCaching)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-invalid-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid YAML
	if _, err := tmpFile.WriteString("invalid: yaml: content: here:"); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	if cfg != nil {
		t.Errorf("Expected nil config on error, got %v", cfg)
	}
}

func TestLoadConfigUnreadableFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-unreadable-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tmpFile.WriteString("max_concurrency: 20\n")
	tmpFile.Close()

	// Make file unreadable (Unix-only)
	if err := os.Chmod(tmpFile.Name(), 0000); err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer os.Chmod(tmpFile.Name(), 0644)

	cfg, err := LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for unreadable file, got nil")
	}

	if cfg != nil {
		t.Errorf("Expected nil config on error, got %v", cfg)
	}
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("ONIGIRAZU_MAX_CONCURRENCY", "50")
	os.Setenv("ONIGIRAZU_LOG_LEVEL", "debug")
	defer os.Unsetenv("ONIGIRAZU_MAX_CONCURRENCY")
	defer os.Unsetenv("ONIGIRAZU_LOG_LEVEL")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cfg.MaxConcurrency != 50 {
		t.Errorf("Expected MaxConcurrency from env to be 50, got %d", cfg.MaxConcurrency)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel from env to be debug, got %s", cfg.LogLevel)
	}
}
